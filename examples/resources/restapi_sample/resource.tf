resource "restapi_sample" "test" {
  headers = {
    "User-agent" = "restapi-agent"
  }
}
