---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_nsxt_nat_rule"
sidebar_current: "docs-vcd-resource-nsxt-nat-rule"
description: |-
  Provides a resource to manage NSX-T NAT rules. To change the source IP address from a private to a
  public IP address, you create a source NAT (SNAT) rule. To change the destination IP address from 
  a public to a private IP address, you create a destination NAT (DNAT) rule.
---

# vcd\_nsxt\_nat\_rule

Supported in provider *v3.3+* and VCD 10.1+ with NSX-T backed VDCs.

Provides a resource to manage NSX-T NAT rules. To change the source IP address from a private to a
public IP address, you create a source NAT (SNAT) rule. To change the destination IP address from 
a public to a private IP address, you create a destination NAT (DNAT) rule.

-> When you configure a SNAT or a DNAT rule on an Edge Gateway in the VMware Cloud Director
environment, you always configure the rule from the perspective of your organization VDC.

## Example Usage 1 (SNAT rule translates to primary Edge Gateway IP for traffic going from 11.11.11.0/24 to 8.8.8.8)

```hcl
resource "vcd_nsxt_nat_rule" "snat" {
  org = "dainius"
  vdc = "nsxt-vdc-dainius"

  edge_gateway_id = data.vcd_nsxt_edgegateway.existing.id

  name        = "SNAT rule"
  rule_type   = "SNAT"
  description = "description"

  # Using primary_ip from edge gateway
  external_address         = tolist(data.vcd_nsxt_edgegateway.existing.subnet)[0].primary_ip
  internal_address         = "11.11.11.0/24"
  snat_destination_address = "8.8.8.8"
  logging                  = true
}
```

## Example Usage 2 (Prevent SNAT for internal addresses in subnet 11.11.11.0/24)
```hcl
resource "vcd_nsxt_nat_rule" "no-snat" {
  org = "dainius"
  vdc = "nsxt-vdc-dainius"

  edge_gateway_id = data.vcd_nsxt_edgegateway.existing.id

  name        = "test-no-snat-rule"
  rule_type   = "NO_SNAT"
  description = "description"

  internal_address = "11.11.11.0/24"
}
```

## Example Usage 3 (DNAT rule translates Edge Gateway primary IP to internal IP 11.11.11.2)
```hcl
resource "vcd_nsxt_nat_rule" "dnat" {
  org = "my-org"
  vdc = "nsxt-vdc"

  edge_gateway_id = data.vcd_nsxt_edgegateway.existing.id

  name        = "test-dnat-rule"
  rule_type   = "DNAT"
  description = "description"

  # Using primary_ip from edge gateway
  external_address = tolist(data.vcd_nsxt_edgegateway.existing.subnet)[0].primary_ip
  internal_address = "11.11.11.2"
  logging          = true
}
```

## Example Usage 4 (No DNAT rule)
```hcl
resource "vcd_nsxt_nat_rule" "no-dnat" {
  org = "my-org"
  vdc = "nsxt-vdc"

  edge_gateway_id = data.vcd_nsxt_edgegateway.existing.id

  name      = "test-no-dnat-rule"
  rule_type = "NO_DNAT"


  # Using primary_ip from edge gateway
  external_address   = tolist(data.vcd_nsxt_edgegateway.existing.subnet)[0].primary_ip
  dnat_external_port = 7777
}
```

## Example Usage 5 (Reflexive NAT rule also known as Stateless NAT)
```hcl
resource "vcd_nsxt_nat_rule" "reflexive" {
  org = "my-org"
  vdc = "nsxt-vdc"

  edge_gateway_id = data.vcd_nsxt_edgegateway.existing.id

  name      = "test-reflexive"
  rule_type = "REFLEXIVE"


  # Using primary_ip from edge gateway
  external_address = tolist(data.vcd_nsxt_edgegateway.existing.subnet)[0].primary_ip
  internal_address = "11.11.11.2"
}
```

## Argument Reference

The following arguments are supported:

* `org` - (Optional) The name of organization to use, optional if defined at provider level. Useful
  when connected as sysadmin working across different organisations.
* `vdc` - (Optional) The name of VDC to use, optional if defined at provider level.
* `edge_gateway_id` - (Required) The ID of the edge gateway (NSX-T only). Can be looked up using
  `vcd_nsxt_edgegateway` data source
* `name` - (Required) A name for NAT rule
* `description` - (Optional) An optional description of the NAT rule
* `enabled` (Optional) - Enables or disables NAT rule (default `true`)
* `rule_type` - (Required) One of `DNAT`, `NO_DNAT`, `SNAT`, `NO_SNAT`, `REFLEXIVE`
  * `DNAT` rule translates the external IP to an internal IP and is used for inbound traffic
  * `NO_DNAT` prevents external IP translation 
  * `SNAT` translates an internal IP to an external IP and is used for outbound traffic
  * `NO_SNAT` prevents internal IP translation
  * `REFLEXIVE` (VCD 10.3+)  is also known as Stateless NAT. This translates an internal IP to an external IP and vice 
    versa. The number of internal addresses should be exactly the same as that of external addresses.
