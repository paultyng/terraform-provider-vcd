//go:build functional || network || extnetwork || nsxt || ALL
// +build functional network extnetwork nsxt ALL

package vcd

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/vmware/go-vcloud-director/v2/govcd"

	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccVcdExternalNetworkV2NsxtVrf(t *testing.T) {
	preTestChecks(t)
	if vcdShortTest {
		t.Skip(acceptanceTestsSkipped)
		return
	}
	// As of 10.1.2 release it is not officially supported (support only introduced in 10.2.0) therefore skipping this
	// tests for 10.1.X. 10.1.1 allowed to create it, but 10.1.2 introduced a validator and throws error.
	client := createTemporaryVCDConnection()
	if client.Client.APIVCDMaxVersionIs("< 35") {
		t.Skip("NSX-T VRF-Lite backed external networks are officially supported only in 10.2.0+")
	}
	testAccVcdExternalNetworkV2Nsxt(t, testConfig.Nsxt.Tier0routerVrf)
	postTestChecks(t)
}

func TestAccVcdExternalNetworkV2Nsxt(t *testing.T) {
	preTestChecks(t)
	testAccVcdExternalNetworkV2Nsxt(t, testConfig.Nsxt.Tier0router)
	postTestChecks(t)
}

func testAccVcdExternalNetworkV2Nsxt(t *testing.T, nsxtTier0Router string) {

	if !usingSysAdmin() {
		t.Skip(t.Name() + " requires system admin privileges")
		return
	}

	skipNoNsxtConfiguration(t)
	vcdClient := createTemporaryVCDConnection()
	if vcdClient.Client.APIVCDMaxVersionIs("< 33.0") {
		t.Skip(t.Name() + " requires at least API v33.0 (vCD 10+)")
	}

	startAddress := "192.168.30.51"
	endAddress := "192.168.30.62"
	description := "Test External Network"
	var params = StringMap{
		"NsxtManager":         testConfig.Nsxt.Manager,
		"NsxtTier0Router":     nsxtTier0Router,
		"ExternalNetworkName": t.Name(),
		"Type":                testConfig.Networking.ExternalNetworkPortGroupType,
		"PortGroup":           testConfig.Networking.ExternalNetworkPortGroup,
		"Vcenter":             testConfig.Networking.Vcenter,
		"StartAddress":        startAddress,
		"EndAddress":          endAddress,
		"Description":         description,
		"Gateway":             "192.168.30.49",
		"Netmask":             "24",
		"Tags":                "network extnetwork nsxt",
	}

	params["FuncName"] = t.Name()
	configText := templateFill(testAccCheckVcdExternalNetworkV2Nsxt, params)
	debugPrintf("#[DEBUG] CONFIGURATION: %s", configText)

	params["FuncName"] = t.Name() + "step1"
	configText1 := templateFill(testAccCheckVcdExternalNetworkV2NsxtStep1, params)
	debugPrintf("#[DEBUG] CONFIGURATION: %s", configText1)

	if vcdShortTest {
		t.Skip(acceptanceTestsSkipped)
		return
	}
	resourceName := "vcd_external_network_v2.ext-net-nsxt"
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccCheckExternalNetworkDestroyV2(t.Name()),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: configText,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", t.Name()),
					resource.TestCheckResourceAttr(resourceName, "description", description),
					resource.TestCheckResourceAttr(resourceName, "vsphere_network.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "nsxt_network.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "ip_scope.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "ip_scope.*", map[string]string{
						"dns1":          "",
						"dns2":          "",
						"dns_suffix":    "",
						"enabled":       "false",
						"gateway":       "192.168.30.49",
						"prefix_length": "24",
					}),

					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "ip_scope.*.static_ip_pool.*", map[string]string{
						"start_address": "192.168.30.51",
						"end_address":   "192.168.30.62",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "ip_scope.*", map[string]string{
						"dns1":          "",
						"dns2":          "",
						"dns_suffix":    "",
						"enabled":       "true",
						"gateway":       "14.14.14.1",
						"prefix_length": "24",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "ip_scope.*.static_ip_pool.*", map[string]string{
						"start_address": "14.14.14.20",
						"end_address":   "14.14.14.25",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "ip_scope.*.static_ip_pool.*", map[string]string{
						"start_address": "14.14.14.10",
						"end_address":   "14.14.14.15",
					}),
					resource.TestCheckResourceAttr(resourceName, "nsxt_network.#", "1"),
					testCheckMatchOutput("nsxt-manager", regexp.MustCompile("^urn:vcloud:nsxtmanager:.*")),
					testCheckOutputNonEmpty("nsxt-tier0-router"), // Match any non empty string
				),
			},
			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: importStateIdTopHierarchy(t.Name()),
			},
			resource.TestStep{
				Config: configText1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", t.Name()),
					resource.TestCheckResourceAttr(resourceName, "description", description),
					resource.TestCheckResourceAttr(resourceName, "vsphere_network.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "nsxt_network.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "ip_scope.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "ip_scope.*", map[string]string{
						"dns1":          "",
						"dns2":          "",
						"dns_suffix":    "",
						"enabled":       "true",
						"gateway":       "192.168.30.49",
						"prefix_length": "24",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "ip_scope.*.static_ip_pool.*", map[string]string{
						"start_address": "192.168.30.51",
						"end_address":   "192.168.30.62",
					}),
					resource.TestCheckResourceAttr(resourceName, "nsxt_network.#", "1"),
					testCheckMatchOutput("nsxt-manager", regexp.MustCompile("^urn:vcloud:nsxtmanager:.*")),
					testCheckOutputNonEmpty("nsxt-tier0-router"), // Match any non empty string
				),
			},
		},
	})
}

