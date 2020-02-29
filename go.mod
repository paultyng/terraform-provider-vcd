module github.com/terraform-providers/terraform-provider-vcd/v2

go 1.13

require (
	github.com/davecgh/go-spew v1.1.1
	github.com/hashicorp/go-version v1.2.0
	github.com/hashicorp/terraform-plugin-sdk v1.5.0
	github.com/vmware/go-vcloud-director/v2 v2.6.0-alpha.6
)
replace github.com/vmware/go-vcloud-director/v2 => ../go-vcloud-director
