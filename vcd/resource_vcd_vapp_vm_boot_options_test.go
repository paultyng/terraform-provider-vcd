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
		vmName   string = t.Name() + "-vm"
	)

	if testConfig.VCD.ProviderVdc.StorageProfile == "" || testConfig.VCD.ProviderVdc.StorageProfile2 == "" {
		t.Skip("Both variables testConfig.VCD.ProviderVdc.StorageProfile and testConfig.VCD.ProviderVdc.StorageProfile2 must be set")
	}

	if checkVersion(testConfig.Provider.ApiVersion, "< 37.1") {
		t.Skip("Most boot options are only available since 37.1")
	}

	var params = StringMap{
		"Org":                    testConfig.VCD.Org,
		"Vdc":                    testConfig.Nsxt.Vdc,
		"CatalogName":            testConfig.VCD.Catalog.NsxtBackedCatalogName,
		"VappTemplateName":       testConfig.VCD.Catalog.CatalogItemWithEfiSupport,
		"VAppName":               vappName,
		"VAppNameWithTemplate":   vappName + "-template",
		"VappVMName":             vappName,
		"VappVMWithTemplateName": vappName + "-template",
		"EmptyVMName":            vmName,
		"VMWithTemplateName":     vmName + "-template",
		"Tags":                   "vapp vm",
	}
	testParamsNotEmpty(t, params)

	params["FuncName"] = t.Name()
	configText1 := templateFill(testAccCheckVcdVAppVmBootOptions, params)
	debugPrintf("#[DEBUG] CONFIGURATION: %s\n", configText1)

	params["FuncName"] = t.Name() + "-step1"
	configText2 := templateFill(testAccCheckVcdVAppVmBootOptionsStep1, params)
	debugPrintf("#[DEBUG] CONFIGURATION: %s\n", configText2)

	params["FuncName"] = t.Name() + "-step2"
	configText3 := templateFill(testAccCheckVcdVAppVmBootOptionsStep2, params)
	debugPrintf("#[DEBUG] CONFIGURATION: %s\n", configText3)

	if vcdShortTest {
		t.Skip(acceptanceTestsSkipped)
		return
	}

	emptyVapp := "vcd_vapp_vm." + vappName
	vappVmWithTemplate := "vcd_vapp_vm." + vappName + "-template"
	emptyVM := "vcd_vm." + vmName
	vmWithTemplate := "vcd_vm." + vmName + "-template"
	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccCheckVcdNsxtVAppVmDestroy(vappName),
			testAccCheckVcdNsxtVAppVmDestroy(vappName+"-template"),
			testAccCheckVcdStandaloneVmDestroy(vmName, testConfig.VCD.Org, testConfig.Nsxt.Vdc),
			testAccCheckVcdStandaloneVmDestroy(vmName+"-template", testConfig.VCD.Org, testConfig.Nsxt.Vdc),
		),
		Steps: []resource.TestStep{
			// Step 0 - create
			{
				// The `enter_bios_setup_on_next_boot flag is set to false on PowerOn, so it returns a non-empty plan`
				ExpectNonEmptyPlan: true,
				Config:             configText1,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckVcdNsxtVAppVmExists(vappName+"-template", vappName+"-template", vappVmWithTemplate, &vapp, &vm),
					testAccCheckVcdNsxtVAppVmExists(vappName, vappName, emptyVapp, &vapp, &vm),
					testAccCheckVcdNsxtStandaloneVmExists(vmName+"-template", vmWithTemplate),
					testAccCheckVcdNsxtStandaloneVmExists(vmName, emptyVM),
					resource.TestCheckResourceAttr(emptyVapp, "os_type", "sles11_64Guest"),
					resource.TestCheckResourceAttr(emptyVapp, "hardware_version", "vmx-13"),
					resource.TestCheckResourceAttr(emptyVapp, "firmware", "efi"),
					resource.TestCheckResourceAttr(vappVmWithTemplate, "firmware", "efi"),
					resource.TestCheckResourceAttr(emptyVM, "os_type", "sles11_64Guest"),
					resource.TestCheckResourceAttr(emptyVM, "hardware_version", "vmx-13"),
					resource.TestCheckResourceAttr(emptyVM, "firmware", "efi"),
					resource.TestCheckResourceAttr(vmWithTemplate, "firmware", "efi"),

					resource.TestCheckResourceAttr(emptyVapp, "boot_options.0.efi_secure_boot", "true"),
					resource.TestCheckResourceAttr(emptyVapp, "boot_options.0.boot_delay", "1200"),
					resource.TestCheckResourceAttr(emptyVapp, "boot_options.0.boot_retry_delay", "12000"),
					resource.TestCheckResourceAttr(emptyVapp, "boot_options.0.boot_retry_enabled", "true"),
					resource.TestCheckResourceAttr("data."+emptyVapp, "firmware", "efi"),
					resource.TestCheckResourceAttr("data."+emptyVapp, "boot_options.0.efi_secure_boot", "true"),
					resource.TestCheckResourceAttr("data."+emptyVapp, "boot_options.0.boot_delay", "1200"),
					resource.TestCheckResourceAttr("data."+emptyVapp, "boot_options.0.boot_retry_delay", "12000"),
					resource.TestCheckResourceAttr("data."+emptyVapp, "boot_options.0.boot_retry_enabled", "true"),

					resource.TestCheckResourceAttr(vappVmWithTemplate, "boot_options.0.efi_secure_boot", "true"),
					resource.TestCheckResourceAttr(vappVmWithTemplate, "boot_options.0.boot_delay", "1200"),
					resource.TestCheckResourceAttr(vappVmWithTemplate, "boot_options.0.boot_retry_delay", "12000"),
					resource.TestCheckResourceAttr(vappVmWithTemplate, "boot_options.0.boot_retry_enabled", "true"),
					resource.TestCheckResourceAttr("data."+vappVmWithTemplate, "firmware", "efi"),
					resource.TestCheckResourceAttr("data."+vappVmWithTemplate, "boot_options.0.efi_secure_boot", "true"),
					resource.TestCheckResourceAttr("data."+vappVmWithTemplate, "boot_options.0.boot_delay", "1200"),
					resource.TestCheckResourceAttr("data."+vappVmWithTemplate, "boot_options.0.boot_retry_delay", "12000"),
					resource.TestCheckResourceAttr("data."+vappVmWithTemplate, "boot_options.0.boot_retry_enabled", "true"),

					resource.TestCheckResourceAttr(emptyVM, "boot_options.0.efi_secure_boot", "true"),
					resource.TestCheckResourceAttr(emptyVM, "boot_options.0.boot_delay", "1200"),
					resource.TestCheckResourceAttr(emptyVM, "boot_options.0.boot_retry_delay", "12000"),
					resource.TestCheckResourceAttr(emptyVM, "boot_options.0.boot_retry_enabled", "true"),
					resource.TestCheckResourceAttr("data."+emptyVM, "firmware", "efi"),
					resource.TestCheckResourceAttr("data."+emptyVM, "boot_options.0.efi_secure_boot", "true"),
					resource.TestCheckResourceAttr("data."+emptyVM, "boot_options.0.boot_delay", "1200"),
					resource.TestCheckResourceAttr("data."+emptyVM, "boot_options.0.boot_retry_delay", "12000"),
					resource.TestCheckResourceAttr("data."+emptyVM, "boot_options.0.boot_retry_enabled", "true"),

					resource.TestCheckResourceAttr(vmWithTemplate, "boot_options.0.efi_secure_boot", "true"),
					resource.TestCheckResourceAttr(vmWithTemplate, "boot_options.0.boot_delay", "1200"),
					resource.TestCheckResourceAttr(vmWithTemplate, "boot_options.0.boot_retry_delay", "12000"),
					resource.TestCheckResourceAttr(vmWithTemplate, "boot_options.0.boot_retry_enabled", "true"),
					resource.TestCheckResourceAttr("data."+vmWithTemplate, "firmware", "efi"),
					resource.TestCheckResourceAttr("data."+vmWithTemplate, "boot_options.0.efi_secure_boot", "true"),
					resource.TestCheckResourceAttr("data."+vmWithTemplate, "boot_options.0.boot_delay", "1200"),
					resource.TestCheckResourceAttr("data."+vmWithTemplate, "boot_options.0.boot_retry_delay", "12000"),
					resource.TestCheckResourceAttr("data."+vmWithTemplate, "boot_options.0.boot_retry_enabled", "true"),
				),
			},
			{
				Config: configText2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(emptyVapp, "boot_options.0.efi_secure_boot", "false"),
					resource.TestCheckResourceAttr(emptyVapp, "boot_options.0.boot_delay", "1100"),
					resource.TestCheckResourceAttr(emptyVapp, "boot_options.0.boot_retry_delay", "11000"),
					resource.TestCheckResourceAttr(emptyVapp, "boot_options.0.boot_retry_enabled", "true"),
					resource.TestCheckResourceAttr("data."+emptyVapp, "firmware", "bios"),
					resource.TestCheckResourceAttr("data."+emptyVapp, "boot_options.0.efi_secure_boot", "false"),
					resource.TestCheckResourceAttr("data."+emptyVapp, "boot_options.0.boot_delay", "1100"),
					resource.TestCheckResourceAttr("data."+emptyVapp, "boot_options.0.boot_retry_delay", "11000"),
					resource.TestCheckResourceAttr("data."+emptyVapp, "boot_options.0.boot_retry_enabled", "true"),

					resource.TestCheckResourceAttr(vappVmWithTemplate, "boot_options.0.efi_secure_boot", "false"),
					resource.TestCheckResourceAttr(vappVmWithTemplate, "boot_options.0.boot_delay", "1100"),
					resource.TestCheckResourceAttr(vappVmWithTemplate, "boot_options.0.boot_retry_delay", "11000"),
					resource.TestCheckResourceAttr(vappVmWithTemplate, "boot_options.0.boot_retry_enabled", "true"),
					resource.TestCheckResourceAttr("data."+vappVmWithTemplate, "firmware", "bios"),
					resource.TestCheckResourceAttr("data."+vappVmWithTemplate, "boot_options.0.efi_secure_boot", "false"),
					resource.TestCheckResourceAttr("data."+vappVmWithTemplate, "boot_options.0.boot_delay", "1100"),
					resource.TestCheckResourceAttr("data."+vappVmWithTemplate, "boot_options.0.boot_retry_delay", "11000"),
					resource.TestCheckResourceAttr("data."+vappVmWithTemplate, "boot_options.0.boot_retry_enabled", "true"),

					resource.TestCheckResourceAttr(emptyVM, "boot_options.0.efi_secure_boot", "false"),
					resource.TestCheckResourceAttr(emptyVM, "boot_options.0.boot_delay", "1100"),
					resource.TestCheckResourceAttr(emptyVM, "boot_options.0.boot_retry_delay", "11000"),
					resource.TestCheckResourceAttr(emptyVM, "boot_options.0.boot_retry_enabled", "true"),
					resource.TestCheckResourceAttr("data."+emptyVM, "firmware", "bios"),
					resource.TestCheckResourceAttr("data."+emptyVM, "boot_options.0.efi_secure_boot", "false"),
					resource.TestCheckResourceAttr("data."+emptyVM, "boot_options.0.boot_delay", "1100"),
					resource.TestCheckResourceAttr("data."+emptyVM, "boot_options.0.boot_retry_delay", "11000"),
					resource.TestCheckResourceAttr("data."+emptyVM, "boot_options.0.boot_retry_enabled", "true"),

					resource.TestCheckResourceAttr(vmWithTemplate, "boot_options.0.efi_secure_boot", "false"),
					resource.TestCheckResourceAttr(vmWithTemplate, "boot_options.0.boot_delay", "1100"),
					resource.TestCheckResourceAttr(vmWithTemplate, "boot_options.0.boot_retry_delay", "11000"),
					resource.TestCheckResourceAttr(vmWithTemplate, "boot_options.0.boot_retry_enabled", "true"),
					resource.TestCheckResourceAttr("data."+vmWithTemplate, "firmware", "bios"),
					resource.TestCheckResourceAttr("data."+vmWithTemplate, "boot_options.0.efi_secure_boot", "false"),
					resource.TestCheckResourceAttr("data."+vmWithTemplate, "boot_options.0.boot_delay", "1100"),
					resource.TestCheckResourceAttr("data."+vmWithTemplate, "boot_options.0.boot_retry_delay", "11000"),
					resource.TestCheckResourceAttr("data."+vmWithTemplate, "boot_options.0.boot_retry_enabled", "true"),
				),
			},
			{
				Config: configText3,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(emptyVapp, "boot_options.0.efi_secure_boot", "true"),
					resource.TestCheckResourceAttr(emptyVapp, "boot_options.0.boot_delay", "1000"),
					resource.TestCheckResourceAttr(emptyVapp, "boot_options.0.boot_retry_delay", "10000"),
					resource.TestCheckResourceAttr(emptyVapp, "boot_options.0.boot_retry_enabled", "false"),
					resource.TestCheckResourceAttr("data."+emptyVapp, "firmware", "efi"),
					resource.TestCheckResourceAttr("data."+emptyVapp, "boot_options.0.efi_secure_boot", "true"),
					resource.TestCheckResourceAttr("data."+emptyVapp, "boot_options.0.boot_delay", "1000"),
					resource.TestCheckResourceAttr("data."+emptyVapp, "boot_options.0.boot_retry_delay", "10000"),
					resource.TestCheckResourceAttr("data."+emptyVapp, "boot_options.0.boot_retry_enabled", "false"),

					resource.TestCheckResourceAttr(vappVmWithTemplate, "boot_options.0.efi_secure_boot", "true"),
					resource.TestCheckResourceAttr(vappVmWithTemplate, "boot_options.0.boot_delay", "1000"),
					resource.TestCheckResourceAttr(vappVmWithTemplate, "boot_options.0.boot_retry_delay", "10000"),
					resource.TestCheckResourceAttr(vappVmWithTemplate, "boot_options.0.boot_retry_enabled", "false"),
					resource.TestCheckResourceAttr("data."+vappVmWithTemplate, "firmware", "efi"),
					resource.TestCheckResourceAttr("data."+vappVmWithTemplate, "boot_options.0.efi_secure_boot", "true"),
					resource.TestCheckResourceAttr("data."+vappVmWithTemplate, "boot_options.0.boot_delay", "1000"),
					resource.TestCheckResourceAttr("data."+vappVmWithTemplate, "boot_options.0.boot_retry_delay", "10000"),
					resource.TestCheckResourceAttr("data."+vappVmWithTemplate, "boot_options.0.boot_retry_enabled", "false"),

					resource.TestCheckResourceAttr(emptyVM, "boot_options.0.efi_secure_boot", "true"),
					resource.TestCheckResourceAttr(emptyVM, "boot_options.0.boot_delay", "1000"),
					resource.TestCheckResourceAttr(emptyVM, "boot_options.0.boot_retry_delay", "10000"),
					resource.TestCheckResourceAttr(emptyVM, "boot_options.0.boot_retry_enabled", "false"),
					resource.TestCheckResourceAttr("data."+emptyVM, "firmware", "efi"),
					resource.TestCheckResourceAttr("data."+emptyVM, "boot_options.0.efi_secure_boot", "true"),
					resource.TestCheckResourceAttr("data."+emptyVM, "boot_options.0.boot_delay", "1000"),
					resource.TestCheckResourceAttr("data."+emptyVM, "boot_options.0.boot_retry_delay", "10000"),
					resource.TestCheckResourceAttr("data."+emptyVM, "boot_options.0.boot_retry_enabled", "false"),

					resource.TestCheckResourceAttr(vmWithTemplate, "boot_options.0.efi_secure_boot", "true"),
					resource.TestCheckResourceAttr(vmWithTemplate, "boot_options.0.boot_delay", "1000"),
					resource.TestCheckResourceAttr(vmWithTemplate, "boot_options.0.boot_retry_delay", "10000"),
					resource.TestCheckResourceAttr(vmWithTemplate, "boot_options.0.boot_retry_enabled", "false"),
					resource.TestCheckResourceAttr("data."+vmWithTemplate, "firmware", "efi"),
					resource.TestCheckResourceAttr("data."+vmWithTemplate, "boot_options.0.efi_secure_boot", "true"),
					resource.TestCheckResourceAttr("data."+vmWithTemplate, "boot_options.0.boot_delay", "1000"),
					resource.TestCheckResourceAttr("data."+vmWithTemplate, "boot_options.0.boot_retry_delay", "10000"),
					resource.TestCheckResourceAttr("data."+vmWithTemplate, "boot_options.0.boot_retry_enabled", "false"),
				),
			},
		},
	})
	postTestChecks(t)
}

