package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/trustbuilder/terraform-provider-restapi/internal/apiclient"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource = &tenantResource{}
)

// tenantResource is the resource implementation.
type tenantResource struct {
	url    string
	client *apiclient.APIClient
}

// tenantResourceModel maps the resource schema data.
type tenantResourceModel struct {
	Headers     types.Map    `tfsdk:"headers"`
	LastUpdated types.String `tfsdk:"last_updated"`
	Id          types.String `tfsdk:"id"`
	Tenant      types.String `tfsdk:"tenant"`
	Path        types.String `tfsdk:"path"`
	Data        types.String `tfsdk:"data"`
}

// NewtenantResource is a helper function to simplify the provider implementation.
func NewTenantResource() resource.Resource {
	return &tenantResource{}
}

// Metadata returns the resource type name.
func (r *tenantResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tenant"
}

// Schema defines the schema for the resource.
func (r *tenantResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Basic resource that does nothing other than interact with the Terraform state",
		Attributes: map[string]schema.Attribute{
			"headers": schema.MapAttribute{
				Description: "A map of header names and values to set on all outbound requests.",
				ElementType: types.StringType,
				Optional:    true,
			},
			"last_updated": schema.StringAttribute{
				Description: "Resource update date in RFC850 format.",
				Computed:    true,
			},
			"id": schema.StringAttribute{
				Description: "The ID of this resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"tenant": schema.StringAttribute{
				Description: "Tenant name used as identifier.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"path": schema.StringAttribute{
				Description: "The API path on top of the base URL set in the provider that represents objects of this type on the API server.",
				Required:    true,
			},
			"data": schema.StringAttribute{
				Description: "Valid JSON object that this provider will manage with the API server.",
				Required:    true,
			},
		},
	}
}

// Create a new resource.
func (r *tenantResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var planResource tenantResourceModel

	diags := req.Plan.Get(ctx, &planResource)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	responseData, err := r.client.SendRequest("POST", planResource.Path.ValueString(), planResource.Data.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Create request error", fmt.Sprintf("Creation request returned the error: %s", err))
		return
	}
	if err := (&planResource).update_computed_fields(responseData); err != nil {
		resp.Diagnostics.AddError("Missing attribute in create API response", fmt.Sprintf("Missing attribute in the creation response : %s", err))
		return
	}

	planResource.LastUpdated = types.StringValue(time.Now().Format(time.RFC3339))

	// Set state to fully populated data
	resp.Diagnostics.Append(resp.State.Set(ctx, planResource)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read resource information.
func (r *tenantResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var stateResource tenantResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &stateResource)...)
	if resp.Diagnostics.HasError() {
		return
	}

	path := strings.TrimRight(stateResource.Path.ValueString(), "/") + "?identifier=" + stateResource.Tenant.ValueString()
	responseData, err := r.client.SendRequest("GET", path, "")
	if err != nil {
		resp.Diagnostics.AddError("Read request error", fmt.Sprintf("Read request returned the error: %s on the path: %s", err, path))
		return
	}
	if err := (&stateResource).update_computed_fields(responseData); err != nil {
		resp.Diagnostics.AddError("Missing attribute in read API response", fmt.Sprintf("Missing attribute in the read response : %s", err))
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *tenantResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan tenantResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC3339))

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *tenantResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
}

// Configure adds the provider configured client to the resource.
func (r *tenantResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {

	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*apiclient.APIClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected string, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
	r.url = client.Uri
}

func (m *tenantResourceModel) update_computed_fields(jsonData string) error {
	//accept a json with an array of 1 object or an object
	var data any
	var mapData map[string]any
	var id string
	var tenant string
	var ok bool

	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		return err
	}

	switch v := data.(type) {
	case map[string]any:
		mapData, ok = data.(map[string]any)
		if !ok {
			return fmt.Errorf("type assertion from any to map[string]any fail")
		}
	case []any:
		mapData, ok = data.([]any)[0].(map[string]any)
		if !ok {
			return fmt.Errorf("type assertion from any to []any fail")
		}
	default:
		return fmt.Errorf("the json data is not an array, neither a map: %T", v)
	}

	if _, ok = mapData["id"]; !ok {
		return fmt.Errorf("id not found")
	}
	id, ok = mapData["id"].(string)
	if !ok {
		return fmt.Errorf("id value can't be casted into string")
	}

	if _, ok = mapData["identifier"]; !ok {
		return fmt.Errorf("identifier not found")
	}
	tenant, ok = mapData["identifier"].(string)
	if !ok {
		return fmt.Errorf("identifier value can't be casted into string")
	}

	m.Id = types.StringValue(id)
	m.Tenant = types.StringValue(tenant)

	return nil
}
