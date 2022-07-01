package vcd

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"log"
	"strings"

	"github.com/vmware/go-vcloud-director/v2/govcd"

	"github.com/vmware/go-vcloud-director/v2/types/v56"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceVcdAlbSettings() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceVcdAlbSettingsCreateUpdate,
		UpdateContext: resourceVcdAlbSettingsCreateUpdate,
		ReadContext:   resourceVcdAlbSettingsRead,
		DeleteContext: resourceVcdAlbSettingsDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceVcdAlbSettingsImport,
		},

		Schema: map[string]*schema.Schema{
			"org": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Description: "The name of organization to use, optional if defined at provider " +
					"level. Useful when connected as sysadmin working across different organizations",
			},
			"vdc": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The name of VDC to use, optional if defined at provider level",
				Deprecated:  "Edge Gateway will be looked up based on 'edge_gateway_id' field",
			},
			"edge_gateway_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Edge gateway ID",
			},
			"is_active": {
				Type:        schema.TypeBool,
				Required:    true,
				Description: "Defines if ALB is enabled on Edge Gateway",
			},
			"service_network_specification": {
				Type:        schema.TypeString,
				ForceNew:    true,
				Optional:    true,
				Computed:    true,
				Description: "Optional custom network CIDR definition for ALB Service Engine placement (VCD default is 192.168.255.1/25)",
			},
			"supported_feature_set": {
				Type:         schema.TypeString,
				Optional:     true,
				Required:     false, // It should be required but for compatibility reasons it is not
				ValidateFunc: validation.StringInSlice([]string{"STANDARD", "PREMIUM"}, false),
				Description:  "Feature set for ALB in this Edge Gateway. One of 'STANDARD', 'PREMIUM'.",
			},
		},
	}
}

// resourceVcdAlbSettingsCreateUpdate covers Create and Update functionality for resource because the API
// endpoint only supports PUT and GET
func resourceVcdAlbSettingsCreateUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vcdClient := meta.(*VCDClient)
	vcdClient.lockParentEdgeGtw(d)
	defer vcdClient.unLockParentEdgeGtw(d)

	orgName := d.Get("org").(string)
	edgeGatewayId := d.Get("edge_gateway_id").(string)

	nsxtEdge, err := vcdClient.GetNsxtEdgeGatewayById(orgName, edgeGatewayId)
	if err != nil {
		return diag.Errorf("error retrieving Edge Gateway: %s", err)
	}

	albConfig := getNsxtAlbConfigurationType(d)
	_, err = nsxtEdge.UpdateAlbSettings(albConfig)
	if err != nil {
		return diag.Errorf("error setting NSX-T ALB General Settings: %s", err)
	}

	// ALB configuration does not have its own ID, but is done for each Edge Gateway therefore
	d.SetId(edgeGatewayId)

	return resourceVcdAlbSettingsRead(ctx, d, meta)
}

func resourceVcdAlbSettingsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return vcdAlbSettingsRead(meta, d, "resource")
}

// vcdAlbSettingsRead is used for read in resource and data source. The only difference between the two is that a
// resource should unset ID, while a data source should return an error
func vcdAlbSettingsRead(meta interface{}, d *schema.ResourceData, resourceType string) diag.Diagnostics {
	vcdClient := meta.(*VCDClient)

	orgName := d.Get("org").(string)
	edgeGatewayId := d.Get("edge_gateway_id").(string)

	nsxtEdge, err := vcdClient.GetNsxtEdgeGatewayById(orgName, edgeGatewayId)
	if err != nil {
		// Edge Gateway being not found means that this resource is removed. Data source should still return error.
		if govcd.ContainsNotFound(err) && resourceType == "resource" {
			d.SetId("")
			return nil
		}

		return diag.Errorf("error retrieving Edge Gateway: %s", err)
	}

	albConfig, err := nsxtEdge.GetAlbSettings()
	if err != nil {
		return diag.Errorf("error retrieve NSX-T ALB General Settings: %s", err)
	}

	setNsxtAlbConfigurationData(albConfig, d)
	d.SetId(edgeGatewayId)

	return nil
}

func resourceVcdAlbSettingsDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vcdClient := meta.(*VCDClient)
	vcdClient.lockParentEdgeGtw(d)
	defer vcdClient.unLockParentEdgeGtw(d)

	orgName := d.Get("org").(string)
	edgeGatewayId := d.Get("edge_gateway_id").(string)

	nsxtEdge, err := vcdClient.GetNsxtEdgeGatewayById(orgName, edgeGatewayId)
	if err != nil {
		return diag.Errorf("error retrieving Edge Gateway: %s", err)
	}

	return diag.FromErr(nsxtEdge.DisableAlb())
}

func resourceVcdAlbSettingsImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	log.Printf("[TRACE] NSX-T ALB General Settings import initiated")

	resourceURI := strings.Split(d.Id(), ImportSeparator)
	if len(resourceURI) != 3 {
		return nil, fmt.Errorf("resource name must be specified as org-name.vdc-or-vdc-group-name.nsxt-edge-gw-name")
	}
	orgName, vdcOrVdcGroupName, edgeName := resourceURI[0], resourceURI[1], resourceURI[2]

	vcdClient := meta.(*VCDClient)
	vdcOrVdcGroup, err := lookupVdcOrVdcGroup(vcdClient, orgName, vdcOrVdcGroupName)
	if err != nil {
		return nil, err
	}

	if !vdcOrVdcGroup.IsNsxt() {
		return nil, fmt.Errorf("this resource is only supported for NSX-T Edge Gateways please use")
	}

	edge, err := vdcOrVdcGroup.GetNsxtEdgeGatewayByName(edgeName)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve NSX-T edge gateway with ID '%s': %s", d.Id(), err)
	}

	dSet(d, "org", orgName)
	dSet(d, "edge_gateway_id", edge.EdgeGateway.ID)

	d.SetId(edge.EdgeGateway.ID)

	return []*schema.ResourceData{d}, nil
}

func getNsxtAlbConfigurationType(d *schema.ResourceData) *types.NsxtAlbConfig {
	albConfig := &types.NsxtAlbConfig{
		Enabled:                  d.Get("is_active").(bool),
		ServiceNetworkDefinition: d.Get("service_network_specification").(string),
	}

	if supportedFeatureSet, ok := d.GetOk("supported_feature_set"); ok {
		albConfig.SupportedFeatureSet = takeStringPointer(supportedFeatureSet.(string))
	}

	return albConfig
}

func setNsxtAlbConfigurationData(config *types.NsxtAlbConfig, d *schema.ResourceData) {
	dSet(d, "is_active", config.Enabled)
	dSet(d, "service_network_specification", config.ServiceNetworkDefinition)
	if config.SupportedFeatureSet != nil {
		dSet(d, "supported_feature_set", *config.SupportedFeatureSet)
	}
}
