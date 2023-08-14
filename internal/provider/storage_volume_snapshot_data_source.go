package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/netapp/terraform-provider-netapp-ontap/internal/interfaces"
	"github.com/netapp/terraform-provider-netapp-ontap/internal/utils"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ datasource.DataSource = &StorageVolumeSnapshotDataSource{}

// NewStorageVolumeSnapshotDataSource is a helper function to simplify the provider implementation.
func NewStorageVolumeSnapshotDataSource() datasource.DataSource {
	return &StorageVolumeSnapshotDataSource{
		config: resourceOrDataSourceConfig{
			name: "storage_volume_snapshot_data_source",
		},
	}
}

// StorageVolumeSnapshotDataSource defines the data source implementation.
type StorageVolumeSnapshotDataSource struct {
	config resourceOrDataSourceConfig
}

// StorageVolumeSnapshotDataSourceModel describes the data source data model.
type StorageVolumeSnapshotDataSourceModel struct {
	CxProfileName types.String `tfsdk:"cx_profile_name"`
	// Snapshot Variables
	CreateTime      types.String  `tfsdk:"create_time"`
	Comment         types.String  `tfsdk:"comment"`
	ExpiryTime      types.String  `tfsdk:"expiry_time"`
	Name            types.String  `tfsdk:"name"`
	Size            types.Float64 `tfsdk:"size"`
	SnapmirrorLabel types.String  `tfsdk:"snapmirror_label"`
	State           types.String  `tfsdk:"state"`
	VolumeUUID      types.String  `tfsdk:"volume_uuid"`
	VolumeName      types.String  `tfsdk:"volume_name"`
}

// Metadata returns the data source type name.
func (d *StorageVolumeSnapshotDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + d.config.name
}

// Schema defines the schema for the data source.
func (d *StorageVolumeSnapshotDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Storage Volume Snapshot data source",

		Attributes: map[string]schema.Attribute{
			"cx_profile_name": schema.StringAttribute{
				MarkdownDescription: "Connection profile name",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Snapshot name",
				Required:            true,
			},
			// TODO: replace UUID with Volume Name, and svm name
			"volume_uuid": schema.StringAttribute{
				MarkdownDescription: "Volume UUID",
				Required:            true,
			},
			"volume_name": schema.StringAttribute{
				MarkdownDescription: "Volume Name",
				Computed:            true,
			},
			"create_time": schema.StringAttribute{
				MarkdownDescription: "Create time",
				Computed:            true,
			},
			"expiry_time": schema.StringAttribute{
				MarkdownDescription: "Expiry time",
				Computed:            true,
			},
			"state": schema.StringAttribute{
				MarkdownDescription: "State",
				Computed:            true,
			},
			"size": schema.Float64Attribute{
				MarkdownDescription: "Size",
				Computed:            true,
			},
			"comment": schema.StringAttribute{
				MarkdownDescription: "Comment",
				Computed:            true,
			},
			"snapmirror_label": schema.StringAttribute{
				MarkdownDescription: "Snapmirror Label",
				Computed:            true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *StorageVolumeSnapshotDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data StorageVolumeSnapshotDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	errorHandler := utils.NewErrorHandler(ctx, &resp.Diagnostics)
	// we need to defer setting the client until we can read the connection profile name
	client, err := getRestClient(errorHandler, d.config, data.CxProfileName)
	if err != nil {
		// error reporting done inside NewClient
		return
	}

	if data.Name.IsNull() {
		errorHandler.MakeAndReportError("error reading snapshot", "Snapshot name is null")
		return
	}
	// TODO change to volume name
	if data.VolumeUUID.IsNull() {
		errorHandler.MakeAndReportError("error reading snapshot", "Volume UUID is null")
		return
	}

	snapshot, err := interfaces.GetStorageVolumeSnapshots(errorHandler, *client, data.Name.ValueString(), data.VolumeUUID.ValueString())
	if err != nil {
		return
	}
	if snapshot == nil {
		errorHandler.MakeAndReportError("No snapshot found", fmt.Sprintf("snapshot %s not found.", data.Name.ValueString()))
		return
	}
	data.CreateTime = types.StringValue(snapshot.CreateTime)
	data.Comment = types.StringValue(snapshot.Comment)
	data.ExpiryTime = types.StringValue(snapshot.ExpiryTime)
	data.Name = types.StringValue(snapshot.Name)
	data.Size = types.Float64Value(snapshot.Size)
	data.SnapmirrorLabel = types.StringValue(snapshot.SnapmirrorLabel)
	data.State = types.StringValue(snapshot.State)
	data.VolumeUUID = types.StringValue(snapshot.Volume.UUID)
	data.VolumeName = types.StringValue(snapshot.Volume.Name)

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Debug(ctx, fmt.Sprintf("read a data source: %#v", data))

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Configure adds the provider configured client to the data source.
func (d *StorageVolumeSnapshotDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	config, ok := req.ProviderData.(Config)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected Config, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
	}
	d.config.providerConfig = config
}
