//go:build vapp || vm || standaloneVm || ALL || functional

package vcd

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/vmware/go-vcloud-director/v2/govcd"
	"golang.org/x/exp/slices"
)

// Terraform codebase for VM management is very complicated and is backed by 4 SDK types of VM:
//  * `types.InstantiateVmTemplateParams` (Standalone VM from template or copy of another VM)
//  * `types.ReComposeVAppParams` (vApp VM from template or copy of another VM)
//  * `types.RecomposeVAppParamsForEmptyVm` (Empty vApp VM)
//  * `types.CreateVmParams` (Empty Standalone VM)
//
// Each of these 4 types have different fields for creation (just like UI differs), but the
// expectation for the user is to get a VM with all configuration available in HCL, no matter the type.
//
// As a result, the architecture of VM creation is such that it uses the above defined types to
// create VMs with minimal configuration and then perform additions API calls. There are still risks
// that some VMs get less configured than others. To overcome this risk, there is a new set of
// tests. Each of these tests aim to ensure that exactly the same configuration is achieved.

// TestAccVcdVAppVm_4types attempts to test the minimal create configuration for all 4 types of VMs
// Template based VMs inherit their CPU/Memory settings from template, while empty ones must have it
// explicitly specified
//
// Additionally such fields are validated:
// * prevent_update_power_off
// * expose_hardware_virtualization
// * cpu_hot_add_enabled
// * memory_hot_add_enabled
// * description
// * network (also checks that 'ip_allocation_mode' and 'secondary_ip' are not set)
// * power_on
// * status
// * status_text
func TestAccVcdVAppVm_4types(t *testing.T) {
	preTestChecks(t)

	var params = StringMap{
		"TestName":        t.Name(),
		"Org":             testConfig.VCD.Org,
		"Vdc":             testConfig.Nsxt.Vdc,
		"Catalog":         testConfig.VCD.Catalog.NsxtBackedCatalogName,
		"CatalogItem":     testConfig.VCD.Catalog.CatalogItemWithMultiVms,
		"Media":           testConfig.Media.NsxtBackedMediaName,
		"NsxtEdgeGateway": testConfig.Nsxt.EdgeGateway,

		"Tags": "vapp vm",
	}
	testParamsNotEmpty(t, params)

	configTextStep1 := templateFill(testAccVcdVAppVm_4types, params)

	if vcdShortTest {
		t.Skip(acceptanceTestsSkipped)
		return
	}

	params["FuncName"] = t.Name() + "-step2"
	configTextStep2 := templateFill(testAccVcdVAppVm_4types+testAccVcdVAppVm_4typesChecks, params)
	debugPrintf("#[DEBUG] CONFIGURATION: %s\n", configTextStep1)
	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccCheckVcdNsxtVAppVmDestroy(t.Name()+"-template-vm"),
			testAccCheckVcdNsxtVAppVmDestroy(t.Name()+"-empty-vm"),
			testAccCheckVcdStandaloneVmDestroy(t.Name()+"-template-standalone-vm", testConfig.VCD.Org, testConfig.Nsxt.Vdc),
			testAccCheckVcdStandaloneVmDestroy(t.Name()+"-empty-standalone-vm", testConfig.VCD.Org, testConfig.Nsxt.Vdc),
		),
		Steps: []resource.TestStep{
			{
				Config: configTextStep1,
				Check: resource.ComposeAggregateTestCheckFunc(

					// vApp checks
					resource.TestCheckResourceAttr("vcd_vapp.template-vm", "name", t.Name()+"-template-vm"),
					resource.TestCheckResourceAttr("vcd_vapp.template-vm", "description", "vApp for Template VM description"),
					resource.TestCheckResourceAttr("vcd_vapp.template-vm", "power_on", "false"),
					resource.TestCheckResourceAttr("vcd_vapp.template-vm", "status", "1"), // 1 - means RESOLVED
					resource.TestCheckResourceAttr("vcd_vapp.template-vm", "status_text", "RESOLVED"),
					testAccCheckVcdVappPowerState(testConfig.VCD.Org, testConfig.Nsxt.Vdc, t.Name()+"-template-vm", []string{"POWERED_OFF"}),

					resource.TestCheckResourceAttr("vcd_vapp.empty-vm", "name", t.Name()+"-empty-vm"),
					resource.TestCheckResourceAttr("vcd_vapp.empty-vm", "description", "vApp for Empty VM description"),
					resource.TestCheckResourceAttr("vcd_vapp.empty-vm", "power_on", "false"),
					resource.TestCheckResourceAttr("vcd_vapp.empty-vm", "status", "1"), // 1 - means RESOLVED
					resource.TestCheckResourceAttr("vcd_vapp.empty-vm", "status_text", "RESOLVED"),
					testAccCheckVcdVappPowerState(testConfig.VCD.Org, testConfig.Nsxt.Vdc, t.Name()+"-empty-vm", []string{"POWERED_OFF"}),

					// Template vApp VM checks
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "vm_type", "vcd_vapp_vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "name", t.Name()+"-template-vapp-vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "description", t.Name()+"-template-vapp-vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "cpu_hot_add_enabled", "false"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "memory_hot_add_enabled", "false"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "expose_hardware_virtualization", "false"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "prevent_update_power_off", "true"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "network.#", "2"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "network.0.type", "org"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "network.0.adapter_type", "VMXNET3"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "network.0.ip_allocation_mode", "POOL"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "network.0.secondary_ip_allocation_mode", "NONE"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "network.0.secondary_ip", ""),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "network.1.type", "vapp"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "network.1.adapter_type", "E1000"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "network.1.ip_allocation_mode", "POOL"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "network.1.mac", "00:00:00:AA:BB:CC"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "power_on", "false"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "status", "8"), // 8 - means POWERED OFF
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "status_text", "POWERED_OFF"),
					testAccCheckVcdVMPowerState(testConfig.VCD.Org, testConfig.Nsxt.Vdc, t.Name()+"-template-vm", t.Name()+"-template-vapp-vm", "POWERED_OFF"),

					// Empty vApp VM checks
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "vm_type", "vcd_vapp_vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "name", t.Name()+"-empty-vapp-vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "description", t.Name()+"-empty-vapp-vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "computer_name", "vapp-vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "cpus", "1"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "memory", "1024"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "os_type", "sles10_64Guest"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "hardware_version", "vmx-14"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "cpu_hot_add_enabled", "false"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "memory_hot_add_enabled", "false"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "expose_hardware_virtualization", "false"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "prevent_update_power_off", "true"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "network.#", "2"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "network.0.type", "org"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "network.0.adapter_type", "VMXNET3"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "network.0.ip_allocation_mode", "POOL"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "network.0.secondary_ip_allocation_mode", "NONE"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "network.0.secondary_ip", ""),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "network.1.type", "vapp"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "network.1.adapter_type", "E1000"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "network.1.ip_allocation_mode", "POOL"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "network.1.mac", "00:00:00:BB:AA:CC"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "power_on", "false"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "status", "8"), // 8 - means POWERED OFF
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "status_text", "POWERED_OFF"),
					testAccCheckVcdVMPowerState(testConfig.VCD.Org, testConfig.Nsxt.Vdc, t.Name()+"-empty-vm", t.Name()+"-empty-vapp-vm", "POWERED_OFF"),

					// Standalone template VM checks
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "vm_type", "vcd_vm"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "name", t.Name()+"-template-standalone-vm"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "description", t.Name()+"-template-standalone-vm"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "cpu_hot_add_enabled", "false"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "memory_hot_add_enabled", "false"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "expose_hardware_virtualization", "false"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "prevent_update_power_off", "true"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "network.#", "2"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "network.0.type", "org"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "network.0.adapter_type", "VMXNET3"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "network.0.ip_allocation_mode", "POOL"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "network.0.secondary_ip_allocation_mode", "NONE"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "network.0.secondary_ip", ""),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "network.1.type", "org"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "network.1.adapter_type", "E1000E"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "network.1.ip_allocation_mode", "POOL"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "network.1.mac", "00:00:00:11:22:33"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "power_on", "false"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "status", "8"), // 8 - means POWERED OFF
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "status_text", "POWERED_OFF"),
					testAccCheckVcdVMPowerState(testConfig.VCD.Org, testConfig.Nsxt.Vdc, "", t.Name()+"-template-standalone-vm", "POWERED_OFF"),

					// Standalone empty VM checks
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "vm_type", "vcd_vm"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "name", t.Name()+"-empty-standalone-vm"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "description", t.Name()+"-standalone"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "cpus", "1"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "memory", "1024"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "os_type", "sles10_64Guest"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "hardware_version", "vmx-14"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "cpu_hot_add_enabled", "false"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "memory_hot_add_enabled", "false"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "expose_hardware_virtualization", "false"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "prevent_update_power_off", "true"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "network.#", "2"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "network.0.type", "org"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "network.0.adapter_type", "VMXNET3"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "network.0.ip_allocation_mode", "POOL"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "network.0.secondary_ip_allocation_mode", "NONE"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "network.0.secondary_ip", ""),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "network.1.type", "org"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "network.1.adapter_type", "E1000E"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "network.1.ip_allocation_mode", "POOL"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "network.1.mac", "00:00:00:22:33:44"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "power_on", "false"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "status", "8"), // 8 - means POWERED OFF
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "status_text", "POWERED_OFF"),
					testAccCheckVcdVMPowerState(testConfig.VCD.Org, testConfig.Nsxt.Vdc, "", t.Name()+"-empty-standalone-vm", "POWERED_OFF"),

					// VM Copy checks

					// vApp checks
					resource.TestCheckResourceAttr("vcd_vapp.vm-copy-destination-template-vm", "name", t.Name()+"-vm-copy-template-destination"),
					resource.TestCheckResourceAttr("vcd_vapp.vm-copy-destination-template-vm", "description", "vApp destination for VM Copy"),
					resource.TestCheckResourceAttr("vcd_vapp.vm-copy-destination-template-vm", "power_on", "false"),
					resource.TestCheckResourceAttr("vcd_vapp.vm-copy-destination-template-vm", "status", "1"), // 1 - means RESOLVED
					resource.TestCheckResourceAttr("vcd_vapp.vm-copy-destination-template-vm", "status_text", "RESOLVED"),
					testAccCheckVcdVappPowerState(testConfig.VCD.Org, testConfig.Nsxt.Vdc, t.Name()+"-vm-copy-template-destination", []string{"POWERED_OFF"}),

					resource.TestCheckResourceAttr("vcd_vapp.vm-copy-destination-empty-vm", "name", t.Name()+"-vm-copy-empty-destination"),
					resource.TestCheckResourceAttr("vcd_vapp.vm-copy-destination-empty-vm", "description", "vApp destination for VM Copy"),
					resource.TestCheckResourceAttr("vcd_vapp.vm-copy-destination-empty-vm", "power_on", "false"),
					resource.TestCheckResourceAttr("vcd_vapp.vm-copy-destination-empty-vm", "status", "1"), // 1 - means RESOLVED
					resource.TestCheckResourceAttr("vcd_vapp.vm-copy-destination-empty-vm", "status_text", "RESOLVED"),
					testAccCheckVcdVappPowerState(testConfig.VCD.Org, testConfig.Nsxt.Vdc, t.Name()+"-vm-copy-empty-destination", []string{"POWERED_OFF"}),

					// Template vApp VM checks
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "vm_type", "vcd_vapp_vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "name", t.Name()+"-template-vapp-vm-copy"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "description", t.Name()+"-template-vapp-vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "cpu_hot_add_enabled", "false"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "memory_hot_add_enabled", "false"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "expose_hardware_virtualization", "false"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "prevent_update_power_off", "true"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "network.#", "2"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "network.0.type", "org"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "network.0.adapter_type", "VMXNET3"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "network.0.ip_allocation_mode", "POOL"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "network.0.secondary_ip_allocation_mode", "NONE"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "network.0.secondary_ip", ""),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "network.1.type", "vapp"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "network.1.adapter_type", "E1000"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "network.1.ip_allocation_mode", "POOL"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "network.1.mac", "00:00:00:AA:AC:CC"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "power_on", "false"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "status", "8"), // 8 - means POWERED OFF
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "status_text", "POWERED_OFF"),
					testAccCheckVcdVMPowerState(testConfig.VCD.Org, testConfig.Nsxt.Vdc, t.Name()+"-vm-copy-template-destination", t.Name()+"-template-vapp-vm-copy", "POWERED_OFF"),

					// Empty vApp VM checks
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "vm_type", "vcd_vapp_vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "name", t.Name()+"-empty-vapp-vm-copy"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "description", t.Name()+"-empty-vapp-vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "computer_name", "vapp-vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "cpus", "1"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "memory", "1024"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "os_type", "sles10_64Guest"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "hardware_version", "vmx-14"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "cpu_hot_add_enabled", "false"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "memory_hot_add_enabled", "false"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "expose_hardware_virtualization", "false"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "prevent_update_power_off", "true"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "network.#", "2"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "network.0.type", "org"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "network.0.adapter_type", "VMXNET3"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "network.0.ip_allocation_mode", "POOL"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "network.0.secondary_ip_allocation_mode", "NONE"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "network.0.secondary_ip", ""),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "network.1.type", "vapp"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "network.1.adapter_type", "E1000"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "network.1.ip_allocation_mode", "POOL"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "network.1.mac", "00:00:00:AA:BB:FC"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "power_on", "false"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "status", "8"), // 8 - means POWERED OFF
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "status_text", "POWERED_OFF"),
					testAccCheckVcdVMPowerState(testConfig.VCD.Org, testConfig.Nsxt.Vdc, t.Name()+"-vm-copy-empty-destination", t.Name()+"-empty-vapp-vm-copy", "POWERED_OFF"),

					// Standalone template VM checks
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "vm_type", "vcd_vm"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "name", t.Name()+"-template-standalone-vm-copy"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "description", t.Name()+"-template-standalone-vm"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "cpu_hot_add_enabled", "false"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "memory_hot_add_enabled", "false"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "expose_hardware_virtualization", "false"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "prevent_update_power_off", "true"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "network.#", "2"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "network.0.type", "org"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "network.0.adapter_type", "VMXNET3"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "network.0.ip_allocation_mode", "POOL"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "network.0.secondary_ip_allocation_mode", "NONE"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "network.0.secondary_ip", ""),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "network.1.type", "org"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "network.1.adapter_type", "E1000E"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "network.1.ip_allocation_mode", "POOL"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "network.1.mac", "00:00:00:11:FF:33"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "power_on", "false"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "status", "8"), // 8 - means POWERED OFF
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "status_text", "POWERED_OFF"),
					testAccCheckVcdVMPowerState(testConfig.VCD.Org, testConfig.Nsxt.Vdc, "", t.Name()+"-template-standalone-vm-copy", "POWERED_OFF"),

					// Standalone empty VM checks
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "vm_type", "vcd_vm"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "name", t.Name()+"-empty-standalone-vm-copy"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "description", t.Name()+"-standalone"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "cpus", "1"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "memory", "1024"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "os_type", "sles10_64Guest"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "hardware_version", "vmx-14"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "cpu_hot_add_enabled", "false"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "memory_hot_add_enabled", "false"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "expose_hardware_virtualization", "false"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "prevent_update_power_off", "true"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "network.#", "2"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "network.0.type", "org"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "network.0.adapter_type", "VMXNET3"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "network.0.ip_allocation_mode", "POOL"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "network.0.secondary_ip_allocation_mode", "NONE"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "network.0.secondary_ip", ""),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "network.1.type", "org"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "network.1.adapter_type", "E1000E"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "network.1.ip_allocation_mode", "POOL"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "network.1.mac", "00:00:00:22:33:44"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "power_on", "false"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "status", "8"), // 8 - means POWERED OFF
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "status_text", "POWERED_OFF"),
					testAccCheckVcdVMPowerState(testConfig.VCD.Org, testConfig.Nsxt.Vdc, "", t.Name()+"-empty-standalone-vm-copy", "POWERED_OFF"),
				),
			},
			{
				Config: configTextStep2,
				Check: resource.ComposeAggregateTestCheckFunc(

					// checking  name collections in vApp resource
					resource.TestCheckResourceAttr("vcd_vapp.template-vm", "vapp_network_names.#", "1"),
					resource.TestCheckResourceAttr("vcd_vapp.template-vm", "vapp_org_network_names.#", "1"),
					resource.TestCheckResourceAttr("vcd_vapp.template-vm", "vm_names.#", "1"),
					resource.TestCheckResourceAttr("vcd_vapp.template-vm", "vm_names.#", "1"),
					resource.TestCheckResourceAttr("vcd_vapp.template-vm", "vm_names.0", t.Name()+"-template-vapp-vm"),
					resource.TestCheckResourceAttr("vcd_vapp.template-vm", "vapp_network_names.0", t.Name()+"-template-vm"),
					resource.TestCheckResourceAttr("vcd_vapp.template-vm", "vapp_org_network_names.0", t.Name()),

					// checking  name collections in vApp data source
					resource.TestCheckResourceAttr("data.vcd_vapp.template-vm-ds", "vapp_network_names.#", "1"),
					resource.TestCheckResourceAttr("data.vcd_vapp.template-vm-ds", "vapp_org_network_names.#", "1"),
					resource.TestCheckResourceAttr("data.vcd_vapp.template-vm-ds", "vm_names.#", "1"),
					resource.TestCheckResourceAttr("data.vcd_vapp.template-vm-ds", "vm_names.0", t.Name()+"-template-vapp-vm"),
					resource.TestCheckResourceAttr("data.vcd_vapp.template-vm-ds", "vapp_network_names.0", t.Name()+"-template-vm"),
					resource.TestCheckResourceAttr("data.vcd_vapp.template-vm-ds", "vapp_org_network_names.0", t.Name()),

					// checking  name collections in vcd_resource_list
					resource.TestCheckResourceAttr("data.vcd_resource_list.vapp_networks", "list.#", "1"),
					checkListForKnownItem("vapp_networks", t.Name()+"-template-vm", "", true, false),
					resource.TestCheckResourceAttr("data.vcd_resource_list.vapp_org_networks", "list.#", "1"),
					checkListForKnownItem("vapp_org_networks", t.Name(), "", true, false),
				),
			},
		},
	})
	postTestChecks(t)
}

