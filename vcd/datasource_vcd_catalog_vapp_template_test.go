//go:build catalog || ALL || functional
// +build catalog ALL functional

package vcd

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// Test catalog and vApp Template data sources
// Using a catalog data source we reference a vApp Template data source
// Using a vApp Template data source we create another vApp Template
// where the description is the first data source ID.
func TestAccVcdCatalogAndVappTemplateDatasource(t *testing.T) {
	preTestChecks(t)
	createdVAppTemplateName := t.Name()

	var params = StringMap{
		"Org":             testConfig.VCD.Org,
		"Vdc":             testConfig.VCD.Vdc,
		"Catalog":         testSuiteCatalogName,
		"VAppTemplate":    testSuiteCatalogOVAItem,
		"NewVappTemplate": createdVAppTemplateName,
		"OvaPath":         testConfig.Ova.OvaPath,
		"UploadPieceSize": testConfig.Ova.UploadPieceSize,
		"Tags":            "catalog",
	}
	testParamsNotEmpty(t, params)

	configText := templateFill(testAccCheckVcdCatalogVAppTemplateDS, params)
	if vcdShortTest {
		t.Skip(acceptanceTestsSkipped)
		return
	}
	debugPrintf("#[DEBUG] CONFIGURATION: %s", configText)

	datasourceCatalog := "data.vcd_catalog." + params["Catalog"].(string)
	datasourceCatalogVappTemplate1 := "data.vcd_catalog_vapp_template." + params["VAppTemplate"].(string) + "_1"
	datasourceCatalogVappTemplate2 := "data.vcd_catalog_vapp_template." + params["VAppTemplate"].(string) + "_2"
	datasourceCatalogVappTemplate3 := "data.vcd_catalog_vapp_template." + params["VAppTemplate"].(string) + "_3"
	datasourceCatalogVappTemplate4 := "data.vcd_catalog_vapp_template." + params["VAppTemplate"].(string) + "_4"
	datasourceCatalogVappTemplate5 := "data.vcd_catalog_vapp_template." + params["VAppTemplate"].(string) + "_5"
	resourceCatalogVappTemplate := "vcd_catalog_vapp_template." + params["NewVappTemplate"].(string)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { preRunChecks(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      catalogVAppTemplateDestroyed(testSuiteCatalogName, createdVAppTemplateName),
		Steps: []resource.TestStep{
			{
				Config: configText,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVcdVAppTemplateExists(resourceCatalogVappTemplate),
					resource.TestCheckResourceAttr(resourceCatalogVappTemplate, "name", createdVAppTemplateName),
					resource.TestCheckResourceAttrPair(datasourceCatalog, "id", resourceCatalogVappTemplate, "catalog_id"),

					// The description of the new catalog item was created using the ID of the catalog item data source
					resource.TestMatchResourceAttr(datasourceCatalogVappTemplate1, "id", regexp.MustCompile(`urn:vcloud:vapptemplate:[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`)),
					resource.TestMatchResourceAttr(datasourceCatalogVappTemplate1, "catalog_id", regexp.MustCompile(`urn:vcloud:catalog:[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`)),
					resource.TestMatchResourceAttr(datasourceCatalogVappTemplate1, "vdc_id", regexp.MustCompile(`urn:vcloud:vdc:[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`)),
					resource.TestCheckResourceAttrPair(datasourceCatalogVappTemplate1, "id", resourceCatalogVappTemplate, "description"), // Deprecated

					// Check both data sources fetched by VDC and Catalog ID are equal
					resource.TestCheckResourceAttrPair(datasourceCatalogVappTemplate1, "id", datasourceCatalogVappTemplate2, "id"),
					resource.TestCheckResourceAttrPair(datasourceCatalogVappTemplate1, "catalog_id", datasourceCatalogVappTemplate2, "catalog_id"),
					resource.TestCheckResourceAttrPair(datasourceCatalogVappTemplate1, "vdc_id", datasourceCatalogVappTemplate2, "vdc_id"),

					resource.TestCheckResourceAttr(resourceCatalogVappTemplate, "metadata.key1", "value1"),
					resource.TestCheckResourceAttr(resourceCatalogVappTemplate, "metadata.key2", "value2"),

					// Check data sources with filter. Not using resourceFieldsEqual here as we'd need to exclude all filtering options by hardcoding the combinations.
					resource.TestCheckResourceAttrPair(datasourceCatalogVappTemplate3, "id", datasourceCatalogVappTemplate1, "id"),
					resource.TestCheckResourceAttrPair(datasourceCatalogVappTemplate3, "catalog_id", datasourceCatalogVappTemplate1, "catalog_id"),
					resource.TestCheckResourceAttrPair(datasourceCatalogVappTemplate3, "vdc_id", datasourceCatalogVappTemplate1, "vdc_id"),
					resource.TestCheckResourceAttrPair(datasourceCatalogVappTemplate4, "id", datasourceCatalogVappTemplate1, "id"),
					resource.TestCheckResourceAttrPair(datasourceCatalogVappTemplate4, "catalog_id", datasourceCatalogVappTemplate1, "catalog_id"),
					resource.TestCheckResourceAttrPair(datasourceCatalogVappTemplate4, "vdc_id", datasourceCatalogVappTemplate1, "vdc_id"),
					resource.TestCheckResourceAttrPair(datasourceCatalogVappTemplate5, "id", resourceCatalogVappTemplate, "id"),
					resource.TestCheckResourceAttrPair(datasourceCatalogVappTemplate5, "catalog_id", resourceCatalogVappTemplate, "catalog_id"),
				),
			},
			{
				ResourceName:      resourceCatalogVappTemplate,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: importStateIdOrgCatalogObject(createdVAppTemplateName),
				// These fields can't be retrieved from vApp Template data
				ImportStateVerifyIgnore: []string{"ova_path", "upload_piece_size"},
			},
		},
	})
	postTestChecks(t)
}

