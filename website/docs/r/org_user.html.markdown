---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_org_user"
sidebar_current: "docs-vcd-resource-org-user"
description: |-
  Provides a VMware Cloud Director Organization user. This can be used to create, update, and delete users.
---

# vcd\_org\_user

Provides a VMware Cloud Director Org User. This can be used to create, update, and delete users, including system and org administrators.

If we have the right permissions, we can create both organization and System users: to do so, we indicate "System" as organization name.

The users being created can be assigned either pre-existing roles or [custom roles](/docs/providers/vcd/r/role.html).

Supported in provider *v2.4+*

~> **Note:** Only `System Administrator` or `Org Administrator` users can create users.

## Example Usage 1 - A simple organization user

```hcl
# A simple user created with the minimum of properties
# uses the "password" field
resource "vcd_org_user" "my-org-admin" {
  org = "my-org"

  name        = "my-org-admin"
  description = "a new org admin"
  role        = "Organization Administrator"
  password    = "change-me"
}

# Another user, created by filling all the fields
# Uses the "password_file" field.
resource "vcd_org_user" "test_user_vapp_author" {
  org = "datacloud"

  name              = "test_user_vapp_author"
  password_file     = "pwd201907101300.txt"
  full_name         = "test user vapp author"
  description       = "Org user test_user_vapp_author"
  role              = "vApp Author"
  enabled           = true
  take_ownership    = true
  provider_type     = "INTEGRATED"
  stored_vm_quota   = 20
  deployed_vm_quota = 20
  instant_messaging = "@test_user_vapp_author"
  email_address     = "test_user_vapp_author@test.company.org"
}
```

## Example Usage 2 - A System user

```hcl
# A new system administrator
resource "vcd_org_user" "my-sys-admin" {
  org         = "System"
  name        = "my-sys-admin"
  description = "a new sys admin"
  role        = "System Administrator"
  password    = "change-me-soon"
}
```

## Example Usage 3 - A System user with custom role

```hcl
resource "vcd_role" "new-sys-role" {
  org         = "System"
  name        = "new-role"
  description = "new role from CLI"
  rights = [
    "Catalog: Add vApp from My Cloud",
    "Catalog: Edit Properties",
    "Catalog: View Private and Shared Catalogs",
    "Organization vDC Compute Policy: View",
    "vApp Template / Media: Edit",
    "vApp Template / Media: View",
  ]
}

resource "vcd_org_user" "test_sys_user" {
  org            = "System"
  name           = "custom-sys-user"
  password       = "mypass"
  role           = vcd_role.new-sys-role.name
  take_ownership = true
}
```


## Argument Reference

The following arguments are supported:

* `org` - (Optional) The name of organization to which the user belongs. Optional if defined at provider level. If we 
  want to create a user at provider level, use "System" as org name.
* `name` - (Required) A unique name for the user.
* `password` - (Optional, but required if `password_file` was not given) The user password. This value is never returned 
  on read. It is inspected on create and modify. To modify, fill with a different value. Note that if you remove the 
  password *on update*, Terraform will indicate that a change was occurring, but the empty password will be ignored by vCD.
* `password_file` (Optional, but required if `password` was not given). A text file containing the password. Recommended
  usage: after changing the password, run an apply again with the password blank.
  Using this property instead of `password` has the advantage that the sensitive data is not saved into Terraform state 
  file. The disadvantage is that a password change requires also changing the file name.
* `provider_type` - (Optional) Identity provider type for this user. One of: `INTEGRATED`, `SAML`, `OAUTH`. The default
   is `INTEGRATED`.
* `role` - (Required) The role of the user. Role names can be retrieved from the organization. Both built-in roles and
  custom built can be used. The roles normally available are:
    * `Organization Administrator`
    * `Catalog Author`
    * `vApp Author`
    * `vApp User`
    * `Console Access Only`
    * `Defer to Identity Provider`
* `full_name` - (Optional) The full name of the user.
* `description` - (Optional) An optional description of the user.
* `telephone` - (Optional) The Org User telephone number.
* `email_address` - (Optional) The Org User email address. Needs to be a properly formatted email address.
* `instant_messaging` - (Optional) The Org User instant messaging.
* `enabled` - (Optional) True if the user is enabled and can log in. The default is `true`.
* `is_group_role` - (Optional) True if this user has a group role. The default is `false`.
* `is_locked` - (Optional)aIf the user account has been locked due to too many invalid login attempts, the value will 
  change to true (only the system can lock the user). To unlock the user re-set this flag to false. 
* `take_ownership` - (Optional) Take ownership of user's objects on deletion.
* `deployed_vm_quota` - (Optional) Quota of vApps that this user can deploy. A value of 0 specifies an unlimited quota.
  The default is 10.
* `stored_vm_quota` - (Optional) Quota of vApps that this user can store. A value of 0 specifies an unlimited quota.
  The default is 10.

## Attribute Reference

The following attributes are exported on this resource:

* `id` - The ID of the Organization user

## Importing

~> **Note:** The current implementation of Terraform import can only import resources into the state. It does not generate
configuration. [More information.][docs-import]

An existing user can be [imported][docs-import] into this resource via supplying the full dot separated path for an
org user. For example, using this structure, representing an existing user that was **not** created using Terraform:

```hcl
resource "vcd_org_user" "my-org-admin" {
  org  = "my-org"
  name = "my-org-admin"
  role = "Organization Administrator"
}
```

You can import such user into terraform state using this command

```
terraform import vcd_org_user.my-org-admin my-org.my-org-admin
```

NOTE: the default separator (.) can be changed using Provider.import_separator or variable VCD_IMPORT_SEPARATOR

[docs-import]:https://www.terraform.io/docs/import/

The state (in `terraform.tfstate`) would look like this:

```json
{
  "version": 4,
  "terraform_version": "0.12.0",
  "serial": 1,
  "lineage": "f3fb8d07-8fe5-4fe3-3afe-c9050ffe68f6",
  "outputs": {},
  "resources": [
    {
      "mode": "managed",
      "type": "vcd_org_user",
      "name": "my-org-user",
      "provider": "provider.vcd",
      "instances": [
        {
          "schema_version": 0,
          "attributes": {
            "deployed_vm_quota": 50,
            "description": "This is my-org main user",
            "email_address": "my-org-admin@mycompany.com",
            "full_name": "My Org Admin",
            "id": "urn:vcloud:user:5fd69dfa-6bbe-40a6-9ee3-70448b6601ef",
            "instant_messaging": "@my_org_admin",
            "enabled": true,
            "is_group_role": false,
            "is_locked": false,
            "name": "my-org-user",
            "org": "my-org",
            "password": null,
            "password_file": null,
            "provider_type": "INTEGRATED",
            "role": "Organization Administrator",
            "stored_vm_quota": 50,
            "take_ownership": null,
            "telephone": "123-456-7890"
          }
        }
      ]
    }
  ]
}
```

After that, you can expand the configuration file and either update or delete the user as needed. Running `terraform plan`
at this stage will show the difference between the minimal configuration file and the user's stored properties.
