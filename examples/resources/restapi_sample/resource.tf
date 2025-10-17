resource "trustbuilder_idhub_tenant" "test" {
  path = "/tenants"
  data = jsonencode(local.tenant_body)
}