const testAccVcdVAppVm_4types = `
data "vcd_org_vdc" "nsxt" {
  org  = "{{.Org}}"
  name = "{{.Vdc}}"
}

data "vcd_nsxt_edgegateway" "t1" {
  org      = "{{.Org}}"
  owner_id = data.vcd_org_vdc.nsxt.id
  name     = "{{.NsxtEdgeGateway}}"
}

data "vcd_catalog" "{{.Catalog}}" {
	org  = "{{.Org}}"
	name = "{{.Catalog}}"
}

data "vcd_catalog_vapp_template" "{{.CatalogItem}}" {
	org         = "{{.Org}}"
	catalog_id = data.vcd_catalog.{{.Catalog}}.id
	name       = "{{.CatalogItem}}"
}

data "vcd_catalog_media" "{{.Media}}" {
	org     = "{{.Org}}"
	catalog = data.vcd_catalog.{{.Catalog}}.name
	name    = "{{.Media}}"
}

resource "vcd_network_routed_v2" "nsxt-backed" {
  org = "{{.Org}}"

  edge_gateway_id = data.vcd_nsxt_edgegateway.t1.id

  name = "{{.TestName}}"

  gateway       = "1.1.1.1"
  prefix_length = 24

  static_ip_pool {
    start_address = "1.1.1.10"
    end_address   = "1.1.1.40"
  }
}

resource "vcd_vapp" "template-vm" {
  org         = "{{.Org}}"
  vdc         = "{{.Vdc}}"
  name        = "{{.TestName}}-template-vm"
  description = "vApp for Template VM description"
  power_on    = false
}

resource "vcd_vapp_network" "template" {
  org  = "{{.Org}}"
  vdc  = "{{.Vdc}}"
  name = "{{.TestName}}-template-vm"

  vapp_name          = (vcd_vapp.template-vm.id == "always-not-equal" ? null : vcd_vapp.template-vm.name)
  gateway            = "192.168.3.1"
  netmask            = "255.255.255.0"

  static_ip_pool {
	start_address = "192.168.3.51"
	end_address   = "192.168.3.100"
  }

  depends_on = [vcd_vapp.template-vm]
}

resource "vcd_vapp_org_network" "template-vapp" {
  org = "{{.Org}}"
  vdc = "{{.Vdc}}"

  vapp_name        = (vcd_vapp.template-vm.id == "always-not-equal" ? null : vcd_vapp.template-vm.name)
  org_network_name = (vcd_network_routed_v2.nsxt-backed.id == "always-not-equal" ? null : vcd_network_routed_v2.nsxt-backed.name)
}

resource "vcd_vapp" "empty-vm" {
  org         = "{{.Org}}"
  vdc         = "{{.Vdc}}"
  name        = "{{.TestName}}-empty-vm"
  description = "vApp for Empty VM description"
  power_on    = false
}

resource "vcd_vapp_network" "empty-vm" {
  org  = "{{.Org}}"
  vdc  = "{{.Vdc}}"
  name = "{{.TestName}}-empty-vm"

  vapp_name          = (vcd_vapp.empty-vm.id == "always-not-equal" ? null : vcd_vapp.empty-vm.name)
  gateway            = "192.168.2.1"
  netmask            = "255.255.255.0"

  static_ip_pool {
	start_address = "192.168.2.51"
	end_address   = "192.168.2.100"
  }

  depends_on = [vcd_vapp.empty-vm]
}

resource "vcd_vapp_org_network" "empty-vapp" {
  org = "{{.Org}}"
  vdc = "{{.Vdc}}"

  vapp_name        = (vcd_vapp.empty-vm.id == "always-not-equal" ? null : vcd_vapp.empty-vm.name)
  org_network_name = (vcd_network_routed_v2.nsxt-backed.id == "always-not-equal" ? null : vcd_network_routed_v2.nsxt-backed.name)
}

resource "vcd_vapp_vm" "template-vm" {
  org  = "{{.Org}}"
  vdc  = "{{.Vdc}}"

  vapp_template_id = data.vcd_catalog_vapp_template.{{.CatalogItem}}.id
  
  vapp_name   = vcd_vapp.template-vm.name
  name        = "{{.TestName}}-template-vapp-vm"
  description = "{{.TestName}}-template-vapp-vm"
  power_on    = false

  network {
	type               = "org"
	name               = (vcd_vapp_org_network.template-vapp.id == "always-not-equal" ? null : vcd_vapp_org_network.template-vapp.org_network_name)
	adapter_type       = "VMXNET3"
	ip_allocation_mode = "POOL"
  }

  network {
	type               = "vapp"
	name               = (vcd_vapp_network.template.id == "always-not-equal" ? null : vcd_vapp_network.template.name)
	adapter_type       = "E1000"
	ip_allocation_mode = "POOL"
	mac                = "00:00:00:AA:BB:CC"
  }

  depends_on = [vcd_vapp_network.template]

  prevent_update_power_off = true
}

resource "vcd_vapp_vm" "empty-vm" {
  org  = "{{.Org}}"
  vdc  = "{{.Vdc}}"
  
  vapp_name     = vcd_vapp.empty-vm.name
  name          = "{{.TestName}}-empty-vapp-vm"
  description   = "{{.TestName}}-empty-vapp-vm"
  computer_name = "vapp-vm"
  power_on      = false

  cpus   = 1
  memory = 1024

  os_type          = "sles10_64Guest"
  hardware_version = "vmx-14"
  boot_image_id    = data.vcd_catalog_media.{{.Media}}.id

  network {
	type               = "org"
	name               = (vcd_vapp_org_network.empty-vapp.id == "always-not-equal" ? null : vcd_vapp_org_network.empty-vapp.org_network_name)
	adapter_type       = "VMXNET3"
	ip_allocation_mode = "POOL"
  }

  network {
	type               = "vapp"
	name               = (vcd_vapp_network.empty-vm.id == "always-not-equal" ? null : vcd_vapp_network.empty-vm.name)
	adapter_type       = "E1000"
	ip_allocation_mode = "POOL"
	mac                = "00:00:00:BB:AA:CC"
  }

  depends_on = [vcd_vapp_network.empty-vm]

  prevent_update_power_off = true
}

resource "vcd_vm" "template-vm" {
  org  = "{{.Org}}"
  vdc  = "{{.Vdc}}"

  vapp_template_id = data.vcd_catalog_vapp_template.{{.CatalogItem}}.id
  
  name        = "{{.TestName}}-template-standalone-vm"
  description = "{{.TestName}}-template-standalone-vm"
  power_on    = false

  network {
	type               = "org"
	name               = (vcd_network_routed_v2.nsxt-backed.id == "always-not-equal" ? null : vcd_network_routed_v2.nsxt-backed.name)
	adapter_type       = "VMXNET3"
	ip_allocation_mode = "POOL"
  }

  network {
	type               = "org"
	name               = (vcd_network_routed_v2.nsxt-backed.id == "always-not-equal" ? null : vcd_network_routed_v2.nsxt-backed.name)
	adapter_type       = "E1000E"
	ip_allocation_mode = "POOL"
	mac                = "00:00:00:11:22:33"
  }

  prevent_update_power_off = true
}

resource "vcd_vm" "empty-vm" {
  org  = "{{.Org}}"
  vdc  = "{{.Vdc}}"

  name          = "{{.TestName}}-empty-standalone-vm"
  description   = "{{.TestName}}-standalone"
  computer_name = "standalone"
  power_on      = false

  cpus   = 1
  memory = 1024

  os_type          = "sles10_64Guest"
  hardware_version = "vmx-14"
  boot_image_id    = data.vcd_catalog_media.{{.Media}}.id

  network {
	type               = "org"
	name               = (vcd_network_routed_v2.nsxt-backed.id == "always-not-equal" ? null : vcd_network_routed_v2.nsxt-backed.name)
	adapter_type       = "VMXNET3"
	ip_allocation_mode = "POOL"
  }

  network {
	type               = "org"
	name               = (vcd_network_routed_v2.nsxt-backed.id == "always-not-equal" ? null : vcd_network_routed_v2.nsxt-backed.name)
	adapter_type       = "E1000E"
	ip_allocation_mode = "POOL"
	mac                = "00:00:00:22:33:44"
  }

  prevent_update_power_off = true
}

# VM Copy from here
resource "vcd_vapp" "vm-copy-destination-template-vm" {
  org         = "{{.Org}}"
  vdc         = "{{.Vdc}}"
  name        = "{{.TestName}}-vm-copy-template-destination"
  description = "vApp destination for VM Copy"
  power_on    = false
}

resource "vcd_vapp_network" "template-copy" {
  org  = "{{.Org}}"
  vdc  = "{{.Vdc}}"
  name = "{{.TestName}}-template-vm-copy"

  vapp_name          = vcd_vapp.vm-copy-destination-template-vm.name
  gateway            = "192.168.3.1"
  netmask            = "255.255.255.0"

  static_ip_pool {
	start_address = "192.168.3.51"
	end_address   = "192.168.3.100"
  }

  depends_on = [vcd_vapp.template-vm]
}

resource "vcd_vapp_org_network" "template-vapp-copy" {
  org = "{{.Org}}"
  vdc = "{{.Vdc}}"

  vapp_name        = vcd_vapp.vm-copy-destination-template-vm.name
  org_network_name = (vcd_network_routed_v2.nsxt-backed.id == "always-not-equal" ? null : vcd_network_routed_v2.nsxt-backed.name)
}

resource "vcd_vapp" "vm-copy-destination-empty-vm" {
  org         = "{{.Org}}"
  vdc         = "{{.Vdc}}"
  name        = "{{.TestName}}-vm-copy-empty-destination"
  description = "vApp destination for VM Copy"
  power_on    = false
}

resource "vcd_vapp_network" "empty-vm-copy" {
  org  = "{{.Org}}"
  vdc  = "{{.Vdc}}"
  name = "{{.TestName}}-empty-vm-copy"

  vapp_name          = vcd_vapp.vm-copy-destination-empty-vm.name
  gateway            = "192.168.2.1"
  netmask            = "255.255.255.0"

  static_ip_pool {
	start_address = "192.168.2.51"
	end_address   = "192.168.2.100"
  }

  depends_on = [vcd_vapp.empty-vm]
}

resource "vcd_vapp_org_network" "empty-vapp-copy" {
  org = "{{.Org}}"
  vdc = "{{.Vdc}}"

  vapp_name        = vcd_vapp.vm-copy-destination-empty-vm.name
  org_network_name = (vcd_network_routed_v2.nsxt-backed.id == "always-not-equal" ? null : vcd_network_routed_v2.nsxt-backed.name)
}

resource "vcd_vapp_vm" "template-vm-copy" {
  org  = "{{.Org}}"
  vdc  = "{{.Vdc}}"
  
  copy_from_vm_id = vcd_vapp_vm.template-vm.id
  vapp_name       = vcd_vapp.vm-copy-destination-template-vm.name
  name            = "{{.TestName}}-template-vapp-vm-copy"
  description     = "{{.TestName}}-template-vapp-vm"
  power_on        = false

  network {
	type               = "org"
	name               = vcd_vapp_org_network.template-vapp-copy.org_network_name
	adapter_type       = "VMXNET3"
	ip_allocation_mode = "POOL"
  }

  network {
	type               = "vapp"
	name               = vcd_vapp_network.template-copy.name
	adapter_type       = "E1000"
	ip_allocation_mode = "POOL"
	mac                = "00:00:00:AA:AC:CC"
  }

  prevent_update_power_off = true

  depends_on = [vcd_vapp_org_network.template-vapp-copy, vcd_vapp_network.template-copy]
}

resource "vcd_vapp_vm" "empty-vm-copy" {
  org  = "{{.Org}}"
  vdc  = "{{.Vdc}}"
  
  vapp_name     = vcd_vapp.vm-copy-destination-empty-vm.name
  name          = "{{.TestName}}-empty-vapp-vm-copy"
  description   = "{{.TestName}}-empty-vapp-vm"
  computer_name = "vapp-vm"
  power_on      = false

  cpus   = 1
  memory = 1024

  copy_from_vm_id = vcd_vapp_vm.empty-vm.id

  network {
	type               = "org"
	name               = vcd_vapp_org_network.empty-vapp-copy.org_network_name
	adapter_type       = "VMXNET3"
	ip_allocation_mode = "POOL"
  }

  network {
	type               = "vapp"
	name               = vcd_vapp_network.empty-vm-copy.name
	adapter_type       = "E1000"
	ip_allocation_mode = "POOL"
	mac                = "00:00:00:AA:BB:FC"
  }

  prevent_update_power_off = true
  depends_on = [vcd_vapp_org_network.template-vapp-copy, vcd_vapp_network.template-copy]
}

resource "vcd_vm" "template-vm-copy" {
  org  = "{{.Org}}"
  vdc  = "{{.Vdc}}"

  copy_from_vm_id = vcd_vm.template-vm.id
  
  name        = "{{.TestName}}-template-standalone-vm-copy"
  description = "{{.TestName}}-template-standalone-vm"
  power_on    = false

  network {
	type               = "org"
	name               = (vcd_network_routed_v2.nsxt-backed.id == "always-not-equal" ? null : vcd_network_routed_v2.nsxt-backed.name)
	adapter_type       = "VMXNET3"
	ip_allocation_mode = "POOL"
  }

  network {
	type               = "org"
	name               = (vcd_network_routed_v2.nsxt-backed.id == "always-not-equal" ? null : vcd_network_routed_v2.nsxt-backed.name)
	adapter_type       = "E1000E"
	ip_allocation_mode = "POOL"
	mac                = "00:00:00:11:FF:33"
  }

  prevent_update_power_off = true
}

resource "vcd_vm" "empty-vm-copy" {
  org  = "{{.Org}}"
  vdc  = "{{.Vdc}}"

  name          = "{{.TestName}}-empty-standalone-vm-copy"
  description   = "{{.TestName}}-standalone"
  computer_name = "standalone"
  power_on      = false

  cpus   = 1
  memory = 1024

  copy_from_vm_id = vcd_vm.empty-vm.id

  network {
	type               = "org"
	name               = (vcd_network_routed_v2.nsxt-backed.id == "always-not-equal" ? null : vcd_network_routed_v2.nsxt-backed.name)
	adapter_type       = "VMXNET3"
	ip_allocation_mode = "POOL"
  }

  network {
	type               = "org"
	name               = (vcd_network_routed_v2.nsxt-backed.id == "always-not-equal" ? null : vcd_network_routed_v2.nsxt-backed.name)
	adapter_type       = "E1000E"
	ip_allocation_mode = "POOL"
	mac                = "00:00:00:22:33:44"
  }

  prevent_update_power_off = true
}
`

const testAccVcdVAppVm_4typesChecks = `
data "vcd_vapp" "template-vm-ds" {
  org  = "{{.Org}}"
  vdc  = "{{.Vdc}}"
  name = vcd_vapp.template-vm.name
}

data "vcd_resource_list" "vapp_networks" {
  org           = "{{.Org}}"
  vdc           = "{{.Vdc}}"
  name          = "vapp_networks"
  parent        = data.vcd_vapp.template-vm-ds.name
  resource_type = "vcd_vapp_network"
}

data "vcd_resource_list" "vapp_org_networks" {
  org           = "{{.Org}}"
  vdc           = "{{.Vdc}}"
  name          = "vapp_org_networks"
  parent        = data.vcd_vapp.template-vm-ds.name
  resource_type = "vcd_vapp_org_network"
}
`