func catalogVAppTemplateDestroyed(catalog, itemName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*VCDClient)
		org, err := conn.GetOrgByName(testConfig.VCD.Org)
		if err != nil {
			return err
		}
		cat, err := org.GetCatalogByName(catalog, false)
		if err != nil {
			return err
		}
		_, err = cat.GetVAppTemplateByName(itemName)
		if err == nil {
			return fmt.Errorf("vApp Template %s not deleted", itemName)
		}
		return nil
	}
}

const testAccCheckVcdCatalogVAppTemplateDS = `
data "vcd_catalog" "{{.Catalog}}" {
  org  = "{{.Org}}"
  name = "{{.Catalog}}"
}

data "vcd_catalog_vapp_template" "{{.VAppTemplate}}_1" {
  org        = "{{.Org}}"
  catalog_id = data.vcd_catalog.{{.Catalog}}.id
  name       = "{{.VAppTemplate}}"
}

data "vcd_org_vdc" "{{.Vdc}}" {
  org  = "{{.Org}}"
  name = "{{.Vdc}}"
}

data "vcd_catalog_vapp_template" "{{.VAppTemplate}}_2" {
  org    = "{{.Org}}"
  vdc_id = data.vcd_org_vdc.{{.Vdc}}.id
  name   = "{{.VAppTemplate}}"
}

resource "vcd_catalog_vapp_template" "{{.NewVappTemplate}}" {
  org        = "{{.Org}}"
  catalog_id = data.vcd_catalog.{{.Catalog}}.id

  name                 = "{{.NewVappTemplate}}"
  description          = data.vcd_catalog_vapp_template.{{.VAppTemplate}}_1.id
  ova_path             = "{{.OvaPath}}"
  upload_piece_size    = {{.UploadPieceSize}}

  metadata = {
    key1 = "value1"
    key2 = "value2"
  }
}

data "vcd_catalog_vapp_template" "{{.VAppTemplate}}_3" {
  org        = "{{.Org}}"
  catalog_id = data.vcd_catalog.{{.Catalog}}.id
  filter {
    name_regex = "{{.VAppTemplate}}"
  }
}

data "vcd_catalog_vapp_template" "{{.VAppTemplate}}_4" {
  org    = "{{.Org}}"
  vdc_id = data.vcd_org_vdc.{{.Vdc}}.id
  filter {
    name_regex = "{{.VAppTemplate}}"
  }
}

data "vcd_catalog_vapp_template" "{{.VAppTemplate}}_5" {
  org    = "{{.Org}}"
  vdc_id = data.vcd_org_vdc.{{.Vdc}}.id
  filter {
    metadata {
      key   = "key1"
      value = "value1"
    }
  }
  depends_on = [ vcd_catalog_vapp_template.{{.NewVappTemplate}} ]
}
`
