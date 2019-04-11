// +build multivm

package vcd

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/vmware/go-vcloud-director/v2/govcd"
)

// To execute this test, run
// go test -v -timeout 0 -tags multivm -run TestAccVcdVAppVmMulti .
// Extends TestAccVcdVappVM with multiple VMs
func TestAccVcdVAppVmMulti(t *testing.T) {
	var (
		vapp              govcd.VApp
		vm                govcd.VM
		diskResourceNameM string = "TestAccVcdVAppVmMulti"
		vappName2         string = "TestAccVcdVAppVmVappM"
		diskName          string = "TestAccVcdIndependentDiskMulti"
		vmName1           string = "TestAccVcdVAppVmVm1"
		vmName2           string = "TestAccVcdVAppVmVm2"
		vmName3           string = "TestAccVcdVAppVmVm3"
	)

	if vcdShortTest {
		t.Skip(acceptanceTestsSkipped)
		return
	}
	var params = StringMap{
		"Org":                testConfig.VCD.Org,
		"Vdc":                testConfig.VCD.Vdc,
		"EdgeGateway":        testConfig.Networking.EdgeGateway,
		"NetworkName":        "TestAccVcdVAppVmNet",
		"Catalog":            testSuiteCatalogName,
		"CatalogItem":        testSuiteCatalogOVAItem,
		"VappName":           vappName2,
		"VmName1":            vmName1,
		"VmName2":            vmName2,
		"VmName3":            vmName3,
		"diskName":           diskName,
		"size":               "5",
		"busType":            "SCSI",
		"busSubType":         "lsilogicsas",
		"storageProfileName": "*",
		"diskResourceName":   diskResourceNameM,
	}

	configText := templateFill(testAccCheckVcdVAppVmMulti, params)
	debugPrintf("#[DEBUG] CONFIGURATION: %s\n", configText)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVcdVAppVmMultiDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: configText,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVcdVAppVmMultiExists("vcd_vapp_vm."+vmName, &vapp, &vm),
					resource.TestCheckResourceAttr(
						"vcd_vapp_vm."+vmName, "name", vmName),
					resource.TestCheckResourceAttr(
						"vcd_vapp_vm."+vmName, "ip", "10.10.102.161"),
					resource.TestCheckResourceAttr(
						"vcd_vapp_vm."+vmName, "power_on", "true"),
				),
			},
		},
	})
}

func testAccCheckVcdVAppVmMultiExists(n string, vapp *govcd.VApp, vm *govcd.VM) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no VAPP ID is set")
		}

		conn := testAccProvider.Meta().(*VCDClient)
		_, vdc, err := conn.GetOrgAndVdc(testConfig.VCD.Org, testConfig.VCD.Vdc)
		if err != nil {
			return fmt.Errorf(errorRetrievingVdcFromOrg, testConfig.VCD.Vdc, testConfig.VCD.Org, err)
		}

		vapp, err := vdc.FindVAppByName(vappName2)

		resp, err := vdc.FindVMByName(vapp, vmName)

		if err != nil {
			return err
		}

		*vm = resp

		return nil
	}
}

func testAccCheckVcdVAppVmMultiDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*VCDClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "vcd_vapp" {
			continue
		}
		_, vdc, err := conn.GetOrgAndVdc(testConfig.VCD.Org, testConfig.VCD.Vdc)
		if err != nil {
			return fmt.Errorf(errorRetrievingVdcFromOrg, testConfig.VCD.Vdc, testConfig.VCD.Org, err)
		}

		_, err = vdc.FindVAppByName(vappName2)

		if err == nil {
			return fmt.Errorf("VPCs still exist")
		}

		return nil
	}

	return nil
}

const testAccCheckVcdVAppVmMulti = `
resource "vcd_network_routed" "{{.NetworkName}}" {
  name         = "{{.NetworkName}}"
  org          = "{{.Org}}"
  vdc          = "{{.Vdc}}"
  edge_gateway = "{{.EdgeGateway}}"
  gateway      = "10.10.102.1"

  static_ip_pool {
    start_address = "10.10.102.2"
    end_address   = "10.10.102.254"
  }
}

resource "vcd_independent_disk" "{{.diskResourceName}}" {
  org             = "{{.Org}}"
  vdc             = "{{.Vdc}}"
  name            = "{{.diskName}}"
  size            = "{{.size}}"
  bus_type        = "{{.busType}}"
  bus_sub_type    = "{{.busSubType}}"
  storage_profile = "{{.storageProfileName}}"
}

resource "vcd_vapp" "{{.VappName}}" {
  name = "{{.VappName}}"
  org  = "{{.Org}}"
  vdc  = "{{.Vdc}}"
}

resource "vcd_vapp_vm" "{{.VmName1}}" {
  org           = "{{.Org}}"
  vdc           = "{{.Vdc}}"
  vapp_name     = "${vcd_vapp.{{.VappName}}.name}"
  network_name  = "${vcd_network_routed.{{.NetworkName}}.name}"
  name          = "{{.VmName1}}"
  catalog_name  = "{{.Catalog}}"
  template_name = "{{.CatalogItem}}"
  memory        = 1024
  cpus          = 2
  cpu_cores     = 1
  ip            = "10.10.102.161"

  disk {
    name = "${vcd_independent_disk.{{.diskResourceName}}.name}"
    bus_number = 1
    unit_number = 0
  }

  depends_on    = ["vcd_vapp.{{.VappName}}","vcd_independent_disk.{{.diskResourceName}}", "vcd_network_routed.{{.NetworkName}}"]
}

resource "vcd_vapp_vm" "{{.VmName2}}" {
  org           = "{{.Org}}"
  vdc           = "{{.Vdc}}"
  vapp_name     = "${vcd_vapp.{{.VappName}}.name}"
  network_name  = "${vcd_network_routed.{{.NetworkName}}.name}"
  name          = "{{.VmName2}}"
  catalog_name  = "{{.Catalog}}"
  template_name = "{{.CatalogItem}}"
  memory        = 1024
  cpus          = 2
  cpu_cores     = 1
  ip            = "10.10.102.162"

  depends_on    = ["vcd_vapp.{{.VappName}}", "vcd_network_routed.{{.NetworkName}}"]
}

resource "vcd_vapp_vm" "{{.VmName3}}" {
  org           = "{{.Org}}"
  vdc           = "{{.Vdc}}"
  vapp_name     = "${vcd_vapp.{{.VappName}}.name}"
  network_name  = "${vcd_network_routed.{{.NetworkName}}.name}"
  name          = "{{.VmName3}}"
  catalog_name  = "{{.Catalog}}"
  template_name = "{{.CatalogItem}}"
  memory        = 1024
  cpus          = 2
  cpu_cores     = 1
  ip            = "10.10.102.163"

  depends_on    = ["vcd_vapp.{{.VappName}}", "vcd_network_routed.{{.NetworkName}}"]
}

`