// TestAccVcdVAppVm_4types_storage_profile validates that storage profile assignment works correctly
// as well as the following fields
// * cpu_hot_add_enabled
// * memory_hot_add_enabled
// * computer_name
// * expose_hardware_virtualization
// * metadata
// * guest_properties
// * power_on
// * description
func TestAccVcdVAppVm_4types_storage_profile(t *testing.T) {
	preTestChecks(t)

	var params = StringMap{
		"TestName":       t.Name(),
		"Org":            testConfig.VCD.Org,
		"Vdc":            testConfig.Nsxt.Vdc,
		"Catalog":        testConfig.VCD.Catalog.NsxtBackedCatalogName,
		"CatalogItem":    testConfig.VCD.Catalog.CatalogItemWithMultiVms,
		"Media":          testConfig.Media.NsxtBackedMediaName,
		"StorageProfile": testConfig.VCD.NsxtProviderVdc.StorageProfile2,

		"Tags": "vapp vm",
	}
	testParamsNotEmpty(t, params)

	configTextStep1 := templateFill(testAccVcdVAppVm_4types_storage_profile, params)

	if vcdShortTest {
		t.Skip(acceptanceTestsSkipped)
		return
	}
	debugPrintf("#[DEBUG] CONFIGURATION: %s\n", configTextStep1)
	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccCheckVcdNsxtVAppVmDestroy(t.Name()+"-template-vm"),
			testAccCheckVcdNsxtVAppVmDestroy(t.Name()+"-empty-vm"),
			testAccCheckVcdStandaloneVmDestroy(t.Name()+"-template-standalone-vm", testConfig.VCD.Org, testConfig.Nsxt.Vdc),
			testAccCheckVcdStandaloneVmDestroy(t.Name()+"-empty-standalone-vm", testConfig.VCD.Org, testConfig.Nsxt.Vdc),
		),
		Steps: []resource.TestStep{
			{
				Config: configTextStep1,
				Check: resource.ComposeAggregateTestCheckFunc(

					// vApp checks
					resource.TestCheckResourceAttr("vcd_vapp.template-vm", "name", t.Name()+"-template-vm"),
					resource.TestCheckResourceAttr("vcd_vapp.template-vm", "description", "vApp for Template VM description"),
					testAccCheckVcdVappPowerState(testConfig.VCD.Org, testConfig.Nsxt.Vdc, t.Name()+"-template-vm", []string{"POWERED_ON"}),

					resource.TestCheckResourceAttr("vcd_vapp.empty-vm", "name", t.Name()+"-empty-vm"),
					resource.TestCheckResourceAttr("vcd_vapp.empty-vm", "description", "vApp for Empty VM description"),
					testAccCheckVcdVappPowerState(testConfig.VCD.Org, testConfig.Nsxt.Vdc, t.Name()+"-empty-vm", []string{"POWERED_ON"}),

					// Template vApp VM checks
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "vm_type", "vcd_vapp_vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "name", t.Name()+"-template-vapp-vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "description", ""),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "storage_profile", params["StorageProfile"].(string)),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "computer_name", "comp-name"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "cpu_hot_add_enabled", "true"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "memory_hot_add_enabled", "true"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "expose_hardware_virtualization", "true"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "metadata.vm1", "VM Metadata"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "metadata.vm2", "VM Metadata2"),
					testMatchResourceAttrWhenVersionMatches("vcd_vapp_vm.template-vm", "inherited_metadata.vm.origin.id", regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`), ">= 38.1"),
					testCheckResourceAttrSetWhenVersionMatches("vcd_vapp_vm.template-vm", "inherited_metadata.vm.origin.name", ">= 38.1"),
					testMatchResourceAttrWhenVersionMatches("vcd_vapp_vm.template-vm", "inherited_metadata.vm.origin.type", regexp.MustCompile(`^com\.vmware\.vcloud\.entity\.\w+$`), ">= 38.1"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", `guest_properties.guest.hostname`, "test-host"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", `guest_properties.guest.another.subkey`, "another-value"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "network.#", "0"),
					testAccCheckVcdVMPowerState(testConfig.VCD.Org, testConfig.Nsxt.Vdc, t.Name()+"-template-vm", t.Name()+"-template-vapp-vm", "POWERED_ON"),

					// Empty vApp VM checks
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "vm_type", "vcd_vapp_vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "name", t.Name()+"-empty-vapp-vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "description", ""),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "computer_name", "comp-name"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "cpus", "1"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "memory", "1024"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "os_type", "rhel8_64Guest"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "hardware_version", "vmx-17"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "storage_profile", params["StorageProfile"].(string)),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "computer_name", "comp-name"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "cpu_hot_add_enabled", "true"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "memory_hot_add_enabled", "true"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "expose_hardware_virtualization", "true"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "metadata.vm1", "VM Metadata"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "metadata.vm2", "VM Metadata2"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", `guest_properties.guest.hostname`, "test-host"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", `guest_properties.guest.another.subkey`, "another-value"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "network.#", "0"),
					testAccCheckVcdVMPowerState(testConfig.VCD.Org, testConfig.Nsxt.Vdc, t.Name()+"-empty-vm", t.Name()+"-empty-vapp-vm", "POWERED_ON"),

					// Standalone template VM checks
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "vm_type", "vcd_vm"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "name", t.Name()+"-template-standalone-vm"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "description", ""), //  Inherited from vApp template
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "storage_profile", params["StorageProfile"].(string)),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "computer_name", "comp-name"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "cpu_hot_add_enabled", "true"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "memory_hot_add_enabled", "true"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "expose_hardware_virtualization", "true"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "metadata.vm1", "VM Metadata"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "metadata.vm2", "VM Metadata2"),
					testMatchResourceAttrWhenVersionMatches("vcd_vapp_vm.template-vm", "inherited_metadata.vm.origin.id", regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`), ">= 38.1"),
					testCheckResourceAttrSetWhenVersionMatches("vcd_vapp_vm.template-vm", "inherited_metadata.vm.origin.name", ">= 38.1"),
					testMatchResourceAttrWhenVersionMatches("vcd_vapp_vm.template-vm", "inherited_metadata.vm.origin.type", regexp.MustCompile(`^com\.vmware\.vcloud\.entity\.\w+$`), ">= 38.1"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", `guest_properties.guest.hostname`, "test-host"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", `guest_properties.guest.another.subkey`, "another-value"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "network.#", "0"),
					testAccCheckVcdVMPowerState(testConfig.VCD.Org, testConfig.Nsxt.Vdc, "", t.Name()+"-template-standalone-vm", "POWERED_ON"),

					// Standalone empty VM checks
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "vm_type", "vcd_vm"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "name", t.Name()+"-empty-standalone-vm"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "description", ""),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "cpus", "1"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "memory", "1024"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "os_type", "rhel8_64Guest"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "hardware_version", "vmx-17"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "storage_profile", params["StorageProfile"].(string)),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "computer_name", "comp-name"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "cpu_hot_add_enabled", "true"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "memory_hot_add_enabled", "true"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "expose_hardware_virtualization", "true"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "metadata.vm1", "VM Metadata"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "metadata.vm2", "VM Metadata2"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", `guest_properties.guest.hostname`, "test-host"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", `guest_properties.guest.another.subkey`, "another-value"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "network.#", "0"),
					testAccCheckVcdVMPowerState(testConfig.VCD.Org, testConfig.Nsxt.Vdc, "", t.Name()+"-empty-standalone-vm", "POWERED_ON"),

					// VM Copy checks

					// vApp checks
					resource.TestCheckResourceAttr("vcd_vapp.vm-copy-destination-template-vm", "name", t.Name()+"-vm-copy-template-destination"),
					resource.TestCheckResourceAttr("vcd_vapp.vm-copy-destination-template-vm", "description", "vApp destination for VM Copy"),
					resource.TestCheckResourceAttr("vcd_vapp.vm-copy-destination-template-vm", "power_on", "true"),
					resource.TestCheckResourceAttr("vcd_vapp.vm-copy-destination-template-vm", "status", "4"), // 4 - means POWERED_ON
					resource.TestCheckResourceAttr("vcd_vapp.vm-copy-destination-template-vm", "status_text", "POWERED_ON"),
					testAccCheckVcdVappPowerState(testConfig.VCD.Org, testConfig.Nsxt.Vdc, t.Name()+"-vm-copy-template-destination", []string{"POWERED_ON"}),

					resource.TestCheckResourceAttr("vcd_vapp.vm-copy-destination-empty-vm", "name", t.Name()+"-vm-copy-empty-destination"),
					resource.TestCheckResourceAttr("vcd_vapp.vm-copy-destination-empty-vm", "description", "vApp destination for VM Copy"),
					resource.TestCheckResourceAttr("vcd_vapp.vm-copy-destination-empty-vm", "power_on", "true"),
					resource.TestCheckResourceAttr("vcd_vapp.vm-copy-destination-empty-vm", "status", "4"), // 4 - means POWERED_ON
					resource.TestCheckResourceAttr("vcd_vapp.vm-copy-destination-empty-vm", "status_text", "POWERED_ON"),
					testAccCheckVcdVappPowerState(testConfig.VCD.Org, testConfig.Nsxt.Vdc, t.Name()+"-vm-copy-empty-destination", []string{"POWERED_ON"}),

					// Template vApp VM checks
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "vm_type", "vcd_vapp_vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "name", t.Name()+"-template-vapp-vm-copy"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "description", t.Name()+"-template-vapp-vm-copy"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "storage_profile", params["StorageProfile"].(string)),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "computer_name", "comp-name"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "cpu_hot_add_enabled", "true"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "memory_hot_add_enabled", "true"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "expose_hardware_virtualization", "true"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "metadata.vm1", "VM Metadata"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "metadata.vm2", "VM Metadata2"),
					testMatchResourceAttrWhenVersionMatches("vcd_vapp_vm.template-vm-copy", "inherited_metadata.vm.origin.id", regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`), ">= 38.1"),
					testCheckResourceAttrSetWhenVersionMatches("vcd_vapp_vm.template-vm-copy", "inherited_metadata.vm.origin.name", ">= 38.1"),
					testMatchResourceAttrWhenVersionMatches("vcd_vapp_vm.template-vm-copy", "inherited_metadata.vm.origin.type", regexp.MustCompile(`^com\.vmware\.vcloud\.entity\.\w+$`), ">= 38.1"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", `guest_properties.guest.hostname`, "test-host"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", `guest_properties.guest.another.subkey`, "another-value"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "network.#", "0"),
					testAccCheckVcdVMPowerState(testConfig.VCD.Org, testConfig.Nsxt.Vdc, t.Name()+"-vm-copy-template-destination", t.Name()+"-template-vapp-vm-copy", "POWERED_ON"),

					// Empty vApp VM checks
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "vm_type", "vcd_vapp_vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "name", t.Name()+"-empty-vapp-vm-copy"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "description", ""),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "computer_name", "comp-name"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "cpus", "1"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "memory", "1024"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "os_type", "rhel8_64Guest"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "hardware_version", "vmx-17"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "storage_profile", params["StorageProfile"].(string)),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "computer_name", "comp-name"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "cpu_hot_add_enabled", "true"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "memory_hot_add_enabled", "true"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "expose_hardware_virtualization", "true"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "metadata.vm1", "VM Metadata"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "metadata.vm2", "VM Metadata2"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", `guest_properties.guest.hostname`, "test-host"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", `guest_properties.guest.another.subkey`, "another-value"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "network.#", "0"),
					testAccCheckVcdVMPowerState(testConfig.VCD.Org, testConfig.Nsxt.Vdc, t.Name()+"-vm-copy-empty-destination", t.Name()+"-empty-vapp-vm-copy", "POWERED_ON"),

					// Standalone template VM checks
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "vm_type", "vcd_vm"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "name", t.Name()+"-template-standalone-vm-copy"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "description", ""), //  Inherited from vApp template
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "storage_profile", params["StorageProfile"].(string)),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "computer_name", "comp-name"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "cpu_hot_add_enabled", "true"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "memory_hot_add_enabled", "true"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "expose_hardware_virtualization", "true"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "metadata.vm1", "VM Metadata"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "metadata.vm2", "VM Metadata2"),
					testMatchResourceAttrWhenVersionMatches("vcd_vm.template-vm-copy", "inherited_metadata.vm.origin.id", regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`), ">= 38.1"),
					testCheckResourceAttrSetWhenVersionMatches("vcd_vm.template-vm-copy", "inherited_metadata.vm.origin.name", ">= 38.1"),
					testMatchResourceAttrWhenVersionMatches("vcd_vm.template-vm-copy", "inherited_metadata.vm.origin.type", regexp.MustCompile(`^com\.vmware\.vcloud\.entity\.\w+$`), ">= 38.1"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", `guest_properties.guest.hostname`, "test-host"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", `guest_properties.guest.another.subkey`, "another-value"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "network.#", "0"),
					testAccCheckVcdVMPowerState(testConfig.VCD.Org, testConfig.Nsxt.Vdc, "", t.Name()+"-template-standalone-vm-copy", "POWERED_ON"),

					// Standalone empty VM checks
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "vm_type", "vcd_vm"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "name", t.Name()+"-empty-standalone-vm-copy"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "description", ""),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "cpus", "1"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "memory", "1024"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "os_type", "rhel8_64Guest"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "hardware_version", "vmx-17"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "storage_profile", params["StorageProfile"].(string)),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "computer_name", "comp-name"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "cpu_hot_add_enabled", "true"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "memory_hot_add_enabled", "true"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "expose_hardware_virtualization", "true"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "metadata.vm1", "VM Metadata"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "metadata.vm2", "VM Metadata2"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", `guest_properties.guest.hostname`, "test-host"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", `guest_properties.guest.another.subkey`, "another-value"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "network.#", "0"),
					testAccCheckVcdVMPowerState(testConfig.VCD.Org, testConfig.Nsxt.Vdc, "", t.Name()+"-empty-standalone-vm-copy", "POWERED_ON"),
				),
			},
		},
	})
	postTestChecks(t)
}

const testAccVcdVAppVm_4types_storage_profile = `
data "vcd_storage_profile" "nsxt-vdc" {
  org  = "{{.Org}}"
  vdc  = "{{.Vdc}}"
  name = "{{.StorageProfile}}"
}

resource "vcd_vapp" "template-vm" {
  org         = "{{.Org}}"
  vdc         = "{{.Vdc}}"
  name        = "{{.TestName}}-template-vm"
  description = "vApp for Template VM description"
}

resource "vcd_vapp" "empty-vm" {
  org         = "{{.Org}}"
  vdc         = "{{.Vdc}}"
  name        = "{{.TestName}}-empty-vm"
  description = "vApp for Empty VM description"
}

data "vcd_catalog" "{{.Catalog}}" {
	org  = "{{.Org}}"
	name = "{{.Catalog}}"
}

data "vcd_catalog_vapp_template" "{{.CatalogItem}}" {
	org         = "{{.Org}}"
	catalog_id = data.vcd_catalog.{{.Catalog}}.id
	name       = "{{.CatalogItem}}"
}

data "vcd_catalog_media" "{{.Media}}" {
	org     = "{{.Org}}"
	catalog = data.vcd_catalog.{{.Catalog}}.name
	name    = "{{.Media}}"
}

resource "vcd_vapp_vm" "template-vm" {
  org  = "{{.Org}}"
  vdc  = "{{.Vdc}}"

  vapp_template_id = data.vcd_catalog_vapp_template.{{.CatalogItem}}.id
  computer_name    = "comp-name"
  
  vapp_name   = vcd_vapp.template-vm.name
  name        = "{{.TestName}}-template-vapp-vm"

  cpu_hot_add_enabled            = true
  memory_hot_add_enabled         = true
  expose_hardware_virtualization = true

  storage_profile = data.vcd_storage_profile.nsxt-vdc.name

  metadata = {
    "vm1" = "VM Metadata"
    "vm2" = "VM Metadata2"
  }

  guest_properties = {
	"guest.hostname"       = "test-host"
	"guest.another.subkey" = "another-value"
  }
}

resource "vcd_vapp_vm" "empty-vm" {
  org  = "{{.Org}}"
  vdc  = "{{.Vdc}}"
  
  vapp_name     = vcd_vapp.empty-vm.name
  name          = "{{.TestName}}-empty-vapp-vm"
  computer_name = "comp-name"

  cpus   = 1
  memory = 1024

  cpu_hot_add_enabled            = true
  memory_hot_add_enabled         = true
  expose_hardware_virtualization = true

  os_type          = "rhel8_64Guest"
  hardware_version = "vmx-17"

  storage_profile = data.vcd_storage_profile.nsxt-vdc.name

  metadata = {
    "vm1" = "VM Metadata"
    "vm2" = "VM Metadata2"
  }

  guest_properties = {
	"guest.hostname"       = "test-host"
	"guest.another.subkey" = "another-value"
  }
}

resource "vcd_vm" "template-vm" {
  org  = "{{.Org}}"
  vdc  = "{{.Vdc}}"

  vapp_template_id = data.vcd_catalog_vapp_template.{{.CatalogItem}}.id
  computer_name    = "comp-name"
  
  name        = "{{.TestName}}-template-standalone-vm"

  cpu_hot_add_enabled            = true
  memory_hot_add_enabled         = true
  expose_hardware_virtualization = true

  storage_profile = data.vcd_storage_profile.nsxt-vdc.name

  metadata = {
    "vm1" = "VM Metadata"
    "vm2" = "VM Metadata2"
  }

  guest_properties = {
	"guest.hostname"       = "test-host"
	"guest.another.subkey" = "another-value"
  }
}

resource "vcd_vm" "empty-vm" {
  org  = "{{.Org}}"
  vdc  = "{{.Vdc}}"

  name          = "{{.TestName}}-empty-standalone-vm"
  computer_name = "comp-name"

  cpus   = 1
  memory = 1024

  cpu_hot_add_enabled            = true
  memory_hot_add_enabled         = true
  expose_hardware_virtualization = true

  os_type          = "rhel8_64Guest"
  hardware_version = "vmx-17"

  storage_profile = data.vcd_storage_profile.nsxt-vdc.name

  metadata = {
    "vm1" = "VM Metadata"
    "vm2" = "VM Metadata2"
  }

  guest_properties = {
	"guest.hostname"       = "test-host"
	"guest.another.subkey" = "another-value"
  }
}

# VM Copy from here
resource "vcd_vapp" "vm-copy-destination-template-vm" {
  org         = "{{.Org}}"
  vdc         = "{{.Vdc}}"
  name        = "{{.TestName}}-vm-copy-template-destination"
  description = "vApp destination for VM Copy"
  power_on    = true
}

resource "vcd_vapp" "vm-copy-destination-empty-vm" {
  org         = "{{.Org}}"
  vdc         = "{{.Vdc}}"
  name        = "{{.TestName}}-vm-copy-empty-destination"
  description = "vApp destination for VM Copy"
  power_on    = true
}

resource "vcd_vapp_vm" "template-vm-copy" {
  org  = "{{.Org}}"
  vdc  = "{{.Vdc}}"

  copy_from_vm_id = vcd_vapp_vm.template-vm.id
  computer_name    = "comp-name"
  description      = "{{.TestName}}-template-vapp-vm-copy"
  
  vapp_name   = vcd_vapp.vm-copy-destination-template-vm.name
  name        = "{{.TestName}}-template-vapp-vm-copy"

  cpu_hot_add_enabled            = true
  memory_hot_add_enabled         = true
  expose_hardware_virtualization = true

  storage_profile = data.vcd_storage_profile.nsxt-vdc.name

  metadata = {
    "vm1" = "VM Metadata"
    "vm2" = "VM Metadata2"
  }

  guest_properties = {
	"guest.hostname"       = "test-host"
	"guest.another.subkey" = "another-value"
  }
}

resource "vcd_vapp_vm" "empty-vm-copy" {
  org  = "{{.Org}}"
  vdc  = "{{.Vdc}}"
  
  vapp_name        = vcd_vapp.vm-copy-destination-empty-vm.name
  copy_from_vm_id  = vcd_vapp_vm.empty-vm.id
  name             = "{{.TestName}}-empty-vapp-vm-copy"
  computer_name    = "comp-name"

  cpus   = 1
  memory = 1024

  cpu_hot_add_enabled            = true
  memory_hot_add_enabled         = true
  expose_hardware_virtualization = true

  os_type          = "rhel8_64Guest"
  hardware_version = "vmx-17"

  storage_profile = data.vcd_storage_profile.nsxt-vdc.name

  metadata = {
    "vm1" = "VM Metadata"
    "vm2" = "VM Metadata2"
  }

  guest_properties = {
	"guest.hostname"       = "test-host"
	"guest.another.subkey" = "another-value"
  }
}

resource "vcd_vm" "template-vm-copy" {
  org  = "{{.Org}}"
  vdc  = "{{.Vdc}}"

  copy_from_vm_id  = vcd_vm.template-vm.id
  computer_name    = "comp-name"
  
  name        = "{{.TestName}}-template-standalone-vm-copy"

  cpu_hot_add_enabled            = true
  memory_hot_add_enabled         = true
  expose_hardware_virtualization = true

  storage_profile = data.vcd_storage_profile.nsxt-vdc.name

  metadata = {
    "vm1" = "VM Metadata"
    "vm2" = "VM Metadata2"
  }

  guest_properties = {
	"guest.hostname"       = "test-host"
	"guest.another.subkey" = "another-value"
  }
}

resource "vcd_vm" "empty-vm-copy" {
  org  = "{{.Org}}"
  vdc  = "{{.Vdc}}"

  copy_from_vm_id  = vcd_vm.empty-vm.id
  name             = "{{.TestName}}-empty-standalone-vm-copy"
  computer_name    = "comp-name"

  cpus   = 1
  memory = 1024

  cpu_hot_add_enabled            = true
  memory_hot_add_enabled         = true
  expose_hardware_virtualization = true

  os_type          = "rhel8_64Guest"
  hardware_version = "vmx-17"

  storage_profile = data.vcd_storage_profile.nsxt-vdc.name

  metadata = {
    "vm1" = "VM Metadata"
    "vm2" = "VM Metadata2"
  }

  guest_properties = {
	"guest.hostname"       = "test-host"
	"guest.another.subkey" = "another-value"
  }
}
`

