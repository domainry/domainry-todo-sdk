package contract

import "encoding/json"

const (
	SaaSDiscoveryPath = "/todo/v1/discovery"
	SaaSInvokePath    = "/todo/v1/invoke"
)

type SaaSRequest struct {
	Operation string          `json:"operation"`
	Input     json.RawMessage `json:"input,omitempty"`
	Grants    []SaaSGrant     `json:"grants,omitempty"`
}
type SaaSResponse struct {
	Result    json.RawMessage `json:"result,omitempty"`
	Error     *SaaSError      `json:"error,omitempty"`
	Challenge *SaaSChallenge  `json:"challenge,omitempty"`
}
type SaaSError struct {
	Class     string `json:"class"`
	Code      string `json:"code"`
	Message   string `json:"message,omitempty"`
	Retryable bool   `json:"retryable,omitempty"`
}
type SaaSChallenge struct {
	Token string          `json:"token"`
	Kind  string          `json:"kind"`
	Input json.RawMessage `json:"input"`
}
type SaaSGrant struct {
	Token  string          `json:"token"`
	Result json.RawMessage `json:"result,omitempty"`
}
