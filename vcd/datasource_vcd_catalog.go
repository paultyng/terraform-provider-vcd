package vcd

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/vmware/go-vcloud-director/v2/govcd"
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
				Optional:    true,
				Description: "Name of the catalog. (Optional if 'filter' is used)",
			},
			"created": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Time stamp of when the catalog was created",
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"filter": &schema.Schema{
				Type:        schema.TypeList,
				MaxItems:    1,
				MinItems:    1,
				Optional:    true,
				Description: "Criteria for retrieving a catalog by various attributes",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name_regex": elementNameRegex,
						"date":       elementDate,
						"latest":     elementLatest,
						"metadata":   elementMetadata,
					},
				},
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

	filter, hasFilter := d.GetOk("filter")

	var catalog *govcd.Catalog

	if hasFilter {
		criteria, err := buildCriteria(filter)
		if err != nil {
			return err
		}
		queryType := govcd.QtAdminCatalog
		queryItems, err := vcdClient.Client.SearchByFilter(queryType, criteria)
		if err != nil {
			return err
		}
		if len(queryItems) == 0 {
			return fmt.Errorf("no catalogs found with given criteria")
		}
		if len(queryItems) > 1 {
			var itemNames = make([]string, len(queryItems))
			for i, item := range queryItems {
				itemNames[i] = item.GetName()
			}
			return fmt.Errorf("more than one catalog found by given criteria: %v", itemNames)
		}
		catalog, err = adminOrg.GetCatalogByHref(queryItems[0].GetHref())
	} else {
		catalog, err = adminOrg.GetCatalogByNameOrId(identifier, false)
	}
	if err != nil {
		log.Printf("[DEBUG] Catalog %s not found. Setting ID to nothing", identifier)
		d.SetId("")
		return fmt.Errorf("error retrieving catalog %s: %s", identifier, err)
	}

	_ = d.Set("description", catalog.Catalog.Description)
	_ = d.Set("created", catalog.Catalog.DateCreated)
	_ = d.Set("name", catalog.Catalog.Name)
	d.SetId(catalog.Catalog.ID)
	return nil
}