const testAccCheckVcdExternalNetworkV2NsxtDS = `
data "vcd_nsxt_manager" "main" {
  name = "{{.NsxtManager}}"
}

data "vcd_nsxt_tier0_router" "router" {
  name            = "{{.NsxtTier0Router}}"
  nsxt_manager_id = data.vcd_nsxt_manager.main.id
}

`

const testAccCheckVcdExternalNetworkV2Nsxt = testAccCheckVcdExternalNetworkV2NsxtDS + `
resource "vcd_external_network_v2" "ext-net-nsxt" {
  name        = "{{.ExternalNetworkName}}"
  description = "{{.Description}}"

  nsxt_network {
    nsxt_manager_id      = data.vcd_nsxt_manager.main.id
    nsxt_tier0_router_id = data.vcd_nsxt_tier0_router.router.id
  }

  ip_scope {
    enabled       = false
    gateway       = "{{.Gateway}}"
    prefix_length = "{{.Netmask}}"

    static_ip_pool {
      start_address = "{{.StartAddress}}"
      end_address   = "{{.EndAddress}}"
    }
  }

  ip_scope {
    gateway       = "14.14.14.1"
    prefix_length = "24"

    static_ip_pool {
      start_address = "14.14.14.10"
      end_address   = "14.14.14.15"
    }
    
    static_ip_pool {
      start_address = "14.14.14.20"
      end_address   = "14.14.14.25"
    }
  }
}

output "nsxt-manager" {
  value = tolist(vcd_external_network_v2.ext-net-nsxt.nsxt_network)[0].nsxt_manager_id
}

output "nsxt-tier0-router" {
  value = tolist(vcd_external_network_v2.ext-net-nsxt.nsxt_network)[0].nsxt_tier0_router_id
}
`
const testAccCheckVcdExternalNetworkV2NsxtStep1 = testAccCheckVcdExternalNetworkV2NsxtDS + `
# skip-binary-test: only for updates
resource "vcd_external_network_v2" "ext-net-nsxt" {
  name        = "{{.ExternalNetworkName}}"
  description = "{{.Description}}"

  nsxt_network {
    nsxt_manager_id      = data.vcd_nsxt_manager.main.id
    nsxt_tier0_router_id = data.vcd_nsxt_tier0_router.router.id
  }

  ip_scope {
    enabled       = true
    gateway       = "{{.Gateway}}"
    prefix_length = "{{.Netmask}}"

    static_ip_pool {
      start_address = "{{.StartAddress}}"
      end_address   = "{{.EndAddress}}"
    }
  }
}

output "nsxt-manager" {
  value = tolist(vcd_external_network_v2.ext-net-nsxt.nsxt_network)[0].nsxt_manager_id
}

output "nsxt-tier0-router" {
  value = tolist(vcd_external_network_v2.ext-net-nsxt.nsxt_network)[0].nsxt_tier0_router_id
}
`