// TestAccVcdVAppVm_4types_sizing_min checks that all types of VMs accept minimal sizing policy
// (without any CPU/Memory values)
func TestAccVcdVAppVm_4types_sizing_min(t *testing.T) {
	preTestChecks(t)
	skipIfNotSysAdmin(t)

	var params = StringMap{
		"TestName":       t.Name(),
		"Org":            testConfig.VCD.Org,
		"Vdc":            testConfig.Nsxt.Vdc,
		"Catalog":        testConfig.VCD.Catalog.NsxtBackedCatalogName,
		"CatalogItem":    testConfig.VCD.Catalog.CatalogItemWithMultiVms,
		"Media":          testConfig.Media.NsxtBackedMediaName,
		"StorageProfile": testConfig.VCD.NsxtProviderVdc.StorageProfile2,

		"ProviderVdc": testConfig.VCD.NsxtProviderVdc.Name,
		"NetworkPool": testConfig.VCD.NsxtProviderVdc.NetworkPool,

		"AllocationModel":           "Flex",
		"Allocated":                 "24000",
		"Reserved":                  "0",
		"Limit":                     "24000",
		"ProviderVdcStorageProfile": testConfig.VCD.ProviderVdc.StorageProfile,
		"FuncName":                  t.Name(),
		"MemoryGuaranteed":          "0.1",
		"CpuGuaranteed":             "0.1",
		// The parameters below are for Flex allocation model
		// Part of HCL is created dynamically and these parameters with values result in the Flex part of the template being filled:
		"equalsChar":                   "=",
		"FlexElasticKey":               "elasticity",
		"FlexElasticValue":             "false",
		"ElasticityValueForAssert":     "false",
		"FlexMemoryOverheadKey":        "include_vm_memory_overhead",
		"FlexMemoryOverheadValue":      "false",
		"MemoryOverheadValueForAssert": "false",

		"Tags": "vapp standaloneVm vm",
	}
	testParamsNotEmpty(t, params)

	params["SizingPolicyId"] = "vcd_vm_sizing_policy.minSize.id"
	configTextStep1 := templateFill(testAccVcdVAppVm_4types_sizing_policy_empty, params)

	if vcdShortTest {
		t.Skip(acceptanceTestsSkipped)
		return
	}
	debugPrintf("#[DEBUG] CONFIGURATION: %s\n", configTextStep1)
	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccCheckVcdNsxtVAppVmDestroy(t.Name()+"-template-vm"),
			testAccCheckVcdNsxtVAppVmDestroy(t.Name()+"-empty-vm"),
			testAccCheckVcdStandaloneVmDestroy(t.Name()+"-template-standalone-vm", testConfig.VCD.Org, testConfig.Nsxt.Vdc),
			testAccCheckVcdStandaloneVmDestroy(t.Name()+"-empty-standalone-vm", testConfig.VCD.Org, testConfig.Nsxt.Vdc),
		),
		Steps: []resource.TestStep{
			{
				Config: configTextStep1,
				Check: resource.ComposeAggregateTestCheckFunc(

					// vApp checks
					resource.TestCheckResourceAttr("vcd_vapp.template-vm", "name", t.Name()+"-template-vm"),
					resource.TestCheckResourceAttr("vcd_vapp.template-vm", "description", "vApp for Template VM description"),
					resource.TestCheckResourceAttr("vcd_vapp.empty-vm", "name", t.Name()+"-empty-vm"),
					resource.TestCheckResourceAttr("vcd_vapp.empty-vm", "description", "vApp for Empty VM description"),

					// Template vApp VM checks
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "vm_type", "vcd_vapp_vm"),
					resource.TestCheckResourceAttrPair("vcd_vapp_vm.template-vm", "sizing_policy_id", "vcd_vm_sizing_policy.minSize", "id"),

					// Empty vApp VM checks
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "vm_type", "vcd_vapp_vm"),
					resource.TestCheckResourceAttrPair("vcd_vapp_vm.template-vm", "sizing_policy_id", "vcd_vm_sizing_policy.minSize", "id"),

					// Standalone template VM checks
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "vm_type", "vcd_vm"),
					resource.TestCheckResourceAttrPair("vcd_vapp_vm.template-vm", "sizing_policy_id", "vcd_vm_sizing_policy.minSize", "id"),

					// Standalone empty VM checks
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "vm_type", "vcd_vm"),
					resource.TestCheckResourceAttrPair("vcd_vapp_vm.template-vm", "sizing_policy_id", "vcd_vm_sizing_policy.minSize", "id"),

					// VM copy checks
					// vApp checks
					resource.TestCheckResourceAttr("vcd_vapp.vm-copy-destination-template-vm", "name", t.Name()+"-vm-copy-template-destination"),
					resource.TestCheckResourceAttr("vcd_vapp.vm-copy-destination-template-vm", "description", "vApp destination for VM Copy"),
					resource.TestCheckResourceAttr("vcd_vapp.vm-copy-destination-empty-vm", "name", t.Name()+"-vm-copy-empty-destination"),
					resource.TestCheckResourceAttr("vcd_vapp.vm-copy-destination-empty-vm", "description", "vApp destination for VM Copy"),

					// Template vApp VM checks
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "vm_type", "vcd_vapp_vm"),
					resource.TestCheckResourceAttrPair("vcd_vapp_vm.template-vm-copy", "sizing_policy_id", "vcd_vm_sizing_policy.minSize", "id"),

					// Empty vApp VM checks
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "vm_type", "vcd_vapp_vm"),
					resource.TestCheckResourceAttrPair("vcd_vapp_vm.template-vm-copy", "sizing_policy_id", "vcd_vm_sizing_policy.minSize", "id"),

					// Standalone template VM checks
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "vm_type", "vcd_vm"),
					resource.TestCheckResourceAttrPair("vcd_vapp_vm.template-vm-copy", "sizing_policy_id", "vcd_vm_sizing_policy.minSize", "id"),

					// Standalone empty VM checks
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "vm_type", "vcd_vm"),
					resource.TestCheckResourceAttrPair("vcd_vapp_vm.template-vm-copy", "sizing_policy_id", "vcd_vm_sizing_policy.minSize", "id"),
				),
			},
		},
	})
	postTestChecks(t)
}

const testAccVcdVAppVm_sizing_policies = `
resource "vcd_vm_sizing_policy" "minSize" {
  name        = "min-size"
}

resource "vcd_vm_sizing_policy" "size_cpu" {
  name        = "size-cpu"

  cpu {
    shares                = "886"
    limit_in_mhz          = "2400"
    count                 = "3"
    speed_in_mhz          = "1500"
    cores_per_socket      = "1"
    reservation_guarantee = "{{.CpuGuaranteed}}"
  }
}

resource "vcd_vm_sizing_policy" "size_full" {
  name = "size-full"

  cpu {
    shares                = "886"
    limit_in_mhz          = "2700"
    count                 = "3"
    speed_in_mhz          = "1500"
    cores_per_socket      = "3"
    reservation_guarantee = "{{.CpuGuaranteed}}"
  }

  memory {
    shares                = "1580"
    size_in_mb            = "512"
    limit_in_mb           = "4800"
    reservation_guarantee = "{{.MemoryGuaranteed}}"
  }
}

resource "vcd_vm_sizing_policy" "size_memory" {
	name        = "size-memory"
  
	memory {
	  shares                = "1580"
	  size_in_mb            = "2048"
	  limit_in_mb           = "4800"
	  reservation_guarantee = "0.5"
	}
}

resource "vcd_org_vdc" "sizing-policy" {
  org = "{{.Org}}"

  name = "{{.TestName}}"

  allocation_model  = "{{.AllocationModel}}"
  network_pool_name = "{{.NetworkPool}}"
  provider_vdc_name = "{{.ProviderVdc}}"
  memory_guaranteed = 1
  cpu_guaranteed    = 1

  compute_capacity {
    cpu {
      limit     = "{{.Limit}}"
    }

    memory {
      limit     = "{{.Limit}}"
    }
  }

  storage_profile {
    name     = "{{.StorageProfile}}"
    enabled  = true
    limit    = 90240
    default  = true
  }

  enabled                    = true
  enable_thin_provisioning   = true
  enable_fast_provisioning   = true
  delete_force               = true
  delete_recursive           = true
  {{.FlexElasticKey}}                 {{.equalsChar}} {{.FlexElasticValue}}
  {{.FlexMemoryOverheadKey}} {{.equalsChar}} {{.FlexMemoryOverheadValue}}

  default_compute_policy_id   = vcd_vm_sizing_policy.size_full.id
  vm_sizing_policy_ids        = [vcd_vm_sizing_policy.minSize.id, vcd_vm_sizing_policy.size_cpu.id, vcd_vm_sizing_policy.size_full.id, vcd_vm_sizing_policy.size_memory.id]
}
`
const testAccVcdVAppVm_4types_sizing_policy_empty = testAccVcdVAppVm_sizing_policies + `
resource "vcd_vapp" "template-vm" {
  org         = "{{.Org}}"
  vdc         = (vcd_org_vdc.sizing-policy.id == "always-not-equal" ? null : vcd_org_vdc.sizing-policy.name)
  name        = "{{.TestName}}-template-vm"
  description = "vApp for Template VM description"
}

resource "vcd_vapp" "empty-vm" {
  org         = "{{.Org}}"
  vdc         = (vcd_org_vdc.sizing-policy.id == "always-not-equal" ? null : vcd_org_vdc.sizing-policy.name)
  name        = "{{.TestName}}-empty-vm"
  description = "vApp for Empty VM description"
}

data "vcd_catalog" "{{.Catalog}}" {
	org  = "{{.Org}}"
	name = "{{.Catalog}}"
}

data "vcd_catalog_vapp_template" "{{.CatalogItem}}" {
	org         = "{{.Org}}"
	catalog_id = data.vcd_catalog.{{.Catalog}}.id
	name       = "{{.CatalogItem}}"
}

data "vcd_catalog_media" "{{.Media}}" {
	org     = "{{.Org}}"
	catalog = data.vcd_catalog.{{.Catalog}}.name
	name    = "{{.Media}}"
}

resource "vcd_vapp_vm" "template-vm" {
  org = "{{.Org}}"
  vdc = (vcd_org_vdc.sizing-policy.id == "always-not-equal" ? null : vcd_org_vdc.sizing-policy.name)

  vapp_template_id = data.vcd_catalog_vapp_template.{{.CatalogItem}}.id
  
  vapp_name   = vcd_vapp.template-vm.name
  name        = "{{.TestName}}-template-vapp-vm"
  description = "{{.TestName}}-template-vapp-vm"

  prevent_update_power_off = true

  sizing_policy_id = {{.SizingPolicyId}}
}

resource "vcd_vapp_vm" "empty-vm" {
  org = "{{.Org}}"
  vdc = (vcd_org_vdc.sizing-policy.id == "always-not-equal" ? null : vcd_org_vdc.sizing-policy.name)
  
  vapp_name     = vcd_vapp.empty-vm.name
  name          = "{{.TestName}}-empty-vapp-vm"
  description   = "{{.TestName}}-empty-vapp-vm"
  computer_name = "vapp-vm"

  cpus   = 1
  memory = 1024

  os_type          = "sles10_64Guest"
  hardware_version = "vmx-14"
  boot_image_id    = data.vcd_catalog_media.{{.Media}}.id

  prevent_update_power_off = true

  sizing_policy_id = {{.SizingPolicyId}}
}

resource "vcd_vm" "template-vm" {
  org = "{{.Org}}"
  vdc = (vcd_org_vdc.sizing-policy.id == "always-not-equal" ? null : vcd_org_vdc.sizing-policy.name)

  vapp_template_id = data.vcd_catalog_vapp_template.{{.CatalogItem}}.id
  
  name        = "{{.TestName}}-template-standalone-vm"
  description = "{{.TestName}}-template-standalone-vm"

  prevent_update_power_off = true

  sizing_policy_id = {{.SizingPolicyId}}
}

resource "vcd_vm" "empty-vm" {
  org = "{{.Org}}"
  vdc = (vcd_org_vdc.sizing-policy.id == "always-not-equal" ? null : vcd_org_vdc.sizing-policy.name)

  name          = "{{.TestName}}-empty-standalone-vm"
  description   = "{{.TestName}}-standalone"
  computer_name = "standalone"

  cpus   = 1
  memory = 1024

  os_type          = "sles10_64Guest"
  hardware_version = "vmx-14"
  boot_image_id    = data.vcd_catalog_media.{{.Media}}.id

  prevent_update_power_off = true

  sizing_policy_id = {{.SizingPolicyId}}
}

# VM Copy from here
resource "vcd_vapp" "vm-copy-destination-template-vm" {
  org         = "{{.Org}}"
  vdc         = (vcd_org_vdc.sizing-policy.id == "always-not-equal" ? null : vcd_org_vdc.sizing-policy.name)
  name        = "{{.TestName}}-vm-copy-template-destination"
  description = "vApp destination for VM Copy"
  power_on    = false
}

resource "vcd_vapp" "vm-copy-destination-empty-vm" {
  org         = "{{.Org}}"
  vdc         = (vcd_org_vdc.sizing-policy.id == "always-not-equal" ? null : vcd_org_vdc.sizing-policy.name)
  name        = "{{.TestName}}-vm-copy-empty-destination"
  description = "vApp destination for VM Copy"
  power_on    = false
}

resource "vcd_vapp_vm" "template-vm-copy" {
  org = "{{.Org}}"
  vdc = (vcd_org_vdc.sizing-policy.id == "always-not-equal" ? null : vcd_org_vdc.sizing-policy.name)

  copy_from_vm_id = vcd_vapp_vm.template-vm.id
  vapp_name       = vcd_vapp.vm-copy-destination-template-vm.name
  name            = "{{.TestName}}-template-vapp-vm-copy"
  description     = "{{.TestName}}-template-vapp-vm"

  prevent_update_power_off = true

  sizing_policy_id = {{.SizingPolicyId}}
}

resource "vcd_vapp_vm" "empty-vm-copy" {
  org = "{{.Org}}"
  vdc = (vcd_org_vdc.sizing-policy.id == "always-not-equal" ? null : vcd_org_vdc.sizing-policy.name)
  
  copy_from_vm_id = vcd_vapp_vm.empty-vm.id
  vapp_name       = vcd_vapp.vm-copy-destination-empty-vm.name
  name            = "{{.TestName}}-empty-vapp-vm-copy"
  description     = "{{.TestName}}-empty-vapp-vm"
  computer_name   = "vapp-vm"

  cpus   = 1
  memory = 1024

  os_type          = "sles10_64Guest"
  hardware_version = "vmx-14"
  boot_image_id    = data.vcd_catalog_media.{{.Media}}.id

  prevent_update_power_off = true

  sizing_policy_id = {{.SizingPolicyId}}
}

resource "vcd_vm" "template-vm-copy" {
  org = "{{.Org}}"
  vdc = (vcd_org_vdc.sizing-policy.id == "always-not-equal" ? null : vcd_org_vdc.sizing-policy.name)

  copy_from_vm_id = vcd_vm.template-vm.id
  name            = "{{.TestName}}-template-standalone-vm-copy"
  description     = "{{.TestName}}-template-standalone-vm"

  prevent_update_power_off = true

  sizing_policy_id = {{.SizingPolicyId}}
}

resource "vcd_vm" "empty-vm-copy" {
  org = "{{.Org}}"
  vdc = (vcd_org_vdc.sizing-policy.id == "always-not-equal" ? null : vcd_org_vdc.sizing-policy.name)

  copy_from_vm_id = vcd_vm.empty-vm.id
  name            = "{{.TestName}}-empty-standalone-vm-copy"
  description     = "{{.TestName}}-standalone"
  computer_name   = "standalone"

  cpus   = 1
  memory = 1024

  os_type          = "sles10_64Guest"
  hardware_version = "vmx-14"
  boot_image_id    = data.vcd_catalog_media.{{.Media}}.id

  prevent_update_power_off = true

  sizing_policy_id = {{.SizingPolicyId}}
}
`

