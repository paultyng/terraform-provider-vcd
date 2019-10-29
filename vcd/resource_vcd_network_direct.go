package vcd

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/vmware/go-vcloud-director/v2/types/v56"
)

func resourceVcdNetworkDirect() *schema.Resource {
	return &schema.Resource{
		Create: resourceVcdNetworkDirectCreate,
		Read:   resourceVcdNetworkDirectRead,
		Delete: resourceVcdNetworkDelete,
		Importer: &schema.ResourceImporter{
			State: resourceVcdNetworkDirectImport,
		},
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "A unique name for this network",
			},
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
				ForceNew:    true,
				Description: "The name of VDC to use, optional if defined at provider level",
			},
			"external_network": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of the external network",
			},
			"external_network_gateway": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Gateway of the external network",
			},
			"external_network_netmask": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Net mask of the external network",
			},
			"external_network_dns1": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Main DNS of the external network",
			},
			"external_network_dns2": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Secondary DNS of the external network",
			},
			"external_network_dns_suffix": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "DNS suffix of the external network",
			},
			"href": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Network Hypertext Reference",
			},
			"shared": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				ForceNew:    true,
				Description: "Defines if this network is shared between multiple VDCs in the Org",
			},
		},
	}
}

func resourceVcdNetworkDirectCreate(d *schema.ResourceData, meta interface{}) error {
	vcdClient := meta.(*VCDClient)

	_, vdc, err := vcdClient.GetOrgAndVdcFromResource(d)
	if err != nil {
		return fmt.Errorf(errorRetrievingOrgAndVdc, err)
	}

	externalNetworkName := d.Get("external_network").(string)
	networkName := d.Get("name").(string)
	externalNetwork, err := vcdClient.GetExternalNetworkByName(externalNetworkName)
	if err != nil {
		return fmt.Errorf("unable to find external network %s (%s)", externalNetworkName, err)
	}

	orgVDCNetwork := &types.OrgVDCNetwork{
		Xmlns: "http://www.vmware.com/vcloud/v1.5",
		Name:  networkName,
		Configuration: &types.NetworkConfiguration{
			ParentNetwork: &types.Reference{
				HREF: externalNetwork.ExternalNetwork.HREF,
				Type: externalNetwork.ExternalNetwork.Type,
				Name: externalNetwork.ExternalNetwork.Name,
			},
			FenceMode:                 "bridged",
			BackwardCompatibilityMode: true,
		},
		IsShared: d.Get("shared").(bool),
	}

	err = vdc.CreateOrgVDCNetworkWait(orgVDCNetwork)
	if err != nil {
		return fmt.Errorf("error: %s", err)
	}

	network, err := vdc.GetOrgVdcNetworkByName(networkName, true)
	if err != nil {
		return fmt.Errorf("error retrieving network %s after creation", networkName)
	}
	d.SetId(network.OrgVDCNetwork.ID)
	return resourceVcdNetworkDirectRead(d, meta)
}

func resourceVcdNetworkDirectRead(d *schema.ResourceData, meta interface{}) error {
	return genericVcdNetworkDirectRead(d, meta, "resource")
}

