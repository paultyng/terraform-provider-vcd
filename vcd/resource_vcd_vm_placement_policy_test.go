//go:build vdc || ALL || functional

package vcd

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccVcdVmPlacementPolicy(t *testing.T) {
	preTestChecks(t)
	skipIfNotSysAdmin(t)

	var params = StringMap{
		"VdcName":     testConfig.Nsxt.Vdc,
		"PolicyName":  t.Name(),
		"VmGroup":     testConfig.VCD.NsxtProviderVdc.PlacementPolicyVmGroup,
		"Description": t.Name() + "_description",
	}
	testParamsNotEmpty(t, params)
	policyName := "vcd_vm_placement_policy." + params["PolicyName"].(string)
	datasourcePolicyName := "data.vcd_vm_placement_policy.data-" + params["PolicyName"].(string)
	configText := templateFill(testAccCheckVmPlacementPolicy_create, params)
	params["FuncName"] = t.Name() + "-Update"
	configTextUpdate := templateFill(testAccCheckVmPlacementPolicy_update, params)

	debugPrintf("#[DEBUG] CONFIGURATION - creation: %s", configText)

	if vcdShortTest {
		t.Skip(acceptanceTestsSkipped)
		return
	}

	vcdClient := createTemporaryVCDConnection(true)
	if vcdClient == nil {
		t.Skip(acceptanceTestsSkipped)
	}
	vmPlacementPolicyDescription := "This is a system generated default compute policy auto assigned to this vDC."
	if vcdClient.Client.APIVCDMaxVersionIs("< 38.0") || vcdClient.Client.APIVCDMaxVersionIs("> 39.0") {
		vmPlacementPolicyDescription = ""
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccCheckComputePolicyDestroyed(t.Name()+"-update", "placement"),
		Steps: []resource.TestStep{
			{
				Config: configText,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(policyName, "id", getUuidRegex("urn:vcloud:vdcComputePolicy:", "$")),
					resource.TestCheckResourceAttr(policyName, "name", params["PolicyName"].(string)),
					resource.TestCheckResourceAttr(policyName, "description", params["Description"].(string)),
					resource.TestMatchResourceAttr(policyName, "provider_vdc_id", getUuidRegex("urn:vcloud:providervdc:", "$")),
					resource.TestCheckResourceAttr(policyName, "vm_group_ids.#", "1"),
					resource.TestCheckResourceAttr(policyName, "logical_vm_group_ids.#", "0"),
					resource.TestMatchResourceAttr(policyName, "vm_group_ids.0", getUuidRegex("^", "$")),
					resource.TestCheckNoResourceAttr(policyName, "vdc_id"),
					resourceFieldsEqual(policyName, datasourcePolicyName, []string{"%"}), // Data source has extra attribute `vdc_id`

					// Checks that the description is correct when it is not populated
					resource.TestCheckResourceAttr(policyName+"_without_description", "name", params["PolicyName"].(string)+"WithoutDescription"),
					resource.TestCheckResourceAttr(policyName+"_without_description", "description", vmPlacementPolicyDescription),
				),
			},
			{
				Config: configTextUpdate,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(policyName, "id", getUuidRegex("urn:vcloud:vdcComputePolicy:", "$")),
					resource.TestCheckResourceAttr(policyName, "name", params["PolicyName"].(string)+"-update"),
					resource.TestCheckResourceAttr(policyName, "description", params["Description"].(string)+"-update"),
					resource.TestMatchResourceAttr(policyName, "provider_vdc_id", getUuidRegex("urn:vcloud:providervdc:", "$")),
					resource.TestCheckResourceAttr(policyName, "vm_group_ids.#", "1"),
					resource.TestCheckResourceAttr(policyName, "logical_vm_group_ids.#", "0"),
					resource.TestMatchResourceAttr(policyName, "vm_group_ids.0", getUuidRegex("^", "$")),
					resource.TestCheckNoResourceAttr(policyName, "vdc_id"),
					resourceFieldsEqual(policyName, datasourcePolicyName, []string{"%"}), // Data source has extra attribute `vdc_id`
				),
			},
			// Tests import by id
			{
				ResourceName:      policyName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: importStateComputePolicyByIdOrName(policyName, true),
			},
			// Tests import by name
			{
				ResourceName:      policyName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: importStateComputePolicyByIdOrName(policyName, false),
			},
		},
	})
	postTestChecks(t)
}

