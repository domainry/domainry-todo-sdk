package contract

import "testing"

func TestSaaSDescriptorRequiresExactAudienceAndCapabilities(t *testing.T) {
	descriptor := Descriptor{ProtocolVersion: SaaSProtocolVersionV1, Mode: DeploymentModeSaaS, Audience: "runtime-a", Capabilities: append([]string(nil), SaaSCapabilitiesV1...)}
	if err := descriptor.Validate(); err != nil {
		t.Fatal(err)
	}
	descriptor.Capabilities = append(descriptor.Capabilities, "store.borrow")
	if err := descriptor.Validate(); err == nil {
		t.Fatal("descriptor accepted an undeclared Store capability")
	}
}
