---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_vm_internal_disk"
sidebar_current: "docs-vcd-vm-internal-disk"
description: |-
  Provides a VMware Cloud Director VM internal disk resource. This can be used to create and delete VM internal disks.
---

# vcd\_vm\_internal\_disk

This can be used to create, update and delete VM internal disks on already created VMs.

~> **Note:** To adjust disk parameters when creating a new VM, please use [override_template_disk](/docs/providers/vcd/r/vapp_vm.html#override-template-disk).

To manage disks which already exist inside a VM, please [import](#importing) them first.

~> **Note:** Managing disks in VM is possible only when VDC fast provisioned is disabled.

Supported in provider *v2.7+*

## Example Usage

```hcl
resource "vcd_vm_internal_disk" "disk1" {
  vapp_name       = "my-vapp"
  vm_name         = "my-vm1"
  bus_type        = "sata"
  size_in_mb      = "13333"
  bus_number      = 0
  unit_number     = 1
  storage_profile = "Development"
  allow_vm_reboot = true
  depends_on      = ["vcd_vapp_vm.web1"]
}
```

## Argument Reference

The following arguments are supported:

* `org` - (Optional) The name of organization to use, optional if defined at provider level. Useful when connected as sysadmin working across different organisations
* `vdc` - (Optional) The name of VDC to use, optional if defined at provider level
* `vapp_name` - (Required) The vAPP this VM internal disk belongs to.
* `vm_name` - (Required) VM in vAPP in which internal disk is created.
* `allow_vm_reboot` - (Optional) Powers off VM when changing any attribute of an IDE disk or unit/bus number of other disk types, after the change is complete VM is powered back on. Without this setting enabled, such changes on a powered-on VM would fail. Defaults to false.
* `bus_type` - (Required) The type of disk controller. Possible values: `ide`, `parallel`( LSI Logic Parallel SCSI), `sas`(LSI Logic SAS (SCSI)), `paravirtual`(Paravirtual (SCSI)), `sata`. 
* `size_in_mb` - (Required) The size of the disk in MB. 
* `bus_number` - (Required) The number of the SCSI or IDE controller itself.
* `unit_number` - (Required) The device number on the SCSI or IDE controller of the disk.
* `iops` - (Optional) Specifies the IOPS for the disk. Default is 0.
* `storage_profile` - (Optional) Storage profile which overrides the VM default one.

## Attribute reference

* `thin_provisioned` - Specifies whether the disk storage is pre-allocated or allocated on demand.

## Importing

~> **Note:** The current implementation of Terraform import can only import resources into the state. It does not generate
configuration. [More information.][docs-import]

An existing VM internal disk can be [imported][docs-import] into this resource via supplying its path.
The path for this resource is made of org-name.vdc-name.vapp-name.vm-name.disk-id
For example, using this structure, representing a VM internal disk that was **not** created using Terraform:

```hcl
resource "vcd_vm_internal_disk" "tf-myInternalDisk" {
  org       = "my-org"
  vdc       = "my-vdc"
  vapp_name = "my-vapp"
  vm_name   = "my-vm"
}
```

You can import such VM internal disk into terraform state using this command

```
terraform import vcd_vm_internal_disk.tf-myInternalDisk my-org.my-vdc.my-vapp.my-vm.my-disk-id
```

[docs-import]:https://www.terraform.io/docs/import/

After importing, if you run `terraform plan` you will see the rest of the values and modify the script accordingly for
further operations.

### Listing VM internal disk IDs

If you want to list IDs there is a special command **`terraform import vcd_vm_internal_disk.imported list@org-name.vcd-name.vapp-name.vm-name`**
where `org-name` is the organization used, `vdc-name` is vDC name, `vapp-name` is vAPP name and `vm_name` is VM name in that vAPP.
The output for this command should look similar to the one below:

```shell
$ terraform import vcd_vm_internal_disk.imported list@org-name.vdc-name.vapp-name.vm-name
vcd_vm_internal_disk.imported: Importing from ID "list@org-name.vdc-name.vapp-name.vm-name"...
Retrieving all disks
No	ID	    BusType		BusNumber	UnitNumber	Size	StoragePofile	Iops	ThinProvisioned
--	--	    -------		---------	----------	----	-------------	----	---------------
1	2000	paravirtual	0		    0		    16384	*               0	    true
2	3001	ide	     	0		    1		    17384	*               0	    true
3	16000	sata		0		    0		    18384	*               0	    true
4	16001	sata		0		    1		    13333	Development     0	    true

Error: resource was not imported! resource id must be specified in one of these formats:
'org-name.vdc-name.vapp-name.vm-name.my-internal-disk-id' to import by rule id
'list@org-name.vdc-name.vapp-name.vm-name' to get a list of internal disks with their IDs

```

Now to import disk with ID 3001 one could supply this command:

```shell
$ terraform import vcd_vm_internal_disk.imported org-name.vdc-name.vapp-name.vm-name.3001
```
