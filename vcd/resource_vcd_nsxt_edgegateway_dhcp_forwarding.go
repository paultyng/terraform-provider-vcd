package vcd

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/vmware/go-vcloud-director/v2/govcd"
	"github.com/vmware/go-vcloud-director/v2/types/v56"
)

func resourceVcdNsxtEdgegatewayDhcpForwarding() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceVcdNsxtEdgegatewayDhcpForwardingCreateUpdate,
		UpdateContext: resourceVcdNsxtEdgegatewayDhcpForwardingCreateUpdate,
		ReadContext:   resourceVcdNsxtEdgegatewayDhcpForwardingRead,
		DeleteContext: resourceVcdNsxtEdgegatewayDhcpForwardingDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceVcdNsxtEdgegatewayDhcpForwardingImport,
		},

		Schema: map[string]*schema.Schema{
			"org": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Description: "The name of organization to use, optional if defined at provider " +
					"level. Useful when connected as sysadmin working across different organizations",
			},
			"edge_gateway_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Edge gateway ID for Rate limiting (QoS) configuration",
			},
			"enabled": {
				Type:        schema.TypeBool,
				Required:    true,
				Description: "Status of DHCP Forwarding for the Edge Gateway",
			},
			"dhcp_servers": {
				Type:        schema.TypeSet,
				Required:    true,
				Description: "IP addresses of the DHCP servers",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func resourceVcdNsxtEdgegatewayDhcpForwardingCreateUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vcdClient := meta.(*VCDClient)

	// Handling locks is conditional. There are two scenarios:
	// * When the parent Edge Gateway is in a VDC - a lock on parent Edge Gateway must be acquired
	// * When the parent Edge Gateway is in a VDC Group - a lock on parent VDC Group must be acquired
	// To find out parent lock object, Edge Gateway must be looked up and its OwnerRef must be checked
	// Note. It is not safe to do multiple locks in the same resource as it can result in a deadlock
	parentEdgeGatewayOwnerId, _, err := getParentEdgeGatewayOwnerId(vcdClient, d)
	if err != nil {
		return diag.Errorf("[DHCP forwarding create/update] error finding parent Edge Gateway: %s", err)
	}

	if govcd.OwnerIsVdcGroup(parentEdgeGatewayOwnerId) {
		vcdClient.lockById(parentEdgeGatewayOwnerId)
		defer vcdClient.unlockById(parentEdgeGatewayOwnerId)
	} else {
		vcdClient.lockParentEdgeGtw(d)
		defer vcdClient.unLockParentEdgeGtw(d)
	}

	orgName := d.Get("org").(string)
	edgeGatewayId := d.Get("edge_gateway_id").(string)

	nsxtEdge, err := vcdClient.GetNsxtEdgeGatewayById(orgName, edgeGatewayId)
	if err != nil {
		return diag.Errorf("[DHCP forwarding create/update] error retrieving Edge Gateway: %s", err)
	}

	dhcpForwardingConfig := &types.NsxtEdgeGatewayDhcpForwarder{
		Enabled:     d.Get("enabled").(bool),
		DhcpServers: convertSchemaSetToSliceOfStrings(d.Get("dhcp_servers").(*schema.Set)),
	}

	_, err = nsxtEdge.UpdateDhcpForwarder(dhcpForwardingConfig)
	if err != nil {
		return diag.Errorf("[DHCP forwarding create/update] error updating QoS configuration: %s", err)
	}

	d.SetId(edgeGatewayId)

	return resourceVcdNsxtEdgegatewayDhcpForwardingRead(ctx, d, meta)
}

