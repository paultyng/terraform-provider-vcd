//go:build tm || ALL || functional

package vcd

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccVcdTmContentLibraryItem(t *testing.T) {
	preTestChecks(t)

	skipIfNotSysAdmin(t)
	skipIfNotTm(t)

	vCenterHcl, vCenterHclRef := getVCenterHcl(t)
	nsxManagerHcl, nsxManagerHclRef := getNsxManagerHcl(t)
	regionHcl, regionHclRef := getRegionHcl(t, vCenterHclRef, nsxManagerHclRef)
	contentLibraryHcl, contentLibraryHclRef := getContentLibraryHcl(t, regionHclRef)

	var params = StringMap{
		"Name":              t.Name() + "6",
		"ContentLibraryRef": fmt.Sprintf("%s.id", contentLibraryHclRef),
		"OvaPath":           "../test-resources/test_vapp_template.ova",
		"Tags":              "tm",
	}
	testParamsNotEmpty(t, params)

	preRequisites := vCenterHcl + nsxManagerHcl + regionHcl + contentLibraryHcl

	configText1 := templateFill(preRequisites+testAccVcdTmContentLibraryItemStep1, params)
	params["FuncName"] = t.Name() + "-step2"
	configText2 := templateFill(preRequisites+testAccVcdTmContentLibraryItemStep2, params)

	debugPrintf("#[DEBUG] CONFIGURATION step1: %s\n", configText1)
	debugPrintf("#[DEBUG] CONFIGURATION step2: %s\n", configText2)
	if vcdShortTest {
		t.Skip(acceptanceTestsSkipped)
		return
	}

	resourceName := "vcd_tm_content_library_item.cli"

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: configText1,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", params["Name"].(string)),
					resource.TestCheckResourceAttr(resourceName, "description", params["Name"].(string)),
					resource.TestCheckResourceAttrPair(resourceName, "content_library_id", contentLibraryHclRef, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "creation_date"),
					resource.TestCheckResourceAttr(resourceName, "is_subscribed", "false"),
					resource.TestCheckResourceAttr(resourceName, "is_published", "false"),
					resource.TestCheckResourceAttrSet(resourceName, "image_identifier"),
					resource.TestMatchResourceAttr(resourceName, "owner_org_id", regexp.MustCompile("urn:vcloud:org:")),
					resource.TestCheckResourceAttr(resourceName, "status", "READY"),
					resource.TestCheckResourceAttr(resourceName, "last_successful_sync", ""),
					resource.TestCheckResourceAttr(resourceName, "version", "1"),
				),
			},
			{
				Config: configText2,
				Check: resource.ComposeTestCheckFunc(
					// file_path and upload_piece_size cannot be obtained during reads, that's why it does not appear in data source schema
					resourceFieldsEqual(resourceName, "data.vcd_tm_content_library_item.cli_ds", []string{"file_path", "upload_piece_size", "%"}),
				),
			},
			{
				ResourceName:            "vcd_tm_content_library_item.cli",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateId:           fmt.Sprintf("%s%s%s", testConfig.Tm.ContentLibrary, ImportSeparator, params["Name"].(string)),
				ImportStateVerifyIgnore: []string{"file_path", "upload_piece_size"}, // file_path and upload_piece_size cannot be obtained during imports, that's why it's Optional
			},
		},
	})

	postTestChecks(t)
}

const testAccVcdTmContentLibraryItemStep1 = `
resource "vcd_tm_content_library_item" "cli" {
  name               = "{{.Name}}"
  description        = "{{.Name}}"
  content_library_id = {{.ContentLibraryRef}}
  file_path          = "{{.OvaPath}}"
}
`

const testAccVcdTmContentLibraryItemStep2 = testAccVcdTmContentLibraryItemStep1 + `
data "vcd_tm_content_library_item" "cli_ds" {
  name               = vcd_tm_content_library_item.cli.name
  content_library_id = vcd_tm_content_library_item.cli.content_library_id
}
`
