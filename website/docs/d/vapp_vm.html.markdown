---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_vapp_vm"
sidebar_current: "docs-vcd-datasource-vapp-vm"
description: |-
  Provides a VMware Cloud Director VM data source. This can be used to access VMs within a vApp.
---

# vcd\_vapp\_vm

Provides a VMware Cloud Director VM data source. This can be used to access VMs within a vApp.

Supported in provider *v2.6+*

## Example Usage

```hcl

data "vcd_vapp" "web" {
  name = "web"
}

data "vcd_vapp_vm" "web1" {
  vapp_name = data.vcd_vapp.web.name
  name      = "web1"
}

output "vm_id" {
  value = data.vcd_vapp_vm.id
}
output "vm" {
  value = data.vcd_vapp_vm.web1
}
```

Sample output:

```
vm = {
  "computer_name" = "TestVM"
  "cpu_cores" = 1
  "cpus" = 2
  "description" = "This OVA provides a minimal installed profile of PhotonOS. Default password for root user is changeme"
  "disk" = []
  "guest_properties" = {}
  "href" = "https://my-vcd.org/api/vApp/vm-ecb449a2-0b11-494d-bbc7-6ae2f2ff9b82"
  "id" = "urn:vcloud:vm:ecb449a2-0b11-494d-bbc7-6ae2f2ff9b82"
  "memory" = 1024
  "metadata" = {
    "vm_metadata" = "VM Metadata."
  }
  "name" = "vm-datacloud"
  "network" = [
    {
      "ip" = "192.168.2.10"
      "ip_allocation_mode" = "MANUAL"
      "is_primary" = true
      "mac" = "00:50:56:29:08:89"
      "name" = "net-datacloud-r"
      "type" = "org"
    },
  ]
  "org" = "datacloud"
  "storage_profile" = "*"
  "vapp_name" = "vapp-datacloud"
  "vdc" = "vdc-datacloud"
}
```

## Argument Reference

The following arguments are supported:

* `org` - (Optional) The name of organization to use, optional if defined at provider level. Useful when connected as sysadmin working across different organisations
* `vdc` - (Optional) The name of VDC to use, optional if defined at provider level
* `vapp_name` - (Required) The vApp this VM belongs to.
* `name` - (Required) A name for the VM, unique within the vApp 
* `network_dhcp_wait_seconds` - (Optional; *v2.7+*) Allows to wait for up to a defined amount of
  seconds before IP address is reported for NICs with `ip_allocation_mode=DHCP` setting. It
  constantly checks if IP is reported so the time given is a maximum. VM must be powered on and 
  __at least one__ of the following __must be true__:
 * VM has guest tools. It waits for IP address to be reported in vCD UI. This is a slower option, but
  does not require for the VM to use Edge Gateways DHCP service.
 * VM DHCP interface is connected to routed Org network and is using Edge Gateways DHCP service (not
  relayed). It works by querying DHCP leases on edge gateway. In general it is quicker than waiting
  until UI reports IP addresses, but is more constrained. However this is the only option if guest
  tools are not present on the VM.

## Attribute reference

* `vm_type` (*3.2+*) - type of the VM (either `vcd_vapp_vm` or `vcd_vm`)
* `computer_name` -  Computer name to assign to this virtual machine. 
* `catalog_name` -  The catalog name in which to find the given vApp Template
* `template_name` -  The name of the vApp Template to use
* `memory` -  The amount of RAM (in MB) allocated to the VM
* `cpus` -  The number of virtual CPUs allocated to the VM
* `cpu_cores` -  The number of cores per socket
* `metadata` -  Key value map of metadata assigned to this VM
* `disk` -  Independent disk attachment configuration.
* `network` -  A block defining a network interface. Multiple can be used.
* `guest_properties` -  Key value map of guest properties
* `description`  -  The VM description. Note: description is read only. Currently, this field has
  the description of the OVA used to create the VM
* `expose_hardware_virtualization` -  Expose hardware-assisted CPU virtualization to guest OS
* `internal_disk` - (*v2.7+*) A block providing internal disk of VM details
* `os_type` - (*v2.9+*) Operating System type.
* `hardware_version` - (*v2.9+*) Virtual Hardware Version (e.g.`vmx-14`, `vmx-13`, `vmx-12`, etc.).
* `sizing_policy_id` (*v3.0+*, *vCD 10.0+*) VM sizing policy ID.


See [VM resource](/docs/providers/vcd/r/vapp_vm.html#attribute-reference) for more info about VM attributes.
