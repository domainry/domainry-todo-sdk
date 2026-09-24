// Package saashost defines the remote Todo deployment boundary. It is separate
// from modulehost so an HTTP client can never pretend to participate in a
// caller-owned SQL transaction.
package saashost

import (
	"context"

	lifecyclecontract "github.com/domainry/domainry-lifecycle-sdk/contract"
	todocontract "github.com/domainry/domainry-todo-sdk/contract"
	toolsdk "github.com/domainry/domainry-tools-sdk"
)

const RuntimeIDHeader = "X-Domainry-Runtime-ID"

type SourceAuthorizer func(context.Context, string, toolsdk.Authority) error

type Binding interface {
	Descriptor() todocontract.Descriptor
	Todos() todocontract.TodoService
	Mutations() todocontract.MutationService
	SubjectLifecycle() lifecyclecontract.SubjectExecutionHandler
	BindSourceAuthorizer(SourceAuthorizer) error
	Close(context.Context) error
}

type Factory interface {
	OpenSaaS(context.Context, todocontract.ApplicationRef) (Binding, error)
}
