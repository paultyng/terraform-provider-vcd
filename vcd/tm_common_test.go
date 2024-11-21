//go:build api || functional || catalog || vapp || network || extnetwork || org || query || vm || vdc || gateway || disk || binary || lb || lbServiceMonitor || lbServerPool || lbAppProfile || lbAppRule || lbVirtualServer || access_control || user || standaloneVm || search || auth || nsxt || role || alb || certificate || vdcGroup || ldap || rde || uiPlugin || providerVdc || cse || slz || multisite || tm || ALL

package vcd

import (
	"github.com/vmware/go-vcloud-director/v3/govcd"
	"testing"
)

// getVCenterHcl gets a vCenter data source as first returned parameter and its HCL reference as second one,
// only if a vCenter is already configured in TM. Otherwise, it returns a vCenter resource HCL as first returned parameter
// and its HCL reference as second one, only if "createVCenter=true" in the testing configuration
func getVCenterHcl(t *testing.T) (string, string) {
	vcdClient := createTemporaryVCDConnection(false)
	vc, err := vcdClient.GetVCenterByUrl(testConfig.Tm.VcenterUrl)
	if err == nil {
		return `
data "vcd_vcenter" "vc" {
  name = "` + vc.VSphereVCenter.Name + `"
}
`, "data.vcd_vcenter.vc"
	}
	if !govcd.ContainsNotFound(err) {
		t.Fatal(err)
		return "", ""
	}
	if !testConfig.Tm.CreateVcenter {
		t.Skip("vCenter is not configured and configuration is not allowed in config file")
		return "", ""
	}
	return `
resource "vcd_vcenter" "vc" {
  name                     = "` + t.Name() + `"
  url                      = "` + testConfig.Tm.VcenterUrl + `"
  auto_trust_certificate   = true
  refresh_vcenter_on_read  = true
  refresh_policies_on_read = true
  username                 = "` + testConfig.Tm.VcenterUsername + `"
  password                 = "` + testConfig.Tm.VcenterPassword + `"
  is_enabled               = true
}
`, "vcd_vcenter.vc"
}

// getNsxManagerHcl gets a NSX Manager data source as first returned parameter and its HCL reference as second one,
// only if a NSX Manager is already configured in TM. Otherwise, it returns a NSX Manager resource HCL as first returned parameter
// and its HCL reference as second one, only if "createNsxManager=true" in the testing configuration
func getNsxManagerHcl(t *testing.T) (string, string) {
	vcdClient := createTemporaryVCDConnection(false)
	nsxtManager, err := vcdClient.GetNsxtManagerOpenApiByUrl(testConfig.Tm.NsxtManagerUrl)
	if err == nil {
		return `
data "vcd_nsxt_manager" "nsx_manager" {
  name = "` + nsxtManager.NsxtManagerOpenApi.Name + `"
}
`, "data.vcd_nsxt_manager.nsx_manager"
	}
	if !govcd.ContainsNotFound(err) {
		t.Fatal(err)
		return "", ""
	}
	if !testConfig.Tm.CreateNsxtManager {
		t.Skip("NSX Manager is not configured and configuration is not allowed in config file")
		return "", ""
	}
	return `
resource "vcd_nsxt_manager" "nsx_manager" {
  name                   = "` + t.Name() + `"
  description            = "` + t.Name() + `"
  username               = "` + testConfig.Tm.NsxtManagerUsername + `"
  password               = "` + testConfig.Tm.NsxtManagerPassword + `"
  url                    = "` + testConfig.Tm.NsxtManagerUrl + `"
  network_provider_scope = ""
  auto_trust_certificate = true
}

`, "vcd_nsxt_manager.nsx_manager"
}

// getRegionHcl gets a Region data source as first returned parameter and its HCL reference as second one,
// only if a Region is already configured in TM. Otherwise, it returns a Region resource HCL as first returned parameter
// and its HCL reference as second one, only if "createRegion=true" in the testing configuration
func getRegionHcl(t *testing.T, vCenterHclRef, nsxManagerHclRef string) (string, string) {
	if testConfig.Tm.Region == "" {
		t.Fatalf("the property tm.region is required but it is not present in testing JSON")
	}
	vcdClient := createTemporaryVCDConnection(false)
	region, err := vcdClient.GetRegionByName(testConfig.Tm.Region)
	if err == nil {
		return `
data "vcd_tm_region" "region" {
  name = "` + region.Region.Name + `"
}
`, "data.vcd_tm_region.region"
	}
	if !govcd.ContainsNotFound(err) {
		t.Fatal(err)
		return "", ""
	}
	if !testConfig.Tm.CreateRegion {
		t.Skip("Region is not configured and configuration is not allowed in config file")
		return "", ""
	}
	return `
data "vcd_tm_supervisor" "supervisor" {
  name       = "` + testConfig.Tm.VcenterSupervisor + `"
  vcenter_id = ` + vCenterHclRef + `.id
  depends_on = [` + vCenterHclRef + `]
}

resource "vcd_tm_region" "region" {
  name                 = "` + testConfig.Tm.Region + `"
  description          = "` + testConfig.Tm.Region + `"
  is_enabled           = true
  nsx_manager_id       = ` + nsxManagerHclRef + `.id
  supervisor_ids       = [data.vcd_tm_supervisor.supervisor.id]
  storage_policy_names = ["` + testConfig.Tm.VcenterStorageProfile + `"]
}
`, "vcd_tm_region.region"
}

// getContentLibraryHcl gets a Content Library data source as first returned parameter and its HCL reference as second one,
// only if a Content Library is already configured in TM. Otherwise, it returns a Content Library resource HCL as first returned parameter
// and its HCL reference as second one
func getContentLibraryHcl(t *testing.T, regionHclRef string) (string, string) {
	if testConfig.Tm.ContentLibrary == "" {
		t.Fatalf("the property tm.contentLibrary is required but it is not present in testing JSON")
	}
	if testConfig.Tm.RegionStoragePolicy == "" {
		t.Fatalf("the property tm.regionStoragePolicy is required but it is not present in testing JSON")
	}
	vcdClient := createTemporaryVCDConnection(false)
	cl, err := vcdClient.GetContentLibraryByName(testConfig.Tm.ContentLibrary)
	if err == nil {
		return `
data "vcd_tm_content_library" "content_library" {
  name = "` + cl.ContentLibrary.Name + `"
}
`, "data.vcd_tm_content_library.content_library"
	}
	if !govcd.ContainsNotFound(err) {
		t.Fatal(err)
		return "", ""
	}
	return `
data "vcd_tm_region_storage_policy" "region_storage_policy" {
  region_id = ` + regionHclRef + `.id 
  name      = "` + testConfig.Tm.RegionStoragePolicy + `"
}

resource "vcd_tm_content_library" "content_library" {
  name                 = "` + testConfig.Tm.ContentLibrary + `"
  description          = "` + testConfig.Tm.ContentLibrary + `"
  storage_class_ids    = [data.vcd_tm_region_storage_policy.region_storage_policy.id]
}
`, "vcd_tm_content_library.content_library"
}
