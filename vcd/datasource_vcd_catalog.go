package vcd

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/vmware/go-vcloud-director/v2/govcd"
)

func datasourceVcdCatalog() *schema.Resource {
	return &schema.Resource{
		ReadContext: datasourceVcdCatalogRead,
		Schema: map[string]*schema.Schema{
			"org": {
				Type:     schema.TypeString,
				Optional: true,
				Description: "The name of organization to use, optional if defined at provider " +
					"level. Useful when connected as sysadmin working across different organizations",
			},
			"name": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "Name of the catalog. (Optional if 'filter' is used)",
				ExactlyOneOf: []string{"name", "filter"},
			},
			"storage_profile_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Storage profile ID",
			},
			"created": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Time stamp of when the catalog was created",
			},

			"description": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"publish_enabled": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "True allows to publish a catalog externally to make its vApp templates and media files available for subscription by organizations outside the Cloud Director installation. Default is `false`.",
			},
			"cache_enabled": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "True enables early catalog export to optimize synchronization",
			},
			"preserve_identity_information": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Include BIOS UUIDs and MAC addresses in the downloaded OVF package. Preserving the identity information limits the portability of the package and you should use it only when necessary.",
			},
			"metadata": {
				Type:        schema.TypeMap,
				Computed:    true,
				Description: "Key and value pairs for catalog metadata",
				Deprecated:  "Use metadata_entry instead",
			},
			"metadata_entry": getMetadataEntrySchema("Catalog", true),
			"catalog_version": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Catalog version number.",
			},
			"owner_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Owner name from the catalog.",
			},
			"number_of_vapp_templates": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Number of vApps this catalog contains.",
			},
			"number_of_media": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Number of Medias this catalog contains.",
			},
			"is_shared": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "True if this catalog is shared.",
			},
			"is_published": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "True if this catalog is shared to all organizations.",
			},
			"publish_subscription_type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "PUBLISHED if published externally, SUBSCRIBED if subscribed to an external catalog, UNPUBLISHED otherwise.",
			},
			"filter": {
				Type:        schema.TypeList,
				MaxItems:    1,
				MinItems:    1,
				Optional:    true,
				Description: "Criteria for retrieving a catalog by various attributes",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name_regex": elementNameRegex,
						"date":       elementDate,
						"earliest":   elementEarliest,
						"latest":     elementLatest,
						"metadata":   elementMetadata,
					},
				},
			},
		},
	}
}

func datasourceVcdCatalogRead(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var (
		vcdClient = meta.(*VCDClient)
		catalog   *govcd.AdminCatalog
	)

	if !nameOrFilterIsSet(d) {
		return diag.Errorf(noNameOrFilterError, "vcd_catalog")
	}

	adminOrg, err := vcdClient.GetAdminOrgFromResource(d)
	if err != nil {
		return diag.Errorf(errorRetrievingOrg, err)
	}
	log.Printf("[TRACE] Org %s found", adminOrg.AdminOrg.Name)

	identifier := d.Get("name").(string)

	filter, hasFilter := d.GetOk("filter")

	if hasFilter {
		catalog, err = getCatalogByFilter(adminOrg, filter, vcdClient.Client.IsSysAdmin)
	} else {
		catalog, err = adminOrg.GetAdminCatalogByNameOrId(identifier, false)
	}
	if err != nil {
		log.Printf("[DEBUG] Catalog %s not found. Setting ID to nothing", identifier)
		return diag.Errorf("error retrieving catalog %s: %s", identifier, err)
	}

	dSet(d, "description", catalog.AdminCatalog.Description)
	dSet(d, "created", catalog.AdminCatalog.DateCreated)
	dSet(d, "name", catalog.AdminCatalog.Name)

	d.SetId(catalog.AdminCatalog.ID)
	if catalog.AdminCatalog.PublishExternalCatalogParams != nil {
		dSet(d, "publish_enabled", catalog.AdminCatalog.PublishExternalCatalogParams.IsPublishedExternally)
		dSet(d, "cache_enabled", catalog.AdminCatalog.PublishExternalCatalogParams.IsCachedEnabled)
		dSet(d, "preserve_identity_information", catalog.AdminCatalog.PublishExternalCatalogParams.PreserveIdentityInfoFlag)
	}

	err = updateMetadataInState(d, catalog, "datasource")
	if err != nil {
		log.Printf("[DEBUG] Unable to set catalog metadata: %s", err)
		return diag.Errorf("There was an issue when setting metadata - %s", err)
	}

	err = setCatalogData(d, adminOrg, catalog.AdminCatalog.Name)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}