// TestAccVcdVAppVm_4types_sizing_max checks that all types of VMs can be created by inheriting
// sizing policy and no compute parameters specified in the VM resource itself
func TestAccVcdVAppVm_4types_sizing_max(t *testing.T) {
	preTestChecks(t)
	skipIfNotSysAdmin(t)

	var params = StringMap{
		"TestName":       t.Name(),
		"Org":            testConfig.VCD.Org,
		"Vdc":            testConfig.Nsxt.Vdc,
		"Catalog":        testConfig.VCD.Catalog.NsxtBackedCatalogName,
		"CatalogItem":    testConfig.VCD.Catalog.CatalogItemWithMultiVms,
		"Media":          testConfig.Media.NsxtBackedMediaName,
		"StorageProfile": testConfig.VCD.NsxtProviderVdc.StorageProfile2,

		"ProviderVdc": testConfig.VCD.NsxtProviderVdc.Name,
		"NetworkPool": testConfig.VCD.NsxtProviderVdc.NetworkPool,

		"AllocationModel":           "Flex",
		"Allocated":                 "40000",
		"Reserved":                  "40000",
		"Limit":                     "40000",
		"ProviderVdcStorageProfile": testConfig.VCD.ProviderVdc.StorageProfile,
		"FuncName":                  t.Name(),
		"MemoryGuaranteed":          "0.2",
		"CpuGuaranteed":             "0.1",
		// The parameters below are for Flex allocation model
		// Part of HCL is created dynamically and these parameters with values result in the Flex part of the template being filled:
		"equalsChar":                   "=",
		"FlexElasticKey":               "elasticity",
		"FlexElasticValue":             "false",
		"ElasticityValueForAssert":     "false",
		"FlexMemoryOverheadKey":        "include_vm_memory_overhead",
		"FlexMemoryOverheadValue":      "false",
		"MemoryOverheadValueForAssert": "false",

		"Tags": "vapp standaloneVm vm",
	}
	testParamsNotEmpty(t, params)

	params["SizingPolicyId"] = "vcd_vm_sizing_policy.size_full.id"
	configTextStep1 := templateFill(testAccVcdVAppVm_4types_sizing_policy_max, params)

	if vcdShortTest {
		t.Skip(acceptanceTestsSkipped)
		return
	}
	debugPrintf("#[DEBUG] CONFIGURATION: %s\n", configTextStep1)
	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccCheckVcdNsxtVAppVmDestroy(t.Name()+"-template-vm"),
			testAccCheckVcdNsxtVAppVmDestroy(t.Name()+"-empty-vm"),
			testAccCheckVcdStandaloneVmDestroy(t.Name()+"-template-standalone-vm", testConfig.VCD.Org, testConfig.Nsxt.Vdc),
			testAccCheckVcdStandaloneVmDestroy(t.Name()+"-empty-standalone-vm", testConfig.VCD.Org, testConfig.Nsxt.Vdc),
		),
		Steps: []resource.TestStep{
			{
				Config: configTextStep1,
				Check: resource.ComposeAggregateTestCheckFunc(

					// vApp checks
					resource.TestCheckResourceAttr("vcd_vapp.template-vm", "name", t.Name()+"-template-vm"),
					resource.TestCheckResourceAttr("vcd_vapp.template-vm", "description", "vApp for Template VM description"),
					resource.TestCheckResourceAttr("vcd_vapp.empty-vm", "name", t.Name()+"-empty-vm"),
					resource.TestCheckResourceAttr("vcd_vapp.empty-vm", "description", "vApp for Empty VM description"),

					// Template vApp VM checks
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "vm_type", "vcd_vapp_vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "cpus", "3"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "cpu_cores", "3"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "memory", "512"),
					resource.TestCheckResourceAttrPair("vcd_vapp_vm.template-vm", "sizing_policy_id", "vcd_vm_sizing_policy.size_full", "id"),

					// Empty vApp VM checks
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "vm_type", "vcd_vapp_vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "cpus", "3"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "cpu_cores", "3"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "memory", "512"),
					resource.TestCheckResourceAttrPair("vcd_vapp_vm.empty-vm", "sizing_policy_id", "vcd_vm_sizing_policy.size_full", "id"),

					// Standalone template VM checks
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "vm_type", "vcd_vm"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "cpus", "3"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "cpu_cores", "3"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "memory", "512"),
					resource.TestCheckResourceAttrPair("vcd_vm.template-vm", "sizing_policy_id", "vcd_vm_sizing_policy.size_full", "id"),

					// Standalone empty VM checks
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "vm_type", "vcd_vm"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "cpus", "3"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "cpu_cores", "3"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "memory", "512"),
					resource.TestCheckResourceAttrPair("vcd_vm.empty-vm", "sizing_policy_id", "vcd_vm_sizing_policy.size_full", "id"),

					// VM copy checks
					// vApp checks
					resource.TestCheckResourceAttr("vcd_vapp.vm-copy-destination-template-vm", "name", t.Name()+"-vm-copy-template-destination"),
					resource.TestCheckResourceAttr("vcd_vapp.vm-copy-destination-template-vm", "description", "vApp destination for VM Copy"),
					resource.TestCheckResourceAttr("vcd_vapp.vm-copy-destination-empty-vm", "name", t.Name()+"-vm-copy-empty-destination"),
					resource.TestCheckResourceAttr("vcd_vapp.vm-copy-destination-empty-vm", "description", "vApp destination for VM Copy"),

					// Template vApp VM checks
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "vm_type", "vcd_vapp_vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "cpus", "3"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "cpu_cores", "3"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "memory", "512"),
					resource.TestCheckResourceAttrPair("vcd_vapp_vm.template-vm-copy", "sizing_policy_id", "vcd_vm_sizing_policy.size_full", "id"),

					// Empty vApp VM checks
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "vm_type", "vcd_vapp_vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "cpus", "3"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "cpu_cores", "3"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "memory", "512"),
					resource.TestCheckResourceAttrPair("vcd_vapp_vm.template-vm-copy", "sizing_policy_id", "vcd_vm_sizing_policy.size_full", "id"),

					// Standalone template VM checks
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "vm_type", "vcd_vm"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "cpus", "3"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "cpu_cores", "3"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "memory", "512"),
					resource.TestCheckResourceAttrPair("vcd_vapp_vm.template-vm-copy", "sizing_policy_id", "vcd_vm_sizing_policy.size_full", "id"),

					// Standalone empty VM checks
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "vm_type", "vcd_vm"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "cpus", "3"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "cpu_cores", "3"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "memory", "512"),
					resource.TestCheckResourceAttrPair("vcd_vapp_vm.template-vm-copy", "sizing_policy_id", "vcd_vm_sizing_policy.size_full", "id"),
				),
			},
		},
	})
	postTestChecks(t)
}

const testAccVcdVAppVm_4types_sizing_policy_max = testAccVcdVAppVm_sizing_policies + `
resource "vcd_vapp" "template-vm" {
  org         = "{{.Org}}"
  vdc         = (vcd_org_vdc.sizing-policy.id == "always-not-equal" ? null : vcd_org_vdc.sizing-policy.name)
  name        = "{{.TestName}}-template-vm"
  description = "vApp for Template VM description"
}

resource "vcd_vapp" "empty-vm" {
  org         = "{{.Org}}"
  vdc         = (vcd_org_vdc.sizing-policy.id == "always-not-equal" ? null : vcd_org_vdc.sizing-policy.name)
  name        = "{{.TestName}}-empty-vm"
  description = "vApp for Empty VM description"
}

data "vcd_catalog" "{{.Catalog}}" {
	org  = "{{.Org}}"
	name = "{{.Catalog}}"
}

data "vcd_catalog_vapp_template" "{{.CatalogItem}}" {
	org         = "{{.Org}}"
	catalog_id = data.vcd_catalog.{{.Catalog}}.id
	name       = "{{.CatalogItem}}"
}

data "vcd_catalog_media" "{{.Media}}" {
	org     = "{{.Org}}"
	catalog = data.vcd_catalog.{{.Catalog}}.name
	name    = "{{.Media}}"
}

resource "vcd_vapp_vm" "template-vm" {
  org = "{{.Org}}"
  vdc = (vcd_org_vdc.sizing-policy.id == "always-not-equal" ? null : vcd_org_vdc.sizing-policy.name)

  vapp_template_id = data.vcd_catalog_vapp_template.{{.CatalogItem}}.id
  
  vapp_name   = vcd_vapp.template-vm.name
  name        = "{{.TestName}}-template-vapp-vm"
  description = "{{.TestName}}-template-vapp-vm"

  prevent_update_power_off = true

  sizing_policy_id = {{.SizingPolicyId}}
}

resource "vcd_vapp_vm" "empty-vm" {
  org = "{{.Org}}"
  vdc = (vcd_org_vdc.sizing-policy.id == "always-not-equal" ? null : vcd_org_vdc.sizing-policy.name)
  
  vapp_name     = vcd_vapp.empty-vm.name
  name          = "{{.TestName}}-empty-vapp-vm"
  description   = "{{.TestName}}-empty-vapp-vm"
  computer_name = "vapp-vm"

  os_type          = "sles10_64Guest"
  hardware_version = "vmx-14"
  boot_image_id    = data.vcd_catalog_media.{{.Media}}.id

  prevent_update_power_off = true

  sizing_policy_id = {{.SizingPolicyId}}
}

resource "vcd_vm" "template-vm" {
  org = "{{.Org}}"
  vdc = (vcd_org_vdc.sizing-policy.id == "always-not-equal" ? null : vcd_org_vdc.sizing-policy.name)

  vapp_template_id = data.vcd_catalog_vapp_template.{{.CatalogItem}}.id
  
  name        = "{{.TestName}}-template-standalone-vm"
  description = "{{.TestName}}-template-standalone-vm"

  prevent_update_power_off = true

  sizing_policy_id = {{.SizingPolicyId}}
}

resource "vcd_vm" "empty-vm" {
  org = "{{.Org}}"
  vdc = (vcd_org_vdc.sizing-policy.id == "always-not-equal" ? null : vcd_org_vdc.sizing-policy.name)

  name          = "{{.TestName}}-empty-standalone-vm"
  description   = "{{.TestName}}-standalone"
  computer_name = "standalone"

  os_type          = "sles10_64Guest"
  hardware_version = "vmx-14"
  boot_image_id    = data.vcd_catalog_media.{{.Media}}.id

  prevent_update_power_off = true

  sizing_policy_id = {{.SizingPolicyId}}
}

# VM Copy from here
resource "vcd_vapp" "vm-copy-destination-template-vm" {
  org         = "{{.Org}}"
  vdc         = (vcd_org_vdc.sizing-policy.id == "always-not-equal" ? null : vcd_org_vdc.sizing-policy.name)
  name        = "{{.TestName}}-vm-copy-template-destination"
  description = "vApp destination for VM Copy"
  power_on    = false
}

resource "vcd_vapp" "vm-copy-destination-empty-vm" {
  org         = "{{.Org}}"
  vdc         = (vcd_org_vdc.sizing-policy.id == "always-not-equal" ? null : vcd_org_vdc.sizing-policy.name)
  name        = "{{.TestName}}-vm-copy-empty-destination"
  description = "vApp destination for VM Copy"
  power_on    = false
}

resource "vcd_vapp_vm" "template-vm-copy" {
  org = "{{.Org}}"
  vdc = (vcd_org_vdc.sizing-policy.id == "always-not-equal" ? null : vcd_org_vdc.sizing-policy.name)

  copy_from_vm_id = vcd_vapp_vm.template-vm.id
  vapp_name       = vcd_vapp.vm-copy-destination-template-vm.name
  name            = "{{.TestName}}-template-vapp-vm-copy"
  description     = "{{.TestName}}-template-vapp-vm"

  prevent_update_power_off = true

  sizing_policy_id = {{.SizingPolicyId}}
}

resource "vcd_vapp_vm" "empty-vm-copy" {
  org = "{{.Org}}"
  vdc = (vcd_org_vdc.sizing-policy.id == "always-not-equal" ? null : vcd_org_vdc.sizing-policy.name)
  
  copy_from_vm_id = vcd_vapp_vm.empty-vm.id
  vapp_name       = vcd_vapp.vm-copy-destination-empty-vm.name
  name            = "{{.TestName}}-empty-vapp-vm-copy"
  description     = "{{.TestName}}-empty-vapp-vm"
  computer_name   = "vapp-vm"

  os_type          = "sles10_64Guest"
  hardware_version = "vmx-14"
  boot_image_id    = data.vcd_catalog_media.{{.Media}}.id

  prevent_update_power_off = true

  sizing_policy_id = {{.SizingPolicyId}}
}

resource "vcd_vm" "template-vm-copy" {
  org = "{{.Org}}"
  vdc = (vcd_org_vdc.sizing-policy.id == "always-not-equal" ? null : vcd_org_vdc.sizing-policy.name)

  copy_from_vm_id = vcd_vm.template-vm.id
  name            = "{{.TestName}}-template-standalone-vm-copy"
  description     = "{{.TestName}}-template-standalone-vm"

  prevent_update_power_off = true

  sizing_policy_id = {{.SizingPolicyId}}
}

resource "vcd_vm" "empty-vm-copy" {
  org = "{{.Org}}"
  vdc = (vcd_org_vdc.sizing-policy.id == "always-not-equal" ? null : vcd_org_vdc.sizing-policy.name)

  copy_from_vm_id = vcd_vm.empty-vm.id
  name            = "{{.TestName}}-empty-standalone-vm-copy"
  description     = "{{.TestName}}-standalone"
  computer_name   = "standalone"

  os_type          = "sles10_64Guest"
  hardware_version = "vmx-14"
  boot_image_id    = data.vcd_catalog_media.{{.Media}}.id

  prevent_update_power_off = true

  sizing_policy_id = {{.SizingPolicyId}}
}
`

// TestAccVcdVAppVm_4types_sizing_cpu_only checks that assigning sizing policy with CPU only setting
// works but memory is still required for empty VMs
func TestAccVcdVAppVm_4types_sizing_cpu_only(t *testing.T) {
	preTestChecks(t)
	skipIfNotSysAdmin(t)

	var params = StringMap{
		"TestName":       t.Name(),
		"Org":            testConfig.VCD.Org,
		"Vdc":            testConfig.Nsxt.Vdc,
		"Catalog":        testConfig.VCD.Catalog.NsxtBackedCatalogName,
		"CatalogItem":    testConfig.VCD.Catalog.CatalogItemWithMultiVms,
		"Media":          testConfig.Media.NsxtBackedMediaName,
		"StorageProfile": testConfig.VCD.NsxtProviderVdc.StorageProfile2,

		"ProviderVdc": testConfig.VCD.NsxtProviderVdc.Name,
		"NetworkPool": testConfig.VCD.NsxtProviderVdc.NetworkPool,

		"AllocationModel":           "Flex",
		"Allocated":                 "30000",
		"Reserved":                  "30000",
		"Limit":                     "32000",
		"ProviderVdcStorageProfile": testConfig.VCD.ProviderVdc.StorageProfile,
		"FuncName":                  t.Name(),
		"MemoryGuaranteed":          "0.3",
		"CpuGuaranteed":             "0.1",
		// The parameters below are for Flex allocation model
		// Part of HCL is created dynamically and these parameters with values result in the Flex part of the template being filled:
		"equalsChar":                   "=",
		"FlexElasticKey":               "elasticity",
		"FlexElasticValue":             "false",
		"ElasticityValueForAssert":     "false",
		"FlexMemoryOverheadKey":        "include_vm_memory_overhead",
		"FlexMemoryOverheadValue":      "false",
		"MemoryOverheadValueForAssert": "false",

		"Tags": "vapp standaloneVm vm",
	}
	testParamsNotEmpty(t, params)

	params["SizingPolicyId"] = "vcd_vm_sizing_policy.size_cpu.id"
	configTextStep1 := templateFill(testAccVcdVAppVm_4types_sizing_policy_cpu_only, params)

	if vcdShortTest {
		t.Skip(acceptanceTestsSkipped)
		return
	}
	debugPrintf("#[DEBUG] CONFIGURATION: %s\n", configTextStep1)
	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccCheckVcdNsxtVAppVmDestroy(t.Name()+"-template-vm"),
			testAccCheckVcdNsxtVAppVmDestroy(t.Name()+"-empty-vm"),
			testAccCheckVcdStandaloneVmDestroy(t.Name()+"-template-standalone-vm", testConfig.VCD.Org, testConfig.Nsxt.Vdc),
			testAccCheckVcdStandaloneVmDestroy(t.Name()+"-empty-standalone-vm", testConfig.VCD.Org, testConfig.Nsxt.Vdc),
		),
		Steps: []resource.TestStep{
			{
				Config: configTextStep1,
				Check: resource.ComposeAggregateTestCheckFunc(

					// vApp checks
					resource.TestCheckResourceAttr("vcd_vapp.template-vm", "name", t.Name()+"-template-vm"),
					resource.TestCheckResourceAttr("vcd_vapp.template-vm", "description", "vApp for Template VM description"),
					resource.TestCheckResourceAttr("vcd_vapp.empty-vm", "name", t.Name()+"-empty-vm"),
					resource.TestCheckResourceAttr("vcd_vapp.empty-vm", "description", "vApp for Empty VM description"),

					// Template vApp VM checks
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "vm_type", "vcd_vapp_vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "cpus", "3"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "cpu_cores", "1"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "memory", "1024"),
					resource.TestCheckResourceAttrPair("vcd_vapp_vm.template-vm", "sizing_policy_id", "vcd_vm_sizing_policy.size_cpu", "id"),

					// Empty vApp VM checks
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "vm_type", "vcd_vapp_vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "cpus", "3"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "cpu_cores", "1"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "memory", "1024"),
					resource.TestCheckResourceAttrPair("vcd_vapp_vm.empty-vm", "sizing_policy_id", "vcd_vm_sizing_policy.size_cpu", "id"),

					// Standalone template VM checks
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "vm_type", "vcd_vm"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "cpus", "3"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "cpu_cores", "1"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "memory", "1024"),
					resource.TestCheckResourceAttrPair("vcd_vm.template-vm", "sizing_policy_id", "vcd_vm_sizing_policy.size_cpu", "id"),

					// Standalone empty VM checks
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "vm_type", "vcd_vm"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "cpus", "3"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "cpu_cores", "1"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "memory", "1024"),
					resource.TestCheckResourceAttrPair("vcd_vm.empty-vm", "sizing_policy_id", "vcd_vm_sizing_policy.size_cpu", "id"),

					// VM copy checks
					// vApp checks
					resource.TestCheckResourceAttr("vcd_vapp.vm-copy-destination-template-vm", "name", t.Name()+"-vm-copy-template-destination"),
					resource.TestCheckResourceAttr("vcd_vapp.vm-copy-destination-template-vm", "description", "vApp destination for VM Copy"),
					resource.TestCheckResourceAttr("vcd_vapp.vm-copy-destination-empty-vm", "name", t.Name()+"-vm-copy-empty-destination"),
					resource.TestCheckResourceAttr("vcd_vapp.vm-copy-destination-empty-vm", "description", "vApp destination for VM Copy"),

					// Template vApp VM checks
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "vm_type", "vcd_vapp_vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "cpus", "3"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "cpu_cores", "1"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "memory", "1024"),
					resource.TestCheckResourceAttrPair("vcd_vapp_vm.template-vm-copy", "sizing_policy_id", "vcd_vm_sizing_policy.size_cpu", "id"),

					// Empty vApp VM checks
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "vm_type", "vcd_vapp_vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "cpus", "3"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "cpu_cores", "1"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "memory", "1024"),
					resource.TestCheckResourceAttrPair("vcd_vapp_vm.template-vm-copy", "sizing_policy_id", "vcd_vm_sizing_policy.size_cpu", "id"),

					// Standalone template VM checks
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "vm_type", "vcd_vm"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "cpus", "3"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "cpu_cores", "1"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "memory", "1024"),
					resource.TestCheckResourceAttrPair("vcd_vapp_vm.template-vm-copy", "sizing_policy_id", "vcd_vm_sizing_policy.size_cpu", "id"),

					// Standalone empty VM checks
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "vm_type", "vcd_vm"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "cpus", "3"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "cpu_cores", "1"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "memory", "1024"),
					resource.TestCheckResourceAttrPair("vcd_vapp_vm.template-vm-copy", "sizing_policy_id", "vcd_vm_sizing_policy.size_cpu", "id"),
				),
			},
		},
	})
	postTestChecks(t)
}