func resourceVcdNsxtEdgegatewayDhcpForwardingRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vcdClient := meta.(*VCDClient)

	orgName := d.Get("org").(string)
	edgeGatewayId := d.Get("edge_gateway_id").(string)

	nsxtEdge, err := vcdClient.GetNsxtEdgeGatewayById(orgName, edgeGatewayId)
	if err != nil {
		if govcd.ContainsNotFound(err) {
			// When parent Edge Gateway is not found - this resource is also not found and should be
			// removed from state
			d.SetId("")
			return nil
		}
		return diag.Errorf("[DHCP forwarding read] error retrieving NSX-T Edge Gateway rate limiting (QoS): %s", err)
	}

	dhcpForwardingConfig, err := nsxtEdge.GetDhcpForwarder()
	if err != nil {
		return diag.Errorf("[DHCP forwarding read] error retrieving NSX-T Edge Gateway rate limiting (QoS): %s", err)
	}
	dSet(d, "enabled", dhcpForwardingConfig.Enabled)
	dSet(d, "dhcp_servers", convertStringsToTypeSet(dhcpForwardingConfig.DhcpServers))

	if !dhcpForwardingConfig.Enabled {
		return diag.Diagnostics{
			diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  "DHCP forwarding IP addresses will not be changed if the service is disabled",
			},
		}
	}

	return nil
}

func resourceVcdNsxtEdgegatewayDhcpForwardingDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vcdClient := meta.(*VCDClient)

	// Handling locks is conditional. There are two scenarios:
	// * When the parent Edge Gateway is in a VDC - a lock on parent Edge Gateway must be acquired
	// * When the parent Edge Gateway is in a VDC Group - a lock on parent VDC Group must be acquired
	// To find out parent lock object, Edge Gateway must be looked up and its OwnerRef must be checked
	// Note. It is not safe to do multiple locks in the same resource as it can result in a deadlock
	parentEdgeGatewayOwnerId, _, err := getParentEdgeGatewayOwnerId(vcdClient, d)
	if err != nil {
		return diag.Errorf("[DHCP forwarding delete] error finding parent Edge Gateway: %s", err)
	}

	if govcd.OwnerIsVdcGroup(parentEdgeGatewayOwnerId) {
		vcdClient.lockById(parentEdgeGatewayOwnerId)
		defer vcdClient.unlockById(parentEdgeGatewayOwnerId)
	} else {
		vcdClient.lockParentEdgeGtw(d)
		defer vcdClient.unLockParentEdgeGtw(d)
	}

	orgName := d.Get("org").(string)
	edgeGatewayId := d.Get("edge_gateway_id").(string)

	nsxtEdge, err := vcdClient.GetNsxtEdgeGatewayById(orgName, edgeGatewayId)
	if err != nil {
		return diag.Errorf("[DHCP forwarding delete] error retrieving Edge Gateway: %s", err)
	}

	// There is no real "delete" for QoS. It can only be updated to empty values (unlimited)
	_, err = nsxtEdge.UpdateDhcpForwarder(&types.NsxtEdgeGatewayDhcpForwarder{})
	if err != nil {
		return diag.Errorf("[DHCP forwarding delete] error updating QoS Profile: %s", err)
	}

	return nil
}

// Edge Gateway.
// The import path for this resource is Edge Gateway. ID of the field is also Edge Gateway ID as it
// rate limiting is a property of Edge Gateway, not a separate entity.
func resourceVcdNsxtEdgegatewayDhcpForwardingImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	log.Printf("[TRACE] NSX-T Edge Gateway DHCP forwarding import initiated")

	resourceURI := strings.Split(d.Id(), ImportSeparator)
	if len(resourceURI) != 3 {
		return nil, fmt.Errorf("resource name must be specified as org-name.vdc-name.nsxt-edge-gw-name or org-name.vdc-group-name.nsxt-edge-gw-name")
	}
	orgName, vdcOrVdcGroupName, edgeName := resourceURI[0], resourceURI[1], resourceURI[2]

	vcdClient := meta.(*VCDClient)
	vdcOrVdcGroup, err := lookupVdcOrVdcGroup(vcdClient, orgName, vdcOrVdcGroupName)
	if err != nil {
		return nil, err
	}

	if !vdcOrVdcGroup.IsNsxt() {
		return nil, fmt.Errorf("please use 'vcd_edgegateway' for NSX-V backed VDC")
	}

	edge, err := vdcOrVdcGroup.GetNsxtEdgeGatewayByName(edgeName)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve NSX-T Edge Gateway with ID '%s': %s", d.Id(), err)
	}

	dSet(d, "org", orgName)
	dSet(d, "edge_gateway_id", edge.EdgeGateway.ID)

	// Storing Edge Gateway ID and Read will retrieve all other data
	d.SetId(edge.EdgeGateway.ID)

	return []*schema.ResourceData{d}, nil
}
