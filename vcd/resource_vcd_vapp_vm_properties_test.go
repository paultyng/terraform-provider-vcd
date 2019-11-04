// +build vapp vm ALL functional

package vcd

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/vmware/go-vcloud-director/v2/govcd"
)

func init() {
	testingTags["vm"] = "resource_vcd_vapp_vm_properties_test.go"
}

func TestAccVcdVAppVmProperties(t *testing.T) {
	var vapp govcd.VApp
	var vm govcd.VM

	var params = StringMap{
		"Org":         testConfig.VCD.Org,
		"Vdc":         testConfig.VCD.Vdc,
		"EdgeGateway": testConfig.Networking.EdgeGateway,
		"Catalog":     testSuiteCatalogName,
		"CatalogItem": testSuiteCatalogOVAItem,
		"VappName":    vappName2,
		"VmName":      vmName,
		"Tags":        "vapp vm",
	}

	configText := templateFill(testAccCheckVcdVAppVm_properties, params)

	params["FuncName"] = t.Name() + "-step1"
	configText1 := templateFill(testAccCheckVcdVAppVm_propertiesUpdate, params)

	params["FuncName"] = t.Name() + "-step2"
	configText2 := templateFill(testAccCheckVcdVAppVm_propertiesRemove, params)

	if vcdShortTest {
		t.Skip(acceptanceTestsSkipped)
		return
	}
	debugPrintf("#[DEBUG] CONFIGURATION: %s\n", configText)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVcdVAppVmDestroy(vappName2),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: configText,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVcdVAppVmExists(vappName2, vmName, "vcd_vapp_vm."+vmName, &vapp, &vm),
					resource.TestCheckResourceAttr("vcd_vapp_vm."+vmName, "name", vmName),
					resource.TestCheckResourceAttr("vcd_vapp_vm."+vmName, `guest_properties.guest.hostname`, "test-host"),
					resource.TestCheckResourceAttr("vcd_vapp_vm."+vmName, `guest_properties.guest.another.subkey`, "another-value"),
				),
			},
			resource.TestStep{
				Config: configText1,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVcdVAppVmExists(vappName2, vmName, "vcd_vapp_vm."+vmName, &vapp, &vm),
					resource.TestCheckResourceAttr("vcd_vapp_vm."+vmName, "name", vmName),
					resource.TestCheckNoResourceAttr("vcd_vapp_vm."+vmName, `guest_properties.guest.hostname`),
					resource.TestCheckResourceAttr("vcd_vapp_vm."+vmName, `guest_properties.guest.another.subkey`, "new-value"),
					resource.TestCheckResourceAttr("vcd_vapp_vm."+vmName, `guest_properties.guest.third.subkey`, "third-value"),
				),
			},
			resource.TestStep{
				Config: configText2,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVcdVAppVmExists(vappName2, vmName, "vcd_vapp_vm."+vmName, &vapp, &vm),
					resource.TestCheckResourceAttr("vcd_vapp_vm."+vmName, "name", vmName),
					resource.TestCheckNoResourceAttr("vcd_vapp_vm."+vmName, `guest_properties`),
				),
			},
		},
	})
}

const testAccCheckVcdVAppVm_properties = `
resource "vcd_vapp" "{{.VappName}}" {
  name = "{{.VappName}}"
  org  = "{{.Org}}"
  vdc  = "{{.Vdc}}"
}

resource "vcd_vapp_vm" "{{.VmName}}" {
  org           = "{{.Org}}"
  vdc           = "{{.Vdc}}"
  vapp_name     = "${vcd_vapp.{{.VappName}}.name}"
  name          = "{{.VmName}}"
  catalog_name  = "{{.Catalog}}"
  template_name = "{{.CatalogItem}}"
  memory        = 512
  cpus          = 2
  cpu_cores     = 1

  guest_properties = {
	"guest.hostname"       = "test-host"
	"guest.another.subkey" = "another-value"
  }
}
`

const testAccCheckVcdVAppVm_propertiesUpdate = `
resource "vcd_vapp" "{{.VappName}}" {
  name = "{{.VappName}}"
  org  = "{{.Org}}"
  vdc  = "{{.Vdc}}"
}

resource "vcd_vapp_vm" "{{.VmName}}" {
  org           = "{{.Org}}"
  vdc           = "{{.Vdc}}"
  vapp_name     = "${vcd_vapp.{{.VappName}}.name}"
  name          = "{{.VmName}}"
  catalog_name  = "{{.Catalog}}"
  template_name = "{{.CatalogItem}}"
  memory        = 512
  cpus          = 2
  cpu_cores     = 1

  guest_properties = {
	"guest.another.subkey" = "new-value"
	"guest.third.subkey"   = "third-value"
  }
}
`

const testAccCheckVcdVAppVm_propertiesRemove = `
resource "vcd_vapp" "{{.VappName}}" {
  name = "{{.VappName}}"
  org  = "{{.Org}}"
  vdc  = "{{.Vdc}}"
}

resource "vcd_vapp_vm" "{{.VmName}}" {
  org           = "{{.Org}}"
  vdc           = "{{.Vdc}}"
  vapp_name     = "${vcd_vapp.{{.VappName}}.name}"
  name          = "{{.VmName}}"
  catalog_name  = "{{.Catalog}}"
  template_name = "{{.CatalogItem}}"
  memory        = 512
  cpus          = 2
  cpu_cores     = 1
}
`