const testSharedBootOptions = `
resource "vcd_vapp" "{{.VAppNameWithTemplate}}" {
  org = "{{.Org}}"
  vdc = "{{.Vdc}}"

  name     = "{{.VAppNameWithTemplate}}"
  power_on = true
}

resource "vcd_vapp" "{{.VAppName}}" {
  org = "{{.Org}}"
  vdc = "{{.Vdc}}"

  name     = "{{.VAppName}}"
  power_on = true
}

data "vcd_vapp_vm" "{{.VappVMWithTemplateName}}" {
  org       = vcd_vapp_vm.{{.VappVMWithTemplateName}}.org
  vdc       = vcd_vapp_vm.{{.VappVMWithTemplateName}}.vdc
  name      = vcd_vapp_vm.{{.VappVMWithTemplateName}}.name
  vapp_name = vcd_vapp_vm.{{.VappVMWithTemplateName}}.vapp_name
}

data "vcd_vapp_vm" "{{.VappVMName}}" {
  org       = vcd_vapp_vm.{{.VappVMName}}.org
  vdc       = vcd_vapp_vm.{{.VappVMName}}.vdc
  name      = vcd_vapp_vm.{{.VappVMName}}.name
  vapp_name = vcd_vapp_vm.{{.VappVMName}}.vapp_name
}

data "vcd_vm" "{{.EmptyVMName}}" {
  org       = vcd_vm.{{.EmptyVMName}}.org
  vdc       = vcd_vm.{{.EmptyVMName}}.vdc
  name      = vcd_vm.{{.EmptyVMName}}.name
}

data "vcd_vm" "{{.VMWithTemplateName}}" {
  org       = vcd_vm.{{.VMWithTemplateName}}.org
  vdc       = vcd_vm.{{.VMWithTemplateName}}.vdc
  name      = vcd_vm.{{.VMWithTemplateName}}.name
}

data "vcd_catalog" "{{.CatalogName}}" {
  org  = "{{.Org}}"
  name = "{{.CatalogName}}"
}

data "vcd_catalog_vapp_template" "{{.VappTemplateName}}" {
  org        = "{{.Org}}"
  catalog_id = data.vcd_catalog.{{.CatalogName}}.id
  name       = "{{.VappTemplateName}}"
}
`

