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
	URI                    types.String  `tfsdk:"uri"`
	Insecure               types.Bool    `tfsdk:"insecure"`
	Username               types.String  `tfsdk:"username"`
	Password               types.String  `tfsdk:"password"`
	Headers                types.Map     `tfsdk:"headers"`
	UseCookies             types.Bool    `tfsdk:"use_cookies"`
	Timeout                types.Int64   `tfsdk:"timeout"`
	IdAttribute            types.String  `tfsdk:"id_attribute"`
	CreateMethod           types.String  `tfsdk:"create_method"`
	ReadMethod             types.String  `tfsdk:"read_method"`
	UpdateMethod           types.String  `tfsdk:"update_method"`
	DestroyMethod          types.String  `tfsdk:"destroy_method"`
	CopyKeys               types.List    `tfsdk:"copy_keys"`
	WriteReturnsObject     types.Bool    `tfsdk:"write_returns_object"`
	CreateReturnsObject    types.Bool    `tfsdk:"create_returns_object"`
	XssiPrefix             types.String  `tfsdk:"xssi_prefix"`
	RateLimit              types.Float64 `tfsdk:"rate_limit"`
	TestPath               types.String  `tfsdk:"test_path"`
	Debug                  types.Bool    `tfsdk:"debug"`
	CertString             types.String  `tfsdk:"cert_string"`
	KeyString              types.String  `tfsdk:"key_string"`
	CertFile               types.String  `tfsdk:"cert_file"`
	KeyFile                types.String  `tfsdk:"key_file"`
	RootCaFile             types.String  `tfsdk:"root_ca_file"`
	RootCaString           types.String  `tfsdk:"root_ca_string"`
	OauthClientCredentials types.Object  `tfsdk:"oauth_client_credentials"`
}