* `external_address` (Optional) The external address for the NAT Rule. This must be supplied as a single IP or Network
  CIDR. For a `DNAT` rule, this is the external facing IP Address for incoming traffic. For an `SNAT` rule, this is the 
  external facing IP Address for outgoing traffic. These IPs are typically allocated/suballocated IP Addresses on the 
  Edge Gateway. For a `REFLEXIVE` rule, these are the external facing IPs.
* `internal_address` (Optional) The internal address for the NAT Rule. This must be supplied as a single IP or
  Network CIDR. For a `DNAT` rule, this is the internal IP address for incoming traffic. For an `SNAT` rule, this is the
  internal IP Address for outgoing traffic. For a `REFLEXIVE` rule, these are the internal IPs.
  These IPs are typically the Private IPs that are allocated to workloads.
* `app_port_profile_id` (Optional) - Application Port Profile to which to apply the rule. The
  Application Port Profile includes a port, and a protocol that the incoming traffic uses on the edge
  gateway to connect to the internal network.  Can be looked up using `vcd_nsxt_app_port_profile`
  data source or created using `vcd_nsxt_app_port_profile` resource
* `dnat_external_port` (Optional) - For `DNAT` only. This represents the external port number or port range when doing 
  `DNAT` port forwarding from external to internal. The default dnatExternalPort is “ANY” meaning traffic on any port
  for the given IPs selected will be translated.
* `snat_destination_address` (Optional) For `SNAT` only. The destination addresses to match in the `SNAT` Rule. This 
  must be supplied as a single IP or Network CIDR. Providing no value for this field results in match with ANY 
  destination network.
* `logging` (Optional) - Enable to have the address translation performed by this rule logged
  (default `false`). **Note** User might lack rights (**Organization Administrator** role by default
  is missing **Gateway -> Configure System Logging** right) to enable logging, but API does not
  return error and it is not possible to validate it. `terraform plan` might show difference on
  every update.
* `firewall_match` (Optional, VCD 10.2.2+) - You can set a firewall match rule to determine how
  firewall is applied during NAT. One of `MATCH_INTERNAL_ADDRESS`, `MATCH_EXTERNAL_ADDRESS`,
  `BYPASS`
  * `MATCH_INTERNAL_ADDRESS` - applies firewall rules to the internal address of a NAT rule
  * `MATCH_EXTERNAL_ADDRESS` - applies firewall rules to the external address of a NAT rule
  * `BYPASS` - skip applying firewall rules to NAT rule
* `priority` (Optional, VCD 10.2.2+) - if an address has multiple NAT rules, you can assign these
  rules different priorities to determine the order in which they are applied. A lower value means a
  higher priority for this rule. 

## Importing

~> The current implementation of Terraform import can only import resources into the state.
It does not generate configuration. [More information.](https://www.terraform.io/docs/import/)

An existing NAT Rule configuration can be [imported][docs-import] into this resource
via supplying the full dot separated path for your NAT Rule name or ID. An example is
below:

[docs-import]: https://www.terraform.io/docs/import/

Supplying Name
```
terraform import vcd_nsxt_nat_rule.imported my-org.my-org-vdc.my-nsxt-edge-gateway.my-nat-rule-name
```



-> When there are multiple NAT rules with the same name they will all be listed so that one can pick
it by ID

```
$ terraform import vcd_nsxt_nat_rule.dnat my-org.nsxt-vdc.nsxt-gw.dnat1

vcd_nsxt_nat_rule.dnat: Importing from ID "my-org.nsxt-vdc.nsxt-gw.dnat1"...
# The following NAT rules with Name 'dnat1' are available
# Please use ID instead of Name in import path to pick exact rule
ID                                   Name  Rule Type Internal Address   External Address
04fde766-2cbd-4986-93bb-7f57e59c6b19 dnat1 DNAT      1.1.1.1            10.1.2.139
f40e3d68-cfa6-42ea-83ed-5571659b3e7b dnat1 DNAT      2.2.2.2            10.1.2.139

$ terraform import vcd_nsxt_nat_rule.imported my-org.my-org-vdc.my-nsxt-edge-gateway.0214a26b-fc30-4202-88e5-7ed551aa6c19
```

The above would import the `my-nat-rule-name` NAT Rule config settings that are defined
on NSX-T Edge Gateway `my-nsxt-edge-gateway` which is configured in organization named `my-org` and
VDC named `my-org-vdc`.