const testAccCheckVcdVAppVmBootOptions = testSharedBootOptions + `
# skip-binary-test: enter_bios_setup_on_next_boot automatically resets to 'false' after boot and causes inconsistent plan
resource "vcd_vapp_vm" "{{.VappVMWithTemplateName}}" {
  org = "{{.Org}}"
  vdc = "{{.Vdc}}"

  power_on = true

  vapp_name     = vcd_vapp.{{.VAppNameWithTemplate}}.name
  name          = "{{.VappVMWithTemplateName}}"
  computer_name = "compNameUp"

  vapp_template_id = data.vcd_catalog_vapp_template.{{.VappTemplateName}}.id
  firmware         = "efi"

  boot_options {
    efi_secure_boot               = true
    boot_retry_delay              = 12000
    boot_retry_enabled            = true
    boot_delay                    = 1200
    enter_bios_setup_on_next_boot = true
  }
}

resource "vcd_vapp_vm" "{{.VappVMName}}" {
  org = "{{.Org}}"
  vdc = "{{.Vdc}}"

  power_on = true

  vapp_name     = vcd_vapp.{{.VAppName}}.name
  name          = "{{.VappVMName}}"
  computer_name = "compNameUp"

  os_type          = "sles11_64Guest"
  firmware         = "efi"
  hardware_version = "vmx-13"

  memory        = 2048
  cpus          = 1

  boot_options {
    efi_secure_boot               = true
    boot_retry_delay              = 12000
    boot_retry_enabled            = true
    boot_delay                    = 1200
    enter_bios_setup_on_next_boot = true
  }
}

resource "vcd_vm" "{{.EmptyVMName}}" {
  org = "{{.Org}}"
  vdc = "{{.Vdc}}"

  power_on = true

  name          = "{{.EmptyVMName}}"
  computer_name = "compNameUp"

  memory        = 2048
  cpus          = 1

  os_type          = "sles11_64Guest"
  firmware         = "efi"
  hardware_version = "vmx-13"

  boot_options {
    efi_secure_boot               = true
    boot_retry_delay              = 12000
    boot_retry_enabled            = true
    boot_delay                    = 1200
    enter_bios_setup_on_next_boot = true
  }
}

resource "vcd_vm" "{{.VMWithTemplateName}}" {
  org = "{{.Org}}"
  vdc = "{{.Vdc}}"

  power_on = true

  name          = "{{.VMWithTemplateName}}"
  computer_name = "compNameUp"

  vapp_template_id = data.vcd_catalog_vapp_template.{{.VappTemplateName}}.id
  firmware         = "efi"

  boot_options {
    efi_secure_boot               = true
    boot_retry_delay              = 12000
    boot_retry_enabled            = true
    boot_delay                    = 1200
    enter_bios_setup_on_next_boot = true
  }
}
`

