module github.com/terraform-providers/terraform-provider-vcd/v2

go 1.13

require (
	github.com/davecgh/go-spew v1.1.1
	github.com/hashicorp/terraform-plugin-sdk v1.0.0
	github.com/vmware/go-vcloud-director/v2 v2.5.0-alpha.5
)

replace github.com/vmware/go-vcloud-director/v2 => ../go-vcloud-director
