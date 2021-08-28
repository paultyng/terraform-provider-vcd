---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_nsxt_network_dhcp"
sidebar_current: "docs-vcd-resource-nsxt-network-dhcp"
description: |-
  Provides a resource to manage DHCP pools for NSX-T Org VDC Routed network.
---

# vcd\_nsxt\_network\_dhcp

Provides a resource to manage DHCP pools for NSX-T Org VDC Routed network.

Supported in provider *v3.2+* and VCD 10.1+ with NSX-T backed VDCs.

## Specific usage notes

**DHCP pool support** for NSX-T Routed networks is **limited** by the API in the following ways:

* DHCP pools can only be added, but not updated/removed one by one

* VCD 10.2+ allows to remove all DHCP pools at once (terraform destroy/resource removal)

* VCD 10.1 **does not allow to remove DHCP pools** after they are created without removing parent network
therefore **destroying the resource will emit warning and do nothing** (to avoid breaking Terraform flow). 
See [Example usage 2](#example-usage-2-pool-removal-on-vcd-101) to see a way for changing/removing DHCP pools

## Example Usage 1

```hcl
resource "vcd_network_routed_v2" "parent-network" {
  name = "nsxt-routed-dhcp"

  edge_gateway_id = data.vcd_nsxt_edgegateway.existing.id

  gateway       = "7.1.1.1"
  prefix_length = 24

  static_ip_pool {
    start_address = "7.1.1.10"
    end_address   = "7.1.1.20"
  }
}

resource "vcd_nsxt_network_dhcp" "pools" {
  org_network_id = vcd_network_routed_v2.parent-network.id

  pool {
    start_address = "7.1.1.100"
    end_address   = "7.1.1.110"
  }

  pool {
    start_address = "7.1.1.111"
    end_address   = "7.1.1.112"
  }
}
```

## Example Usage 2 (Pool removal on VCD 10.1)

DHCP pool definitions cannot be removed once defined therefore the trick is to recreate parent Org VDC network.
This can be done by following such procedure:

* Define the network and DHCP pools as in [Example Usage 1](#example-usage-1)
* Use `terraform taint` on the parent network to force recreation:
   ```sh
   # terraform taint vcd_network_routed_v2.parent-network
   Resource instance vcd_network_routed_v2.parent-network has been marked as tainted.
   ```
* Modify/remove `vcd_nsxt_network_dhcp` definition as per your needs
* Perform `terraform apply`. This will recreate tainted parent Org VDC network and new DHCP pools if defined.
You will see a WARNING during removal but it will not break :
```sh
vcd_nsxt_network_dhcp.pools: Destroying... [id=urn:vcloud:network:209666ec-6253-418a-a816-076b15413fea]
vcd_nsxt_network_dhcp WARNING: for VCD versions < 10.2 DHCP pool removal is not supported. Destroy is a NO-OP operation for VCD versions < 10.2. Please recreate parent network to remove DHCP pools.
vcd_nsxt_network_dhcp.pools: Destruction complete after 0s
```

## Argument Reference

The following arguments are supported:

* `org` - (Optional) The name of organization to use, optional if defined at provider level. Useful
  when connected as sysadmin working across different organisations.
* `vdc` - (Optional) The name of VDC to use, optional if defined at provider level.
* `org_network_id` - (Required) ID of parent Org VDC Routed network
* `pool` - (Required) One or more blocks to define DHCP pool ranges. See [Pools](#pools) and example 
for usage details.

## Pools

* `start_address` - (Required) Start address of DHCP pool range
* `end_address` - (Required) End address of DHCP pool range

## Importing

~> The current implementation of Terraform import can only import resources into the state.
It does not generate configuration. [More information.](https://www.terraform.io/docs/import/)

An existing DHCP configuration can be [imported][docs-import] into this resource
via supplying the full dot separated path for your Org VDC network. An example is
below:

[docs-import]: https://www.terraform.io/docs/import/

```
terraform import vcd_nsxt_network_dhcp.imported my-org.my-org-vdc.my-nsxt-vdc-network-name
```

The above would import the DHCP config settings that are defined on VDC network 
`my-nsxt-vdc-network-name` which is configured in organization named `my-org` and VDC named 
`my-org-vdc`.
