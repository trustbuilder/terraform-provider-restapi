resource "trustbuilder_tenant" "test" {
  path = "/tenants"
  data = jsonencode(local.tenant_body)
}
