//go:build vapp || ALL || functional

package vcd

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccVcdClonedVApp(t *testing.T) {
	preTestChecks(t)
	var vappFromTemplate = "TestAccClonedVAppFromTemplate"
	var vappFromVapp = "TestAccClonedVAppFromVapp"
	var vappDescription = "Test cloned vApp from Template"
	vappTemplateName := "small-3VM"
	firstVmName := "firstVM"
	orgName := testConfig.VCD.Org
	nsxtVdcName := testConfig.Nsxt.Vdc

	var params = StringMap{
		"Org":                  orgName,
		"Vdc":                  nsxtVdcName,
		"Catalog":              testConfig.VCD.Catalog.NsxtBackedCatalogName,
		"CatalogItem":          vappTemplateName,
		"VappFromTemplateName": vappFromTemplate,
		"VappFromVappName":     vappFromVapp,
		"VappDescription":      vappDescription,
		"FirstVmName":          firstVmName,
		"FuncName":             t.Name(),
		"Tags":                 "vapp",
	}
	testParamsNotEmpty(t, params)

	configText := templateFill(testAccVcdClonedVApp, params)

	debugPrintf("#[DEBUG] CONFIGURATION cloned vApp: %s\n", configText)

	resourceVappFromTemplate := "vcd_cloned_vapp." + vappFromTemplate
	datasourceVappFromTemplate := "data.vcd_vapp." + vappFromTemplate
	resourceVappFromVapp := "vcd_cloned_vapp." + vappFromVapp
	datasourceVappFromVapp := "data.vcd_vapp." + vappFromVapp
	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		CheckDestroy: resource.ComposeTestCheckFunc(
			testAccCheckVAppExists(resourceVappFromTemplate, orgName, nsxtVdcName, vappFromTemplate, false),
			testAccCheckVAppExists(resourceVappFromVapp, orgName, nsxtVdcName, vappFromVapp, false),
		),
		Steps: []resource.TestStep{
			{
				Config: configText,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVAppExists(resourceVappFromTemplate, orgName, nsxtVdcName, vappFromTemplate, true),
					testAccCheckVAppExists(resourceVappFromVapp, orgName, nsxtVdcName, vappFromVapp, true),
					resource.TestCheckResourceAttr(resourceVappFromTemplate, "name", vappFromTemplate),
					resource.TestCheckResourceAttr(resourceVappFromTemplate, "description", vappDescription),
					resource.TestCheckResourceAttr(resourceVappFromTemplate, "status", "1"),
					resource.TestCheckResourceAttr(resourceVappFromVapp, "name", vappFromVapp),
					resource.TestCheckResourceAttr(resourceVappFromVapp, "description", vappDescription),
					resource.TestCheckResourceAttr(resourceVappFromVapp, "status", "1"),
					resource.TestCheckResourceAttrPair(resourceVappFromTemplate, "name", datasourceVappFromTemplate, "name"),
					resource.TestCheckResourceAttrPair(resourceVappFromTemplate, "href", datasourceVappFromTemplate, "href"),
					resource.TestCheckResourceAttrPair(resourceVappFromTemplate, "description", datasourceVappFromTemplate, "description"),
					resource.TestCheckResourceAttrPair(resourceVappFromVapp, "name", datasourceVappFromVapp, "name"),
					resource.TestCheckResourceAttrPair(resourceVappFromVapp, "href", datasourceVappFromVapp, "href"),
					resource.TestCheckResourceAttrPair(resourceVappFromVapp, "description", datasourceVappFromVapp, "description"),
					resource.TestCheckResourceAttr("data.vcd_vapp_vm.first_vm_from_template", "name", firstVmName),
					resource.TestCheckResourceAttr("data.vcd_vapp_vm.first_vm_from_vapp", "name", firstVmName),
				),
			},
		},
	})
	postTestChecks(t)
}

func testAccCheckVAppExists(resourceDef, orgName, vdcName, vAppName string, wantExist bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceDef]
		if !ok {
			return fmt.Errorf("not found: %s", resourceDef)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no vApp ID is set")
		}

		conn := testAccProvider.Meta().(*VCDClient)

		_, vdc, err := conn.GetOrgAndVdc(orgName, vdcName)
		if err != nil {
			return fmt.Errorf(errorRetrievingVdcFromOrg, vdcName, orgName, err)
		}

		_, err = vdc.GetVAppByName(vAppName, false)

		if err != nil {
			if wantExist {
				return err
			}
			return nil
		} else {
			if !wantExist {
				return fmt.Errorf("vApp %s not deleted yet", vAppName)
			}
		}
		return nil
	}
}

const testAccVcdClonedVApp = `
data "vcd_catalog" "cat" {
  org  = "{{.Org}}"
  name = "{{.Catalog}}"
}

data "vcd_catalog_vapp_template" "tmpl" {
  org        = "{{.Org}}"
  catalog_id = data.vcd_catalog.cat.id
  name       = "{{.CatalogItem}}"
}

resource "vcd_cloned_vapp" "vapp_from_template" {
  org           = "{{.Org}}"
  vdc           = "{{.Vdc}}"
  name          = "{{.VappFromTemplateName}}"
  description   = "{{.VappDescription}}"
  power_on      = true
  source_id     = data.vcd_catalog_vapp_template.tmpl.id
  source_type   = "template"
  delete_source = false
}

resource "vcd_cloned_vapp" "vapp_from_vapp" {
  org           = "{{.Org}}"
  vdc           = "{{.Vdc}}"
  name          = "{{.VappFromVappName}}"
  description   = "{{.VappDescription}}"
  power_on      = true
  source_id     = vcd_cloned_vapp.vapp_from_template.id
  source_type   = "vapp"
  delete_source = false
}

data "vcd_vapp" "vapp_from_template" {
  org  = "{{.Org}}"
  vdc  = "{{.Vdc}}"
  name = vcd_cloned_vapp.vapp_from_template.name
}

data "vcd_vapp" "vapp_from_vapp" {
  org  = "{{.Org}}"
  vdc  = "{{.Vdc}}"
  name = vcd_cloned_vapp.vapp_from_vapp.name
}

data "vcd_vapp_vm" "first_vm_from_template" {
  org       = "{{.Org}}"
  vdc       = "{{.Vdc}}"
  name      = "{{.FirstVmName}}"
  vapp_name = data.vcd_vapp.vapp_from_template.name
}

data "vcd_vapp_vm" "first_vm_from_vapp" {
  org       = "{{.Org}}"
  vdc       = "{{.Vdc}}"
  name      = "{{.FirstVmName}}"
  vapp_name = data.vcd_vapp.vapp_from_vapp.name
}

`
