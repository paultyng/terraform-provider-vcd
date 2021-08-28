---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_vapp_nat_rules"
sidebar_current: "docs-vcd-resource-vapp-nat-rules"
description: |-
  Provides a VMware Cloud Director vApp NAT resource. This can be used to create, modify, and delete NAT rules.
---

# vcd\_vapp\_nat\_rules

Provides a VMware Cloud Director vApp NAT resource. This can be used to create,
modify, and delete NAT rules in a [vApp network](/docs/providers/vcd/r/vapp_network.html).
NAT rules can be applied to [vApp networks connected to Org network](/docs/providers/vcd/r/vapp_network.html) or [vApp org networks](/docs/providers/vcd/r/vapp_org_network.html) which are fenced.

!> **Warning:** Using this resource overrides any existing NAT rules on vApp network. It's recommended to have only one resource per vApp and vApp network. 

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

resource "vcd_vapp_org_network" "vapp-org-net" {
  vapp_name        = vcd_vapp.web.name
  org_network_name = vcd_network_routed.network-routed.name
  is_fenced        = true
}

# In order to enable NAT service, firewall needs to be enabled as well
resource "vcd_vapp_firewall_rules" "vapp_fw1" {
  vapp_id        = vcd_vapp.web.id
  network_id     = vcd_vapp_network.vapp-net.id
  default_action = "drop"
  enabled = true
}

resource "vcd_vapp_nat_rules" "vapp-nat" {
  vapp_id              = vcd_vapp.web.id
  network_id           = vcd_vapp_network.vapp-net.id
  nat_type             = "portForwarding"
  enable_ip_masquerade = false

  rule {
    external_port   = "22"
    forward_to_port = "-1"
    protocol        = "TCP_UDP"
    vm_nic_id       = "0"
    vm_id           = vcd_vapp_vm.vm1.id
  }

  rule {
    external_port   = "-1"
    forward_to_port = "80"
    protocol        = "TCP"
    vm_nic_id       = "0"
    vm_id           = vcd_vapp_vm.vm2.id
  }
}

# In order to enable NAT service, firewall needs to be enabled as well
resource "vcd_vapp_firewall_rules" "vapp_fw2" {
  vapp_id    = vcd_vapp.web.id
  network_id = vcd_vapp_network.vapp-org-net.id
  default_action = "drop"
  enabled = true
}

resource "vcd_vapp_nat_rules" "vapp-nat2" {
  vapp_id    = vcd_vapp.web.id
  network_id = vcd_vapp_network.vapp-org-net.id
  nat_type   = "ipTranslation"

  rule {
    mapping_mode = "manual"
    external_ip  = "10.10.103.13"
    vm_nic_id    = 0
    vm_id        = vcd_vapp_vm.vm1.id
  }

  rule {
    mapping_mode = "automatic"
    vm_nic_id    = 0
    vm_id        = vcd_vapp_vm.vm2.id
  }
}

```

## Argument Reference

The following arguments are supported:

* `org` - (Optional) The name of organization to use, optional if defined at provider level. Useful when connected as sysadmin working across different organisations.
* `vdc` - (Optional) The name of VDC to use, optional if defined at provider level.
* `vapp_id` - (Required) The identifier of [vApp](/docs/providers/vcd/r/vapp.html).
* `network_id` - (Required) The identifier of [vApp network](/docs/providers/vcd/r/vapp_network.html).
* `enabled` - (Optional) Enable or disable NAT. Default is `true`. To enable the NAT service, [vcd_vapp_firewall_rules](/docs/providers/vcd/r/vapp_firewall_rules.html) needs to be enabled as well.
* `nat_type` - (Required) "One of: `ipTranslation` (use IP translation), `portForwarding` (use port forwarding). For `ipTranslation` fields `vm_id`, `vm_nic_id`, `mapping_mode` are required and `external_ip` is optional. For `portForwarding` fields `vm_id`, `vm_nic_id`, `protocol`, `external_port` and `forward_to_port` are required.
* `enable_ip_masquerade` - (Optional) When enabled translates a virtual machine's private, internal IP address to a public IP address for outbound traffic. Default value is `false`.
* `rule` - (Optional) Configures a NAT rule; see [Rules](#rules) below for details.

<a id="rules"></a>
## Rules

Each NAT rule supports the following attributes:

* `mapping_mode` - (Optional) Mapping mode. One of: `automatic`, `manual`.
* `vm_id` - (Required) VM to which this rule applies.
* `vm_nic_id` - (Required) VM NIC ID to which this rule applies.
* `external_ip` - (Optional) External IP address to forward to or External IP address to map to VM.
* `external_port` - (Optional) External port to forward. `-1` value for any port.
* `forward_to_port` - (Optional) Internal port to forward. `-1` value for any port.
* `protocol` - (Optional) Protocol to forward. One of: `TCP` (forward TCP packets), `UDP` (forward UDP packets), `TCP_UDP` (forward TCP and UDP packets).

## Importing

~> **Note:** The current implementation of Terraform import can only import resources into the state.
It does not generate configuration. [More information.](https://www.terraform.io/docs/import/)

An existing vApp network NAT rules can be [imported][docs-import] into this resource
via supplying the full dot separated path to vapp network. An example is
below:

```
terraform import vcd_vapp_nat_rules.my-rules my-org.my-vdc.vapp_name.network_name
```
or using IDs:
```
terraform import vcd_vapp_nat_rules.my-rules my-org.my-vdc.vapp_id.network_id
```

NOTE: the default separator (.) can be changed using Provider.import_separator or variable VCD_IMPORT_SEPARATOR

[docs-import]:https://www.terraform.io/docs/import/

After that, you can expand the configuration file and either update or delete the vApp network rules as needed. Running `terraform plan`
at this stage will show the difference between the minimal configuration file and the vApp network rules stored properties.

### Listing vApp Network IDs

If you want to list IDs there is a special command **`terraform import vcd_vapp_nat_rules.imported list@org-name.vcd-name.vapp-name`**
where `org-name` is the organization used, `vdc-name` is VDC name and `vapp-name` is vApp name. 
The output for this command should look similar to the one below:

```shell
$ terraform import vcd_vapp_nat_rules.imported list@org-name.vdc-name.vapp-name
vcd_vapp_nat_rules.imported: Importing from ID "list@org-name.vdc-name.vapp-name"...
Retrieving all vApp networks by name
No	vApp ID                                                 ID                                      Name	
--	-------                                                 --                                      ----	
1	urn:vcloud:vapp:77755b9c-5ec9-41f7-aceb-4cf158786482	0027c6ae-7d59-457e-b33e-a89e97f0bdc1	Net2
2	urn:vcloud:vapp:77755b9c-5ec9-41f7-aceb-4cf158786482	36986073-8051-4f6d-a1c6-bda648bdf6ba	Net1      		

Error: resource id must be specified in one of these formats:
'org-name.vdc-name.vapp-name.network_name', 'org.vdc-name.vapp-id.network-id' or 
'list@org-name.vdc-name.vapp-name' to get a list of vapp networks with their IDs

```

Now to import vApp network NAT rules with ID 0027c6ae-7d59-457e-b33e-a89e97f0bdc1 one could supply this command:

```shell
$ terraform import vcd_vapp_nat_rules.imported org-name.vdc-name.urn:vcloud:vapp:77755b9c-5ec9-41f7-aceb-4cf158786482.0027c6ae-7d59-457e-b33e-a89e97f0bdc1
```