const testAccVcdVAppVm_4types_sizing_policy_cpu_only = testAccVcdVAppVm_sizing_policies + `
resource "vcd_vapp" "template-vm" {
  org         = "{{.Org}}"
  vdc         = (vcd_org_vdc.sizing-policy.id == "always-not-equal" ? null : vcd_org_vdc.sizing-policy.name)
  name        = "{{.TestName}}-template-vm"
  description = "vApp for Template VM description"
}

resource "vcd_vapp" "empty-vm" {
  org         = "{{.Org}}"
  vdc         = (vcd_org_vdc.sizing-policy.id == "always-not-equal" ? null : vcd_org_vdc.sizing-policy.name)
  name        = "{{.TestName}}-empty-vm"
  description = "vApp for Empty VM description"
}

data "vcd_catalog" "{{.Catalog}}" {
	org  = "{{.Org}}"
	name = "{{.Catalog}}"
}

data "vcd_catalog_vapp_template" "{{.CatalogItem}}" {
	org         = "{{.Org}}"
	catalog_id = data.vcd_catalog.{{.Catalog}}.id
	name       = "{{.CatalogItem}}"
}

data "vcd_catalog_media" "{{.Media}}" {
	org     = "{{.Org}}"
	catalog = data.vcd_catalog.{{.Catalog}}.name
	name    = "{{.Media}}"
}

resource "vcd_vapp_vm" "template-vm" {
  org = "{{.Org}}"
  vdc = (vcd_org_vdc.sizing-policy.id == "always-not-equal" ? null : vcd_org_vdc.sizing-policy.name)

  vapp_template_id = data.vcd_catalog_vapp_template.{{.CatalogItem}}.id
  
  vapp_name   = vcd_vapp.template-vm.name
  name        = "{{.TestName}}-template-vapp-vm"
  description = "{{.TestName}}-template-vapp-vm"

  memory = 1024

  prevent_update_power_off = true

  sizing_policy_id = {{.SizingPolicyId}}
}

resource "vcd_vapp_vm" "empty-vm" {
  org = "{{.Org}}"
  vdc = (vcd_org_vdc.sizing-policy.id == "always-not-equal" ? null : vcd_org_vdc.sizing-policy.name)
  
  vapp_name     = vcd_vapp.empty-vm.name
  name          = "{{.TestName}}-empty-vapp-vm"
  description   = "{{.TestName}}-empty-vapp-vm"
  computer_name = "vapp-vm"

  memory = 1024

  os_type          = "sles10_64Guest"
  hardware_version = "vmx-14"
  boot_image_id    = data.vcd_catalog_media.{{.Media}}.id

  prevent_update_power_off = true

  sizing_policy_id = {{.SizingPolicyId}}
}

resource "vcd_vm" "template-vm" {
  org = "{{.Org}}"
  vdc = (vcd_org_vdc.sizing-policy.id == "always-not-equal" ? null : vcd_org_vdc.sizing-policy.name)

  vapp_template_id = data.vcd_catalog_vapp_template.{{.CatalogItem}}.id
  
  name        = "{{.TestName}}-template-standalone-vm"
  description = "{{.TestName}}-template-standalone-vm"

  memory = 1024

  prevent_update_power_off = true

  sizing_policy_id = {{.SizingPolicyId}}
}

resource "vcd_vm" "empty-vm" {
  org = "{{.Org}}"
  vdc = (vcd_org_vdc.sizing-policy.id == "always-not-equal" ? null : vcd_org_vdc.sizing-policy.name)

  name          = "{{.TestName}}-empty-standalone-vm"
  description   = "{{.TestName}}-standalone"
  computer_name = "standalone"

  memory = 1024

  os_type          = "sles10_64Guest"
  hardware_version = "vmx-14"
  boot_image_id    = data.vcd_catalog_media.{{.Media}}.id

  prevent_update_power_off = true

  sizing_policy_id = {{.SizingPolicyId}}
}


# VM Copy from here
resource "vcd_vapp" "vm-copy-destination-template-vm" {
  org         = "{{.Org}}"
  vdc         = (vcd_org_vdc.sizing-policy.id == "always-not-equal" ? null : vcd_org_vdc.sizing-policy.name)
  name        = "{{.TestName}}-vm-copy-template-destination"
  description = "vApp destination for VM Copy"
  power_on    = false
}

resource "vcd_vapp" "vm-copy-destination-empty-vm" {
  org         = "{{.Org}}"
  vdc         = (vcd_org_vdc.sizing-policy.id == "always-not-equal" ? null : vcd_org_vdc.sizing-policy.name)
  name        = "{{.TestName}}-vm-copy-empty-destination"
  description = "vApp destination for VM Copy"
  power_on    = false
}

resource "vcd_vapp_vm" "template-vm-copy" {
  org = "{{.Org}}"
  vdc = (vcd_org_vdc.sizing-policy.id == "always-not-equal" ? null : vcd_org_vdc.sizing-policy.name)

  copy_from_vm_id = vcd_vapp_vm.template-vm.id
  vapp_name       = vcd_vapp.vm-copy-destination-template-vm.name
  name            = "{{.TestName}}-template-vapp-vm-copy"
  description     = "{{.TestName}}-template-vapp-vm"

  memory = 1024

  prevent_update_power_off = true

  sizing_policy_id = {{.SizingPolicyId}}
}

resource "vcd_vapp_vm" "empty-vm-copy" {
  org = "{{.Org}}"
  vdc = (vcd_org_vdc.sizing-policy.id == "always-not-equal" ? null : vcd_org_vdc.sizing-policy.name)
  
  copy_from_vm_id = vcd_vapp_vm.empty-vm.id
  vapp_name       = vcd_vapp.vm-copy-destination-empty-vm.name
  name            = "{{.TestName}}-empty-vapp-vm-copy"
  description     = "{{.TestName}}-empty-vapp-vm"
  computer_name   = "vapp-vm"

  memory = 1024

  os_type          = "sles10_64Guest"
  hardware_version = "vmx-14"
  boot_image_id    = data.vcd_catalog_media.{{.Media}}.id

  prevent_update_power_off = true

  sizing_policy_id = {{.SizingPolicyId}}
}

resource "vcd_vm" "template-vm-copy" {
  org = "{{.Org}}"
  vdc = (vcd_org_vdc.sizing-policy.id == "always-not-equal" ? null : vcd_org_vdc.sizing-policy.name)

  copy_from_vm_id = vcd_vm.template-vm.id
  name            = "{{.TestName}}-template-standalone-vm-copy"
  description     = "{{.TestName}}-template-standalone-vm"

  memory = 1024

  prevent_update_power_off = true

  sizing_policy_id = {{.SizingPolicyId}}
}

resource "vcd_vm" "empty-vm-copy" {
  org = "{{.Org}}"
  vdc = (vcd_org_vdc.sizing-policy.id == "always-not-equal" ? null : vcd_org_vdc.sizing-policy.name)

  copy_from_vm_id = vcd_vm.empty-vm.id
  name            = "{{.TestName}}-empty-standalone-vm-copy"
  description     = "{{.TestName}}-standalone"
  computer_name   = "standalone"

  memory = 1024

  os_type          = "sles10_64Guest"
  hardware_version = "vmx-14"
  boot_image_id    = data.vcd_catalog_media.{{.Media}}.id

  prevent_update_power_off = true

  sizing_policy_id = {{.SizingPolicyId}}
}
`

func TestAccVcdVAppVm_4typesAdvancedComputeSettings(t *testing.T) {
	preTestChecks(t)

	var params = StringMap{
		"TestName":    t.Name(),
		"Org":         testConfig.VCD.Org,
		"Vdc":         testConfig.Nsxt.Vdc,
		"Catalog":     testConfig.VCD.Catalog.NsxtBackedCatalogName,
		"CatalogItem": testConfig.VCD.Catalog.CatalogItemWithMultiVms,
		"Media":       testConfig.Media.NsxtBackedMediaName,

		"Tags": "vapp vm",
	}
	testParamsNotEmpty(t, params)

	configTextStep1 := templateFill(testAccVcdVAppVm_4types_advancedComputeSettings, params)

	if vcdShortTest {
		t.Skip(acceptanceTestsSkipped)
		return
	}
	debugPrintf("#[DEBUG] CONFIGURATION: %s\n", configTextStep1)
	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccCheckVcdNsxtVAppVmDestroy(t.Name()+"-template-vm"),
			testAccCheckVcdNsxtVAppVmDestroy(t.Name()+"-empty-vm"),
			testAccCheckVcdStandaloneVmDestroy(t.Name()+"-template-standalone-vm", testConfig.VCD.Org, testConfig.Nsxt.Vdc),
			testAccCheckVcdStandaloneVmDestroy(t.Name()+"-empty-standalone-vm", testConfig.VCD.Org, testConfig.Nsxt.Vdc),
		),
		Steps: []resource.TestStep{
			{
				Config: configTextStep1,
				Check: resource.ComposeAggregateTestCheckFunc(

					// vApp checks
					resource.TestCheckResourceAttr("vcd_vapp.template-vm", "name", t.Name()+"-template-vm"),
					resource.TestCheckResourceAttr("vcd_vapp.template-vm", "description", "vApp for Template VM description"),

					resource.TestCheckResourceAttr("vcd_vapp.empty-vm", "name", t.Name()+"-empty-vm"),
					resource.TestCheckResourceAttr("vcd_vapp.empty-vm", "description", "vApp for Empty VM description"),

					// Template vApp VM checks
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "vm_type", "vcd_vapp_vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "name", t.Name()+"-template-vapp-vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "description", ""),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "cpu_hot_add_enabled", "false"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "memory_hot_add_enabled", "false"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "expose_hardware_virtualization", "false"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "memory_priority", "CUSTOM"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "memory_shares", "480"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "memory_reservation", "8"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "memory_limit", "48"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "cpu_priority", "CUSTOM"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "cpu_shares", "512"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "cpu_reservation", "200"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "cpu_limit", "1000"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "cpus", "1"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "memory", "1024"),

					// Empty vApp VM checks
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "vm_type", "vcd_vapp_vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "name", t.Name()+"-empty-vapp-vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "description", ""),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "computer_name", "vapp-vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "cpus", "1"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "memory", "1024"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "os_type", "sles10_64Guest"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "hardware_version", "vmx-14"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "cpu_hot_add_enabled", "false"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "memory_hot_add_enabled", "false"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "expose_hardware_virtualization", "false"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "memory_priority", "CUSTOM"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "memory_shares", "480"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "memory_reservation", "8"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "memory_limit", "48"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "cpu_priority", "CUSTOM"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "cpu_shares", "512"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "cpu_reservation", "200"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "cpu_limit", "1000"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "cpus", "1"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "memory", "1024"),

					// Standalone template VM checks
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "vm_type", "vcd_vm"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "name", t.Name()+"-template-standalone-vm"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "description", ""),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "cpu_hot_add_enabled", "false"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "memory_hot_add_enabled", "false"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "expose_hardware_virtualization", "false"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "memory_priority", "CUSTOM"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "memory_shares", "480"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "memory_reservation", "8"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "memory_limit", "48"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "cpu_priority", "CUSTOM"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "cpu_shares", "512"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "cpu_reservation", "200"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "cpu_limit", "1000"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "cpus", "1"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "memory", "1024"),

					// Standalone empty VM checks
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "vm_type", "vcd_vm"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "name", t.Name()+"-empty-standalone-vm"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "description", ""),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "cpus", "1"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "memory", "1024"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "os_type", "sles10_64Guest"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "hardware_version", "vmx-14"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "cpu_hot_add_enabled", "false"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "memory_hot_add_enabled", "false"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "expose_hardware_virtualization", "false"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "memory_priority", "CUSTOM"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "memory_shares", "480"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "memory_reservation", "8"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "memory_limit", "48"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "cpu_priority", "CUSTOM"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "cpu_shares", "512"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "cpu_reservation", "200"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "cpu_limit", "1000"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "cpus", "1"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "memory", "1024"),

					// VM copy checks
					// vApp checks
					resource.TestCheckResourceAttr("vcd_vapp.vm-copy-destination-template-vm", "name", t.Name()+"-vm-copy-template-destination"),
					resource.TestCheckResourceAttr("vcd_vapp.vm-copy-destination-template-vm", "description", "vApp destination for VM Copy"),
					resource.TestCheckResourceAttr("vcd_vapp.vm-copy-destination-empty-vm", "name", t.Name()+"-vm-copy-empty-destination"),
					resource.TestCheckResourceAttr("vcd_vapp.vm-copy-destination-empty-vm", "description", "vApp destination for VM Copy"),

					// Template vApp VM checks
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "vm_type", "vcd_vapp_vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "name", t.Name()+"-template-vapp-vm-copy"),
					resource.TestCheckResourceAttrSet("vcd_vapp_vm.template-vm-copy", "description"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "cpu_hot_add_enabled", "false"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "memory_hot_add_enabled", "false"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "expose_hardware_virtualization", "false"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "memory_priority", "CUSTOM"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "memory_shares", "480"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "memory_reservation", "8"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "memory_limit", "48"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "cpu_priority", "CUSTOM"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "cpu_shares", "512"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "cpu_reservation", "200"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "cpu_limit", "1000"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "cpus", "1"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "memory", "1024"),

					// Empty vApp VM checks
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "vm_type", "vcd_vapp_vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "name", t.Name()+"-empty-vapp-vm-copy"),
					resource.TestCheckResourceAttrSet("vcd_vapp_vm.empty-vm-copy", "description"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "computer_name", "vapp-vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "cpus", "1"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "memory", "1024"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "os_type", "sles10_64Guest"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "hardware_version", "vmx-14"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "cpu_hot_add_enabled", "false"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "memory_hot_add_enabled", "false"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "expose_hardware_virtualization", "false"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "memory_priority", "CUSTOM"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "memory_shares", "480"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "memory_reservation", "8"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "memory_limit", "48"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "cpu_priority", "CUSTOM"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "cpu_shares", "512"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "cpu_reservation", "200"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "cpu_limit", "1000"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "cpus", "1"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "memory", "1024"),

					// Standalone template VM checks
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "vm_type", "vcd_vm"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "name", t.Name()+"-template-standalone-vm-copy"),
					resource.TestCheckResourceAttrSet("vcd_vm.template-vm-copy", "description"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "cpu_hot_add_enabled", "false"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "memory_hot_add_enabled", "false"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "expose_hardware_virtualization", "false"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "memory_priority", "CUSTOM"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "memory_shares", "480"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "memory_reservation", "8"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "memory_limit", "48"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "cpu_priority", "CUSTOM"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "cpu_shares", "512"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "cpu_reservation", "200"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "cpu_limit", "1000"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "cpus", "1"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "memory", "1024"),

					// Standalone empty VM checks
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "vm_type", "vcd_vm"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "name", t.Name()+"-empty-standalone-vm-copy"),
					resource.TestCheckResourceAttrSet("vcd_vm.empty-vm-copy", "description"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "cpus", "1"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "memory", "1024"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "os_type", "sles10_64Guest"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "hardware_version", "vmx-14"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "cpu_hot_add_enabled", "false"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "memory_hot_add_enabled", "false"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "expose_hardware_virtualization", "false"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "memory_priority", "CUSTOM"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "memory_shares", "480"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "memory_reservation", "8"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "memory_limit", "48"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "cpu_priority", "CUSTOM"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "cpu_shares", "512"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "cpu_reservation", "200"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "cpu_limit", "1000"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "cpus", "1"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "memory", "1024"),
				),
			},
		},
	})
	postTestChecks(t)
}