const testAccCheckVcdVAppVmBootOptionsStep1 = testSharedBootOptions + `
# skip-binary-test - Can't set boot_options and firmware when creating a VM instantiated from a vApp template
resource "vcd_vapp_vm" "{{.VappVMWithTemplateName}}" {
  org = "{{.Org}}"
  vdc = "{{.Vdc}}"

  power_on = true

  vapp_name     = vcd_vapp.{{.VAppNameWithTemplate}}.name
  name          = "{{.VappVMWithTemplateName}}"
  computer_name = "compNameUp"

  vapp_template_id = data.vcd_catalog_vapp_template.{{.VappTemplateName}}.id
  firmware         = "bios"

  boot_options {
    efi_secure_boot    = false
    boot_retry_delay   = 11000
    boot_delay         = 1100
    boot_retry_enabled = true
  }
}

resource "vcd_vapp_vm" "{{.VappVMName}}" {
  org = "{{.Org}}"
  vdc = "{{.Vdc}}"

  power_on = true

  vapp_name     = vcd_vapp.{{.VAppName}}.name
  name          = "{{.VappVMName}}"
  computer_name = "compNameUp"

  memory        = 2048
  cpus          = 1

  os_type          = "sles11_64Guest"
  hardware_version = "vmx-13"
  firmware         = "bios"

  boot_options {
    efi_secure_boot    = false
    boot_retry_delay   = 11000
    boot_delay         = 1100
    boot_retry_enabled = true
  }
}

resource "vcd_vm" "{{.EmptyVMName}}" {
  org = "{{.Org}}"
  vdc = "{{.Vdc}}"

  power_on = true

  name          = "{{.EmptyVMName}}"
  computer_name = "compNameUp"

  memory        = 2048
  cpus          = 1

  os_type          = "sles11_64Guest"
  hardware_version = "vmx-13"
  firmware         = "bios"

  boot_options {
    efi_secure_boot    = false
    boot_retry_delay   = 11000
    boot_delay         = 1100
    boot_retry_enabled = true
  }
}

resource "vcd_vm" "{{.VMWithTemplateName}}" {
  org = "{{.Org}}"
  vdc = "{{.Vdc}}"

  power_on = true

  name             = "{{.VMWithTemplateName}}"
  computer_name    = "compNameUp"
  vapp_template_id = data.vcd_catalog_vapp_template.{{.VappTemplateName}}.id

  firmware = "bios"

  boot_options {
    efi_secure_boot    = false
    boot_retry_delay   = 11000
    boot_delay         = 1100
    boot_retry_enabled = true
  }
}
`