func TestAccVcdExternalNetworkV2Nsxv(t *testing.T) {
	preTestChecks(t)
	if vcdShortTest {
		t.Skip(acceptanceTestsSkipped)
		return
	}
	if !usingSysAdmin() {
		t.Skip(t.Name() + " requires system admin privileges")
		return
	}

	vcdClient := createTemporaryVCDConnection()
	if vcdClient.Client.APIVCDMaxVersionIs("< 33.0") {
		t.Skip(t.Name() + " requires at least API v33.0 (vCD 10+)")
	}

	description := "Test External Network"
	var params = StringMap{
		"ExternalNetworkName": t.Name(),
		"Type":                testConfig.Networking.ExternalNetworkPortGroupType,
		"PortGroup":           testConfig.Networking.ExternalNetworkPortGroup,
		"Vcenter":             testConfig.Networking.Vcenter,
		"StartAddress":        "192.168.30.51",
		"EndAddress":          "192.168.30.62",
		"Description":         description,
		"Gateway":             "192.168.30.49",
		"Netmask":             "24",
		"Dns1":                "192.168.0.164",
		"Dns2":                "192.168.0.196",
		"Tags":                "network extnetwork nsxt",
	}

	configText := templateFill(testAccCheckVcdExternalNetworkV2Nsxv, params)
	params["FuncName"] = t.Name() + "step1"
	configText1 := templateFill(testAccCheckVcdExternalNetworkV2NsxvUpdate, params)

	debugPrintf("#[DEBUG] CONFIGURATION: %s", configText)
	debugPrintf("#[DEBUG] CONFIGURATION: %s", configText1)

	resourceName := "vcd_external_network_v2.ext-net-nsxv"
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccCheckExternalNetworkDestroyV2(t.Name()),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: configText,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", t.Name()),
					resource.TestCheckResourceAttr(resourceName, "description", description),
					resource.TestCheckResourceAttr(resourceName, "ip_scope.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "ip_scope.*", map[string]string{
						"dns1":          "192.168.0.164",
						"dns2":          "192.168.0.196",
						"dns_suffix":    "company.biz",
						"enabled":       "true",
						"gateway":       "192.168.30.49",
						"prefix_length": "24",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "ip_scope.0.static_ip_pool.*", map[string]string{
						"start_address": "192.168.30.51",
						"end_address":   "192.168.30.62",
					}),
					resource.TestCheckResourceAttr(resourceName, "ip_scope.0.static_ip_pool.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "nsxt_network.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "vsphere_network.#", "1"),
					testCheckOutputNonEmpty("vcenter-id"),   // Match any non empty string
					testCheckOutputNonEmpty("portgroup-id"), // Match any non empty string
				),
			},
			resource.TestStep{
				Config: configText1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", t.Name()),
					resource.TestCheckResourceAttr(resourceName, "description", description),
					resource.TestCheckResourceAttr(resourceName, "ip_scope.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "ip_scope.*", map[string]string{
						"dns1":          "192.168.0.164",
						"dns2":          "192.168.0.196",
						"dns_suffix":    "company.biz",
						"enabled":       "false",
						"gateway":       "192.168.30.49",
						"prefix_length": "24",
					}),
					resource.TestCheckResourceAttr(resourceName, "ip_scope.0.static_ip_pool.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "ip_scope.*.static_ip_pool.*", map[string]string{
						"start_address": "192.168.30.51",
						"end_address":   "192.168.30.62",
					}),

					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "ip_scope.*", map[string]string{
						"dns1":          "8.8.8.8",
						"dns2":          "8.8.4.4",
						"dns_suffix":    "asd.biz",
						"enabled":       "true",
						"gateway":       "88.88.88.1",
						"prefix_length": "24",
					}),
					resource.TestCheckResourceAttr(resourceName, "ip_scope.0.static_ip_pool.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "ip_scope.*.static_ip_pool.*", map[string]string{
						"start_address": "88.88.88.10",
						"end_address":   "88.88.88.100",
					}),
					resource.TestCheckResourceAttr(resourceName, "nsxt_network.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "vsphere_network.#", "1"),
					testCheckMatchOutput("vcenter-id", regexp.MustCompile("^urn:vcloud:vimserver:.*")),
					testCheckOutputNonEmpty("portgroup-id"), // Match any non empty string because IDs may differ
				),
			},
			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: importStateIdTopHierarchy(t.Name()),
			},
		},
	})
	postTestChecks(t)
}

