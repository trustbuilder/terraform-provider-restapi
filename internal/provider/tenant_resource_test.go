package provider

import (
	"encoding/json"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/trustbuilder/terraform-provider-restapi/fakeserver"
)

var tenantsDataObjects = map[string]map[string]any{
	"1": {
		"Test_case":  "normal",
		"identifier": "tenant_1",
		"Id":         "1",
		"Revision":   1,
		"Thing":      "potato",
		"Is_cat":     false,
		"Colors":     []string{"orange", "white"},
		"Attrs": map[string]any{
			"size":   "6 in",
			"weight": "10 oz",
		},
	},
	"2": {
		"Test_case":  "minimal",
		"identifier": "tenant_2",
		"Id":         "2",
		"Thing":      "fork",
	},
	"3": {
		"Test_case":  "no Colors",
		"identifier": "tenant_name",
		"Id":         "3",
		"Thing":      "paper",
		"Is_cat":     false,
		"Attrs": map[string]any{
			"height": "8.5 in",
			"width":  "11 in",
		},
	},
	"4": {
		"Test_case":  "no Attrs",
		"identifier": "tenant_4",
		"Id":         "4",
		"Thing":      "nothing",
		"Is_cat":     false,
		"Colors":     []string{"none"},
	},
	"5": {
		"Test_case":  "pet",
		"identifier": "tenant_5",
		"Id":         "5",
		"Thing":      "cat",
		"Is_cat":     true,
		"Colors":     []string{"orange", "white"},
		"Attrs": map[string]any{
			"size":   "1.5 ft",
			"weight": "15 lb",
		},
	},
}

func testAccTenantPreCheck(t *testing.T) {
	debug := false
	svr := fakeserver.NewFakeServer(19090, tenantsDataObjects, true, debug, "")

	t.Cleanup(func() {
		svr.Shutdown()
	})
}

func TestAccTenantResource_basic(t *testing.T) {
	var firstUpdatedTime string

	resourceName := "restapi_tenant.api_data"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccTenantPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + generateTenantResource("api_data", `{"Test_case": "newTest", "identifier": "tenant_6", "id": "6", "Thing": "basic"}`, nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("restapi_tenant.api_data", "id"),
					resource.TestCheckResourceAttrSet("restapi_tenant.api_data", "last_updated"),
					//Setup the last_updated change verification
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources[resourceName]
						if !ok {
							return fmt.Errorf("Custom resource check: resource not found in state")
						}
						lastUpdated := rs.Primary.Attributes["last_updated"]
						if lastUpdated == "" {
							return fmt.Errorf("Custom resource check: last_updated is empty")
						}
						firstUpdatedTime = lastUpdated
						return nil
					},
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("restapi_tenant.api_data", tfjsonpath.New("id"), knownvalue.StringExact("6")),
					statecheck.ExpectKnownValue("restapi_tenant.api_data", tfjsonpath.New("tenant"), knownvalue.StringExact("tenant_6")),
					statecheck.ExpectKnownValue("restapi_tenant.api_data", tfjsonpath.New("data"), knownvalue.StringExact(`{"Test_case": "newTest", "identifier": "tenant_6", "id": "6", "Thing": "basic"}`)),
					statecheck.ExpectKnownValue("restapi_tenant.api_data", tfjsonpath.New("last_updated"), knownvalue.StringRegexp(regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(Z|[+-]\d{2}:\d{2})$`))),
				},
			},
			// Update and Read testing
			{
				Config: providerConfig + generateTenantResource("api_data", `{"Test_case": "newTest", "identifier": "tenant_6", "id": "6", "Thing": "basic", "additional_field": "new API feature"}`, nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("restapi_tenant.api_data", "id"),
					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("restapi_tenant.api_data", "last_updated"),
					//Verify that the last_updated has changed
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources[resourceName]
						if !ok {
							return fmt.Errorf("Custom resource check: resource not found in state")
						}
						lastUpdated := rs.Primary.Attributes["last_updated"]
						if lastUpdated == firstUpdatedTime {
							return fmt.Errorf("Custom resource check: expected last_updated to change, but it did not")
						}
						return nil
					},
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("restapi_tenant.api_data", tfjsonpath.New("id"), knownvalue.StringExact("6")),
					statecheck.ExpectKnownValue("restapi_tenant.api_data", tfjsonpath.New("tenant"), knownvalue.StringExact("tenant_6")),
					statecheck.ExpectKnownValue("restapi_tenant.api_data", tfjsonpath.New("data"), knownvalue.StringExact(`{"Test_case": "newTest", "identifier": "tenant_6", "id": "6", "Thing": "basic", "additional_field": "new API feature"}`)),
				},
			},
			// Returns API error when the API object to create already exists
			{
				Config:      providerConfig + generateTenantResource("error", `{"id":"1"}`, map[string]any{}),
				ExpectError: regexp.MustCompile(".*Create request error.*"),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func generateTenantResource(name string, data string, params map[string]any) string {
	strData, _ := json.Marshal(data)
	config := []string{
		`path = "/api/objects"`,
		fmt.Sprintf("data = %s", strData),
	}
	for k, v := range params {
		entry := fmt.Sprintf(`%s = %v`, k, v)
		config = append(config, entry)
	}
	strConfig := ""
	for _, v := range config {
		strConfig = strConfig + v + "\n"
	}

	return fmt.Sprintf(`
		resource "restapi_tenant" "%s" {
		%s
	}`, name, strConfig)
}