const testAccCheckVcdVAppVmBootOptionsStep2 = testSharedBootOptions + `
# skip-binary-test - Can't set boot_options and firmware when creating a VM instantiated from a vApp template
resource "vcd_vapp_vm" "{{.VappVMWithTemplateName}}" {
  org = "{{.Org}}"
  vdc = "{{.Vdc}}"

  power_on = true

  vapp_name     = vcd_vapp.{{.VAppNameWithTemplate}}.name
  name          = "{{.VappVMWithTemplateName}}"
  computer_name = "compNameUp"

  vapp_template_id = data.vcd_catalog_vapp_template.{{.VappTemplateName}}.id
  firmware         = "efi"

  boot_options {
    efi_secure_boot    = true
    boot_retry_delay   = 10000
    boot_delay         = 1000
    boot_retry_enabled = false
  }
}

resource "vcd_vapp_vm" "{{.VappVMName}}" {
  org = "{{.Org}}"
  vdc = "{{.Vdc}}"

  power_on = true

  vapp_name     = vcd_vapp.{{.VAppName}}.name
  name          = "{{.VappVMName}}"
  computer_name = "compNameUp"

  memory        = 2048
  cpus          = 1

  os_type          = "sles11_64Guest"
  hardware_version = "vmx-13"
  firmware         = "efi"

  boot_options {
    efi_secure_boot    = true
    boot_retry_delay   = 10000
    boot_delay         = 1000
    boot_retry_enabled = false
  }
}

resource "vcd_vm" "{{.EmptyVMName}}" {
  org = "{{.Org}}"
  vdc = "{{.Vdc}}"

  power_on = true

  name          = "{{.EmptyVMName}}"
  computer_name = "compNameUp"

  memory        = 2048
  cpus          = 1

  os_type          = "sles11_64Guest"
  hardware_version = "vmx-13"
  firmware         = "efi"

  boot_options {
    efi_secure_boot    = true
    boot_retry_delay   = 10000
    boot_delay         = 1000
    boot_retry_enabled = false
  }
}

resource "vcd_vm" "{{.VMWithTemplateName}}" {
  org = "{{.Org}}"
  vdc = "{{.Vdc}}"

  power_on = true

  name             = "{{.VMWithTemplateName}}"
  computer_name    = "compNameUp"
  vapp_template_id = data.vcd_catalog_vapp_template.{{.VappTemplateName}}.id

  firmware = "efi"

  boot_options {
    efi_secure_boot    = true
    boot_retry_delay   = 10000
    boot_delay         = 1000
    boot_retry_enabled = false
  }
}
`