const testAccCheckVmPlacementPolicy_create = `
data "vcd_org_vdc" "vdc" {
  name = "{{.VdcName}}"
}

data "vcd_provider_vdc" "pvdc" {
  name = data.vcd_org_vdc.vdc.provider_vdc_name
}

data "vcd_vm_group" "vm-group" {
  name            = "{{.VmGroup}}"
  provider_vdc_id = data.vcd_provider_vdc.pvdc.id
}

resource "vcd_vm_placement_policy" "{{.PolicyName}}" {
  name            = "{{.PolicyName}}"
  description     = "{{.Description}}"
  provider_vdc_id = data.vcd_provider_vdc.pvdc.id
  vm_group_ids    = [ data.vcd_vm_group.vm-group.id ]
}

resource "vcd_vm_placement_policy" "{{.PolicyName}}_without_description" {
  name            = "{{.PolicyName}}WithoutDescription"
  provider_vdc_id = data.vcd_provider_vdc.pvdc.id
  vm_group_ids    = [ data.vcd_vm_group.vm-group.id ]
}

data "vcd_vm_placement_policy" "data-{{.PolicyName}}" {
  name            = vcd_vm_placement_policy.{{.PolicyName}}.name
  provider_vdc_id = vcd_vm_placement_policy.{{.PolicyName}}.provider_vdc_id
}
`

const testAccCheckVmPlacementPolicy_update = `
data "vcd_org_vdc" "vdc" {
  name = "{{.VdcName}}"
}

data "vcd_provider_vdc" "pvdc" {
  name = data.vcd_org_vdc.vdc.provider_vdc_name
}

data "vcd_vm_group" "vm-group" {
  name            = "{{.VmGroup}}"
  provider_vdc_id = data.vcd_provider_vdc.pvdc.id
}

resource "vcd_vm_placement_policy" "{{.PolicyName}}" {
  name            = "{{.PolicyName}}-update"
  description     = "{{.Description}}-update"
  provider_vdc_id = data.vcd_provider_vdc.pvdc.id
  vm_group_ids    = [ data.vcd_vm_group.vm-group.id ]
}

data "vcd_vm_placement_policy" "data-{{.PolicyName}}" {
  name            = vcd_vm_placement_policy.{{.PolicyName}}.name
  provider_vdc_id = vcd_vm_placement_policy.{{.PolicyName}}.provider_vdc_id
}
`

// TestAccVcdVmPlacementPolicyInVdc tests fetching a VM Placement Policy using the `vdc_id` instead
// of Provider VDC, with System Administrator.
func TestAccVcdVmPlacementPolicyInVdc(t *testing.T) {
	preTestChecks(t)
	skipIfNotSysAdmin(t)

	var params = StringMap{
		"OrgName":                   testConfig.VCD.Org,
		"VdcName":                   t.Name(),
		"PolicyName":                t.Name(),
		"ProviderVdc":               testConfig.VCD.NsxtProviderVdc.Name,
		"NetworkPool":               testConfig.VCD.NsxtProviderVdc.NetworkPool,
		"ProviderVdcStorageProfile": testConfig.VCD.NsxtProviderVdc.StorageProfile,
		"VmGroup":                   testConfig.VCD.NsxtProviderVdc.PlacementPolicyVmGroup,
	}
	testParamsNotEmpty(t, params)
	policyName := "vcd_vm_placement_policy." + params["PolicyName"].(string)
	datasourcePolicyName := "data.vcd_vm_placement_policy.data-" + params["PolicyName"].(string)
	configText := templateFill(testAccCheckVmPlacementPolicyFromVdcId, params)

	debugPrintf("#[DEBUG] CONFIGURATION - creation: %s", configText)

	if vcdShortTest {
		t.Skip(acceptanceTestsSkipped)
		return
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccCheckComputePolicyDestroyed(t.Name(), "placement"),
		Steps: []resource.TestStep{
			{
				Config: configText,
				Check: resource.ComposeTestCheckFunc(
					// System administrator
					resource.TestMatchResourceAttr(datasourcePolicyName, "id", getUuidRegex("urn:vcloud:vdcComputePolicy:", "$")),
					resource.TestCheckResourceAttr(datasourcePolicyName, "name", params["PolicyName"].(string)),
					resource.TestCheckResourceAttr(datasourcePolicyName, "description", "foo"),
					resource.TestMatchResourceAttr(datasourcePolicyName, "vdc_id", getUuidRegex("urn:vcloud:vdc:", "$")),
					resource.TestCheckResourceAttr(datasourcePolicyName, "vm_group_ids.#", "1"),
					resource.TestCheckResourceAttr(datasourcePolicyName, "logical_vm_group_ids.#", "0"),
					resource.TestMatchResourceAttr(datasourcePolicyName, "vm_group_ids.0", getUuidRegex("^", "$")),
					resource.TestCheckNoResourceAttr(datasourcePolicyName, "provider_vdc_id"),
					resourceFieldsEqual(policyName, datasourcePolicyName, []string{"%", "provider_vdc_id"}), // Resource doesn't have attribute `vdc_id` and we didn't use `provider_vdc_id` in data source
				),
			},
		},
	})
	postTestChecks(t)
}

