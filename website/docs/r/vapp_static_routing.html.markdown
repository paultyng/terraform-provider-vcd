---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_vapp_static_routing"
sidebar_current: "docs-vcd-resource-vapp-static-routing"
description: |-
  Provides a VMware Cloud Director vApp static routing resource. This can be used to create, modify, and delete static routing rules.
---

# vcd\_vapp\_static\_routing

Provides a VMware Cloud Director vApp static routing resource. This can be used to create,
modify, and delete static routing rules in a [vApp network](/docs/providers/vcd/r/vapp_network.html).

~> **Note:** Resource used for vApp network, not vApp Org network.

!> **Warning:** Using this resource overrides any existing static routing rules on vApp network. It's recommended to have only one resource per vApp and vApp network. 

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
​
  static_ip_pool {
    start_address = "192.168.2.51"
    end_address   = "192.168.2.100"
  }
}

resource "vcd_vapp_static_routing" "vapp1-static-routing" {
  vapp_id    = vcd_vapp.web.id
  network_id = vcd_vapp_network.vapp-net.id
  enabled    = true

  rule {
    name         = "rule1"
    network_cidr = "10.10.0.0/24"
    next_hop_ip  = "192.168.2.2"
  }

  rule {
    name         = "rule2"
    network_cidr = "10.10.1.0/24"
    next_hop_ip  = "192.168.2.3"
  }
}
```

## Argument Reference

The following arguments are supported:

* `org` - (Optional) The name of organization to use, optional if defined at provider level. Useful when connected as sysadmin working across different organisations.
* `vdc` - (Optional) The name of VDC to use, optional if defined at provider level.
* `vapp_id` - (Required) The identifier of [vApp](/docs/providers/vcd/r/vapp.html).
* `network_id` - (Required) The identifier of [vApp network](/docs/providers/vcd/r/vapp_network.html).
* `enabled` - (Optional) Enable or disable static Routing. Default is `true`.
* `rule` - (Optional) Configures a static routing rule; see [Rules](#rules) below for details.

<a id="rules"></a>
## Rules

Each static routing rule supports the following attributes:

* `name` - (Required) Name for the static route.
* `network_cidr` - (Required) Network specification in CIDR.
* `next_hop_ip` - (Required) IP address of Next Hop router/gateway.

## Importing

~> **Note:** The current implementation of Terraform import can only import resources into the state.
It does not generate configuration. [More information.](https://www.terraform.io/docs/import/)

An existing an vApp network static routing rules can be [imported][docs-import] into this resource
via supplying the full dot separated path to vApp network. An example is
below:

```
terraform import vcd_vapp_static_routing.my-rules my-org.my-vdc.vapp_name.network_name
```
or using IDs:
```
terraform import vcd_vapp_static_routing.my-rules my-org.my-vdc.vapp_id.network_id
```

NOTE: the default separator (.) can be changed using Provider.import_separator or variable VCD_IMPORT_SEPARATOR

[docs-import]:https://www.terraform.io/docs/import/

After that, you can expand the configuration file and either update or delete the vApp network rules as needed. Running `terraform plan`
at this stage will show the difference between the minimal configuration file and the vApp network rules stored properties.

### Listing vApp Network IDs

If you want to list IDs there is a special command **`terraform import vcd_vapp_static_routing.imported list@org-name.vcd-name.vapp-name`**
where `org-name` is the organization used, `vdc-name` is VDC name and `vapp-name` is vApp name. 
The output for this command should look similar to the one below:

```shell
$ terraform import vcd_vapp_static_routing.imported list@org-name.vdc-name.vapp-name
vcd_vapp_static_routing.imported: Importing from ID "list@org-name.vdc-name.vapp-name"...
Retrieving all vApp networks by name
No	vApp ID                                                 ID                                      Name	
--	-------                                                 --                                      ----	
1	urn:vcloud:vapp:77755b9c-5ec9-41f7-aceb-4cf158786482	0027c6ae-7d59-457e-b33e-a89e97f0bdc1	Net2
2	urn:vcloud:vapp:77755b9c-5ec9-41f7-aceb-4cf158786482	36986073-8051-4f6d-a1c6-bda648bdf6ba	Net1      		

Error: resource id must be specified in one of these formats:
'org-name.vdc-name.vapp-name.network_name', 'org.vdc-name.vapp-id.network-id' or 
'list@org-name.vdc-name.vapp-name' to get a list of vapp networks with their IDs

```

Now to import vApp network static routing rules with ID 0027c6ae-7d59-457e-b33e-a89e97f0bdc1 one could supply this command:

```shell
$ terraform import vcd_vapp_static_routing.imported org-name.vdc-name.urn:vcloud:vapp:77755b9c-5ec9-41f7-aceb-4cf158786482.0027c6ae-7d59-457e-b33e-a89e97f0bdc1
```
