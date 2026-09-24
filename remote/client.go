package remote

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	todocontract "github.com/domainry/domainry-todo-sdk/contract"
	"github.com/domainry/domainry-todo-sdk/saashost"
	toolsdk "github.com/domainry/domainry-tools-sdk"
)

type client struct {
	base                        *url.URL
	runtimeID, token            string
	http                        *http.Client
	requestLimit, responseLimit int64
	authorizeSource             SourceAuthorizer
}

func newClient(config Config, runtimeID string) (*client, error) {
	config = normalizeConfig(config)
	base, err := url.Parse(strings.TrimSpace(config.Endpoint))
	if err != nil || base == nil || (base.Scheme != "http" && base.Scheme != "https") || strings.TrimSpace(base.Host) == "" || base.RawQuery != "" || base.Fragment != "" {
		return nil, fmt.Errorf("Todo SaaS endpoint is invalid")
	}
	base.Path = strings.TrimRight(base.Path, "/")
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: config.RequestTimeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	return &client{base: base, runtimeID: runtimeID, token: strings.TrimSpace(config.ServiceAccessToken), http: httpClient, requestLimit: config.MaxRequestBytes, responseLimit: config.MaxResponseBytes, authorizeSource: config.AuthorizeSource}, nil
}
func (c *client) request(ctx context.Context, method, path string, input, output any) error {
	if ctx == nil {
		return remoteError("unavailable", "todo.context_required", false, nil)
	}
	var body io.Reader
	if input != nil {
		raw, err := json.Marshal(input)
		if err != nil {
			return remoteError("bad_request", "todo.request_invalid", false, err)
		}
		if int64(len(raw)) > c.requestLimit {
			return remoteError("bad_request", "todo.request_too_large", false, nil)
		}
		body = bytes.NewReader(raw)
	}
	endpoint := *c.base
	endpoint.Path = strings.TrimRight(c.base.Path, "/") + "/" + strings.TrimLeft(path, "/")
	request, err := http.NewRequestWithContext(ctx, method, endpoint.String(), body)
	if err != nil {
		return remoteError("unavailable", "todo.request_invalid", false, err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "domainry-todo-sdk-go")
	request.Header.Set(saashost.RuntimeIDHeader, c.runtimeID)
	if input != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		request.Header.Set("Authorization", "Bearer "+c.token)
	}
	response, err := c.http.Do(request)
	if err != nil {
		return remoteError("unavailable", "todo.remote_unavailable", true, err)
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(response.Body, c.responseLimit+1))
	if err != nil {
		return remoteError("unavailable", "todo.response_read_failed", true, err)
	}
	if int64(len(raw)) > c.responseLimit {
		return remoteError("unavailable", "todo.response_too_large", false, nil)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		var payload todocontract.SaaSResponse
		if json.Unmarshal(raw, &payload) == nil && payload.Error != nil {
			return codedError(payload.Error)
		}
		return remoteError(statusClass(response.StatusCode), "todo.remote_request_failed", response.StatusCode >= 500, nil)
	}
	if output == nil {
		return nil
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(output) != nil || decoder.Decode(&struct{}{}) != io.EOF {
		return remoteError("unavailable", "todo.response_invalid", false, nil)
	}
	return nil
}
func (c *client) invoke(ctx context.Context, operation string, input, output any) error {
	raw, err := json.Marshal(input)
	if err != nil {
		return remoteError("bad_request", "todo.request_invalid", false, err)
	}
	request := todocontract.SaaSRequest{Operation: operation, Input: raw}
	for attempt := 0; attempt < 4; attempt++ {
		var response todocontract.SaaSResponse
		if err = c.request(ctx, http.MethodPost, todocontract.SaaSInvokePath, request, &response); err != nil {
			return err
		}
		if response.Challenge != nil {
			if response.Challenge.Kind != "source.authorize" || c.authorizeSource == nil {
				return remoteError("unavailable", "todo.source_authorizer_required", false, nil)
			}
			var challenge struct {
				Reference string            `json:"reference"`
				Authority toolsdk.Authority `json:"authority"`
			}
			if json.Unmarshal(response.Challenge.Input, &challenge) != nil {
				return remoteError("unavailable", "todo.challenge_invalid", false, nil)
			}
			if err = c.authorizeSource(ctx, challenge.Reference, challenge.Authority); err != nil {
				return err
			}
			request.Grants = append(request.Grants, todocontract.SaaSGrant{Token: response.Challenge.Token, Result: json.RawMessage(`{}`)})
			continue
		}
		if response.Error != nil {
			return codedError(response.Error)
		}
		if output == nil || len(response.Result) == 0 {
			return nil
		}
		decoder := json.NewDecoder(bytes.NewReader(response.Result))
		decoder.DisallowUnknownFields()
		if decoder.Decode(output) != nil {
			return remoteError("unavailable", "todo.response_invalid", false, nil)
		}
		return nil
	}
	return remoteError("unavailable", "todo.challenge_limit_exceeded", false, nil)
}
func codedError(value *todocontract.SaaSError) error {
	return &toolsdk.Error{Class: safeClass(value.Class), Code: safeCode(value.Code), Message: safeMessage(value.Message), Retryable: value.Retryable}
}
func remoteError(class, code string, retryable bool, cause error) error {
	return &toolsdk.Error{Class: class, Code: code, Retryable: retryable, Cause: cause}
}
func safeClass(v string) string {
	switch strings.TrimSpace(v) {
	case "bad_request", "forbidden", "not_found", "conflict", "unavailable":
		return strings.TrimSpace(v)
	default:
		return "unavailable"
	}
}
func safeCode(v string) string {
	v = strings.TrimSpace(v)
	if v == "" || len(v) > 128 {
		return "todo.remote_request_failed"
	}
	for _, r := range v {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '.' || r == '_') {
			return "todo.remote_request_failed"
		}
	}
	return v
}
func safeMessage(v string) string {
	v = strings.TrimSpace(v)
	if len(v) > 256 {
		return ""
	}
	return v
}
func statusClass(status int) string {
	switch status {
	case http.StatusBadRequest, http.StatusRequestEntityTooLarge:
		return "bad_request"
	case http.StatusUnauthorized, http.StatusForbidden:
		return "forbidden"
	case http.StatusNotFound:
		return "not_found"
	case http.StatusConflict:
		return "conflict"
	default:
		return "unavailable"
	}
}
