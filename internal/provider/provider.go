package provider

import (
	"context"
	"fmt"
	"os"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/trustbuilder/terraform-provider-restapi/internal/apiclient"
	"github.com/trustbuilder/terraform-provider-restapi/internal/envvar"
)

var _ provider.Provider = &RestapiProvider{}

// Defines the provider implementation.
type RestapiProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &RestapiProvider{
			version: version,
		}
	}
}

// Describes the provider data model.
type RestapiProviderModel struct {
	URI                 types.String `tfsdk:"uri"`
	Headers             types.Map    `tfsdk:"headers"`
	Timeout             types.Int64  `tfsdk:"timeout"`
	IdAttribute         types.String `tfsdk:"id_attribute"`
	CreateMethod        types.String `tfsdk:"create_method"`
	ReadMethod          types.String `tfsdk:"read_method"`
	UpdateMethod        types.String `tfsdk:"update_method"`
	DestroyMethod       types.String `tfsdk:"destroy_method"`
	WriteReturnsObject  types.Bool   `tfsdk:"write_returns_object"`
	CreateReturnsObject types.Bool   `tfsdk:"create_returns_object"`
	TestPath            types.String `tfsdk:"test_path"`
	Debug               types.Bool   `tfsdk:"debug"`
}

func (p *RestapiProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "restapi"
	resp.Version = p.version
}

func (p *RestapiProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"uri": schema.StringAttribute{
				Description: "URI of the REST API endpoint. This serves as the base of all requests.",
				Required:    false,
				Validators: []validator.String{
					stringvalidator.LengthBetween(10, 2048),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^https?://.*$`),
						"Must be in https?:// format",
					),
				},
				Optional: true,
			},
			"headers": schema.MapAttribute{
				Description: "A map of header names and values to set on all outbound requests. This is useful if you want to use a script via the 'external' provider or provide a pre-approved token or change Content-Type from `application/json`. If `username` and `password` are set and Authorization is one of the headers defined here, the BASIC auth credentials take precedence.",
				ElementType: types.StringType,
				Optional:    true,
			},
			"timeout": schema.Int64Attribute{
				Description: "When set, will cause requests taking longer than this time (in seconds) to be aborted.",
				Optional:    true,
			},
			"id_attribute": schema.StringAttribute{
				Description: "When set, this key will be used to operate on REST objects. For example, if the ID is set to 'name', changes to the API object will be to http://foo.com/bar/VALUE_OF_NAME. This value may also be a '/'-delimeted path to the id attribute if it is multple levels deep in the data (such as `attributes/id` in the case of an object `{ \"attributes\": { \"id\": 1234 }, \"config\": { \"name\": \"foo\", \"something\": \"bar\"}}`",
				Optional:    true,
			},
			"create_method": schema.StringAttribute{
				Description: "Defaults to `POST`. The HTTP method used to CREATE objects of this type on the API server.",
				Optional:    true,
			},
			"read_method": schema.StringAttribute{
				Description: "Defaults to `GET`. The HTTP method used to READ objects of this type on the API server.",
				Optional:    true,
			},
			"update_method": schema.StringAttribute{
				Description: "Defaults to `PUT`. The HTTP method used to UPDATE objects of this type on the API server.",
				Optional:    true,
			},
			"destroy_method": schema.StringAttribute{
				Description: "Defaults to `DELETE`. The HTTP method used to DELETE objects of this type on the API server.",
				Optional:    true,
			},
			"write_returns_object": schema.BoolAttribute{
				Description: "Set this when the API returns the object created on all write operations (POST, PUT). This is used by the provider to refresh internal data structures.",
				Optional:    true,
			},
			"create_returns_object": schema.BoolAttribute{
				Description: "Set this when the API returns the object created only on creation operations (POST). This is used by the provider to refresh internal data structures.",
				Optional:    true,
			},
			"test_path": schema.StringAttribute{
				Description: "If set, the provider will issue a read_method request to this path after instantiation requiring a 200 OK response before proceeding. This is useful if your API provides a no-op endpoint that can signal if this provider is configured correctly. Response data will be ignored.",
				Optional:    true,
			},
			"debug": schema.BoolAttribute{
				Description: "Enabling this will cause lots of debug information to be printed to STDOUT by the API client.",
				Optional:    true,
			},
		},
		Description: "Provider managing REST API queries. The only authenthication way is JWT.",
	}
}

func (p *RestapiProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {

	var config RestapiProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uri := os.Getenv(envvar.RestApiUri)

	if !config.URI.IsNull() {
		uri = config.URI.ValueString()
	}

	tflog.Debug(ctx, "uri content: "+uri)

	if uri == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("uri"),
			"The uri is mandatory",
			"The provider has unknown configuration value for the uri. "+
				"Set the uri value in the configuration or use the "+envvar.RestApiUri+" environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// headers := make(map[string]string)
	// if iHeaders := config.Headers.ToMapValue(); iHeaders != nil {
	// 	for k, v := range iHeaders.(map[string]interface{}) {
	// 		headers[k] = v.(string)
	// 	}
	// }
	headers := make(map[string]string)
	for k, v := range config.Headers.Elements() {
		headers[k] = v.String()
	}

	opt := &apiclient.ApiClientOpt{
		Uri:                 config.URI.ValueString(),
		Headers:             headers,
		Timeout:             config.Timeout.ValueInt64(),
		IdAttribute:         config.IdAttribute.ValueString(),
		WriteReturnsObject:  config.WriteReturnsObject.ValueBool(),
		CreateReturnsObject: config.CreateReturnsObject.ValueBool(),
		Debug:               config.Debug.ValueBool(),

		CreateMethod:  config.CreateMethod.ValueString(),
		ReadMethod:    config.ReadMethod.ValueString(),
		UpdateMethod:  config.UpdateMethod.ValueString(),
		DestroyMethod: config.DestroyMethod.ValueString(),
	}

	client, err := apiclient.NewAPIClient(opt)
	if err != nil {
		resp.Diagnostics.AddError(
			"API client creation fail",
			fmt.Sprintf("The creation of the API client failed. Verify the provider configuration. %v", err),
		)
	}

	testPath := config.TestPath.ValueString()
	_, err = client.SendRequest(client.ReadMethod, testPath, "")
	if err != nil {
		resp.Diagnostics.AddError(
			"Test send request fail",
			fmt.Sprintf("a test request to %v after setting up the provider did not return an OK response - is your configuration correct? %v", testPath, err),
		)
	}

	resp.DataSourceData = client
	resp.ResourceData = client

}

func (p *RestapiProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewSampleResource,
	}
}

func (p *RestapiProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return nil
}