const testAccVcdVAppVm_4types_advancedComputeSettings = `
data "vcd_org_vdc" "nsxt" {
  org  = "{{.Org}}"
  name = "{{.Vdc}}"
}

resource "vcd_vapp" "template-vm" {
  org         = "{{.Org}}"
  vdc         = "{{.Vdc}}"
  name        = "{{.TestName}}-template-vm"
  description = "vApp for Template VM description"
}

resource "vcd_vapp" "empty-vm" {
  org         = "{{.Org}}"
  vdc         = "{{.Vdc}}"
  name        = "{{.TestName}}-empty-vm"
  description = "vApp for Empty VM description"
}

data "vcd_catalog" "{{.Catalog}}" {
	org  = "{{.Org}}"
	name = "{{.Catalog}}"
}

data "vcd_catalog_vapp_template" "{{.CatalogItem}}" {
	org         = "{{.Org}}"
	catalog_id = data.vcd_catalog.{{.Catalog}}.id
	name       = "{{.CatalogItem}}"
}

data "vcd_catalog_media" "{{.Media}}" {
	org     = "{{.Org}}"
	catalog = data.vcd_catalog.{{.Catalog}}.name
	name    = "{{.Media}}"
}

resource "vcd_vapp_vm" "template-vm" {
  org  = "{{.Org}}"
  vdc  = "{{.Vdc}}"

  vapp_template_id = data.vcd_catalog_vapp_template.{{.CatalogItem}}.id
  
  vapp_name   = vcd_vapp.template-vm.name
  name        = "{{.TestName}}-template-vapp-vm"

  cpus   = 1
  memory = 1024

  memory_priority    = "CUSTOM"
  memory_shares      = "480"
  memory_reservation = "8"
  memory_limit       = "48"

  cpu_priority    = "CUSTOM"
  cpu_shares      = "512"
  cpu_reservation = "200"
  cpu_limit       = "1000"
}

resource "vcd_vapp_vm" "empty-vm" {
  org  = "{{.Org}}"
  vdc  = "{{.Vdc}}"
  
  vapp_name     = vcd_vapp.empty-vm.name
  name          = "{{.TestName}}-empty-vapp-vm"
  computer_name = "vapp-vm"

  os_type          = "sles10_64Guest"
  hardware_version = "vmx-14"
  boot_image_id    = data.vcd_catalog_media.{{.Media}}.id

  cpus   = 1
  memory = 1024

  memory_priority    = "CUSTOM"
  memory_shares      = "480"
  memory_reservation = "8"
  memory_limit       = "48"

  cpu_priority    = "CUSTOM"
  cpu_shares      = "512"
  cpu_reservation = "200"
  cpu_limit       = "1000"
}

resource "vcd_vm" "template-vm" {
  org  = "{{.Org}}"
  vdc  = "{{.Vdc}}"

  vapp_template_id = data.vcd_catalog_vapp_template.{{.CatalogItem}}.id
  
  name        = "{{.TestName}}-template-standalone-vm"

  cpus   = 1
  memory = 1024

  memory_priority    = "CUSTOM"
  memory_shares      = "480"
  memory_reservation = "8"
  memory_limit       = "48"

  cpu_priority    = "CUSTOM"
  cpu_shares      = "512"
  cpu_reservation = "200"
  cpu_limit       = "1000"
}

resource "vcd_vm" "empty-vm" {
  org  = "{{.Org}}"
  vdc  = "{{.Vdc}}"

  name          = "{{.TestName}}-empty-standalone-vm"
  computer_name = "standalone"

  os_type          = "sles10_64Guest"
  hardware_version = "vmx-14"
  boot_image_id    = data.vcd_catalog_media.{{.Media}}.id

  cpus   = 1
  memory = 1024

  memory_priority    = "CUSTOM"
  memory_shares      = "480"
  memory_reservation = "8"
  memory_limit       = "48"

  cpu_priority    = "CUSTOM"
  cpu_shares      = "512"
  cpu_reservation = "200"
  cpu_limit       = "1000"
}

# VM Copy from here
resource "vcd_vapp" "vm-copy-destination-template-vm" {
  org         = "{{.Org}}"
  vdc         = "{{.Vdc}}"
  name        = "{{.TestName}}-vm-copy-template-destination"
  description = "vApp destination for VM Copy"
  power_on    = false
}

resource "vcd_vapp" "vm-copy-destination-empty-vm" {
  org         = "{{.Org}}"
  vdc         = "{{.Vdc}}"
  name        = "{{.TestName}}-vm-copy-empty-destination"
  description = "vApp destination for VM Copy"
  power_on    = false
}

resource "vcd_vapp_vm" "template-vm-copy" {
  org = "{{.Org}}"
  vdc = "{{.Vdc}}"

  copy_from_vm_id = vcd_vapp_vm.template-vm.id
  vapp_name       = vcd_vapp.vm-copy-destination-template-vm.name
  name            = "{{.TestName}}-template-vapp-vm-copy"
  description     = "{{.TestName}}-template-vapp-vm"

  cpus   = 1
  memory = 1024

  memory_priority    = "CUSTOM"
  memory_shares      = "480"
  memory_reservation = "8"
  memory_limit       = "48"

  cpu_priority    = "CUSTOM"
  cpu_shares      = "512"
  cpu_reservation = "200"
  cpu_limit       = "1000"
}

resource "vcd_vapp_vm" "empty-vm-copy" {
  org = "{{.Org}}"
  vdc = "{{.Vdc}}"
  
  copy_from_vm_id = vcd_vapp_vm.empty-vm.id
  vapp_name       = vcd_vapp.vm-copy-destination-empty-vm.name
  name            = "{{.TestName}}-empty-vapp-vm-copy"
  description     = "{{.TestName}}-empty-vapp-vm"
  computer_name   = "vapp-vm"

  cpus   = 1
  memory = 1024

  memory_priority    = "CUSTOM"
  memory_shares      = "480"
  memory_reservation = "8"
  memory_limit       = "48"

  cpu_priority    = "CUSTOM"
  cpu_shares      = "512"
  cpu_reservation = "200"
  cpu_limit       = "1000"

  os_type          = "sles10_64Guest"
  hardware_version = "vmx-14"
  boot_image_id    = data.vcd_catalog_media.{{.Media}}.id
}

resource "vcd_vm" "template-vm-copy" {
  org = "{{.Org}}"
  vdc = "{{.Vdc}}"

  copy_from_vm_id = vcd_vm.template-vm.id
  name            = "{{.TestName}}-template-standalone-vm-copy"
  description     = "{{.TestName}}-template-standalone-vm"

  cpus   = 1
  memory = 1024

  memory_priority    = "CUSTOM"
  memory_shares      = "480"
  memory_reservation = "8"
  memory_limit       = "48"

  cpu_priority    = "CUSTOM"
  cpu_shares      = "512"
  cpu_reservation = "200"
  cpu_limit       = "1000"
}

resource "vcd_vm" "empty-vm-copy" {
  org = "{{.Org}}"
  vdc = "{{.Vdc}}"

  copy_from_vm_id = vcd_vm.empty-vm.id
  name            = "{{.TestName}}-empty-standalone-vm-copy"
  description     = "{{.TestName}}-standalone"
  computer_name   = "standalone"

  cpus   = 1
  memory = 1024

  memory_priority    = "CUSTOM"
  memory_shares      = "480"
  memory_reservation = "8"
  memory_limit       = "48"

  cpu_priority    = "CUSTOM"
  cpu_shares      = "512"
  cpu_reservation = "200"
  cpu_limit       = "1000"

  os_type          = "sles10_64Guest"
  hardware_version = "vmx-14"
  boot_image_id    = data.vcd_catalog_media.{{.Media}}.id
}
`

// TestAccVcdVAppVm_4types_PowerState aims to test if power management works correctly for vApps and
// VMs
// Step 1 creates 4 types of powered off VMs. Two of these VMs are placed in powered-on vApps.
// The result is that vApps power state should resolve as MIXED, while all VMs must be POWERED OFF
//
// Step 2 additionally adds two more VM to existing vApps. Both of them are powered on and all power
// states are verified again. It also checks that adding a new VM did not change power statuses of
// any other existing VMs
func TestAccVcdVAppVm_4types_PowerState(t *testing.T) {
	preTestChecks(t)

	var params = StringMap{
		"TestName":        t.Name(),
		"Org":             testConfig.VCD.Org,
		"Vdc":             testConfig.Nsxt.Vdc,
		"Catalog":         testConfig.VCD.Catalog.NsxtBackedCatalogName,
		"CatalogItem":     testConfig.VCD.Catalog.CatalogItemWithMultiVms,
		"Media":           testConfig.Media.NsxtBackedMediaName,
		"NsxtEdgeGateway": testConfig.Nsxt.EdgeGateway,

		"Tags": "vapp vm",
	}
	testParamsNotEmpty(t, params)

	configTextStep1 := templateFill(testAccVcdVAppVm_4types_PowerStateStep1, params)

	params["FuncName"] = t.Name() + "-step2"
	configTextStep2 := templateFill(testAccVcdVAppVm_4types_PowerStateStep2, params)

	debugPrintf("#[DEBUG] CONFIGURATION: %s\n", configTextStep1)
	debugPrintf("#[DEBUG] CONFIGURATION: %s\n", configTextStep2)

	if vcdShortTest {
		t.Skip(acceptanceTestsSkipped)
		return
	}
	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccCheckVcdNsxtVAppVmDestroy(t.Name()+"-template-vm"),
			testAccCheckVcdNsxtVAppVmDestroy(t.Name()+"-empty-vm"),
			testAccCheckVcdStandaloneVmDestroy(t.Name()+"-template-standalone-vm", testConfig.VCD.Org, testConfig.Nsxt.Vdc),
			testAccCheckVcdStandaloneVmDestroy(t.Name()+"-empty-standalone-vm", testConfig.VCD.Org, testConfig.Nsxt.Vdc),
		),
		Steps: []resource.TestStep{
			{
				Config: configTextStep1,
				Check: resource.ComposeAggregateTestCheckFunc(

					// vApp checks
					resource.TestCheckResourceAttr("vcd_vapp.template-vm", "name", t.Name()+"-template-vm"),
					resource.TestCheckResourceAttr("vcd_vapp.template-vm", "description", "vApp for Template VM description"),
					resource.TestCheckResourceAttr("vcd_vapp.template-vm", "power_on", "true"),
					// Ignoring these two checks and only relying on a function for "live" check because vApp status
					// changes once a VM is spawned inside it. Due to Terraform inner workings the vApp does not get
					// refreshed until next read. It reports POWERED_OFF instead of MIXED as its state is stored after creation.
					//
					// resource.TestCheckResourceAttr("vcd_vapp.template-vm", "status", "10"), // 10 - means MIXED
					// resource.TestCheckResourceAttr("vcd_vapp.template-vm", "status_text", "MIXED"),
					// VCD 10.3.0 report "POWERED_OFF" instead of "MIXED" state
					testAccCheckVcdVappPowerState(testConfig.VCD.Org, testConfig.Nsxt.Vdc, t.Name()+"-template-vm", []string{"MIXED", "POWERED_OFF"}),

					resource.TestCheckResourceAttr("vcd_vapp.empty-vm", "name", t.Name()+"-empty-vm"),
					resource.TestCheckResourceAttr("vcd_vapp.empty-vm", "description", "vApp for Empty VM description"),
					resource.TestCheckResourceAttr("vcd_vapp.empty-vm", "power_on", "true"),
					// Ignoring these two checks and only relying on a function for "live" check because vApp status
					// changes once a VM is spawned inside it. Due to Terraform inner workings the vApp does not get
					// refreshed until next read. It reports POWERED_OFF instead of MIXED as its state is stored after creation.
					//
					// resource.TestCheckResourceAttr("vcd_vapp.empty-vm", "status", "10"), // 10 - means MIXED
					// resource.TestCheckResourceAttr("vcd_vapp.empty-vm", "status_text", "MIXED"),
					// VCD 10.3.0 report "POWERED_OFF" instead of "MIXED" state
					testAccCheckVcdVappPowerState(testConfig.VCD.Org, testConfig.Nsxt.Vdc, t.Name()+"-empty-vm", []string{"MIXED", "POWERED_OFF"}),

					// Template vApp VM checks
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "vm_type", "vcd_vapp_vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "name", t.Name()+"-template-vapp-vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "description", t.Name()+"-template-vapp-vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "power_on", "false"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "status", "8"), // 8 - means POWERED OFF
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "status_text", "POWERED_OFF"),
					testAccCheckVcdVMPowerState(testConfig.VCD.Org, testConfig.Nsxt.Vdc, t.Name()+"-template-vm", t.Name()+"-template-vapp-vm", "POWERED_OFF"),

					// Empty vApp VM checks
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "vm_type", "vcd_vapp_vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "name", t.Name()+"-empty-vapp-vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "description", t.Name()+"-empty-vapp-vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "computer_name", "vapp-vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "power_on", "false"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "status", "8"), // 8 - means POWERED OFF
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "status_text", "POWERED_OFF"),
					testAccCheckVcdVMPowerState(testConfig.VCD.Org, testConfig.Nsxt.Vdc, t.Name()+"-empty-vm", t.Name()+"-empty-vapp-vm", "POWERED_OFF"),

					// Standalone template VM checks
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "vm_type", "vcd_vm"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "name", t.Name()+"-template-standalone-vm"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "description", t.Name()+"-template-standalone-vm"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "power_on", "false"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "status", "8"), // 8 - means POWERED OFF
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "status_text", "POWERED_OFF"),
					testAccCheckVcdVMPowerState(testConfig.VCD.Org, testConfig.Nsxt.Vdc, "", t.Name()+"-template-standalone-vm", "POWERED_OFF"),

					// Standalone empty VM checks
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "vm_type", "vcd_vm"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "name", t.Name()+"-empty-standalone-vm"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "description", t.Name()+"-standalone"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "power_on", "false"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "status", "8"), // 8 - means POWERED OFF
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "status_text", "POWERED_OFF"),
					testAccCheckVcdVMPowerState(testConfig.VCD.Org, testConfig.Nsxt.Vdc, "", t.Name()+"-empty-standalone-vm", "POWERED_OFF"),

					// VM copy checks
					// vApp checks
					resource.TestCheckResourceAttr("vcd_vapp.vm-copy-destination-template-vm", "name", t.Name()+"-vm-copy-template-destination"),
					resource.TestCheckResourceAttr("vcd_vapp.vm-copy-destination-template-vm", "power_on", "true"),
					testAccCheckVcdVappPowerState(testConfig.VCD.Org, testConfig.Nsxt.Vdc, t.Name()+"-vm-copy-template-destination", []string{"MIXED", "POWERED_OFF"}),

					resource.TestCheckResourceAttr("vcd_vapp.vm-copy-destination-empty-vm", "name", t.Name()+"-vm-copy-empty-destination"),
					resource.TestCheckResourceAttr("vcd_vapp.vm-copy-destination-empty-vm", "power_on", "true"),
					testAccCheckVcdVappPowerState(testConfig.VCD.Org, testConfig.Nsxt.Vdc, t.Name()+"-vm-copy-empty-destination", []string{"MIXED", "POWERED_OFF"}),

					// Template vApp VM checks
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "vm_type", "vcd_vapp_vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "name", t.Name()+"-template-vapp-vm-copy"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "power_on", "false"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "status", "8"), // 8 - means POWERED OFF
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm-copy", "status_text", "POWERED_OFF"),
					testAccCheckVcdVMPowerState(testConfig.VCD.Org, testConfig.Nsxt.Vdc,
						t.Name()+"-vm-copy-template-destination",
						t.Name()+"-template-vapp-vm-copy",
						"POWERED_OFF"),

					// Empty vApp VM checks
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "vm_type", "vcd_vapp_vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "name", t.Name()+"-empty-vapp-vm-copy"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "description", t.Name()+"-empty-vapp-vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "computer_name", "vapp-vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "power_on", "false"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "status", "8"), // 8 - means POWERED OFF
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm-copy", "status_text", "POWERED_OFF"),
					testAccCheckVcdVMPowerState(testConfig.VCD.Org, testConfig.Nsxt.Vdc, t.Name()+"-vm-copy-empty-destination", t.Name()+"-empty-vapp-vm-copy", "POWERED_OFF"),

					// Standalone template VM checks
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "vm_type", "vcd_vm"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "name", t.Name()+"-template-standalone-vm-copy"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "description", t.Name()+"-template-standalone-vm"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "power_on", "false"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "status", "8"), // 8 - means POWERED OFF
					resource.TestCheckResourceAttr("vcd_vm.template-vm-copy", "status_text", "POWERED_OFF"),
					testAccCheckVcdVMPowerState(testConfig.VCD.Org, testConfig.Nsxt.Vdc, "", t.Name()+"-template-standalone-vm-copy", "POWERED_OFF"),

					// Standalone empty VM checks
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "vm_type", "vcd_vm"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "name", t.Name()+"-empty-standalone-vm-copy"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "description", t.Name()+"-standalone-copy"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "power_on", "false"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "status", "8"), // 8 - means POWERED OFF
					resource.TestCheckResourceAttr("vcd_vm.empty-vm-copy", "status_text", "POWERED_OFF"),
					testAccCheckVcdVMPowerState(testConfig.VCD.Org, testConfig.Nsxt.Vdc, "", t.Name()+"-empty-standalone-vm-copy", "POWERED_OFF"),
				),
			},
			{
				Config: configTextStep2,
				Check: resource.ComposeAggregateTestCheckFunc(

					// vApp checks
					resource.TestCheckResourceAttr("vcd_vapp.template-vm", "name", t.Name()+"-template-vm"),
					resource.TestCheckResourceAttr("vcd_vapp.template-vm", "description", "vApp for Template VM description"),
					resource.TestCheckResourceAttr("vcd_vapp.template-vm", "power_on", "true"),
					// Ignoring these two checks and only relying on a function for "live" check because vApp status
					// changes once a VM is spawned inside it. Due to Terraform inner workings the vApp does not get
					// refreshed until next read. It reports POWERED_OFF instead of MIXED as its state is stored after creation.
					//
					// resource.TestCheckResourceAttr("vcd_vapp.template-vm", "status", "10"), // 10 - means MIXED
					// resource.TestCheckResourceAttr("vcd_vapp.template-vm", "status_text", "MIXED"),
					testAccCheckVcdVappPowerState(testConfig.VCD.Org, testConfig.Nsxt.Vdc, t.Name()+"-template-vm", []string{"MIXED"}),

					resource.TestCheckResourceAttr("vcd_vapp.empty-vm", "name", t.Name()+"-empty-vm"),
					resource.TestCheckResourceAttr("vcd_vapp.empty-vm", "description", "vApp for Empty VM description"),
					resource.TestCheckResourceAttr("vcd_vapp.empty-vm", "power_on", "true"),
					// Ignoring these two checks and only relying on a function for "live" check because vApp status
					// changes once a VM is spawned inside it. Due to Terraform inner workings the vApp does not get
					// refreshed until next read. It reports POWERED_OFF instead of MIXED as its state is stored after creation.
					//
					// resource.TestCheckResourceAttr("vcd_vapp.empty-vm", "status", "10"), // 10 - means MIXED
					// resource.TestCheckResourceAttr("vcd_vapp.empty-vm", "status_text", "MIXED"),
					testAccCheckVcdVappPowerState(testConfig.VCD.Org, testConfig.Nsxt.Vdc, t.Name()+"-empty-vm", []string{"MIXED"}),

					// Template vApp VM checks
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "vm_type", "vcd_vapp_vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "name", t.Name()+"-template-vapp-vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "description", t.Name()+"-template-vapp-vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "power_on", "false"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "status", "8"), // 8 - means POWERED OFF
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "status_text", "POWERED_OFF"),
					testAccCheckVcdVMPowerState(testConfig.VCD.Org, testConfig.Nsxt.Vdc, t.Name()+"-template-vm", t.Name()+"-template-vapp-vm", "POWERED_OFF"),

					// Template vApp VM 2 checks
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm2", "vm_type", "vcd_vapp_vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm2", "name", t.Name()+"-template-vapp-vm-2"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm2", "description", t.Name()+"-template-vapp-vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm2", "power_on", "true"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm2", "status", "4"), // 4 - means POWERED ON
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm2", "status_text", "POWERED_ON"),
					testAccCheckVcdVMPowerState(testConfig.VCD.Org, testConfig.Nsxt.Vdc, t.Name()+"-template-vm", t.Name()+"-template-vapp-vm-2", "POWERED_ON"),

					// Empty vApp VM checks
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "vm_type", "vcd_vapp_vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "name", t.Name()+"-empty-vapp-vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "description", t.Name()+"-empty-vapp-vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "computer_name", "vapp-vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "power_on", "false"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "status", "8"), // 8 - means POWERED OFF
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm", "status_text", "POWERED_OFF"),
					testAccCheckVcdVMPowerState(testConfig.VCD.Org, testConfig.Nsxt.Vdc, t.Name()+"-empty-vm", t.Name()+"-empty-vapp-vm", "POWERED_OFF"),

					// Empty vApp VM 2 checks
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm2", "vm_type", "vcd_vapp_vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm2", "name", t.Name()+"-empty-vapp-vm-2"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm2", "description", t.Name()+"-empty-vapp-vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm2", "computer_name", "vapp-vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm2", "power_on", "true"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm2", "status", "4"), // 4 - means POWERED ON
					resource.TestCheckResourceAttr("vcd_vapp_vm.empty-vm2", "status_text", "POWERED_ON"),
					testAccCheckVcdVMPowerState(testConfig.VCD.Org, testConfig.Nsxt.Vdc, t.Name()+"-empty-vm", t.Name()+"-empty-vapp-vm-2", "POWERED_ON"),

					// Standalone template VM checks
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "vm_type", "vcd_vm"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "name", t.Name()+"-template-standalone-vm"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "description", t.Name()+"-template-standalone-vm"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "power_on", "false"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "status", "8"), // 8 - means POWERED OFF
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "status_text", "POWERED_OFF"),
					testAccCheckVcdVMPowerState(testConfig.VCD.Org, testConfig.Nsxt.Vdc, "", t.Name()+"-template-standalone-vm", "POWERED_OFF"),

					// Standalone empty VM checks
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "vm_type", "vcd_vm"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "name", t.Name()+"-empty-standalone-vm"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "description", t.Name()+"-standalone"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "power_on", "false"),
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "status", "8"), // 8 - means POWERED OFF
					resource.TestCheckResourceAttr("vcd_vm.empty-vm", "status_text", "POWERED_OFF"),
					testAccCheckVcdVMPowerState(testConfig.VCD.Org, testConfig.Nsxt.Vdc, "", t.Name()+"-empty-standalone-vm", "POWERED_OFF"),
				),
			},
		},
	})
	postTestChecks(t)
}

