package contract

import (
	"context"
	"fmt"
	"slices"
	"strings"

	toolsdk "github.com/domainry/domainry-tools-sdk"
)

const (
	SaaSProtocolVersionV1 = "todo.saas.v1"
	DeploymentModeSaaS    = "saas"
)

var SaaSCapabilitiesV1 = []string{
	"todos.read",
	"todos.write",
	"mutations.apply",
	"mutations.receipt",
	"subjects.lifecycle",
}

// Descriptor is returned by the SaaS discovery endpoint before a client is
// allowed to use the service for one Runtime audience.
type Descriptor struct {
	ProtocolVersion string   `json:"protocol_version"`
	Mode            string   `json:"mode"`
	Audience        string   `json:"audience"`
	Capabilities    []string `json:"capabilities"`
}

func (descriptor Descriptor) Validate() error {
	if descriptor.ProtocolVersion != SaaSProtocolVersionV1 || descriptor.Mode != DeploymentModeSaaS {
		return fmt.Errorf("Todo SaaS protocol descriptor is unsupported")
	}
	if strings.TrimSpace(descriptor.Audience) == "" || descriptor.Audience != strings.TrimSpace(descriptor.Audience) || len(descriptor.Audience) > 255 {
		return fmt.Errorf("Todo SaaS protocol audience is invalid")
	}
	capabilities := slices.Clone(descriptor.Capabilities)
	slices.Sort(capabilities)
	expected := slices.Clone(SaaSCapabilitiesV1)
	slices.Sort(expected)
	if !slices.Equal(capabilities, expected) {
		return fmt.Errorf("Todo SaaS protocol capabilities are unsupported")
	}
	return nil
}

// MutationService is the deployment-neutral idempotent mutation boundary.
// SaaS callers never receive or supply a database transaction. Replaying the
// same Mutation.Key must return the original result through ApplyMutation or
// MutationReceipt.
type MutationService interface {
	ApplyMutation(context.Context, Mutation, toolsdk.Authority) (MutationResult, error)
	MutationReceipt(context.Context, Mutation, toolsdk.Authority) (MutationResult, bool, error)
}
