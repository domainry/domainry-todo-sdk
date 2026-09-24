package remote

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	todocontract "github.com/domainry/domainry-todo-sdk/contract"
	"github.com/domainry/domainry-todo-sdk/saashost"
	toolsdk "github.com/domainry/domainry-tools-sdk"
)

func TestFactoryValidatesAudienceAndCallsMutationBoundary(t *testing.T) {
	var invoked bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer service-token" || r.Header.Get(saashost.RuntimeIDHeader) != "runtime-a" {
			t.Fatalf("headers=%v", r.Header)
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == todocontract.SaaSDiscoveryPath {
			_ = json.NewEncoder(w).Encode(todocontract.Descriptor{ProtocolVersion: todocontract.SaaSProtocolVersionV1, Mode: todocontract.DeploymentModeSaaS, Audience: "runtime-a", Capabilities: append([]string(nil), todocontract.SaaSCapabilitiesV1...)})
			return
		}
		var envelope todocontract.SaaSRequest
		if json.NewDecoder(r.Body).Decode(&envelope) != nil || envelope.Operation != "mutations.apply" {
			t.Fatalf("request=%+v", envelope)
		}
		invoked = true
		raw, _ := json.Marshal(todocontract.MutationResult{ResourceID: "todo-a"})
		_ = json.NewEncoder(w).Encode(todocontract.SaaSResponse{Result: raw})
	}))
	defer server.Close()
	binding, err := NewFactory(Config{Endpoint: server.URL, ServiceAccessToken: "service-token"}).OpenSaaS(t.Context(), todocontract.ApplicationRef{RuntimeID: "runtime-a"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := binding.Mutations().ApplyMutation(t.Context(), todocontract.Mutation{Key: "one", Operation: "todo_delete"}, toolsdk.Authority{Known: true, RuntimeID: "runtime-a", WorkspaceID: "workspace", UserID: "user"})
	if err != nil || result.ResourceID != "todo-a" || !invoked {
		t.Fatalf("result=%+v invoked=%v err=%v", result, invoked, err)
	}
}