const testAccCheckVcdExternalNetworkV2NsxvDs = `
data "vcd_vcenter" "vc" {
  name = "{{.Vcenter}}"
}

data "vcd_portgroup" "sw" {
  name = "{{.PortGroup}}"
  type = "{{.Type}}"
}

`

const testAccCheckVcdExternalNetworkV2Nsxv = testAccCheckVcdExternalNetworkV2NsxvDs + `
resource "vcd_external_network_v2" "ext-net-nsxv" {
  name        = "{{.ExternalNetworkName}}"
  description = "{{.Description}}"

  vsphere_network {
    vcenter_id     = data.vcd_vcenter.vc.id
    portgroup_id   = data.vcd_portgroup.sw.id
  }

  ip_scope {
    gateway       = "{{.Gateway}}"
    prefix_length = "{{.Netmask}}"
    dns1          = "{{.Dns1}}"
    dns2          = "{{.Dns2}}"
    dns_suffix    = "company.biz"

    static_ip_pool {
      start_address = "{{.StartAddress}}"
      end_address   = "{{.EndAddress}}"
    }
  }
}

output "vcenter-id" {
  value = tolist(vcd_external_network_v2.ext-net-nsxv.vsphere_network)[0].vcenter_id
}

output "portgroup-id" {
  value = tolist(vcd_external_network_v2.ext-net-nsxv.vsphere_network)[0].portgroup_id
}
`

const testAccCheckVcdExternalNetworkV2NsxvUpdate = testAccCheckVcdExternalNetworkV2NsxvDs + `
# skip-binary-test: only for updates
resource "vcd_external_network_v2" "ext-net-nsxv" {
  name        = "{{.ExternalNetworkName}}"
  description = "{{.Description}}"

  vsphere_network {
    vcenter_id     = data.vcd_vcenter.vc.id
    portgroup_id   = data.vcd_portgroup.sw.id
  }

  ip_scope {
    enabled       = false
    gateway       = "{{.Gateway}}"
    prefix_length = "{{.Netmask}}"
    dns1          = "{{.Dns1}}"
    dns2          = "{{.Dns2}}"
    dns_suffix    = "company.biz"

    static_ip_pool {
      start_address = "{{.StartAddress}}"
      end_address   = "{{.EndAddress}}"
    }
  }

  ip_scope {
    gateway       = "88.88.88.1"
    prefix_length = "24"
    dns1          = "8.8.8.8"
    dns2          = "8.8.4.4"
    dns_suffix    = "asd.biz"

    static_ip_pool {
      start_address = "88.88.88.10"
      end_address   = "88.88.88.100"
    }
  }
}

output "vcenter-id" {
  value = tolist(vcd_external_network_v2.ext-net-nsxv.vsphere_network)[0].vcenter_id
}

output "portgroup-id" {
  value = tolist(vcd_external_network_v2.ext-net-nsxv.vsphere_network)[0].portgroup_id
}
`

func testAccCheckExternalNetworkDestroyV2(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "vcd_external_network_v2" && rs.Primary.Attributes["name"] != name {
				continue
			}

			conn := testAccProvider.Meta().(*VCDClient)
			_, err := govcd.GetExternalNetworkV2ByName(conn.VCDClient, rs.Primary.ID)
			if err == nil {
				return fmt.Errorf("external network v2 %s still exists", rs.Primary.ID)
			}
		}

		return nil
	}
}
