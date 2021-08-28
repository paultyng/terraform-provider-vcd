---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_org_vdc"
sidebar_current: "docs-vcd-resource-org-vdc"
description: |-
  Provides a VMware Cloud Director Organization VDC resource. This can be used to create and delete an Organization VDC.
---

# vcd\_org\_vdc

Provides a VMware Cloud Director Organization VDC resource. This can be used to create and delete an Organization VDC.
Requires system administrator privileges.

-> **Note:** This resource supports NSX-T and NSX-V based Org Vdcs by providing relevant
`network_pool_name` and `provider_vdc_name`

Supported in provider *v2.2+*

## Example Usage

```hcl
provider "vcd" {
  user     = "${var.admin_user}"
  password = "${var.admin_password}"
  org      = "System"
  url      = "https://AcmeVcd/api"
}

resource "vcd_org_vdc" "my-vdc" {
  name        = "my-vdc"
  description = "The pride of my work"
  org         = "my-org"

  allocation_model  = "ReservationPool"
  network_pool_name = "vDC1-VXLAN-NP"
  provider_vdc_name = "vDC1"

  compute_capacity {
    cpu {
      allocated = 2048
    }

    memory {
      allocated = 2048
    }
  }

  storage_profile {
    name    = "storage-name"
    limit   = 10240
    default = true
  }

  metadata = {
    role    = "customerName"
    env     = "staging"
    version = "v1"
  }

  enabled                  = true
  enable_thin_provisioning = true
  enable_fast_provisioning = true
  delete_force             = true
  delete_recursive         = true
}
```

## Example Usage (NSX-T)
```hcl
resource "vcd_org_vdc" "nsxt-vdc" {
  name = "NSXT-VDC"
  org  = "main-org"

  allocation_model  = "ReservationPool"
  network_pool_name = "NSX-T Overlay 1"
  provider_vdc_name = "nsxTPvdc1"

  compute_capacity {
    cpu {
      allocated = "1024"
      limit     = "1024"
    }

    memory {
      allocated = "1024"
      limit     = "1024"
    }
  }

  storage_profile {
    name    = "*"
    enabled = true
    limit   = 10240
    default = true
  }

  enabled                  = true
  enable_thin_provisioning = true
  enable_fast_provisioning = true
  delete_force             = true
  delete_recursive         = true
}
```

## Example Usage (With VM sizing policies)

```hcl
resource "vcd_vm_sizing_policy" "size_1" {
  name        = "size-one"

  cpu {
    shares                = "886"
    limit_in_mhz          = "2400"
    count                 = "9"
    speed_in_mhz          = "2500"
    cores_per_socket      = "3"
    reservation_guarantee = "0.55"
  }

}

resource "vcd_vm_sizing_policy" "size_2" {
  name        = "size-two"

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
resource "vcd_org_vdc" "my-vdc" {
  name        = "my-vdc"
  description = "The pride of my work"
  org         = "my-org"
  ...  
  default_vm_sizing_policy_id = vcd_vm_sizing_policy.size_1.id
  vm_sizing_policy_ids        = [vcd_vm_sizing_policy.size_1.id, vcd_vm_sizing_policy.size_2.id]
}
```

## Argument Reference

The following arguments are supported:

~> **Note:** Only part of fields are read if user is Organization administrator. With System Admin user all fields are populated.

* `org` - (Optional) Organization to create the VDC in, optional if defined at provider level
* `name` - (Required) VDC name
* `description` - (Optional) VDC friendly description
* `provider_vdc_name` - (Required, System Admin) Name of the Provider VDC from which this organization VDC is provisioned.
* `allocation_model` - (Required) The allocation model used by this VDC; must be one of 
    * AllocationVApp ("Pay as you go")
    * AllocationPool ("Allocation pool")
    * ReservationPool ("Reservation pool")
    * Flex ("Flex") (*v2.7+*, *vCD 9.7+*)
