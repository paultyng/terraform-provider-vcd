package vcd

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	govcd "github.com/vmware/go-vcloud-director/govcd"
)

func TestAccVcdNetwork_Basic(t *testing.T) {
	var network govcd.OrgVDCNetwork
	generatedHrefRegexp := regexp.MustCompile("^https://")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVcdNetworkDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: fmt.Sprintf(testAccCheckVcdNetwork_basic, testOrg, testVDC, os.Getenv("VCD_EDGE_GATEWAY")),
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
				),
			},
		},
	})
}

func testAccCheckVcdNetworkExists(n string, network *govcd.OrgVDCNetwork) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No VAPP ID is set")
		}

		conn := testAccProvider.Meta().(*VCDClient)
		org, err := govcd.GetOrgByName(conn.VCDClient, testOrg)
		if err != nil || org == (govcd.Org{}) {
			return fmt.Errorf("Could not find test Org")
		}
		vdc, err := org.GetVdcByName(testVDC)
		if err != nil || vdc == (govcd.Vdc{}) {
			return fmt.Errorf("Could not find test Vdc")
		}
		resp, err := vdc.FindVDCNetwork(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("Network does not exist.")
		}

		*network = resp

		return nil
	}
}

func testAccCheckVcdNetworkDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*VCDClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "vcd_network" {
			continue
		}
		org, err := govcd.GetOrgByName(conn.VCDClient, testOrg)
		if err != nil || org == (govcd.Org{}) {
			return fmt.Errorf("Could not find test Org")
		}
		vdc, err := org.GetVdcByName(testVDC)
		if err != nil || vdc == (govcd.Vdc{}) {
			return fmt.Errorf("Could not find test Vdc")
		}
		_, err = vdc.FindVDCNetwork(rs.Primary.ID)

		if err == nil {
			return fmt.Errorf("Network still exists.")
		}

		return nil
	}

	return nil
}

func testAccCheckVcdNetworkAttributes(network *govcd.OrgVDCNetwork) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if network.OrgVDCNetwork.Name != "foonet" {
			return fmt.Errorf("Bad name: %s", network.OrgVDCNetwork.Name)
		}

		return nil
	}
}

const testAccCheckVcdNetwork_basic = `
resource "vcd_network" "foonet" {
	name = "foonet"
	org  = "%s"
	vdc  = "%s"
	edge_gateway = "%s"
	gateway = "10.10.102.1"
	static_ip_pool {
		start_address = "10.10.102.2"
		end_address = "10.10.102.254"
	}
}
`
