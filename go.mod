module github.com/terraform-providers/terraform-provider-vcd/v2

go 1.12

require (
	github.com/hashicorp/terraform v0.12.0
	github.com/vmware/go-vcloud-director/v2 v2.3.0-beta.1
)

replace github.com/vmware/go-vcloud-director/v2 => github.com/vbauzysvmware/go-vcloud-director/v2 v2.0.0-20190715111302-2c35536e6def
