package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSampleResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + testAccExampleResourceConfig("{\"User-agent\": \"restapi-agent\"}"),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify the lonely attribute
					resource.TestCheckResourceAttr("restapi_sample.test", "headers.User-agent", "restapi-agent"),
					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("restapi_sample.test", "last_updated"),
				),
			},
			// Update and Read testing
			{
				Config: providerConfig + testAccExampleResourceConfig("{\"User-agent\": \"restapi-agent/latest\"}"),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify the lonely attribute
					resource.TestCheckResourceAttr("restapi_sample.test", "headers.User-agent", "restapi-agent/latest"),
					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("restapi_sample.test", "last_updated"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccExampleResourceConfig(configurableAttribute string) string {
	return fmt.Sprintf(`
resource "restapi_sample" "test" {
  headers = %v
}
`, configurableAttribute)
}
