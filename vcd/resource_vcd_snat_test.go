package vcd

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	govcd "github.com/ukcloud/govcloudair"
	"regexp"
)

func TestAccVcdSNAT_Basic(t *testing.T) {
	if v := os.Getenv("VCD_EXTERNAL_IP"); v == "" {
		t.Skip("Environment variable VCD_EXTERNAL_IP must be set to run SNAT tests")
		return
	}

	var e govcd.EdgeGateway

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVcdSNATDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckVcdSnat_basic, os.Getenv("VCD_EDGE_GATEWAY"), os.Getenv("VCD_EXTERNAL_IP")),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVcdSNATExists("vcd_snat.bar", &e),
					resource.TestCheckResourceAttr(
						"vcd_snat.bar", "external_ip", os.Getenv("VCD_EXTERNAL_IP")),
					resource.TestCheckResourceAttr(
						"vcd_snat.bar", "internal_ip", "10.10.102.0/24"),
				),
			},
		},
	})
}

func TestAccVcdSNAT_network(t *testing.T) {
	if v := os.Getenv("VCD_EXTERNAL_IP"); v == "" {
		t.Skip("Environment variable VCD_EXTERNAL_IP must be set to run DNAT tests")
		return
	}

	var network govcd.OrgVDCNetwork
	generatedHrefRegexp := regexp.MustCompile("^https://")

	var e govcd.EdgeGateway

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVcdSNATDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckVcdSnat_network, os.Getenv("VCD_EDGE_GATEWAY"), os.Getenv("VCD_EDGE_GATEWAY"), "foonet", os.Getenv("VCD_EXTERNAL_IP")),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVcdNetworkExists("vcd_network.foonet", &network),
					testAccCheckVcdNetworkAttributes(&network),
					resource.TestCheckResourceAttr(
						"vcd_network.foonet", "name", "foonet"),
					resource.TestCheckResourceAttr(
						"vcd_network.foonet", "static_ip_pool.#", "1"),
					resource.TestCheckResourceAttr(
						"vcd_network.foonet", "gateway", "10.10.102.1"),
					resource.TestMatchResourceAttr(
						"vcd_network.foonet", "href", generatedHrefRegexp),
					testAccCheckVcdSNATExists("vcd_snat.bar", &e),
					resource.TestCheckResourceAttr(
						"vcd_snat.bar", "external_ip", os.Getenv("VCD_EXTERNAL_IP")),
					resource.TestCheckResourceAttr(
						"vcd_snat.bar", "internal_ip", "10.10.102.0/24"),
				),
			},
		},
	})
}

func testAccCheckVcdSNATExists(n string, gateway *govcd.EdgeGateway) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		//return fmt.Errorf("Check this: %#v", rs.Primary)

		if rs.Primary.ID == "" {
			return fmt.Errorf("No SNAT ID is set")
		}

		conn := testAccProvider.Meta().(*VCDClient)

		gatewayName := rs.Primary.Attributes["edge_gateway"]
		edgeGateway, err := conn.OrgVdc.FindEdgeGateway(gatewayName)

		if err != nil {
			return fmt.Errorf("Could not find edge gateway")
		}

		var found bool
		for _, v := range edgeGateway.EdgeGateway.Configuration.EdgeGatewayServiceConfiguration.NatService.NatRule {
			if v.RuleType == "SNAT" &&
				v.GatewayNatRule.OriginalIP == "10.10.102.0/24" &&
				v.GatewayNatRule.OriginalPort == "" &&
				v.GatewayNatRule.TranslatedIP == os.Getenv("VCD_EXTERNAL_IP") {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("SNAT rule was not found")
		}

		*gateway = edgeGateway

		return nil
	}
}

func testAccCheckVcdSNATDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*VCDClient)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "vcd_snat" {
			continue
		}

		gatewayName := rs.Primary.Attributes["edge_gateway"]
		edgeGateway, err := conn.OrgVdc.FindEdgeGateway(gatewayName)

		if err != nil {
			return fmt.Errorf("Could not find edge gateway")
		}

		var found bool
		for _, v := range edgeGateway.EdgeGateway.Configuration.EdgeGatewayServiceConfiguration.NatService.NatRule {
			if v.RuleType == "SNAT" &&
				v.GatewayNatRule.OriginalIP == "10.10.102.0/24" &&
				v.GatewayNatRule.OriginalPort == "" &&
				v.GatewayNatRule.TranslatedIP == os.Getenv("VCD_EXTERNAL_IP") {
				found = true
			}
		}

		if found {
			return fmt.Errorf("SNAT rule still exists.")
		}
	}

	return nil
}

const testAccCheckVcdSnat_basic = `
resource "vcd_snat" "bar" {
	edge_gateway = "%s"
	external_ip = "%s"
	internal_ip = "10.10.102.0/24"
}
`
const testAccCheckVcdSnat_network = `
resource "vcd_network" "foonet" {
	name = "foonet"
	edge_gateway = "%s"
	gateway = "10.10.102.1"
	static_ip_pool {
		start_address = "10.10.102.2"
		end_address = "10.10.102.254"
	}
}

resource "vcd_snat" "bar" {
	depends_on = ["vcd_network.foonet"]
	edge_gateway = "%s"
	network_name = "%s"
	external_ip = "%s"
	internal_ip = "10.10.102.0/24"
}
`
