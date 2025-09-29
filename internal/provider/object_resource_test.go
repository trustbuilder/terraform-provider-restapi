package provider

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccObjectResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + generateTestResource("header", `{"id":"1"}`, map[string]any{"headers": "{\"User-agent\": \"restapi-agent\"}"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("restapi_object.header", "headers.User-agent", "restapi-agent"),
					resource.TestCheckResourceAttrSet("restapi_object.header", "last_updated"),
				),
			},
			// Update and Read testing
			{
				Config: providerConfig + generateTestResource("header", `{"id":"1"}`, map[string]any{"headers": "{\"User-agent\": \"restapi-agent/latest\"}"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify the lonely attribute
					resource.TestCheckResourceAttr("restapi_object.header", "headers.User-agent", "restapi-agent/latest"),
					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("restapi_object.header", "last_updated"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func generateTestResource(name string, data string, params map[string]any) string {
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
		resource "restapi_object" "%s" {
		%s
	}`, name, strConfig)
}