type OauthClientCredentialsModel struct {
	OauthClientId      types.String `tfsdk:"oauth_client_id"`
	OauthClientSecret  types.String `tfsdk:"oauth_client_secret"`
	OauthTokenEndpoint types.String `tfsdk:"oauth_token_endpoint"`
	OauthScopes        types.List   `tfsdk:"oauth_scopes"`
	EndpointParams     types.Map    `tfsdk:"endpoint_params"`
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
			"insecure": schema.BoolAttribute{
				Description: "When using https, this disables TLS verification of the host.",
				Optional:    true,
			},
			"username": schema.StringAttribute{
				Description: "When set, will use this username for BASIC auth to the API.",
				Optional:    true,
			},
			"password": schema.StringAttribute{
				Description: "When set, will use this password for BASIC auth to the API.",
				Optional:    true,
				Sensitive:   true,
			},
			"headers": schema.MapAttribute{
				Description: "A map of header names and values to set on all outbound requests. This is useful if you want to use a script via the 'external' provider or provide a pre-approved token or change Content-Type from `application/json`. If `username` and `password` are set and Authorization is one of the headers defined here, the BASIC auth credentials take precedence.",
				ElementType: types.StringType,
				Optional:    true,
			},
			"use_cookies": schema.BoolAttribute{
				Description: "Enable cookie jar to persist session.",
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
			"copy_keys": schema.ListAttribute{
				Description: "When set, any PUT to the API for an object will copy these keys from the data the provider has gathered about the object. This is useful if internal API information must also be provided with updates, such as the revision of the object.",
				ElementType: types.StringType,
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
			"xssi_prefix": schema.StringAttribute{
				Description: "Trim the xssi prefix from response string, if present, before parsing.",
				Optional:    true,
			},
			"rate_limit": schema.Float64Attribute{
				Description: "Set this to limit the number of requests per second made to the API.",
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
			"cert_string": schema.StringAttribute{
				Description: "When set with the key_string parameter, the provider will load a client certificate as a string for mTLS authentication.",
				Optional:    true,
			},
			"key_string": schema.StringAttribute{
				Description: "When set with the cert_string parameter, the provider will load a client certificate as a string for mTLS authentication. Note that this mechanism simply delegates to golang's tls.LoadX509KeyPair which does not support passphrase protected private keys. The most robust security protections available to the key_file are simple file system permissions.",
				Optional:    true,
			},
			"cert_file": schema.StringAttribute{
				Description: "When set with the key_file parameter, the provider will load a client certificate as a file for mTLS authentication.",
				Optional:    true,
			},
			"key_file": schema.StringAttribute{
				Description: "When set with the cert_file parameter, the provider will load a client certificate as a file for mTLS authentication. Note that this mechanism simply delegates to golang's tls.LoadX509KeyPair which does not support passphrase protected private keys. The most robust security protections available to the key_file are simple file system permissions.",
				Optional:    true,
			},
			"root_ca_file": schema.StringAttribute{
				Description: "When set, the provider will load a root CA certificate as a file for mTLS authentication. This is useful when the API server is using a self-signed certificate and the client needs to trust it.",
				Optional:    true,
			},
			"root_ca_string": schema.StringAttribute{
				Description: "When set, the provider will load a root CA certificate as a string for mTLS authentication. This is useful when the API server is using a self-signed certificate and the client needs to trust it.",
				Optional:    true,
			},
			"oauth_client_credentials": schema.SingleNestedAttribute{
				Description: "Configuration for oauth client credential flow using the https://pkg.go.dev/golang.org/x/oauth2 implementation",
				Optional:    true,
				Attributes:  oauthClientCredentialsResourceSchema(),
			},
		},
	}
}

func oauthClientCredentialsResourceSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"oauth_client_id": schema.StringAttribute{
			Description: "client id",
			Required:    true,
		},
		"oauth_client_secret": schema.StringAttribute{
			Description: "client secret",
			Required:    true,
		},
		"oauth_token_endpoint": schema.StringAttribute{
			Description: "oauth token endpoint",
			Required:    true,
		},
		"oauth_scopes": schema.ListAttribute{
			Description: "scopes",
			ElementType: types.StringType,
			Optional:    true,
		},
		"endpoint_params": schema.MapAttribute{
			Description: "Additional key/values to pass to the underlying Oauth client library (as EndpointParams)",
			Optional:    true,
			ElementType: types.StringType,
		},
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

	/* As "data-safe" as terraform says it is, you'd think
	   it would have already coaxed this to a slice FOR me */
	// copyKeys := make([]string, 0)
	// if iCopyKeys := config.CopyKeys.ToListValue(); iCopyKeys != nil {
	// 	for _, v := range iCopyKeys.([]interface{}) {
	// 		copyKeys = append(copyKeys, v.(string))
	// 	}
	// }
	copyKeys := make([]string, 0)
	for _, elem := range config.CopyKeys.Elements() {
		copyKeys = append(copyKeys, elem.String())
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
		Insecure:            config.Insecure.ValueBool(),
		Username:            config.Username.ValueString(),
		Password:            config.Password.ValueString(),
		Headers:             headers,
		UseCookies:          config.UseCookies.ValueBool(),
		Timeout:             config.Timeout.ValueInt64(),
		IdAttribute:         config.IdAttribute.ValueString(),
		CopyKeys:            copyKeys,
		WriteReturnsObject:  config.WriteReturnsObject.ValueBool(),
		CreateReturnsObject: config.CreateReturnsObject.ValueBool(),
		XssiPrefix:          config.XssiPrefix.ValueString(),
		RateLimit:           config.RateLimit.ValueFloat64(),
		Debug:               config.Debug.ValueBool(),

		CreateMethod:  config.CreateMethod.ValueString(),
		ReadMethod:    config.ReadMethod.ValueString(),
		UpdateMethod:  config.UpdateMethod.ValueString(),
		DestroyMethod: config.DestroyMethod.ValueString(),
	}

	var oauthClientCredentials OauthClientCredentialsModel
	if !config.OauthClientCredentials.IsNull() && !config.OauthClientCredentials.IsUnknown() {
		diags := req.Config.GetAttribute(ctx, path.Root("oauth_client_credentials"), &oauthClientCredentials)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		opt.OauthClientID = oauthClientCredentials.OauthClientId.ValueString()
		opt.OauthClientSecret = oauthClientCredentials.OauthClientSecret.ValueString()
		opt.OauthTokenURL = oauthClientCredentials.OauthTokenEndpoint.ValueString()

		var stringSlice []string
		oauthScopeDiags := oauthClientCredentials.OauthScopes.ElementsAs(ctx, &stringSlice, false)
		resp.Diagnostics.Append(oauthScopeDiags...)
		if oauthScopeDiags.HasError() {
			resp.Diagnostics.AddError(
				"Error when converting oauth_client_credentials.oauth_scopes",
				fmt.Sprintf("can not convert oauth_client_credentials.oauth_scopes list. %v", oauthScopeDiags.Errors()),
			)
			return
		}
		opt.OauthScopes = stringSlice
	}

	// if tmp, ok := oauthConfig["endpoint_params"]; ok {
	// 	m := tmp.(map[string]interface{})
	// 	setVals := url.Values{}
	// 	for k, val := range m {
	// 		setVals.Add(k, val.(string))
	// 	}
	// 	opt.oauthEndpointParams = setVals
	// }

	opt.CertFile = config.CertFile.ValueString()
	opt.KeyFile = config.KeyFile.ValueString()
	opt.CertString = config.CertString.ValueString()
	opt.KeyString = config.KeyString.ValueString()
	opt.RootCaFile = config.RootCaFile.ValueString()
	opt.RootCaString = config.RootCaString.ValueString()

	client, err := apiclient.NewAPIClient(opt)

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
