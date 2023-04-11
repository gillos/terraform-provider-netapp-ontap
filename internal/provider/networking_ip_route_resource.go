package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/netapp/terraform-provider-netapp-ontap/internal/interfaces"
	"github.com/netapp/terraform-provider-netapp-ontap/internal/utils"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ resource.Resource = &IPRouteResource{}
var _ resource.ResourceWithImportState = &IPRouteResource{}

// NewIPRouteResource is a helper function to simplify the provider implementation.
func NewIPRouteResource() resource.Resource {
	return &IPRouteResource{
		config: resourceOrDataSourceConfig{
			name: "networking_ip_route_resource",
		},
	}
}

// IPRouteResource defines the resource implementation.
type IPRouteResource struct {
	config resourceOrDataSourceConfig
}

// IPRouteResourceModel describes the resource data model.
type IPRouteResourceModel struct {
	CxProfileName types.String                `tfsdk:"cx_profile_name"`
	SVMName       types.String                `tfsdk:"svm_name"`
	Destination   *DestinationDataSourceModel `tfsdk:"destination"`
	Gateway       types.String                `tfsdk:"gateway"`
	Metric        types.Int64                 `tfsdk:"metric"`
	UUID          types.String                `tfsdk:"uuid"`
}

// Metadata returns the resource type name.
func (r *IPRouteResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + r.config.name
}

// Schema defines the schema for the resource.
func (r *IPRouteResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "NetRoute resource",

		Attributes: map[string]schema.Attribute{
			"cx_profile_name": schema.StringAttribute{
				MarkdownDescription: "Connection profile name",
				Required:            true,
			},
			"destination": schema.SingleNestedAttribute{
				Required:            true,
				MarkdownDescription: "destination IP address information",
				Attributes: map[string]schema.Attribute{
					"address": schema.StringAttribute{
						MarkdownDescription: "IPv4 or IPv6 address",
						Required:            true,
					},
					"netmask": schema.StringAttribute{
						MarkdownDescription: "netmask length (16) or IPv4 mask (255.255.0.0). For IPv6, valid range is 1 to 127.",
						Required:            true,
					},
				},
			},
			"svm_name": schema.StringAttribute{
				MarkdownDescription: "IPInterface vserver name",
				Optional:            true,
			},
			"gateway": schema.StringAttribute{
				MarkdownDescription: "The IP address of the gateway router leading to the destination.",
				Optional:            true,
			},
			"metric": schema.Int64Attribute{
				MarkdownDescription: "Indicates a preference order between several routes to the same destination.",
				Optional:            true,
			},
			"uuid": schema.StringAttribute{
				MarkdownDescription: "IP Route UUID",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *IPRouteResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	config, ok := req.ProviderData.(Config)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected Config, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
	}
	r.config.providerConfig = config
}

// Read refreshes the Terraform state with the latest data.
func (r *IPRouteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data IPRouteResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	errorHandler := utils.NewErrorHandler(ctx, &resp.Diagnostics)
	// we need to defer setting the client until we can read the connection profile name
	client, err := getRestClient(errorHandler, r.config, data.CxProfileName)
	if err != nil {
		// error reporting done inside NewClient
		return
	}

	cluster, err := interfaces.GetCluster(errorHandler, *client)
	if err != nil {
		// error reporting done inside GetCluster
		return
	}
	restInfo, err := interfaces.GetIPRoute(errorHandler, *client, data.Destination.Address.ValueString(), data.SVMName.ValueString(), cluster.Version)
	if err != nil {
		// error reporting done inside GetIPInterface
		return
	}

	data.Destination.Address = types.StringValue(restInfo.Destination.Address)
	data.Destination.Netmask = types.StringValue(restInfo.Destination.Netmask)
	data.Gateway = types.StringValue(restInfo.Gateway)
	data.Metric = types.Int64Value(restInfo.Metric)
	data.SVMName = types.StringValue(restInfo.SVMName.Name)

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Debug(ctx, fmt.Sprintf("read a resource: %#v", data))

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Create a resource and retrieve UUID
func (r *IPRouteResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *IPRouteResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	var body interfaces.IPRouteResourceBodyDataModelONTAP
	errorHandler := utils.NewErrorHandler(ctx, &resp.Diagnostics)

	if resp.Diagnostics.HasError() {
		return
	}

	body.Destination.Address = data.Destination.Address.ValueString()
	body.Destination.Netmask = data.Destination.Netmask.ValueString()
	if !data.SVMName.IsNull() {
		body.SVM.Name = data.SVMName.ValueString()
	}
	if !data.Gateway.IsNull() {
		body.Gateway = data.Gateway.ValueString()
	}
	if !data.Metric.IsNull() {
		body.Metric = data.Metric.ValueInt64()
	}

	client, err := getRestClient(errorHandler, r.config, data.CxProfileName)
	if err != nil {
		// error reporting done inside NewClient
		return
	}

	resource, err := interfaces.CreateIPRoute(errorHandler, *client, body)
	if err != nil {
		return
	}

	data.UUID = types.StringValue(resource.UUID)

	tflog.Trace(ctx, fmt.Sprintf("created a resource, UUID=%s", data.UUID))

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *IPRouteResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *IPRouteResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *IPRouteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *IPRouteResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	errorHandler := utils.NewErrorHandler(ctx, &resp.Diagnostics)
	client, err := getRestClient(errorHandler, r.config, data.CxProfileName)
	if err != nil {
		// error reporting done inside NewClient
		return
	}

	if data.UUID.IsNull() {
		errorHandler.MakeAndReportError("UUID is null", "ip_interface UUID is null")
		return
	}

	err = interfaces.DeleteIPRoute(errorHandler, *client, data.UUID.ValueString())
	if err != nil {
		return
	}

}

// ImportState imports a resource using ID from terraform import command by calling the Read method.
func (r *IPRouteResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
