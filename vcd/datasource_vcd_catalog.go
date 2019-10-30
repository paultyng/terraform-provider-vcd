package vcd

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func datasourceVcdCatalog() *schema.Resource {
	return &schema.Resource{
		Read: datasourceVcdCatalogRead,
		Schema: map[string]*schema.Schema{
			"org": {
				Type:     schema.TypeString,
				Required: true,
				Description: "The name of organization to use, optional if defined at provider " +
					"level. Useful when connected as sysadmin working across different organizations",
			},
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the catalog",
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func datasourceVcdCatalogRead(d *schema.ResourceData, meta interface{}) error {
	vcdClient := meta.(*VCDClient)

	orgName := d.Get("org").(string)
	identifier := d.Get("name").(string)
	log.Printf("[TRACE] Reading Org %s", orgName)
	adminOrg, err := vcdClient.VCDClient.GetAdminOrgByName(orgName)

	if err != nil {
		log.Printf("[DEBUG] Org %s not found. Setting ID to nothing", orgName)
		d.SetId("")
		return nil
	}
	log.Printf("[TRACE] Org %s found", orgName)

	catalog, err := adminOrg.GetCatalogByNameOrId(identifier, false)
	if err != nil {
		log.Printf("[DEBUG] Catalog %s not found. Setting ID to nothing", identifier)
		d.SetId("")
		return fmt.Errorf("error retrieving catalog %s", identifier)
	}

	_ = d.Set("description", catalog.Catalog.Description)
	_ = d.Set("name", catalog.Catalog.Name)
	d.SetId(catalog.Catalog.ID)
	return nil
}