func genericVcdNetworkDirectRead(d *schema.ResourceData, meta interface{}, origin string) error {
	vcdClient := meta.(*VCDClient)

	_, vdc, err := vcdClient.GetOrgAndVdcFromResource(d)
	if err != nil {
		return fmt.Errorf("[network direct read] "+errorRetrievingOrgAndVdc, err)
	}

	identifier := d.Id()

	if identifier == "" {
		identifier = d.Get("name").(string)
	}
	network, err := vdc.GetOrgVdcNetworkByNameOrId(identifier, false)
	if err != nil {
		if origin == "resource" {
			log.Printf("[DEBUG] Network %s no longer exists. Removing from tfstate", identifier)
			d.SetId("")
			return nil
		}
		return fmt.Errorf("[network direct read] network %s not found: %s", identifier, err)
	}

	_ = d.Set("name", network.OrgVDCNetwork.Name)
	_ = d.Set("href", network.OrgVDCNetwork.HREF)
	_ = d.Set("shared", network.OrgVDCNetwork.IsShared)

	parentNetwork := network.OrgVDCNetwork.Configuration.ParentNetwork
	if parentNetwork == nil {
		return fmt.Errorf("[network direct read] no parent network found for %s", network.OrgVDCNetwork.Name)
	}
	_ = d.Set("external_network", parentNetwork.Name)

	externalNetwork, err := vcdClient.GetExternalNetworkByName(parentNetwork.Name)
	if err != nil {
		return fmt.Errorf("[network direct read] error fetching external network %s ", parentNetwork.Name)
	}

	enConf := externalNetwork.ExternalNetwork.Configuration
	if enConf == nil || enConf.IPScopes == nil || len(enConf.IPScopes.IPScope) == 0 {
		return fmt.Errorf("[network direct read] error retrieving details from external network %s", externalNetwork.ExternalNetwork.Name)
	}
	_ = d.Set("external_network_gateway", externalNetwork.ExternalNetwork.Configuration.IPScopes.IPScope[0].Gateway)
	_ = d.Set("external_network_netmask", externalNetwork.ExternalNetwork.Configuration.IPScopes.IPScope[0].Netmask)
	_ = d.Set("external_network_dns1", externalNetwork.ExternalNetwork.Configuration.IPScopes.IPScope[0].DNS1)
	_ = d.Set("external_network_dns2", externalNetwork.ExternalNetwork.Configuration.IPScopes.IPScope[0].DNS2)
	_ = d.Set("external_network_dns_suffix", externalNetwork.ExternalNetwork.Configuration.IPScopes.IPScope[0].DNSSuffix)

	d.SetId(network.OrgVDCNetwork.ID)

	return nil
}

// resourceVcdNetworkDirectImport is responsible for importing the resource.
// The following steps happen as part of import
// 1. The user supplies `terraform import _resource_name_ _the_id_string_` command
// 2. `_the_id_string_` contains a dot formatted path to resource as in the example below
// 3. The functions splits the dot-formatted path and tries to lookup the object
// 4. If the lookup succeeds it sets the ID field for `_resource_name_` resource in statefile
// (the resource must be already defined in .tf config otherwise `terraform import` will complain)
// 5. `terraform refresh` is being implicitly launched. The Read method looks up all other fields
// based on the known ID of object.
//
// Example resource name (_resource_name_): vcd_network_direct.my-network
// Example import path (_the_id_string_): org.vdc.my-network
// Note: the separator can be changed using Provider.import_separator or variable VCD_IMPORT_SEPARATOR
func resourceVcdNetworkDirectImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	resourceURI := strings.Split(d.Id(), ImportSeparator)
	if len(resourceURI) != 3 {
		return nil, fmt.Errorf("[network direct import] resource name must be specified as org-name.vdc-name.network-name")
	}
	orgName, vdcName, networkName := resourceURI[0], resourceURI[1], resourceURI[2]

	vcdClient := meta.(*VCDClient)
	_, vdc, err := vcdClient.GetOrgAndVdc(orgName, vdcName)
	if err != nil {
		return nil, fmt.Errorf("[network direct import] unable to find VDC %s: %s ", vdcName, err)
	}

	network, err := vdc.GetOrgVdcNetworkByName(networkName, false)
	if err != nil {
		return nil, fmt.Errorf("[network direct import] error retrieving network %s: %s", networkName, err)
	}
	parentNetwork := network.OrgVDCNetwork.Configuration.ParentNetwork
	if parentNetwork == nil || parentNetwork.Name == "" {
		return nil, fmt.Errorf("[network direct import] no parent network found for %s", network.OrgVDCNetwork.Name)
	}

	_ = d.Set("org", orgName)
	_ = d.Set("vdc", vdcName)
	_ = d.Set("external_network", parentNetwork.Name)
	d.SetId(network.OrgVDCNetwork.ID)
	return []*schema.ResourceData{d}, nil
}
