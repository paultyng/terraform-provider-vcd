//go:build vapp || vm || ALL || functional

package vcd

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/vmware/go-vcloud-director/v2/govcd"
)

func TestAccVcdVAppVmBootOptions(t *testing.T) {
	preTestChecks(t)
	var (
		vapp     govcd.VApp
		vm       govcd.VM
		vappName string = t.Name()
		vmName   string = t.Name() + "VM"
	)

	if testConfig.VCD.ProviderVdc.StorageProfile == "" || testConfig.VCD.ProviderVdc.StorageProfile2 == "" {
		t.Skip("Both variables testConfig.VCD.ProviderVdc.StorageProfile and testConfig.VCD.ProviderVdc.StorageProfile2 must be set")
	}

	var params = StringMap{
		"Org":         testConfig.VCD.Org,
		"Vdc":         testConfig.VCD.Vdc,
		"EdgeGateway": testConfig.Networking.EdgeGateway,
		"Catalog":     testSuiteCatalogName,
		"CatalogItem": testSuiteCatalogOVAItem,
		"VAppName":    vappName,
		"VMName":      vmName,
		"Tags":        "vapp vm",
	}
	testParamsNotEmpty(t, params)

	params["FuncName"] = t.Name()
	configText1 := templateFill(testAccCheckVcdVAppVmBootOptions, params)
	debugPrintf("#[DEBUG] CONFIGURATION: %s\n", configText1)

	params["FuncName"] = t.Name() + "-step1"
	configText2 := templateFill(testAccCheckVcdVAppVmBootOptionsStep1, params)
	debugPrintf("#[DEBUG] CONFIGURATION: %s\n", configText2)

	params["FuncName"] = t.Name() + "-step3"
	configText3 := templateFill(testAccCheckVcdVAppVmBootOptionsStep2, params)
	debugPrintf("#[DEBUG] CONFIGURATION: %s\n", configText3)

	if vcdShortTest {
		t.Skip(acceptanceTestsSkipped)
		return
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccCheckVcdVAppVmDestroy(vappName),
		Steps: []resource.TestStep{
			// Step 0 - create
			{
				Config: configText1,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckVcdVAppVmExists(vappName, vmName, "vcd_vapp_vm."+vmName, &vapp, &vm),
					resource.TestCheckResourceAttr("vcd_vapp_vm."+vmName, "name", vmName),

					resource.TestCheckResourceAttr("vcd_vapp_vm."+vmName, "memory", "2048"),
					resource.TestCheckResourceAttr("vcd_vapp_vm."+vmName, "cpus", "1"),

					resource.TestCheckResourceAttr("vcd_vapp_vm."+vmName, "os_type", "sles11_64Guest"),
					resource.TestCheckResourceAttr("vcd_vapp_vm."+vmName, "hardware_version", "vmx-13"),
					resource.TestCheckResourceAttr("vcd_vapp_vm."+vmName, "firmware", "efi"),

					resource.TestCheckResourceAttr("vcd_vapp_vm."+vmName, "boot_options.0.enter_bios_setup", "true"),
					resource.TestCheckResourceAttr("vcd_vapp_vm."+vmName, "boot_options.0.efi_secure_boot", "true"),
					resource.TestCheckResourceAttr("vcd_vapp_vm."+vmName, "boot_options.0.boot_delay", "20"),
					resource.TestCheckResourceAttr("vcd_vapp_vm."+vmName, "boot_options.0.boot_retry_delay", "20"),
					resource.TestCheckResourceAttr("vcd_vapp_vm."+vmName, "boot_options.0.boot_retry_enabled", "true"),
				),
			},
			{
				Config: configText2,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckVcdVAppVmExists(vappName, vmName, "vcd_vapp_vm."+vmName, &vapp, &vm),
					resource.TestCheckResourceAttr("vcd_vapp_vm."+vmName, "name", vmName),

					resource.TestCheckResourceAttr("vcd_vapp_vm."+vmName, "memory", "2048"),
					resource.TestCheckResourceAttr("vcd_vapp_vm."+vmName, "cpus", "1"),

					resource.TestCheckResourceAttr("vcd_vapp_vm."+vmName, "os_type", "sles11_64Guest"),
					resource.TestCheckResourceAttr("vcd_vapp_vm."+vmName, "hardware_version", "vmx-13"),
					resource.TestCheckResourceAttr("vcd_vapp_vm."+vmName, "firmware", "bios"),

					resource.TestCheckResourceAttr("vcd_vapp_vm."+vmName, "boot_options.0.enter_bios_setup", "true"),
					resource.TestCheckResourceAttr("vcd_vapp_vm."+vmName, "boot_options.0.efi_secure_boot", "false"),
					resource.TestCheckResourceAttr("vcd_vapp_vm."+vmName, "boot_options.0.boot_delay", "20"),
					resource.TestCheckResourceAttr("vcd_vapp_vm."+vmName, "boot_options.0.boot_retry_delay", "20"),
					resource.TestCheckResourceAttr("vcd_vapp_vm."+vmName, "boot_options.0.boot_retry_enabled", "true"),
				),
			},
			{
				Config: configText3,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckVcdVAppVmExists(vappName, vmName, "vcd_vapp_vm."+vmName, &vapp, &vm),
					resource.TestCheckResourceAttr("vcd_vapp_vm."+vmName, "name", vmName),

					resource.TestCheckResourceAttr("vcd_vapp_vm."+vmName, "memory", "2048"),
					resource.TestCheckResourceAttr("vcd_vapp_vm."+vmName, "cpus", "1"),

					resource.TestCheckResourceAttr("vcd_vapp_vm."+vmName, "os_type", "sles11_64Guest"),
					resource.TestCheckResourceAttr("vcd_vapp_vm."+vmName, "hardware_version", "vmx-13"),
					resource.TestCheckResourceAttr("vcd_vapp_vm."+vmName, "firmware", "efi"),

					resource.TestCheckResourceAttr("vcd_vapp_vm."+vmName, "boot_options.0.enter_bios_setup", "false"),
					resource.TestCheckResourceAttr("vcd_vapp_vm."+vmName, "boot_options.0.efi_secure_boot", "true"),
					resource.TestCheckResourceAttr("vcd_vapp_vm."+vmName, "boot_options.0.boot_delay", "50"),
					resource.TestCheckResourceAttr("vcd_vapp_vm."+vmName, "boot_options.0.boot_retry_delay", "50"),
					resource.TestCheckResourceAttr("vcd_vapp_vm."+vmName, "boot_options.0.boot_retry_enabled", "true"),
				),
			},
		},
	})
	postTestChecks(t)
}

