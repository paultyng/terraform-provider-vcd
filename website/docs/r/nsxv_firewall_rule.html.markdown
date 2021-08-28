---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_nsxv_firewall_rule"
sidebar_current: "docs-vcd-resource-nsxv-firewall-rule"
description: |-
  Provides a VMware Cloud Director firewall rule resource for advanced edge gateways (NSX-V). This can be
  used to create, modify, and delete firewall rules.
---

# vcd\_nsxv\_firewall\_rule

Provides a VMware Cloud Director firewall rule resource for advanced edge gateways (NSX-V). This can be
used to create, modify, and delete firewall rules.

~> **Note:** This resource requires advanced edge gateway (NSX-V).

## Example Usage 1 (Minimal input with dynamic edge gateway IP)

```hcl
data "vcd_edgegateway" "mygw" {
  org  = "my-org"
  vdc  = "my-vdc"
  name = "my-edge-gateway-name"
}

resource "vcd_nsxv_firewall_rule" "my-rule-1" {
  org          = "my-org"
  vdc          = "my-vdc"
  edge_gateway = "my-edge-gateway"

  source {
    ip_sets = [vcd_ipset.test-ipset2.name]
  }

  destination {
    ip_addresses = ["${data.vcd_edgegateway.mygw.default_external_network_ip}"]
  }

  service {
    protocol = "any"
  }
}
```


## Example Usage 2 (Multiple services)

```hcl
resource "vcd_nsxv_firewall_rule" "my-rule-1" {
  org          = "my-org"
  vdc          = "my-vdc"
  edge_gateway = "my-edge-gateway"

  source {
    ip_addresses       = ["any"]
    gateway_interfaces = ["internal"]
  }

  destination {
    ip_addresses = ["192.168.1.110"]
  }

  service {
    protocol = "icmp"
  }

  service {
    protocol = "tcp"
    port     = "443"
  }
}
```

## Example Usage 3 (Use exclusion in source)

```hcl
resource "vcd_nsxv_firewall_rule" "my-rule-1" {
  org          = "my-org"
  vdc          = "my-vdc"
  edge_gateway = "my-edge-gateway"

  source {
    exclude            = true
    gateway_interfaces = ["internal"]
  }

  destination {
    ip_addresses = ["any"]
  }

  service {
    protocol = "icmp"
  }
}
```

## Example Usage 4 (Deny rule using exclusion and priority set using above_rule_id)

```hcl
resource "vcd_nsxv_firewall_rule" "my-rule-1" {
  org          = "my-org"
  vdc          = "my-vdc"
  edge_gateway = "my-edge-gateway"

  logging_enabled = "true"
  action          = "deny"

  source {
    ip_addresses = ["30.10.10.0/24", "31.10.10.0/24"]
    org_networks = ["org-net-1", "org-net-2"]
  }

  destination {
    ip_addresses = ["any"]
  }

  service {
    protocol = "icmp"
  }
}

resource "vcd_nsxv_firewall_rule" "my-rule-2" {
  org          = "my-org"
  vdc          = "my-vdc"
  edge_gateway = "my-edge-gateway"

  # This attribute allows to ensure rule is inserted above the referred one
  # in rule processing engine
  above_rule_id = "${vcd_nsxv_firewall_rule.my-rule-1.id}"
  name          = "my-friendly-name"

  source {
    ip_addresses = ["30.10.10.0/24", "31.10.10.0/24"]
    org_networks = ["org-net-1", "org-net-2"]
  }

  destination {
    ip_addresses = ["any"]
  }

  service {
    protocol = "icmp"
  }
}
```


## Argument Reference

The following arguments are supported:

