module github.com/vmware/terraform-provider-vcd/v2

go 1.13

require (
	github.com/aws/aws-sdk-go v1.30.12 // indirect
	github.com/hashicorp/go-getter v1.4.2-0.20200106182914-9813cbd4eb02 // indirect
	github.com/hashicorp/go-version v1.2.0
	github.com/hashicorp/hcl/v2 v2.3.0 // indirect
	github.com/hashicorp/terraform-config-inspect v0.0.0-20191212124732-c6ae6269b9d7 // indirect
	github.com/hashicorp/terraform-plugin-sdk v1.8.0
	github.com/kr/pretty v0.2.0
	github.com/vmware/go-vcloud-director/v2 v2.9.0-alpha.1
	golang.org/x/crypto v0.0.0-20200510223506-06a226fb4e37 // indirect
)

// replace github.com/vmware/go-vcloud-director/v2 => ../go-vcloud-director
replace github.com/vmware/go-vcloud-director/v2 => github.com/dataclouder/go-vcloud-director/v2 v2.5.0-alpha.4.0.20200817144118-26a93722ee02