const testSharedBootOptions = `
resource "vcd_vapp" "{{.VAppName}}" {
  org = "{{.Org}}"
  vdc = "{{.Vdc}}"

  name     = "{{.VAppName}}"
  power_on = true
}
`

const testAccCheckVcdVAppVmBootOptions = testSharedBootOptions + `
resource "vcd_vapp_vm" "{{.VMName}}" {
  org = "{{.Org}}"
  vdc = "{{.Vdc}}"

  power_on = true

  vapp_name     = vcd_vapp.{{.VAppName}}.name
  name          = "{{.VMName}}"
  computer_name = "compNameUp"

  memory        = 2048
  cpus          = 1

  os_type          = "sles11_64Guest"
  firmware         = "efi"
  hardware_version = "vmx-13"

  boot_options {
    efi_secure_boot = true
    boot_retry_delay = 20
    boot_retry_enabled = true
    boot_delay = 20
    enter_bios_setup = true
  }
 }
`

const testAccCheckVcdVAppVmBootOptionsStep1 = testSharedBootOptions + `
resource "vcd_vapp_vm" "{{.VMName}}" {
  org = "{{.Org}}"
  vdc = "{{.Vdc}}"

  power_on = false

  vapp_name     = vcd_vapp.{{.VAppName}}.name
  name          = "{{.VMName}}"
  computer_name = "compNameUp"

  memory        = 2048
  cpus          = 1

  os_type          = "sles11_64Guest"
  firmware         = "bios"
  hardware_version = "vmx-13"

  boot_options {
    efi_secure_boot = false
    boot_retry_delay = 20
    boot_retry_enabled = true
    boot_delay = 20
    enter_bios_setup = true
  }
 }
`

const testAccCheckVcdVAppVmBootOptionsStep2 = testSharedBootOptions + `
resource "vcd_vapp_vm" "{{.VMName}}" {
  org = "{{.Org}}"
  vdc = "{{.Vdc}}"

  power_on = true

  vapp_name     = vcd_vapp.{{.VAppName}}.name
  name          = "{{.VMName}}"
  computer_name = "compNameUp"

  memory        = 2048
  cpus          = 1

  os_type          = "sles11_64Guest"
  firmware         = "efi"
  hardware_version = "vmx-13"

  boot_options {
    efi_secure_boot = true
    boot_retry_delay = 50
    boot_retry_enabled = true
    boot_delay = 50
    enter_bios_setup = false
  }
 }
`
