package vcd

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func datasourceVcdNsxtEdgegatewayL2VpnTunnel() *schema.Resource {
	return &schema.Resource{
		ReadContext: datasourceVcdNsxtEdgegatewayL2VpnTunnelRead,
		Schema: map[string]*schema.Schema{
			"org": {
				Type:     schema.TypeString,
				Optional: true,
				Description: "The name of organization to use, optional if defined at provider " +
					"level. Useful when connected as sysadmin working across different organizations",
			},
			"edge_gateway_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Edge gateway ID for the tunnel",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the L2 VPN Tunnel session",
			},
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Description of the L2 VPN Tunnel session",
			},
			"session_mode": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Mode of the tunnel session, must be CLIENT or SERVER",
			},
			"enabled": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Status of the L2 VPN Tunnel session",
			},
			"local_endpoint_ip": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Local endpoint IP of the tunnel session, the IP must be sub-allocated to the Edge Gateway",
			},
			"remote_endpoint_ip": {
				Type:     schema.TypeString,
				Computed: true,
				Description: "The IP address of the remote endpoint, which corresponds to the device" +
					"on the remote site terminating the VPN tunnel.",
			},
			"tunnel_interface": {
				Type:     schema.TypeString,
				Computed: true,
				Description: "Network CIDR block over which the session interfaces. Only relevant if " +
					"`session_mode` is set to `SERVER`",
			},
			"connector_initiation_mode": {
				Type:     schema.TypeString,
				Optional: true,
				Description: "Connector initation mode of the session describing how a connection is made. " +
					"Needs to be set only if `session_mode` is set to `SERVER`",
			},
			"pre_shared_key": {
				Type:     schema.TypeString,
				Computed: true,
				Description: "Pre-shared key used for authentication, needs to be provided only on creation of" +
					"`SERVER` sessions",
			},
			"peer_code": {
				Type:     schema.TypeString,
				Computed: true,
				Description: "Base64 encoded string of the full configuration of the tunnel, " +
					"generated by the server-side L2 VPN session. This value contains the encoded pre-shared key " +
					"so it is user's responsibility to secure it. Needs to be provided only on creation of" +
					"`CLIENT` sessions",
			},
			"stretched_network": {
				Type:     schema.TypeSet,
				Computed: true,
				// DHCP forwarding supports up to 8 IP addresses
				Description: "Org VDC networks that are attached to this L2 VPN tunnel",
				Elem:        stretchedNetwork,
			},
		},
	}
}

func datasourceVcdNsxtEdgegatewayL2VpnTunnelRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vcdClient := meta.(*VCDClient)

	orgName := d.Get("org").(string)
	edgeGatewayId := d.Get("edge_gateway_id").(string)
	nsxtEdge, err := vcdClient.GetNsxtEdgeGatewayById(orgName, edgeGatewayId)
	if err != nil {
		return diag.Errorf("[L2 VPN Tunnel DS read] error retrieving edge gateway: %s", err)
	}

	// Getting tunnel by name doesn't provide all the information (peer_code, pre_shared_key) so we need to get it by ID after
	// for all the data
	tunnelName := d.Get("name").(string)
	tunnelConfig, err := nsxtEdge.GetL2VpnTunnelByName(tunnelName)
	if err != nil {
		return diag.Errorf("[L2 VPN Tunnel DS read] error retrieving tunnel by name: %s", err)
	}

	tunnel, err := nsxtEdge.GetL2VpnTunnelById(tunnelConfig.NsxtL2VpnTunnel.ID)
	if err != nil {
		return diag.Errorf("[L2 VPN Tunnel DS read] error retrieving tunnel by ID: %s", err)
	}

	err = readL2VpnTunnelToSchema(tunnel.NsxtL2VpnTunnel, d)
	if err != nil {
		return diag.Errorf("[L2 VPN Tunnel DS read] error reading retrieved tunnel into schema: %s", err)
	}

	return nil
}
