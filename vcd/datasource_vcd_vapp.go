package vcd

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func datasourceVcdVApp() *schema.Resource {
	return &schema.Resource{
		ReadContext: datasourceVcdVAppRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "A name for the vApp, unique within the VDC",
			},
			"org": {
				Type:     schema.TypeString,
				Optional: true,
				Description: "The name of organization to use, optional if defined at provider " +
					"level. Useful when connected as sysadmin working across different organizations",
			},
			"vdc": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The name of VDC to use, optional if defined at provider level",
			},
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Optional description of the vApp",
			},
			"metadata": {
				Type:        schema.TypeMap,
				Computed:    true,
				Description: "Key value map of metadata to assign to this vApp. Key and value can be any string.",
				Deprecated:  "Use metadata_entry instead",
			},
			"metadata_entry": metadataEntryDatasourceSchema("vApp"),
			"href": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "vApp Hyper Reference",
			},
			"guest_properties": {
				Type:        schema.TypeMap,
				Computed:    true,
				Description: "Key/value settings for guest properties",
			},
			"status": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Shows the status code of the vApp",
			},
			"status_text": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Shows the status of the vApp",
			},
			"lease": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Defines lease parameters for this vApp",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"runtime_lease_in_sec": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "How long any of the VMs in the vApp can run before the vApp is automatically powered off or suspended. 0 means never expires",
						},
						"storage_lease_in_sec": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "How long the vApp is available before being automatically deleted or marked as expired. 0 means never expires",
						},
					},
				},
			},
			"inherited_metadata": {
				Computed:    true,
				Type:        schema.TypeList,
				Description: "Metadata that is automatically set by VCD",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"vapp_origin_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "ID of the vApp from which this vApp was generated",
						},
						"vapp_origin_name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Name of the vApp from which this vApp was generated",
						},
						"vapp_origin_type": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Type of the vApp from which this vApp was generated",
						},
					},
				},
			},
		},
	}
}

func datasourceVcdVAppRead(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return genericVcdVAppRead(d, meta, "datasource")
}
