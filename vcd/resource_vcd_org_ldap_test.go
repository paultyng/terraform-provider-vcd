//go:build ldap || user || org || ALL || functional
// +build ldap user org ALL functional

package vcd

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func init() {
	testingTags["ldap"] = "resource_vcd_org_ldap_test.go"
}

func TestAccVcdOrgLdap(t *testing.T) {
	preTestChecks(t)
	if !usingSysAdmin() {
		t.Skip("TestAccVcdOrgLdap requires system admin privileges")
		return
	}
	if testConfig.Networking.LdapServer == "" {
		t.Skip("TestAccVcdOrgLdap requires a working LDAP server (set the IP in testConfig.Networking.LdapServer)")
		return
	}
	var orgName = testConfig.VCD.Org

	var params = StringMap{
		"OrgName":      orgName,
		"LdapServerIp": testConfig.Networking.LdapServer,
		"Tags":         "ldap org",
	}
	testParamsNotEmpty(t, params)

	configText := templateFill(testAccOrgLdap, params)

	params["FuncName"] = t.Name()
	if vcdShortTest {
		t.Skip(acceptanceTestsSkipped)
		return
	}
	debugPrintf("#[DEBUG] CONFIGURATION: %s\n", configText)

	ldapResourceDef := "vcd_org_ldap." + orgName
	// Note: don't run this test in parallel, as it would clash with TestAccVcdOrgGroup
	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccCheckOrgLdapDestroy(ldapResourceDef),
		Steps: []resource.TestStep{
			{
				Config: configText,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOrgLdapExists(ldapResourceDef),
					resource.TestCheckResourceAttr(ldapResourceDef, "name", orgName),
					resource.TestCheckResourceAttr(ldapResourceDef, "ldap_mode", "CUSTOM"),
					resource.TestCheckResourceAttr(ldapResourceDef, "custom_settings.0.server", testConfig.Networking.LdapServer),
					resource.TestCheckResourceAttr(ldapResourceDef, "custom_settings.0.authentication_method", "SIMPLE"),
					resource.TestCheckResourceAttr(ldapResourceDef, "custom_settings.0.connector_type", "OPEN_LDAP"),
					resource.TestCheckResourceAttr(ldapResourceDef, "custom_settings.0.user_attributes.0.object_class", "inetOrgPerson"),
					resource.TestCheckResourceAttr(ldapResourceDef, "custom_settings.0.group_attributes.0.object_class", "group"),
				),
			},
			{
				ResourceName:      ldapResourceDef,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: importStateIdTopHierarchy(orgName),
			},
		},
	})
	postTestChecks(t)
}

func testAccCheckOrgLdapExists(identifier string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[identifier]
		if !ok {
			return fmt.Errorf("not found: %s", identifier)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no Org ID is set")
		}

		conn := testAccProvider.Meta().(*VCDClient)

		adminOrg, err := conn.GetAdminOrgById(rs.Primary.ID)
		if err != nil {
			return err
		}
		config, err := adminOrg.GetLdapConfiguration()
		if err != nil {
			return err
		}
		if config.OrgLdapMode == "NONE" {
			return fmt.Errorf("resource %s not configured", identifier)
		}
		return nil
	}
}

func testAccCheckOrgLdapDestroy(identifier string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[identifier]
		if !ok {
			return fmt.Errorf("not found: %s", identifier)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no Org ID is set")
		}

		conn := testAccProvider.Meta().(*VCDClient)

		adminOrg, err := conn.GetAdminOrgById(rs.Primary.ID)
		if err != nil {
			return err
		}
		config, err := adminOrg.GetLdapConfiguration()
		if err != nil {
			return err
		}
		if config.OrgLdapMode != "NONE" {
			return fmt.Errorf("resource %s still configured", identifier)
		}
		return nil

	}
}

const testAccOrgLdap = `
resource "vcd_org_ldap" "{{.OrgName}}" {
  name      = "{{.OrgName}}"
  ldap_mode = "CUSTOM"
  custom_settings {
    server                  = "{{.LdapServerIp}}"
    port                    = 389
    is_ssl                  = false
    username                = "cn=admin,dc=planetexpress,dc=com"
    password                = "GoodNewsEveryone"
    authentication_method   = "SIMPLE"
    base_distinguished_name = "dc=planetexpress,dc=com"
    connector_type          = "OPEN_LDAP"
    user_attributes {
      object_class                = "inetOrgPerson"
      unique_identifier           = "uid"
      full_name                   = "cn"
      username                    = "uid"
      given_name                  = "givenName"
      surname                     = "sn"
      telephone                   = "telephoneNumber"
      group_membership_identifier = "dn"
      email                       = "mail"
    }
    group_attributes {
      name                        = "cn"
      object_class                = "group"
      membership                  = "member"
      unique_identifier           = "cn"
      group_membership_identifier = "dn"
    }
  }
  lifecycle {
    # password value does not get returned by GET
    ignore_changes = [custom_settings[0].password]
  }
}
`
