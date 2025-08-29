package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/trustbuilder/terraform-provider-restapi/fakeserver"
)

const (
	providerConfig = `
provider "restapi" {
  uri = "http://localhost:19090/api/object_list"
}
`
)

// testAccProtoV6ProviderFactories is used to instantiate a provider during acceptance testing.
// The factory function is called for each Terraform CLI command to create a provider
// server that the CLI can connect to and interact with.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"restapi": providerserver.NewProtocol6WithError(New("test")()),
}

func testAccPreCheck(t *testing.T) {
	debug := false
	apiServerObjects := make(map[string]map[string]interface{})

	svr := fakeserver.NewFakeServer(19090, apiServerObjects, true, debug, "")

	t.Cleanup(func() {
		svr.Shutdown()
	})
}

// func TestResourceProvider_RequireTestPath(t *testing.T) {
// 	debug := false
// 	apiServerObjects := make(map[string]map[string]interface{})

// 	svr := fakeserver.NewFakeServer(8085, apiServerObjects, true, debug, "")
// 	svr.StartInBackground()

// 	provider := New("test")()
// 	raw := map[string]interface{}{
// 		"uri":       "http://127.0.0.1:8085/",
// 		"test_path": "/api/objects",
// 	}

// 	err := provider.Configure(context.TODO(), terraform.NewResourceConfigRaw(raw))
// 	if err != nil {
// 		t.Fatalf("Provider config failed when visiting %v at %v but it did not!", raw["test_path"], raw["uri"])
// 	}

// 	/* Now test the inverse */
// 	provider = New("test")()
// 	raw = map[string]interface{}{
// 		"uri":       "http://127.0.0.1:8085/",
// 		"test_path": "/api/apaththatdoesnotexist",
// 	}

// 	err = provider.Configure(context.TODO(), terraform.NewResourceConfigRaw(raw))
// 	if err == nil {
// 		t.Fatalf("Provider was expected to fail when visiting %v at %v but it did not!", raw["test_path"], raw["uri"])
// 	}

// 	svr.Shutdown()
// }

func TestProvider_basic(t *testing.T) {
	ctx := context.Background()
	provider := New("test")()

	// Create the provider server
	providerServer, err := createProviderServer(provider)
	if err != nil {
		t.Fatalf("Failed to create provider server: %s", err)
	}
	// Perform config validation

	validateResponse, err := providerServer.ValidateProviderConfig(ctx, &tfprotov6.ValidateProviderConfigRequest{})
	if err != nil {
		t.Fatalf("Provider config validation failed, error: %v", err)
	}

	if hasError(validateResponse.Diagnostics) {
		t.Fatalf("Provider config validation failed, diagnostics: %v", validateResponse.Diagnostics)
	}
}

func createProviderServer(provider provider.Provider) (tfprotov6.ProviderServer, error) {
	providerServerFunc := providerserver.NewProtocol6WithError(provider)
	server, err := providerServerFunc()
	if err != nil {
	} else {
	}
	return server, err
}

func hasError(diagnostics []*tfprotov6.Diagnostic) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == tfprotov6.DiagnosticSeverityError {
			return true
		}
	}
	return false
}
