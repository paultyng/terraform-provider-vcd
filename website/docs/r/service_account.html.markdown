---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_service_account"
sidebar_current: "docs-vcd-resource-service-account"
description: |-
  Provides a resource to manage Service Accounts. Service Accounts can have defined roles
  and act just like a VCD user. Service Accounts, when activated, provide one-time use
  access tokens for authentication to the VCD API, during which a new access token is generated.
---

# vcd\_service\_account 

Provides a resource to manage Service Accounts. Service Accounts can have defined roles
and act just like a VCD user. Service Accounts, when activated, provide one-time use
access tokens for authentication to the VCD API, during which a new access token is generated.
Explained in more detail [here][service-accounts]

Supported in provider *v3.10+* and VCD 10.4+.

## Example Usage 

```hcl
data "vcd_role" "vapp_author" {
  org  = "my-org"
  name = "vApp Author"
}

resource "vcd_service_account" "example_service" {
  org     = "my-org"
  name    = "example"
  role_id = data.vcd_role.vapp_author.id

  software_id      = "12345678-1234-1234-1234-1234567890ab"
  software_version = "1.0.0"
  uri              = "example.com"

  file_name = "example_service.json"

  active = true
}
```

-> After creation, the file can be used to authenticate the provider using the [`service_account_token_file`][provider-service-account-token-file] field.

## Argument Reference

The following arguments are supported:

* `org` - (Optional) The name of organization to use, optional if defined at provider level. Useful
  when connected as sysadmin working across different organisations.
* `name` - (Required) A unique name for the Service Account in an organisation.
* `role_id` - (Required) ID of a Role.
* `software_id` - (Required) UUID of the Service Account.
* `software_version` - (Optional) Version of the service using the Service Account
* `uri` - (Optional) URI of the service using the Service Account
* `active` - (Required) Status of the Service Account. Can be set to `false` and back to `true` if
  the access token was lost to get a new one.
* `file_name` - (Optional) Required only when `active` is set to `true`. Contains the access token
  that can be used for authenticating to VCD.
* `allow_token_file` - (Optional) If set to false, will output a warning about the service account file
  containing sensitive information.

## Importing

~> The current implementation of Terraform import can only import resources into the state.
It does not generate configuration. [More information.][docs-import]

An existing service account can be [imported][docs-import] into this resource via supplying
the full dot separated path. An example is below:

```
terraform import vcd_service_account.imported my-org.my-service-account 
```

[service-accounts]: https://blogs.vmware.com/cloudprovider/2022/07/cloud-director-service-accounts.html
[docs-import]: https://www.terraform.io/docs/import/
[provider-service-account-token-file]: /providers/vmware/vcd/latest/docs#service_account_token_file