func TestAccVcdVAppVmFromTemplateOverrideFirmware(t *testing.T) {
	preTestChecks(t)
	var (
		vapp govcd.VApp
		vm   govcd.VM
	)

	if checkVersion(testConfig.Provider.ApiVersion, "< 37.1") {
		t.Skip("Most boot options are only available since 37.1")
	}

	var params = StringMap{
		"Org":              testConfig.VCD.Org,
		"Vdc":              testConfig.Nsxt.Vdc,
		"CatalogName":      testConfig.VCD.Catalog.NsxtBackedCatalogName,
		"VappTemplateName": testConfig.VCD.Catalog.NsxtCatalogItem,
		"TestName":         t.Name(),

		"Tags": "vapp vm",
	}
	testParamsNotEmpty(t, params)

	params["FuncName"] = t.Name()
	configText1 := templateFill(testAccVcdVAppVmFromTemplateOverrideFirmware, params)
	debugPrintf("#[DEBUG] CONFIGURATION: %s\n", configText1)

	if vcdShortTest {
		t.Skip(acceptanceTestsSkipped)
		return
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccCheckVcdNsxtVAppVmDestroy(t.Name()),
			testAccCheckVcdNsxtVAppVmDestroy(t.Name()+"-standalone"),
		),
		Steps: []resource.TestStep{
			{
				Config: configText1,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckVcdNsxtVAppVmExists(t.Name(), t.Name(), "vcd_vapp_vm.vappvm", &vapp, &vm),
					testAccCheckVcdNsxtStandaloneVmExists(t.Name()+"-standalone", "vcd_vm.vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.vappvm", "firmware", "efi"),
					resource.TestCheckResourceAttr("vcd_vm.vm", "firmware", "efi"),
				),
			},
		},
	})
	postTestChecks(t)
}

