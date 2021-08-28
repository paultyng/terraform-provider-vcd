---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_edgegateway_settings"
sidebar_current: "docs-vcd-resource-edgegateway-settings"
description: |-
  Provides a VMware Cloud Director edge gateway global settings. This can be used to update global edge gateways settings related to firewall and load balancing.
---

# vcd\_edgegateway\_settings

Provides a resource that can update VMware Cloud Director edge gateway global settings either as System Administrator or as
Organization user.

The main use case of this resource is to allow both providers and tenants to change edge gateways global settings (such as
enabling load balancing or firewall) when the edge gateway was created outside of Terraform.
A second use case is when the provider creates the edge gateway using Terraform, and then delegates the tenant to change
some settings for further operations.

~> **Warning:** The edge gateway settings info is tied to an edge gateway. Thus, there could be only one instance per 
edge gateway. Using a different definition for the same edge gateway ID will result in a previous instance to be overwritten.

!> **Warning:** Using a `vcd_edgegateway` and a `vcd_edgegateway_settings` for the same entity does not work correctly,
as the main purpose of this resource is to handle general settings when the edge gateway was created outside of Terraform.
If users can create an edge gateway, they don't need `vcd_edgegateway_settings`, as they can set the same properties
directly during creation.

Supported in provider *v3.0+*

## Example Usage

```hcl
data "vcd_edgegateway" "egw" {
  name = "my-egw"
}

resource "vcd_edgegateway_settings" "egw-settings" {
  edge_gateway_id         = data.vcd_edgegateway.egw.id
  lb_enabled              = true
  lb_acceleration_enabled = true
  lb_logging_enabled      = true
  lb_loglevel             = "debug"

  fw_enabled                      = true
  fw_default_rule_logging_enabled = true
  fw_default_rule_action          = "accept"
}
```

-> **Tip:** Although this resource changes values in the edge gateway referenced as a data source, due to how Terraform works, the state
of the edge gateway doesn't get updated. To reconcile the state of the data source with the values modified in `vcd_edgegateway_settings`,
you need to run `terraform refresh` after `apply`.

=> **Note:** Although tenants can enable load balancing using this resource, they can't set the properties `lb_logging_enabled` and `lb_loglevel`.

## Argument Reference

The following arguments are supported:

* `org` - (Optional) The name of organization to which the VDC belongs. Optional if defined at provider level.
* `vdc` - (Optional) The name of VDC that owns the edge gateway. Optional if defined at provider level. 
* `edge_gateway_name` - (Optional) A unique name for the edge gateway. (Required if `edge_gateway_id` is not set)
* `edge_gateway_id` - (Optional) The edge gateway ID. (Required if `edge_gateway_name` is not set)
* `lb_enabled` - (Optional) Enable load balancing. Default is `false`.
* `lb_acceleration_enabled` - (Optional) Enable to configure the load balancer.
* `lb_logging_enabled` - (Optional) Enables the edge gateway load balancer to collect traffic logs.
Default is `false`. Note: **only System administrators can change this property**. It is ignored by API for Org users.
* `lb_loglevel` - (Optional) Choose the severity of events to be logged. One of `emergency`,
`alert`, `critical`, `error`, `warning`, `notice`, `info`, `debug`. Note: **only System administrators can change this property**. It is ignored by API for Org users.
* `fw_enabled` (Optional) Enable firewall. Default `true`.
* `fw_default_rule_logging_enabled` (Optional) Enable default firewall rule (last in the processing 
order) logging. Default `false`.
* `fw_default_rule_action` (Optional) Default firewall rule (last in the processing order) action.
One of `accept` or `deny`. Default `deny`.

## Importing

~> **Note:** The current implementation of Terraform import can only import resources into the state. It does not generate
configuration. [More information.][docs-import]

An existing edge gateway settings can be [imported][docs-import] into this resource via supplying its path. 

The path for this resource is made of org-name.vdc-name.edge-name

For example, using this structure, representing an edge gateway settings that was **not** created using Terraform:

```hcl
resource "vcd_edgegateway_settings" "tf-egw" {
  edge_gateway_name = "my-edge-gw"
}
```

You can import such resource into terraform state using one of the commands below

```
terraform import vcd_edgegateway_settings.tf-egw my-org.my-vdc.my-edge-gw

terraform import vcd_edgegateway_settings.tf-egw my-org.my-vdc.63ed92de-4001-450c-879f-deadbeef0123
```

* **Note 1**: the name to provide here is the name of the edge gateway, as this resource is tied to it.
* **Note 2**: the separator can be changed using `Provider.import_separator` or variable `VCD_IMPORT_SEPARATOR`
* **Note 3**: the identifier of the resource could be either the edge gateway name or the ID

[docs-import]:https://www.terraform.io/docs/import/

After importing, if you run `terraform plan` you will see the rest of the values and modify the script accordingly for 
further operations.
