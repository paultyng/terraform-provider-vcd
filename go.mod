module github.com/terraform-providers/terraform-provider-vcd/v2

go 1.13

require (
	github.com/davecgh/go-spew v1.1.1
	github.com/hashicorp/terraform v0.12.8
	github.com/vmware/go-vcloud-director/v2 v2.4.0-alpha.10
)

replace github.com/vmware/go-vcloud-director/v2 => ../go-vcloud-director