const testAccVcdVAppVmFromTemplateOverrideFirmware = `
resource "vcd_vapp" "invapp" {
  name     = "{{.TestName}}"
  org      = "{{.Org}}"
  vdc      = "{{.Vdc}}"
  power_on = false
}

resource "vcd_vapp_vm" "vappvm" {
  org           = "{{.Org}}"
  vdc           = "{{.Vdc}}"
  vapp_name     = vcd_vapp.invapp.name
  name          = "{{.TestName}}"
  computer_name = "{{.TestName}}"
  catalog_name  = "{{.CatalogName}}"
  template_name = "{{.VappTemplateName}}"
  power_on      = false
  memory        = 1024
  cpus          = 2
  cpu_cores     = 1

  hardware_version       = "vmx-17"
  firmware               = "efi"
  os_type                = "ubuntu64Guest"
  boot_options {
    efi_secure_boot = true
  }
}


resource "vcd_vm" "vm" {
  org           = "{{.Org}}"
  vdc           = "{{.Vdc}}"
  name          = "{{.TestName}}-standalone"
  computer_name = "{{.TestName}}"
  catalog_name  = "{{.CatalogName}}"
  template_name = "{{.VappTemplateName}}"
  power_on      = false
  memory        = 1024
  cpus          = 2
  cpu_cores     = 1

  hardware_version       = "vmx-17"
  firmware               = "efi"
  os_type                = "ubuntu64Guest"
  boot_options {
    efi_secure_boot = true
  }
}
`
