---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_vm_sizing_policy"
sidebar_current: "docs-vcd-resource-vm-sizing-policy"
description: |-
  Provides a VMware Cloud Director VM sizing policy resource. This can be
  used to create, modify, and delete VM sizing policy.
---

# vcd\_vm\_sizing\_policy

Provides a VMware Cloud Director VM sizing policy resource. This can be
used to create, modify, and delete VM sizing policy.

Supported in provider *v3.0+* and requires VCD 10.0+

-> **Note:** This resource requires system administrator privileges.

-> **Note:** 
CPU and memory properties of a VM sizing policy can't be updated in-place, so updating them will force a re-create. For such re-creation to succeed, the policy can't be used by VDC and VM. Hence, the policy usage has to be removed from VDC and VM beforehand. For the cases when that is not trivial, a two-step approach may be easier: to create a new policy with the new values, assign it to VDC and VM, and afterwards remove the old policy.

## Example Usage

```hcl
resource "vcd_vm_sizing_policy" "minSize" {
  org         = "my-org" # Optional
  name        = "min-size"
  description = "smallest size"

  cpu {
    shares                = "886"
    limit_in_mhz          = "2400"
    count                 = "9"
    speed_in_mhz          = "2500"
    cores_per_socket      = "3"
    reservation_guarantee = "0.55"
  }

  memory {
    shares                = "1580"
    size_in_mb            = "3200"
    limit_in_mb           = "2800"
    reservation_guarantee = "0.3"
  }
}
```
## Argument Reference

The following arguments are supported:

* `org` - (Optional) The name of organization to use, optional if defined at provider level. Useful when connected as sysadmin working across different organizations
* `name` - (Required) The name of VM sizing policy.
* `description` - (Optional) description of VM sizing policy.
* `cpu` - (Optional) Configures cpu policy; see [Cpu](#cpu) below for details.
* `memory` - (Optional) Configures memory policy; see [Memory](#memory) below for details.
 
<a id="cpu"></a>
## CPU
 
 Each VM sizing policy supports the following attributes:
 
 * `shares` - (Optional) Defines the number of CPU shares for a VM. Shares specify the relative importance of a VM within a virtual data center. If a VM has twice as many shares of CPU as another VM, it is entitled to consume twice as much CPU when these two virtual machines are competing for resources. If not defined in the VDC compute policy, normal shares are applied to the VM.
 * `limit_in_mhz` - (Optional) Defines the CPU limit in MHz for a VM. If not defined in the VDC compute policy, CPU limit is equal to the vCPU speed multiplied by the number of vCPUs.
 * `count` - (Required) Defines the number of vCPUs configured for a VM. This is a VM hardware configuration. When a tenant assigns the VM sizing policy to a VM, this count becomes the configured number of vCPUs for the VM.
 * `speed_in_mhz` - (Optional) Defines the vCPU speed of a core in MHz.
 * `cores_per_socket` - (Optional) The number of cores per socket for a VM. This is a VM hardware configuration. The number of vCPUs that is defined in the VM sizing policy must be divisible by the number of cores per socket. If the number of vCPUs is not divisible by the number of cores per socket, the number of cores per socket becomes invalid.
 * `reservation_guarantee` - (Optional) Defines how much of the CPU resources of a VM are reserved. The allocated CPU for a VM equals the number of vCPUs times the vCPU speed in MHz. The value of the attribute ranges between 0 and one. Value of 0 CPU reservation guarantee defines no CPU reservation. Value of 1 defines 100% of CPU reserved.
 
<a id="memory"></a>
## Memory
  
  Each VM sizing policy supports the following attributes:
  
  * `shares` - (Optional) Defines the number of memory shares for a VM. Shares specify the relative importance of a VM within a virtual data center. If a VM has twice as many shares of memory as another VM, it is entitled to consume twice as much memory when these two virtual machines are competing for resources. If not defined in the VDC compute policy, normal shares are applied to the VM.
  * `size_in_mb` - (Optional) Defines the memory configured for a VM in MB. This is a VM hardware configuration. When a tenant assigns the VM sizing policy to a VM, the VM receives the amount of memory defined by this attribute.
  * `limit_in_mb` - (Optional) Defines the memory limit in MB for a VM. If not defined in the VM sizing policy, memory limit is equal to the allocated memory for the VM.
  * `reservation_guarantee` - (Optional) Defines the reserved amount of memory that is configured for a VM. The value of the attribute ranges between 0 and one. Value of 0 memory reservation guarantee defines no memory reservation. Value of 1 defines 100% of memory reserved.

# Importing

~> **Note:** The current implementation of Terraform import can only import resources into the state.
It does not generate configuration. [More information.](https://www.terraform.io/docs/import/)

An existing an VM sizing policy can be [imported][docs-import] into this resource
via supplying the full dot separated path to VM sizing policy. An example is
below:

```
terraform import vcd_vm_sizing_policy.my-policy my-org.policy_name
```
or using IDs:
```
terraform import vcd_vm_sizing_policy.my-policy my-org.policy_id
```

NOTE: the default separator (.) can be changed using Provider.import_separator or variable VCD_IMPORT_SEPARATOR

[docs-import]:https://www.terraform.io/docs/import/

After that, you can expand the configuration file and either update or delete the VM sizing policy as needed. Running `terraform plan`
at this stage will show the difference between the minimal configuration file and the VM sizing policy stored properties.

### Listing VM sizing policies

If you want to list IDs there is a special command **`terraform import vcd_vm_sizing_policy.imported list@org-name`**
where `org-name` is the organization used. 
The output for this command should look similar to the one below:

```
terraform import vcd_vm_sizing_policy.imported list@org-name
vcd_vm_sizing_policy.import: Importing from ID "list@org-name"...
Retrieving all VM sizing policies
No	ID									Name	
--	--									----	
1	urn:vcloud:vdcComputePolicy:100dc35a-572b-4876-a762-c734d67c56ef	tf_policy_3
2	urn:vcloud:vdcComputePolicy:446d623e-1eec-4c8c-8a14-2f7e6086546b	tf_policy_2

```

Now to import VM sizing policy with ID urn:vcloud:vdcComputePolicy:446d623e-1eec-4c8c-8a14-2f7e6086546b one could supply this command:

```shell
$ terraform import vcd_vm_sizing_policy.imported org-name.urn:vcloud:vdcComputePolicy:446d623e-1eec-4c8c-8a14-2f7e6086546b
```