const testAccCheckVmPlacementPolicyFromVdcId_prereqs = `
data "vcd_provider_vdc" "pvdc" {
  name = "{{.ProviderVdc}}"
}

data "vcd_vm_group" "vm-group" {
  name            = "{{.VmGroup}}"
  provider_vdc_id = data.vcd_provider_vdc.pvdc.id
}

resource "vcd_vm_placement_policy" "{{.PolicyName}}" {
  name            = "{{.PolicyName}}"
  description     = "foo"
  provider_vdc_id = data.vcd_provider_vdc.pvdc.id
  vm_group_ids    = [ data.vcd_vm_group.vm-group.id ]
}

resource "vcd_org_vdc" "{{.VdcName}}" {
  name = "{{.VdcName}}"
  org  = "{{.OrgName}}"

  allocation_model  = "ReservationPool"
  network_pool_name = "{{.NetworkPool}}"
  provider_vdc_name = data.vcd_provider_vdc.pvdc.name

  compute_capacity {
    cpu {
      allocated = 1024
      limit     = 1024
    }

    memory {
      allocated = 1024
      limit     = 1024
    }
  }

  storage_profile {
    name     = "{{.ProviderVdcStorageProfile}}"
    enabled  = true
    limit    = 10240
    default  = true
  }

  enabled                    = true
  enable_thin_provisioning   = true
  enable_fast_provisioning   = true
  delete_force               = true
  delete_recursive           = true

  default_compute_policy_id = vcd_vm_placement_policy.{{.PolicyName}}.id
  vm_placement_policy_ids   = [vcd_vm_placement_policy.{{.PolicyName}}.id]
}
`

const testAccCheckVmPlacementPolicyFromVdcId = testAccCheckVmPlacementPolicyFromVdcId_prereqs + `
data "vcd_vm_placement_policy" "data-{{.PolicyName}}" {
  name   = vcd_vm_placement_policy.{{.PolicyName}}.name
  vdc_id = vcd_org_vdc.{{.VdcName}}.id
}
`

