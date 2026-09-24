// Package remote implements the official Todo SaaS transport.
package remote

import (
	"context"
	"encoding/json"
	"net/http"

	lifecyclecontract "github.com/domainry/domainry-lifecycle-sdk/contract"
	lifecyclemodel "github.com/domainry/domainry-lifecycle-sdk/model"
	todocontract "github.com/domainry/domainry-todo-sdk/contract"
	"github.com/domainry/domainry-todo-sdk/saashost"
	toolsdk "github.com/domainry/domainry-tools-sdk"
)

type Factory struct{ config Config }

func NewFactory(config Config) *Factory { return &Factory{config: config} }
func (factory *Factory) OpenSaaS(ctx context.Context, ref todocontract.ApplicationRef) (saashost.Binding, error) {
	if err := ref.Validate(); err != nil {
		return nil, err
	}
	client, err := newClient(factory.config, ref.RuntimeID)
	if err != nil {
		return nil, err
	}
	var descriptor todocontract.Descriptor
	if err = client.request(ctx, http.MethodGet, todocontract.SaaSDiscoveryPath, nil, &descriptor); err != nil {
		return nil, err
	}
	if err = descriptor.Validate(); err != nil {
		return nil, remoteError("unavailable", "todo.protocol_incompatible", false, err)
	}
	if descriptor.Audience != ref.RuntimeID {
		return nil, remoteError("forbidden", "todo.runtime_mismatch", false, nil)
	}
	return &binding{descriptor: descriptor, client: client}, nil
}

type binding struct {
	descriptor todocontract.Descriptor
	client     *client
}

func (v *binding) Descriptor() todocontract.Descriptor     { return v.descriptor }
func (v *binding) Todos() todocontract.TodoService         { return todoService{v.client} }
func (v *binding) Mutations() todocontract.MutationService { return mutationService{v.client} }
func (v *binding) SubjectLifecycle() lifecyclecontract.SubjectExecutionHandler {
	return subjectLifecycle{v.client}
}
func (*binding) Close(context.Context) error { return nil }

type todoService struct{ client *client }

func (v todoService) Todos(c context.Context, i todocontract.TodoQuery, a toolsdk.Authority) (todocontract.TodoPage, error) {
	var o todocontract.TodoPage
	e := v.client.invoke(c, "todos.list", struct {
		Input     todocontract.TodoQuery `json:"input"`
		Authority toolsdk.Authority      `json:"authority"`
	}{i, a}, &o)
	return o, e
}
func (v todoService) Todo(c context.Context, id string, a toolsdk.Authority) (todocontract.Todo, error) {
	var o todocontract.Todo
	e := v.client.invoke(c, "todos.get", struct {
		ID        string            `json:"id"`
		Authority toolsdk.Authority `json:"authority"`
	}{id, a}, &o)
	return o, e
}
func (v todoService) CreateTodos(c context.Context, i todocontract.TodoCreate, a toolsdk.Authority) (todocontract.TodoBatch, error) {
	var o todocontract.TodoBatch
	e := v.client.invoke(c, "todos.create", struct {
		Input     todocontract.TodoCreate `json:"input"`
		Authority toolsdk.Authority       `json:"authority"`
	}{i, a}, &o)
	return o, e
}
func (v todoService) UpdateTodo(c context.Context, id string, i todocontract.TodoUpdate, a toolsdk.Authority) (todocontract.Todo, error) {
	var o todocontract.Todo
	e := v.client.invoke(c, "todos.update", struct {
		ID        string                  `json:"id"`
		Input     todocontract.TodoUpdate `json:"input"`
		Authority toolsdk.Authority       `json:"authority"`
	}{id, i, a}, &o)
	return o, e
}
func (v todoService) DeleteTodo(c context.Context, id string, i todocontract.TodoDelete, a toolsdk.Authority) error {
	return v.client.invoke(c, "todos.delete", struct {
		ID        string                  `json:"id"`
		Input     todocontract.TodoDelete `json:"input"`
		Authority toolsdk.Authority       `json:"authority"`
	}{id, i, a}, nil)
}

type mutationService struct{ client *client }

func (v mutationService) ApplyMutation(c context.Context, i todocontract.Mutation, a toolsdk.Authority) (todocontract.MutationResult, error) {
	var o todocontract.MutationResult
	e := v.client.invoke(c, "mutations.apply", struct {
		Input     todocontract.Mutation `json:"input"`
		Authority toolsdk.Authority     `json:"authority"`
	}{i, a}, &o)
	return o, e
}
func (v mutationService) MutationReceipt(c context.Context, i todocontract.Mutation, a toolsdk.Authority) (todocontract.MutationResult, bool, error) {
	var o struct {
		Result todocontract.MutationResult `json:"result"`
		Found  bool                        `json:"found"`
	}
	e := v.client.invoke(c, "mutations.receipt", struct {
		Input     todocontract.Mutation `json:"input"`
		Authority toolsdk.Authority     `json:"authority"`
	}{i, a}, &o)
	return o.Result, o.Found, e
}

type subjectLifecycle struct{ client *client }

func (subjectLifecycle) Owner(context.Context) string { return "todo" }
func (v subjectLifecycle) PreviewSubject(c context.Context, w, s string) (json.RawMessage, error) {
	return v.invoke(c, "subjects.preview", "", w, s, nil)
}
func (v subjectLifecycle) ExportSubjectForRequest(c context.Context, r, w, s string) (json.RawMessage, error) {
	return v.invoke(c, "subjects.export", r, w, s, nil)
}
func (v subjectLifecycle) EraseSubjectForRequest(c context.Context, r, w, s string, h []lifecyclemodel.LegalHold) (json.RawMessage, error) {
	return v.invoke(c, "subjects.erase", r, w, s, h)
}
func (v subjectLifecycle) invoke(c context.Context, o, r, w, s string, h []lifecyclemodel.LegalHold) (json.RawMessage, error) {
	var out json.RawMessage
	e := v.client.invoke(c, o, struct {
		RequestID   string                     `json:"request_id,omitempty"`
		WorkspaceID string                     `json:"workspace_id"`
		SubjectID   string                     `json:"subject_id"`
		LegalHolds  []lifecyclemodel.LegalHold `json:"legal_holds,omitempty"`
	}{r, w, s, h}, &out)
	return out, e
}

var _ saashost.Factory = (*Factory)(nil)