const testAccVcdVAppVm_4types_PowerStateStep1 = `
resource "vcd_vapp" "template-vm" {
  org         = "{{.Org}}"
  vdc         = "{{.Vdc}}"
  name        = "{{.TestName}}-template-vm"
  description = "vApp for Template VM description"
  power_on    = true
}

resource "vcd_vapp" "empty-vm" {
  org         = "{{.Org}}"
  vdc         = "{{.Vdc}}"
  name        = "{{.TestName}}-empty-vm"
  description = "vApp for Empty VM description"
  power_on    = true
}

data "vcd_catalog" "{{.Catalog}}" {
  org  = "{{.Org}}"
  name = "{{.Catalog}}"
}

data "vcd_catalog_vapp_template" "{{.CatalogItem}}" {
  org         = "{{.Org}}"
  catalog_id = data.vcd_catalog.{{.Catalog}}.id
  name       = "{{.CatalogItem}}"
}

data "vcd_catalog_media" "{{.Media}}" {
  org     = "{{.Org}}"
  catalog = data.vcd_catalog.{{.Catalog}}.name
  name    = "{{.Media}}"
}

resource "vcd_vapp_vm" "template-vm" {
  org  = "{{.Org}}"
  vdc  = "{{.Vdc}}"

  vapp_template_id = data.vcd_catalog_vapp_template.{{.CatalogItem}}.id
  
  vapp_name   = vcd_vapp.template-vm.name
  name        = "{{.TestName}}-template-vapp-vm"
  description = "{{.TestName}}-template-vapp-vm"
  power_on    = false
}

resource "vcd_vapp_vm" "empty-vm" {
  org  = "{{.Org}}"
  vdc  = "{{.Vdc}}"
  
  vapp_name     = vcd_vapp.empty-vm.name
  name          = "{{.TestName}}-empty-vapp-vm"
  description   = "{{.TestName}}-empty-vapp-vm"
  computer_name = "vapp-vm"
  power_on      = false

  cpus   = 1
  memory = 1024

  os_type          = "sles10_64Guest"
  hardware_version = "vmx-14"
  boot_image_id    = data.vcd_catalog_media.{{.Media}}.id
}

resource "vcd_vm" "template-vm" {
  org  = "{{.Org}}"
  vdc  = "{{.Vdc}}"

  vapp_template_id = data.vcd_catalog_vapp_template.{{.CatalogItem}}.id
  
  name        = "{{.TestName}}-template-standalone-vm"
  description = "{{.TestName}}-template-standalone-vm"
  power_on    = false
}

resource "vcd_vm" "empty-vm" {
  org  = "{{.Org}}"
  vdc  = "{{.Vdc}}"

  name          = "{{.TestName}}-empty-standalone-vm"
  description   = "{{.TestName}}-standalone"
  computer_name = "standalone"
  power_on      = false

  cpus   = 1
  memory = 1024

  os_type          = "sles10_64Guest"
  hardware_version = "vmx-14"
  boot_image_id    = data.vcd_catalog_media.{{.Media}}.id
}

# VM Copy from here
resource "vcd_vapp" "vm-copy-destination-template-vm" {
  org         = "{{.Org}}"
  vdc         = "{{.Vdc}}"
  name        = "{{.TestName}}-vm-copy-template-destination"
  description = "vApp destination for VM Copy"
  power_on    = vcd_vapp.template-vm.power_on
}

resource "vcd_vapp" "vm-copy-destination-empty-vm" {
  org         = "{{.Org}}"
  vdc         = "{{.Vdc}}"
  name        = "{{.TestName}}-vm-copy-empty-destination"
  description = "vApp destination for VM Copy"
  power_on    = vcd_vapp.empty-vm.power_on
}

resource "vcd_vapp_vm" "template-vm-copy" {
  org = "{{.Org}}"
  vdc = "{{.Vdc}}"

  copy_from_vm_id = vcd_vapp_vm.template-vm.id
  vapp_name       = vcd_vapp.vm-copy-destination-template-vm.name
  name            = "{{.TestName}}-template-vapp-vm-copy"
  description     = "{{.TestName}}-template-vapp-vm"
  power_on        = vcd_vapp_vm.template-vm.power_on

  cpus   = 1
  memory = 1024
}

resource "vcd_vapp_vm" "empty-vm-copy" {
  org = "{{.Org}}"
  vdc = "{{.Vdc}}"
  
  copy_from_vm_id = vcd_vapp_vm.empty-vm.id
  vapp_name       = vcd_vapp.vm-copy-destination-empty-vm.name
  name            = "{{.TestName}}-empty-vapp-vm-copy"
  description     = "{{.TestName}}-empty-vapp-vm"
  computer_name   = "vapp-vm"
  power_on        = vcd_vapp_vm.empty-vm.power_on

  cpus   = 1
  memory = 1024
}

resource "vcd_vm" "template-vm-copy" {
  org = "{{.Org}}"
  vdc = "{{.Vdc}}"

  copy_from_vm_id = vcd_vm.template-vm.id
  name            = "{{.TestName}}-template-standalone-vm-copy"
  description     = "{{.TestName}}-template-standalone-vm"
  power_on        = vcd_vm.template-vm.power_on

  cpus   = 1
  memory = 1024
}

resource "vcd_vm" "empty-vm-copy" {
  org = "{{.Org}}"
  vdc = "{{.Vdc}}"

  copy_from_vm_id = vcd_vm.empty-vm.id
  name            = "{{.TestName}}-empty-standalone-vm-copy"
  description     = "{{.TestName}}-standalone-copy"
  computer_name   = "standalone"
  power_on        = vcd_vm.empty-vm.power_on

  cpus   = 1
  memory = 1024
}
`

const testAccVcdVAppVm_4types_PowerStateStep2 = testAccVcdVAppVm_4types_PowerStateStep1 + `
resource "vcd_vapp_vm" "template-vm2" {
  org  = "{{.Org}}"
  vdc  = "{{.Vdc}}"

  vapp_template_id = data.vcd_catalog_vapp_template.{{.CatalogItem}}.id
  
  vapp_name   = vcd_vapp.template-vm.name
  name        = "{{.TestName}}-template-vapp-vm-2"
  description = "{{.TestName}}-template-vapp-vm"
  power_on    = true
}

resource "vcd_vapp_vm" "empty-vm2" {
  org  = "{{.Org}}"
  vdc  = "{{.Vdc}}"
  
  vapp_name     = vcd_vapp.empty-vm.name
  name          = "{{.TestName}}-empty-vapp-vm-2"
  description   = "{{.TestName}}-empty-vapp-vm"
  computer_name = "vapp-vm"
  power_on      = true

  cpus   = 1
  memory = 1024

  os_type          = "sles10_64Guest"
  hardware_version = "vmx-14"
  boot_image_id    = data.vcd_catalog_media.{{.Media}}.id
}
`

// testAccCheckVcdVMPowerState checks if a given VM has expected status
// `expectedStatus` comes from types.VAppStatuses
func testAccCheckVcdVMPowerState(orgName, vdcName string, vappName, vmName, expectedStatus string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*VCDClient)
		_, vdc, err := conn.GetOrgAndVdc(orgName, vdcName)
		if err != nil {
			return fmt.Errorf(errorRetrievingVdcFromOrg, vdcName, orgName, err)
		}

		var vm *govcd.VM

		// vApp VM
		if vappName != "" {
			vapp, err := vdc.GetVAppByName(vappName, false)
			if err != nil {
				return err
			}
			vm, err = vapp.GetVMByName(vmName, false)
			if err != nil {
				return err
			}
		} else { // Standalone VM lookup
			vm, _, err = getVmByName(conn, vdc, vmName)
			if err != nil {
				return fmt.Errorf("error looking up standalone VM '%s': %s", vmName, err)
			}
		}

		// getVmByName

		vmStatus, err := vm.GetStatus()
		if err != nil {
			return fmt.Errorf("error retrieving VM power status: %s", err)
		}

		if vcdTestVerbose {
			fmt.Printf("VM '%s' status expected '%s', got '%s'\n", vm.VM.Name, expectedStatus, vmStatus)
		}

		if vmStatus != expectedStatus {
			return fmt.Errorf("expected VM '%s' to have status '%s', got '%s'", vm.VM.Name, expectedStatus, vmStatus)
		}

		return nil
	}
}

// testAccCheckVcdVappPowerState checks if given vApp has expected status
// `expectedStatus` comes from types.VAppStatuses
func testAccCheckVcdVappPowerState(orgName, vdcName string, vappName string, expectedStatuses []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*VCDClient)
		_, vdc, err := conn.GetOrgAndVdc(orgName, vdcName)
		if err != nil {
			return fmt.Errorf(errorRetrievingVdcFromOrg, vdcName, orgName, err)
		}

		vapp, err := vdc.GetVAppByName(vappName, false)
		if err != nil {
			return err
		}

		vappStatus, err := vapp.GetStatus()
		if err != nil {
			return fmt.Errorf("error retrieving vApp power status: %s", err)
		}

		if vcdTestVerbose {
			fmt.Printf("vApp '%s' status expected '%s', got '%s'\n", vapp.VApp.Name, expectedStatuses, vappStatus)
		}

		if !slices.Contains(expectedStatuses, vappStatus) {
			return fmt.Errorf("expected vApp '%s' to have status '%s', got '%s'", vapp.VApp.Name, expectedStatuses, vappStatus)
		}

		return nil
	}
}

// TestAccVcdVAppVm_2typesOverrideDiskFastProvisionedVdc checks that `consolidate_disks_on_create`
// performs disk consolidation, which in turn allows to use 'override_template_disk' for growing
// template based VMs (vApp and standalone) at the time of creation in fast provisioned VDCs
func TestAccVcdVAppVm_2typesOverrideDiskFastProvisionedVdc(t *testing.T) {
	preTestChecks(t)

	var params = StringMap{
		"TestName":           t.Name(),
		"Org":                testConfig.VCD.Org,
		"Vdc":                testConfig.Nsxt.Vdc,
		"Catalog":            testConfig.VCD.Catalog.NsxtBackedCatalogName,
		"CatalogItem":        testConfig.VCD.Catalog.CatalogItemWithMultiVms,
		"VmNameInTemplate1":  testConfig.VCD.Catalog.VmName1InMultiVmItem,
		"VmNameInTemplate2":  testConfig.VCD.Catalog.VmName2InMultiVmItem,
		"Media":              testConfig.Media.NsxtBackedMediaName,
		"NsxtEdgeGateway":    testConfig.Nsxt.EdgeGateway,
		"StorageProfileName": testConfig.VCD.NsxtProviderVdc.StorageProfile,

		"Tags": "vapp vm",
	}
	testParamsNotEmpty(t, params)

	configTextStep1 := templateFill(testAccVcdVAppVm_4typesOverrideDiskFastProvisionedVdc, params)

	if vcdShortTest {
		t.Skip(acceptanceTestsSkipped)
		return
	}
	debugPrintf("#[DEBUG] CONFIGURATION: %s\n", configTextStep1)
	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			testAccCheckVcdNsxtVAppVmDestroy(t.Name()+"-template-vm"),
			testAccCheckVcdStandaloneVmDestroy(t.Name()+"-template-standalone-vm", testConfig.VCD.Org, testConfig.Nsxt.Vdc),
		),
		Steps: []resource.TestStep{
			{
				Config: configTextStep1,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Template vApp VM checks
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "vm_type", "vcd_vapp_vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "name", t.Name()+"-template-vapp-vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "description", t.Name()+"-template-vapp-vm"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "power_on", "true"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "consolidate_disks_on_create", "true"),
					resource.TestCheckResourceAttr("vcd_vapp_vm.template-vm", "status_text", "POWERED_ON"),
					testAccCheckVcdVMPowerState(testConfig.VCD.Org, testConfig.Nsxt.Vdc, t.Name()+"-template-vm", t.Name()+"-template-vapp-vm", "POWERED_ON"),
					resource.TestCheckOutput("vcd_vapp_vm_disk_size", "20480"),

					// Standalone template VM checks
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "vm_type", "vcd_vm"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "name", t.Name()+"-template-standalone-vm"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "description", t.Name()+"-template-standalone-vm"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "power_on", "true"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "consolidate_disks_on_create", "true"),
					resource.TestCheckResourceAttr("vcd_vm.template-vm", "status_text", "POWERED_ON"),
					testAccCheckVcdVMPowerState(testConfig.VCD.Org, testConfig.Nsxt.Vdc, "", t.Name()+"-template-standalone-vm", "POWERED_ON"),
					resource.TestCheckOutput("vcd_vm_disk_size", "20480"),
				),
			},
		},
	})
	postTestChecks(t)
}

const testAccVcdVAppVm_4typesOverrideDiskFastProvisionedVdc = `
data "vcd_catalog" "{{.Catalog}}" {
  org  = "{{.Org}}"
  name = "{{.Catalog}}"
}

data "vcd_catalog_vapp_template" "multivm" {
  org         = "{{.Org}}"
  catalog_id = data.vcd_catalog.{{.Catalog}}.id
  name       = "{{.CatalogItem}}"
}

resource "vcd_vapp" "template-vm" {
  org         = "{{.Org}}"
  vdc         = "{{.Vdc}}"
  name        = "{{.TestName}}-template-vm"
  description = "vApp for Template VM description"
  power_on    = true
}

resource "vcd_vapp_vm" "template-vm" {
  org  = "{{.Org}}"
  vdc  = "{{.Vdc}}"

  vapp_template_id    = data.vcd_catalog_vapp_template.multivm.id
  vm_name_in_template = "{{.VmNameInTemplate1}}"
  
  vapp_name   = vcd_vapp.template-vm.name
  name        = "{{.TestName}}-template-vapp-vm"
  description = "{{.TestName}}-template-vapp-vm"
  power_on    = true

  consolidate_disks_on_create = true

  override_template_disk {
    bus_type         = "parallel"
    size_in_mb       = "20480"
    bus_number       = 0
    unit_number      = 0
    iops             = 0
    storage_profile  = "{{.StorageProfileName}}"
  }

  prevent_update_power_off = true
}

resource "vcd_vm" "template-vm" {
  org  = "{{.Org}}"
  vdc  = "{{.Vdc}}"

  vapp_template_id    = data.vcd_catalog_vapp_template.multivm.id
  vm_name_in_template = "{{.VmNameInTemplate1}}"
  
  name        = "{{.TestName}}-template-standalone-vm"
  description = "{{.TestName}}-template-standalone-vm"
  power_on    = true

  consolidate_disks_on_create = true

  override_template_disk {
    bus_type         = "parallel"
    size_in_mb       = "20480"
    bus_number       = 0
    unit_number      = 0
    iops             = 0
    storage_profile  = "{{.StorageProfileName}}"
  }

  prevent_update_power_off = true
}

output "vcd_vapp_vm_disk_size" {
  value = tolist(vcd_vapp_vm.template-vm.override_template_disk)[0].size_in_mb
}

output "vcd_vm_disk_size" {
	value = tolist(vcd_vm.template-vm.override_template_disk)[0].size_in_mb
}
`