// TestAccVcdVmPlacementPolicyInVdcTenant complements TestAccVcdVmPlacementPolicyInVdc and uses SDK
// connection to build up prerequisites with System user so that this test can run in both System
// and Org user modes.
func TestAccVcdVmPlacementPolicyInVdcTenant(t *testing.T) {
	preTestChecks(t)

	// This test cannot run in Short mode because it uses go-vcloud-director SDK to setup prerequisites
	if vcdShortTest {
		t.Skip(acceptanceTestsSkipped)
		return
	}

	vcdClient := createSystemTemporaryVCDConnection()

	// Setup prerequisites using temporary admin version and defer cleanup
	systemPrerequisites := &vdcPlacementPolicyOrgUserPrerequisites{t: t, vcdClient: vcdClient}
	fmt.Println("## Setting up prerequisites using System user")
	systemPrerequisites.setup()
	fmt.Println("## Running Terraform test")

	defer func() {
		fmt.Println("## Cleaning up prerequisites")
		systemPrerequisites.teardown()
		fmt.Println("## Finished cleaning up prerequisites")
	}()

	var params = StringMap{
		// These fields come from prerequisite builder
		"PlacementPolicyName": systemPrerequisites.placementPolicy.VdcComputePolicyV2.Name,
		"VdcId":               systemPrerequisites.vdc.Vdc.ID,
	}
	testParamsNotEmpty(t, params)

	configText := templateFill(testAccVcdVmPlacementPolicyInVdcTenant, params)
	debugPrintf("#[DEBUG] CONFIGURATION - creation: %s", configText)

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: configText,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.vcd_vm_placement_policy.tenant", "id", systemPrerequisites.placementPolicy.VdcComputePolicyV2.ID),
					resource.TestCheckResourceAttr("data.vcd_vm_placement_policy.tenant", "vdc_id", systemPrerequisites.vdc.Vdc.ID),
					resource.TestCheckResourceAttr("data.vcd_vm_placement_policy.tenant", "logical_vm_group_ids.#", "0"),
					resource.TestCheckNoResourceAttr("data.vcd_vm_placement_policy.tenant", "provider_vdc_id"),
				),
			},
		},
	})
	postTestChecks(t)
}

const testAccVcdVmPlacementPolicyInVdcTenant = `
data "vcd_vm_placement_policy" "tenant" {
  name     = "{{.PlacementPolicyName}}"
  vdc_id   = "{{.VdcId}}"
}
`

// TestAccVcdVmPlacementPolicyWithoutDescription checks that a VM Placement Policy without description specified in the HCL
// corresponds to a VM Placement Policy with an empty description in VCD.
func TestAccVcdVmPlacementPolicyWithoutDescription(t *testing.T) {
	preTestChecks(t)
	skipIfNotSysAdmin(t)
	if testConfig.VCD.ProviderVdc.Name == "" {
		t.Skip("Variable providerVdc.Name must be set to run VDC tests")
	}

	var params = StringMap{
		"PvdcName":   testConfig.VCD.NsxtProviderVdc.Name,
		"PolicyName": t.Name(),
		"VmGroup":    testConfig.VCD.NsxtProviderVdc.PlacementPolicyVmGroup,
	}
	testParamsNotEmpty(t, params)
	policyName := "vcd_vm_placement_policy." + params["PolicyName"].(string)
	configText := templateFill(testAccCheckVmPlacementPolicyWithoutDescription, params)

	debugPrintf("#[DEBUG] CONFIGURATION - creation: %s", configText)

	if vcdShortTest {
		t.Skip(acceptanceTestsSkipped)
		return
	}

	vcdClient := createTemporaryVCDConnection(true)
	if vcdClient == nil {
		t.Skip(acceptanceTestsSkipped)
	}
	vmPlacementPolicyDescription := "This is a system generated default compute policy auto assigned to this vDC."
	if vcdClient.Client.APIVCDMaxVersionIs("< 38.0") || vcdClient.Client.APIVCDMaxVersionIs("> 39.0") {
		vmPlacementPolicyDescription = ""
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccCheckComputePolicyDestroyed(t.Name(), "placement"),
		Steps: []resource.TestStep{
			{
				Config: configText,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(policyName, "description", vmPlacementPolicyDescription),
				),
			},
		},
	})
	postTestChecks(t)
}

const testAccCheckVmPlacementPolicyWithoutDescription = `
data "vcd_provider_vdc" "pvdc" {
  name = "{{.PvdcName}}"
}

data "vcd_vm_group" "vm-group" {
  name            = "{{.VmGroup}}"
  provider_vdc_id = data.vcd_provider_vdc.pvdc.id
}

resource "vcd_vm_placement_policy" "{{.PolicyName}}" {
  name            = "{{.PolicyName}}"
  provider_vdc_id = data.vcd_provider_vdc.pvdc.id
  vm_group_ids    = [ data.vcd_vm_group.vm-group.id ]
}
`
