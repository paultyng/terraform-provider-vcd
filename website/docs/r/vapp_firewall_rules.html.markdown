---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_vapp_firewall_rules"
sidebar_current: "docs-vcd-resource-vapp-firewall-rules"
description: |-
  Provides a VMware Cloud Director vApp Firewall resource. This can be used to create, modify, and delete firewall settings and rules.
---

# vcd\_vapp\_firewall\_rules

Provides a VMware Cloud Director vApp Firewall resource. This can be used to create,
modify, and delete firewall settings and rules in a [vApp network](/docs/providers/vcd/r/vapp_network.html).
Firewall rules can be applied to [vApp networks connected to Org network](/docs/providers/vcd/r/vapp_network.html) or [vApp org networks](/docs/providers/vcd/r/vapp_org_network.html) which are fenced. 

!> **Warning:** Using this resource overrides any existing firewall rules on vApp network. It's recommended to have only one resource per vApp and vApp network. 

## Example Usage

```hcl
resource "vcd_vapp" "web" {
  name = "web"
}
​
resource "vcd_vapp_network" "vapp-net" {
  name               = "my-vapp-net"
  vapp_name          = vcd_vapp.web.name
  org_network_name   = "my-vdc-int-net"
  gateway            = "192.168.2.1"
  netmask            = "255.255.255.0"
  dns1               = "192.168.2.1"

  static_ip_pool {
    start_address = "192.168.2.51"
    end_address   = "192.168.2.100"
  }
}

resource "vcd_vapp_firewall_rules" "vapp-fw" {
  vapp_id        = vcd_vapp.web.id
  network_id     = vcd_vapp_network.vapp-net.id
  default_action = "drop"

  rule {
    name             = "drop-ftp-out"
    policy           = "drop"
    protocol         = "tcp"
    destination_port = "21"
    destination_ip   = "any"
    source_port      = "any"
    source_ip        = "10.10.0.0/24"
  }

  rule {
    name             = "allow-outbound"
    policy           = "allow"
    protocol         = "any"
    destination_port = "any"
    destination_ip   = "any"
    source_port      = "any"
    source_ip        = "10.10.0.0/24"
  }
}
```

## Argument Reference

The following arguments are supported:

* `org` - The name of organization to use, optional if defined at provider level. Useful when connected as sysadmin working across different organisations.
* `vdc` - The name of VDC to use, optional if defined at provider level.
* `vapp_id` - (Required) The identifier of [vApp](/docs/providers/vcd/r/vapp.html).
* `network_id` - (Required) The identifier of [vApp network](/docs/providers/vcd/r/vapp_network.html).
* `enabled` - (Optional) Enable or disable firewall. Default is `true`.
* `default_action` - (Required) Either 'allow' or 'drop'. Specifies what to do should none of the rules match.
* `log_default_action` - (Optional) Flag to enable logging for default action. Default value is `false`.
* `rule` - (Optional) Configures a firewall rule; see [Rules](#rules) below for details.

<a id="rules"></a>
## Rules

Each firewall rule supports the following attributes:

* `name` - (Optional) Name of the firewall rule.
* `enabled` - (Optional) `true` value will enable firewall rule. Default is `true`.
* `policy` - (Optional) Specifies what to do when this rule is matched. Either `allow` or `drop`.
* `protocol` - (Optional) The protocol to match. One of `tcp`, `udp`, `icmp`, `any` or `tcp&udp`. Default is `any`.
* `destination_port` - (Optional) The destination port to match. Either a port number or `any`.
* `destination_ip` - (Optional) The destination IP to match. Either an IP address, IP range or `any`.
* `destination_vm_id` - (Optional) Destination VM identifier.
* `destination_vm_ip_type` - (Optional) The value can be one of: `assigned` - assigned internal IP will be automatically chosen, `NAT` - NATed external IP will be automatically chosen.
* `destination_vm_nic_id` - (Optional) VM NIC ID to which this rule applies.
* `source_port` - (Optional) The source port to match. Either a port number or `any`.
* `source_ip` - (Optional) The source IP to match. Either an IP address, IP range or `any`.
* `source_vm_id` - (Optional) Source VM identifier.
* `source_vm_ip_type` - (Optional) The value can be one of: `assigned` - assigned internal IP will be automatically chosen, `NAT` - NATed external IP will be automatically chosen.
* `source_vm_nic_id` - (Optional) VM NIC ID to which this rule applies.
* `enable_logging`- (Optional) `true` value will enable rule logging. Default is `false`.

## Importing

~> **Note:** The current implementation of Terraform import can only import resources into the state.
It does not generate configuration. [More information.](https://www.terraform.io/docs/import/)

An existing an vApp network firewall rules can be [imported][docs-import] into this resource
via supplying the full dot separated path to vApp network. An example is
below:

```
terraform import vcd_vapp_firewall_rules.my-rules my-org.my-vdc.vapp-name.network-name
```

or using IDs:

```
terraform import vcd_vapp_firewall_rules.my-rules my-org.my-vdc.vapp-id.network-id
```

NOTE: the default separator (.) can be changed using Provider.import_separator or variable VCD_IMPORT_SEPARATOR

[docs-import]:https://www.terraform.io/docs/import/

After that, you can expand the configuration file and either update or delete the vApp network rules as needed. Running `terraform plan`
at this stage will show the difference between the minimal configuration file and the vApp network rules stored properties.

### Listing vApp Network IDs

If you want to list IDs there is a special command **`terraform import vcd_vapp_firewall_rules.imported list@org-name.vcd-name.vapp-name`**
where `org-name` is the organization used, `vdc-name` is VDC name and `vapp-name` is vApp name. 
The output for this command should look similar to the one below:

```shell
$ terraform import vcd_vapp_firewall_rules.imported list@org-name.vdc-name.vapp-name
vcd_vapp_firewall_rules.imported: Importing from ID "list@org-name.vdc-name.vapp-name"...
Retrieving all vApp networks by name
No	vApp ID                                                 ID                                      Name	
--	-------                                                 --                                      ----	
1	urn:vcloud:vapp:77755b9c-5ec9-41f7-aceb-4cf158786482	0027c6ae-7d59-457e-b33e-a89e97f0bdc1	Net2
2	urn:vcloud:vapp:77755b9c-5ec9-41f7-aceb-4cf158786482	36986073-8051-4f6d-a1c6-bda648bdf6ba	Net1      		

Error: resource id must be specified in one of these formats:
'org-name.vdc-name.vapp-name.network_name', 'org.vdc-name.vapp-id.network-id' or 
'list@org-name.vdc-name.vapp-name' to get a list of vapp networks with their IDs

```

Now to import vApp network firewall rules with ID 0027c6ae-7d59-457e-b33e-a89e97f0bdc1 one could supply this command:

```shell
$ terraform import vcd_vapp_firewall_rules.imported org-name.vdc-name.urn:vcloud:vapp:77755b9c-5ec9-41f7-aceb-4cf158786482.0027c6ae-7d59-457e-b33e-a89e97f0bdc1
```
