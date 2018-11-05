package vcd

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	govcd "github.com/ukcloud/govcloudair"
)

func resourceVcdDNAT() *schema.Resource {
	return &schema.Resource{
		Create: resourceVcdDNATCreate,
		Delete: resourceVcdDNATDelete,
		Read:   resourceVcdDNATRead,

		Schema: map[string]*schema.Schema{
			"edge_gateway": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"org": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"vdc": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"external_ip": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"port": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},

			"translated_port": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},

			"internal_ip": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceVcdDNATCreate(d *schema.ResourceData, meta interface{}) error {
	vcdClient := meta.(*VCDClient)
	org, err := govcd.GetOrgFromName(vcdClient.VCDClient, d.Get("org").(string))
	if err != nil {
		return fmt.Errorf("Could not find Org: %v", err)
	}
	vdc, err := org.GetVDCFromName(d.Get("vdc").(string))
	if err != nil {
		return fmt.Errorf("Could not find vdc: %v", err)
	}

	// Multiple VCD components need to run operations on the Edge Gateway, as
	// the edge gatway will throw back an error if it is already performing an
	// operation we must wait until we can aquire a lock on the client
	vcdClient.Mutex.Lock()
	defer vcdClient.Mutex.Unlock()
	portString := getPortString(d.Get("port").(int))
	translatedPortString := portString // default
	if d.Get("translated_port").(int) > 0 {
		translatedPortString = getPortString(d.Get("translated_port").(int))
	}

	edgeGateway, err := vdc.FindEdgeGateway(d.Get("edge_gateway").(string))

	if err != nil {
		return fmt.Errorf("Unable to find edge gateway: %#v", err)
	}

	// Creating a loop to offer further protection from the edge gateway erroring
	// due to being busy eg another person is using another client so wouldn't be
	// constrained by out lock. If the edge gateway reurns with a busy error, wait
	// 3 seconds and then try again. Continue until a non-busy error or success

	err = retryCall(vcdClient.MaxRetryTimeout, func() *resource.RetryError {
		task, err := edgeGateway.AddNATPortMapping("DNAT",
			d.Get("external_ip").(string),
			portString,
			d.Get("internal_ip").(string),
			translatedPortString)
		if err != nil {
			return resource.RetryableError(
				fmt.Errorf("Error setting DNAT rules: %#v", err))
		}

		return resource.RetryableError(task.WaitTaskCompletion())
	})

	if err != nil {
		return fmt.Errorf("Error completing tasks: %#v", err)
	}

	d.SetId(d.Get("external_ip").(string) + ":" + portString + " > " + d.Get("internal_ip").(string) + ":" + translatedPortString)
	return nil
}

func resourceVcdDNATRead(d *schema.ResourceData, meta interface{}) error {
	vcdClient := meta.(*VCDClient)

	org, err := govcd.GetOrgFromName(vcdClient.VCDClient, d.Get("org").(string))
	if err != nil {
		return fmt.Errorf("Could not find Org: %v", err)
	}
	vdc, err := org.GetVDCFromName(d.Get("vdc").(string))
	if err != nil {
		return fmt.Errorf("Could not find vdc: %v", err)
	}

	e, err := vdc.FindEdgeGateway(d.Get("edge_gateway").(string))

	if err != nil {
		return fmt.Errorf("Unable to find edge gateway: %#v", err)
	}

	var found bool

	for _, r := range e.EdgeGateway.Configuration.EdgeGatewayServiceConfiguration.NatService.NatRule {
		if r.RuleType == "DNAT" &&
			r.GatewayNatRule.OriginalIP == d.Get("external_ip").(string) &&
			r.GatewayNatRule.OriginalPort == getPortString(d.Get("port").(int)) {
			found = true
			d.Set("internal_ip", r.GatewayNatRule.TranslatedIP)
		}
	}

	if !found {
		d.SetId("")
	}

	return nil
}

func resourceVcdDNATDelete(d *schema.ResourceData, meta interface{}) error {
	vcdClient := meta.(*VCDClient)
	org, err := govcd.GetOrgFromName(vcdClient.VCDClient, d.Get("org").(string))
	if err != nil {
		return fmt.Errorf("Could not find Org: %v", err)
	}
	vdc, err := org.GetVDCFromName(d.Get("vdc").(string))
	if err != nil {
		return fmt.Errorf("Could not find vdc: %v", err)
	}

	// Multiple VCD components need to run operations on the Edge Gateway, as
	// the edge gatway will throw back an error if it is already performing an
	// operation we must wait until we can aquire a lock on the client
	vcdClient.Mutex.Lock()
	defer vcdClient.Mutex.Unlock()
	portString := getPortString(d.Get("port").(int))
	translatedPortString := portString // default
	if d.Get("translated_port").(int) > 0 {
		translatedPortString = getPortString(d.Get("translated_port").(int))
	}

	edgeGateway, err := vdc.FindEdgeGateway(d.Get("edge_gateway").(string))

	if err != nil {
		return fmt.Errorf("Unable to find edge gateway: %#v", err)
	}
	err = retryCall(vcdClient.MaxRetryTimeout, func() *resource.RetryError {
		task, err := edgeGateway.RemoveNATPortMapping("DNAT",
			d.Get("external_ip").(string),
			portString,
			d.Get("internal_ip").(string),
			translatedPortString)
		if err != nil {
			return resource.RetryableError(
				fmt.Errorf("Error setting DNAT rules: %#v", err))
		}

		return resource.RetryableError(task.WaitTaskCompletion())
	})
	if err != nil {
		return fmt.Errorf("Error completing tasks: %#v", err)
	}
	return nil
}