* `compute_capacity` - (Required) The compute capacity allocated to this VDC.  See [Compute Capacity](#computecapacity) below for details.
* `nic_quota` - (Optional) Maximum number of virtual NICs allowed in this VDC. Defaults to 0, which specifies an unlimited number.
* `network_quota` - (Optional) Maximum number of network objects that can be deployed in this VDC. Defaults to 0, which means no networks can be deployed.
* `vm_quota` - (Optional) The maximum number of VMs that can be created in this VDC. Includes deployed and undeployed VMs in vApps and vApp templates. Defaults to 0, which specifies an unlimited number.
* `enabled` - (Optional) True if this VDC is enabled for use by the organization VDCs. Default is true.
* `storage_profile` - (Required, System Admin) Storage profiles supported by this VDC.  See [Storage Profile](#storageprofile) below for details.
* `memory_guaranteed` - (Optional, System Admin) Percentage of allocated memory resources guaranteed to vApps deployed in this VDC. For example, if this value is 0.75, then 75% of allocated resources are guaranteed. Required when `allocation_model` is AllocationVApp, AllocationPool or Flex. When Allocation model is AllocationPool minimum value is 0.2. If left empty, vCD sets a value.
* `cpu_guaranteed` - (Optional, System Admin) Percentage of allocated CPU resources guaranteed to vApps deployed in this VDC. For example, if this value is 0.75, then 75% of allocated resources are guaranteed. Required when `allocation_model` is AllocationVApp, AllocationPool or Flex. If left empty, vCD sets a value.
* `cpu_speed` - (Optional, System Admin) Specifies the clock frequency, in Megahertz, for any virtual CPU that is allocated to a VM. A VM with 2 vCPUs will consume twice as much of this value. Ignored for ReservationPool. Required when `allocation_model` is AllocationVApp, AllocationPool or Flex, and may not be less than 256 MHz. Defaults to 1000 MHz if value isn't provided.
* `metadata` - (Optional; *v2.4+*) Key value map of metadata to assign to this VDC
* `enable_thin_provisioning` - (Optional, System Admin) Boolean to request thin provisioning. Request will be honored only if the underlying data store supports it. Thin provisioning saves storage space by committing it on demand. This allows over-allocation of storage.
* `enable_fast_provisioning` - (Optional, System Admin) Request fast provisioning. Request will be honored only if the underlying datastore supports it. Fast provisioning can reduce the time it takes to create virtual machines by using vSphere linked clones. If you disable fast provisioning, all provisioning operations will result in full clones.
* `network_pool_name` - (Optional, System Admin) Reference to a network pool in the Provider VDC. Required if this VDC will contain routed or isolated networks.
* `allow_over_commit` - (Optional) Set to false to disallow creation of the VDC if the `allocation_model` is AllocationPool or ReservationPool and the ComputeCapacity you specified is greater than what the backing Provider VDC can supply. Default is true.
* `enable_vm_discovery` - (Optional) If true, discovery of vCenter VMs is enabled for resource pools backing this VDC. If false, discovery is disabled. If left unspecified, the actual behaviour depends on enablement at the organization level and at the system level.
* `elasticity` - (Optional, *v2.7+*, *vCD 9.7+*) Indicates if the Flex VDC should be elastic. Required with the Flex allocation model.
* `include_vm_memory_overhead` - (Optional, *v2.7+*, *vCD 9.7+*) Indicates if the Flex VDC should include memory overhead into its accounting for admission control. Required with the Flex allocation model.
* `delete_force` - (Required) When destroying use `delete_force=True` to remove a VDC and any objects it contains, regardless of their state.
* `delete_recursive` - (Required) When destroying use `delete_recursive=True` to remove the VDC and any objects it contains that are in a state that normally allows removal.
* `default_vm_sizing_policy_id` - (Optional, *v3.0+*, *vCD 10.0+*) Set of VM sizing policy IDs. This field requires `vm_sizing_policy_ids` to be configured together. 
* `vm_sizing_policy_ids` - (Optional, *v3.0+*, *vCD 10.0+*) Default VM sizing policy ID. This field requires `default_vm_sizing_policy_id` to be configured together.

<a id="storageprofile"></a>
## Storage Profile

* `name` - (Required) Name of Provider VDC storage profile.
* `enabled` - (Optional) True if this storage profile is enabled for use in the VDC. Default is true.
* `limit` - (Required) Maximum number of MB allocated for this storage profile. A value of 0 specifies unlimited MB.
* `default` - (Required) True if this is default storage profile for this VDC. The default storage profile is used when an object that can specify a storage profile is created with no storage profile specified.
* `storage_used_in_mb` - (Computed, *v3.1+*) Storage used, in Megabytes.

<a id="computecapacity"></a>
## Compute Capacity

Capacity must be specified twice, once for `memory` and another for `cpu`.  Each has the same structure:

* `allocated` - (Optional) Capacity that is committed to be available. Value in MB or MHz. Used with AllocationPool ("Allocation pool"), ReservationPool ("Reservation pool"), Flex.
* `limit` - (Optional) Capacity limit relative to the value specified for Allocation. It must not be less than that value. If it is greater than that value, it implies over provisioning. A value of 0 specifies unlimited units. Value in MB or MHz. Used with AllocationVApp ("Pay as you go") or Flex (only for `cpu`).

## Importing

Supported in provider *v2.5+*

~> **Note:** The current implementation of Terraform import can only import resources into the state.
It does not generate configuration. [More information.](https://www.terraform.io/docs/import/)

An existing an organization VDC can be [imported][docs-import] into this resource
via supplying the full dot separated path to VDC. An example is
below:

```
terraform import vcd_org_vdc.my-vdc my-org.my-vdc
```

NOTE: the default separator (.) can be changed using Provider.import_separator or variable VCD_IMPORT_SEPARATOR

[docs-import]:https://www.terraform.io/docs/import/

After that, you can expand the configuration file and either update or delete the VDC as needed. Running `terraform plan`
at this stage will show the difference between the minimal configuration file and the VDC's stored properties.

