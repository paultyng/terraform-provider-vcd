module github.com/terraform-providers/terraform-provider-vcd/v2

go 1.13

require (
	github.com/hashicorp/go-version v1.2.0
	github.com/hashicorp/terraform-plugin-sdk v1.8.0
	github.com/vmware/go-vcloud-director/v2 v2.8.0-alpha.5
)

replace github.com/vmware/go-vcloud-director/v2 => github.com/Didainius/go-vcloud-director/v2 v2.7.1-0.20200518073933-6a7a60237d90
