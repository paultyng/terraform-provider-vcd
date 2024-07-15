package vcd

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/vmware/go-vcloud-director/v2/govcd"
	"github.com/vmware/go-vcloud-director/v2/types/v56"
)

func resourceVcdExternalEndpoint() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceVcdExternalEndpointCreate,
		ReadContext:   resourceVcdExternalEndpointRead,
		UpdateContext: resourceVcdExternalEndpointUpdate,
		DeleteContext: resourceVcdExternalEndpointDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceVcdExternalEndpointImport,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the External Endpoint",
			},
			"vendor": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Vendor of the External Endpoint",
			},
			"version": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Version of the External Endpoint",
			},
			"enabled": {
				Type:        schema.TypeBool,
				Required:    true,
				Description: "Whether the External Endpoint is enabled or not",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Description of the External Endpoint",
			},
			"root_url": {
				Type:             schema.TypeString,
				Required:         true,
				Description:      "The URL which requests will be redirected to. It must be a valid URL using https protocol",
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsURLWithHTTPS),
			},
		},
	}
}

func resourceVcdExternalEndpointCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vcdClient := meta.(*VCDClient)
	_, err := vcdClient.CreateExternalEndpoint(&types.ExternalEndpoint{
		Name:        d.Get("name").(string),
		Version:     d.Get("version").(string),
		Vendor:      d.Get("vendor").(string),
		Enabled:     d.Get("enabled").(bool),
		Description: d.Get("description").(string),
		RootUrl:     d.Get("root_url").(string),
	})
	if err != nil {
		return diag.Errorf("could not create the External Endpoint: %s", err)
	}
	return resourceVcdExternalEndpointRead(ctx, d, meta)
}

func resourceVcdExternalEndpointRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vcdClient := meta.(*VCDClient)

	var ep *govcd.ExternalEndpoint
	var err error
	if d.Id() == "" {
		ep, err = vcdClient.GetExternalEndpoint(d.Get("vendor").(string), d.Get("name").(string), d.Get("version").(string))
	} else {
		ep, err = vcdClient.GetExternalEndpointById(d.Id())
	}
	if err != nil {
		return diag.Errorf("could not read the External Endpoint: %s", err)
	}

	dSet(d, "name", ep.ExternalEndpoint.Name)
	dSet(d, "vendor", ep.ExternalEndpoint.Vendor)
	dSet(d, "version", ep.ExternalEndpoint.Version)
	dSet(d, "enabled", ep.ExternalEndpoint.Enabled)
	dSet(d, "description", ep.ExternalEndpoint.Description)
	dSet(d, "root_url", ep.ExternalEndpoint.RootUrl)
	d.SetId(ep.ExternalEndpoint.ID)
	return nil
}

func resourceVcdExternalEndpointUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vcdClient := meta.(*VCDClient)
	ep, err := vcdClient.GetExternalEndpointById(d.Id())
	if err != nil {
		return diag.Errorf("could not retrieve the External Endpoint for update: %s", err)
	}
	err = ep.Update(types.ExternalEndpoint{
		Name:        d.Get("name").(string),
		Version:     d.Get("version").(string),
		Vendor:      d.Get("vendor").(string),
		Enabled:     d.Get("enabled").(bool),
		Description: d.Get("description").(string),
		RootUrl:     d.Get("root_url").(string),
	})
	if err != nil {
		return diag.Errorf("could not update the External Endpoint: %s", err)
	}
	return resourceVcdExternalEndpointRead(ctx, d, meta)
}

func resourceVcdExternalEndpointDelete(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vcdClient := meta.(*VCDClient)
	ep, err := vcdClient.GetExternalEndpointById(d.Id())
	if err != nil {
		return diag.Errorf("could not retrieve the External Endpoint for update: %s", err)
	}
	err = ep.Delete()
	if err != nil {
		return diag.Errorf("could not delete the External Endpoint: %s", err)
	}
	return nil
}

func resourceVcdExternalEndpointImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	return nil, nil
}