* `org` - (Optional) The name of organization to use, optional if defined at provider level. Useful
when connected as sysadmin working across different organisations.
* `vdc` - (Optional) The name of VDC to use, optional if defined at provider level.
* `edge_gateway` - (Required) The name of the edge gateway on which to apply the firewall rule.
* `action` - (Optional) Defines if the rule is set to `accept` or `deny` traffic. Default `accept`
* `enabled` - (Optional) Defines if the rule is enabaled. Default `true`.
* `logging_enabled` - (Optional) Defines if the logging for this rule is enabaled. Default `false`.
* `name` - (Optional) Free text name. Can be duplicate.
* `rule_tag` - (Optional) This can be used to specify user-controlled rule tag. If not specified,
it will report rule ID after creation. Must be between 65537-131072.
* `above_rule_id` - (Optional) This can be used to alter default rule placement order. By default
every rule is appended to the end of firewall rule list. When a value of another rule is set - this
rule will be placed above the specified rule.
* `source` - (Required) Exactly one block to define source criteria for firewall. See
[Endpoint](#endpoint) and example for usage details.
* `destination` - (Required) Exactly one block to define source criteria for firewall. See
[Endpoint](#endpoint) and example for usage details.
* `service` - (Required) One or more blocks to define protocol and port details. Use multiple blocks
if you want to define multiple port/protocol combinations for the same rule. See
[Service](#service) and example for usage details.


<a id="endpoint"></a>
## Endpoint (source or destination)

* `exclude` - (Optional) When the toggle exclusion is selected, the rule is applied to
traffic on all sources except for the locations you excluded. When the toggle exclusion is not
selected, the rule applies to traffic you specified. Default `false`. This
[example](#example-usage-3-use-exclusion-in-source-) uses it.
* `ip_addresses` - (Optional) A set of IP addresses, CIDRs or ranges. A keyword `any` is also
accepted as a parameter.
* `gateway_interfaces` - (Optional) A set of with either three keywords `vse` (UI names it as `any`), `internal`, `external` or an org network name. It automatically looks up vNic in the backend.
* `vm_ids` - (Optional) A set of `.id` fields of `vcd_vapp_vm` resources.
* `org_networks` - (Optional) A set of org network names.
* `ip_sets` - (Optional) A set of existing IP set names (either created manually or configured using `vcd_nsxv_ip_set` resource)


<a id="service"></a>
## Service

* `protocol` - (Required) One of `any`, `tcp`, `udp`, `icmp` to apply.
* `port` - (Optional) Port number or range separated by `-` for port number. Default 'any'.
* `source_port` - (Optional) Port number or range separated by `-` for port number. Default 'any'.

## Attribute Reference

The following additional attributes are exported:

* `rule_type` - Possible values - `user`, `internal_high`.

## Importing

~> **Note:** The current implementation of Terraform import can only import resources into the state.
It does not generate configuration. [More information.](https://www.terraform.io/docs/import/)



An existing firewall rule can be [imported][docs-import] into this resource
via supplying the full dot separated path for firewall rule. There are a few ways as per examples
below.

NOTE: The default separator (.) can be changed using Provider.import_separator or variable VCD_IMPORT_SEPARATOR

[docs-import]: https://www.terraform.io/docs/import/

!> **Warning:** The UI shows only firewall rule order numbers (not their real IDs). Real firewall
rules have IDs with larger integer numbers like `132730`. See below for possible options to use
import.


### Import by real firewall rule ID

```
terraform import vcd_nsxv_firewall_rule.imported my-org-name.my-org-vdc-name.my-edge-gw-name.my-firewall-rule-id
```

The above would import the application rule named `my-firewall-rule-id` that is defined on edge
gateway `my-edge-gw-name` which is configured in organization named `my-org-name` and vDC named
`my-org-vdc-name`.


### Import by firewall rule number as shown in the UI ("No." field)

```
terraform import vcd_nsxv_firewall_rule.imported my-org-name.my-org-vdc-name.my-edge-gw-name.ui-no.3
```

**Pay attention** to the specific format of firewall rule number `ui-no.3`. The `ui-no.` flags that
import must be performed by UI number of firewall rule rather than real ID.

### Listing real firewall rule IDs and their numbers

If you want to list the real IDs and firewall rule numbers there is a
special command **`terraform import vcd_nsxv_firewall_rule.imported list@my-org-name.my-org-vdc-name.my-edge-gw-name`**
where `my-org-name` is the organization used, `my-org-vdc-name` is vDC name and `my-edge-gw-name`
is edge gateway name. The output for this command should look similar to below one:

```shell
$ terraform import vcd_nsxv_firewall_rule.import list@my-org-name.my-org-vdc-name.my-edge-gw-name
vcd_nsxv_firewall_rule.import: Importing from ID "list@my-org-name.my-org-vdc-name.my-edge-gw-name"...
Retrieving all firewall rules
UI No   ID      Name                                    Action  Type
-----   --      ----                                    ------  ----
1       132589  firewall                                accept  internal_high
2       132730  My deny rule                            deny    user
3       132729  My accept rule                          accept  user
4       132588  default rule for ingress traffic        deny    default_policy

Error: Resource was not imported! Please use the above ID to format the command as:
terraform import vcd_nsxv_firewall_rule.resource-name org-name.vdc-name.edge-gw-name.firewall-rule-id
```

Now to import rule with UI ID 2 (real ID 132730) one could supply this command:

```shell
$ terraform import vcd_nsxv_firewall_rule.import my-org-name.my-org-vdc-name.my-edge-gw-name.132730
vcd_nsxv_firewall_rule.import: Importing from ID "my-org-name.my-org-vdc-name.my-edge-gw-name.132730"...
vcd_nsxv_firewall_rule.import: Import prepared!
  Prepared vcd_nsxv_firewall_rule for import
vcd_nsxv_firewall_rule.import: Refreshing state... [id=132730]

Import successful!

The resources that were imported are shown above. These resources are now in
your Terraform state and will henceforth be managed by Terraform.
```
