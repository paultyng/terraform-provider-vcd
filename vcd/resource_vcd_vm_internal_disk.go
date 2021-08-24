package vcd

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/vmware/go-vcloud-director/v2/govcd"
	"github.com/vmware/go-vcloud-director/v2/types/v56"
	"log"
	"strings"
	"text/tabwriter"
)

func resourceVmInternalDisk() *schema.Resource {
	return &schema.Resource{
		Create: resourceVmInternalDiskCreate,
		Read:   resourceVmInternalDiskRead,
		Update: resourceVmInternalDiskUpdate,
		Delete: resourceVmInternalDiskDelete,
		Importer: &schema.ResourceImporter{
			State: resourceVcdVmInternalDiskImport,
		},
		Schema: map[string]*schema.Schema{
			"org": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Description: "The name of organization to use, optional if defined at provider " +
					"level. Useful when connected as sysadmin working across different organizations",
			},
			"vdc": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "The name of VDC to use, optional if defined at provider level",
			},
			"vapp_name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The vApp this VM internal disk belongs to",
			},
			"vm_name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "VM in vApp in which internal disk is created",
			},
			"bus_type": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{"ide", "parallel", "sas", "paravirtual", "sata", "nvme"}, false),
				Description:  "The type of disk controller. Possible values: ide, parallel( LSI Logic Parallel SCSI), sas(LSI Logic SAS (SCSI)), paravirtual(Paravirtual (SCSI)), sata, nvme",
			},
			"size_in_mb": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "The size of the disk in MB.",
			},
			"bus_number": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "The number of the SCSI or IDE controller itself.",
			},
			"unit_number": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "The device number on the SCSI or IDE controller of the disk.",
			},
			"thin_provisioned": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Specifies whether the disk storage is pre-allocated or allocated on demand.",
			},
			"iops": {
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: "Specifies the IOPS for the disk.",
			},
			"storage_profile": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Storage profile to override the VM default one",
			},
			"allow_vm_reboot": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Powers off VM when changing any attribute of an IDE disk or unit/bus number of other disk types, after the change is complete VM is powered back on. Without this setting enabled, such changes on a powered-on VM would fail.",
			},
		},
	}
}

var internalDiskBusTypes = map[string]string{
	"ide":         "1",
	"parallel":    "3",
	"sas":         "4",
	"paravirtual": "5",
	"sata":        "6",
	"nvme":        "7",
}
var internalDiskBusTypesFromValues = map[string]string{
	"1": "ide",
	"3": "parallel",
	"4": "sas",
	"5": "paravirtual",
	"6": "sata",
	"7": "nvme",
}

// resourceVmInternalDiskCreate creates an internal disk for VM
func resourceVmInternalDiskCreate(d *schema.ResourceData, meta interface{}) error {
	vcdClient := meta.(*VCDClient)

	vcdClient.lockParentVm(d)
	defer vcdClient.unLockParentVm(d)

	vm, vdc, err := getVm(vcdClient, d)
	if err != nil {
		return err
	}

	var storageProfilePrt *types.Reference
	var overrideVmDefault bool

	if storageProfileName, ok := d.GetOk("storage_profile"); ok {
		storageProfile, err := vdc.FindStorageProfileReference(storageProfileName.(string))
		if err != nil {
			return fmt.Errorf("[internal disk creation] error retrieving storage profile %s : %s", storageProfileName, err)
		}
		storageProfilePrt = &storageProfile
		overrideVmDefault = true
	} else {
		storageProfilePrt = vm.VM.StorageProfile
		overrideVmDefault = false
	}

	iops := int64(d.Get("iops").(int))
	// value is required but not treated.
	isThinProvisioned := true

	diskSetting := &types.DiskSettings{
		SizeMb:              int64(d.Get("size_in_mb").(int)),
		UnitNumber:          d.Get("unit_number").(int),
		BusNumber:           d.Get("bus_number").(int),
		AdapterType:         internalDiskBusTypes[d.Get("bus_type").(string)],
		ThinProvisioned:     &isThinProvisioned,
		StorageProfile:      storageProfilePrt,
		Iops:                &iops,
		VirtualQuantityUnit: "byte",
		OverrideVmDefault:   overrideVmDefault,
	}

	vmStatusBefore, err := powerOffIfNeeded(d, vm)
	if err != nil {
		return err
	}

	diskId, err := vm.AddInternalDisk(diskSetting)
	if err != nil {
		return fmt.Errorf("error updating VM disks: %s", err)
	}

	d.SetId(diskId)

	err = powerOnIfNeeded(d, vm, vmStatusBefore)
	if err != nil {
		return err
	}

	return resourceVmInternalDiskRead(d, meta)
}

func powerOnIfNeeded(d *schema.ResourceData, vm *govcd.VM, vmStatusBefore string) error {
	vmStatus, err := vm.GetStatus()
	if err != nil {
		return fmt.Errorf("error getting VM status before ensuring it is powered on: %s", err)
	}

	if vmStatusBefore == "POWERED_ON" && vmStatus != "POWERED_ON" && d.Get("bus_type").(string) == "ide" && d.Get("allow_vm_reboot").(bool) {
		log.Printf("[DEBUG] Powering on VM %s after adding internal disk.", vm.VM.Name)

		task, err := vm.PowerOn()
		if err != nil {
			return fmt.Errorf("error powering on VM for adding/updating internal disk: %s", err)
		}
		err = task.WaitTaskCompletion()
		if err != nil {
			return fmt.Errorf(errorCompletingTask, err)
		}
	}
	return nil
}

func powerOffIfNeeded(d *schema.ResourceData, vm *govcd.VM) (string, error) {
	vmStatus, err := vm.GetStatus()
	if err != nil {
		return "", fmt.Errorf("error getting VM status before ensuring it is powered off: %s", err)
	}
	vmStatusBefore := vmStatus

	if vmStatus != "POWERED_OFF" && d.Get("bus_type").(string) == "ide" && d.Get("allow_vm_reboot").(bool) {
		log.Printf("[DEBUG] Powering off VM %s for adding/updating internal disk.", vm.VM.Name)

		task, err := vm.PowerOff()
		if err != nil {
			return vmStatusBefore, fmt.Errorf("error powering off VM for adding internal disk: %s", err)
		}
		err = task.WaitTaskCompletion()
		if err != nil {
			return vmStatusBefore, fmt.Errorf(errorCompletingTask, err)
		}
	}
	return vmStatusBefore, nil
}

// resourceVmInternalDiskDelete deletes disk from VM
func resourceVmInternalDiskDelete(d *schema.ResourceData, m interface{}) error {
	vcdClient := m.(*VCDClient)

	vcdClient.lockParentVm(d)
	defer vcdClient.unLockParentVm(d)

	vm, _, err := getVm(vcdClient, d)
	if err != nil {
		return err
	}

	vmStatusBefore, err := powerOffIfNeeded(d, vm)
	if err != nil {
		return err
	}

	err = vm.DeleteInternalDisk(d.Id())
	if err != nil {
		return fmt.Errorf("[resourceVmInternalDiskDelete] failed to delete internal disk: %s", err)
	}

	err = powerOnIfNeeded(d, vm, vmStatusBefore)
	if err != nil {
		return err
	}

	log.Printf("[TRACE] VM internal disk %s deleted", d.Id())
	d.SetId("")
	return nil
}

func getVm(vcdClient *VCDClient, d *schema.ResourceData) (*govcd.VM, *govcd.Vdc, error) {
	_, vdc, err := vcdClient.GetOrgAndVdcFromResource(d)
	if err != nil {
		return nil, nil, fmt.Errorf(errorRetrievingOrgAndVdc, err)
	}
	vapp, err := vdc.GetVAppByName(d.Get("vapp_name").(string), false)
	if err != nil {
		return nil, nil, fmt.Errorf("[getVm] failed to get vApp: %s", err)
	}
	vm, err := vapp.GetVMByName(d.Get("vm_name").(string), false)
	if err != nil {
		return nil, nil, fmt.Errorf("[getVm] failed to get VM: %s", err)
	}
	return vm, vdc, err
}

// Update the resource
func resourceVmInternalDiskUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[TRACE] Update Internal Disk with ID: %s started.", d.Id())
	vcdClient := meta.(*VCDClient)

	vcdClient.lockParentVm(d)
	defer vcdClient.unLockParentVm(d)

	// ignore only allow_vm_reboot change, allows to avoid empty update
	if d.HasChange("allow_vm_reboot") && !d.HasChange("iops") && !d.HasChange("size_in_mb") && !d.HasChange("storage_profile") {
		return nil
	}
	vm, vdc, err := getVm(vcdClient, d)
	if err != nil {
		return err
	}

	// has refresh inside
	vmStatusBefore, err := powerOffIfNeeded(d, vm)
	if err != nil {
		return err
	}

	diskSettingsToUpdate, err := vm.GetInternalDiskById(d.Id(), false)
	if err != nil {
		return err
	}
	log.Printf("[TRACE] Internal Disk with id %s found", d.Id())
	iops := int64(d.Get("iops").(int))
	diskSettingsToUpdate.Iops = &iops
	diskSettingsToUpdate.SizeMb = int64(d.Get("size_in_mb").(int))
	// Note can't change adapter type, bus number, unit number as vSphere changes diskId

	var storageProfilePrt *types.Reference
	var overrideVmDefault bool

	storageProfileName := d.Get("storage_profile").(string)
	if storageProfileName != "" {
		storageProfile, err := vdc.FindStorageProfileReference(storageProfileName)
		if err != nil {
			return fmt.Errorf("[Error] error retrieving storage profile %s : %s", storageProfileName, err)
		}
		storageProfilePrt = &storageProfile
		overrideVmDefault = true
	} else {
		storageProfilePrt = vm.VM.StorageProfile
		overrideVmDefault = false
	}

	diskSettingsToUpdate.StorageProfile = storageProfilePrt
	diskSettingsToUpdate.OverrideVmDefault = overrideVmDefault

	_, err = vm.UpdateInternalDisks(vm.VM.VmSpecSection)
	if err != nil {
		return err
	}

	err = powerOnIfNeeded(d, vm, vmStatusBefore)
	if err != nil {
		return err
	}

	log.Printf("[TRACE] Inernal Disk %s updated", d.Id())
	return resourceVmInternalDiskRead(d, meta)
}

// Retrieves internal disk from VM and updates terraform state
func resourceVmInternalDiskRead(d *schema.ResourceData, m interface{}) error {
	vcdClient := m.(*VCDClient)

	vm, _, err := getVm(vcdClient, d)
	if err != nil {
		return err
	}

	diskSettings, err := vm.GetInternalDiskById(d.Id(), true)
	if err == govcd.ErrorEntityNotFound {
		log.Printf("[DEBUG] Unable to find disk with Id: %s. Removing from tfstate", d.Id())
		d.SetId("")
		return nil
	}
	if err != nil {
		return err
	}

	_ = d.Set("bus_type", internalDiskBusTypesFromValues[strings.ToLower(diskSettings.AdapterType)])
	_ = d.Set("size_in_mb", diskSettings.SizeMb)
	_ = d.Set("bus_number", diskSettings.BusNumber)
	_ = d.Set("unit_number", diskSettings.UnitNumber)
	if diskSettings.ThinProvisioned != nil {
		_ = d.Set("thin_provisioned", *diskSettings.ThinProvisioned)
	}
	_ = d.Set("iops", diskSettings.Iops)
	_ = d.Set("storage_profile", diskSettings.StorageProfile.Name)

	return nil
}

var errHelpInternalDiskImport = fmt.Errorf(`resource id must be specified in one of these formats:
'org-name.vdc-name.vapp-name.vm-name.my-internal-disk-id' to import by rule id
'list@org-name.vdc-name.vapp-name.vm-name' to get a list of internal disks with their IDs`)

// resourceVcdVmInternalDiskImport is responsible for importing the resource.
// The following steps happen as part of import
// 1. The user supplies `terraform import _resource_name_ _the_id_string_` command
// 2a. If the `_the_id_string_` contains a dot formatted path to resource as in the example below
// it will try to import it. If it is found - the ID is set
// 2b. If the `_the_id_string_` starts with `list@` and contains path to VM name similar to
// `list@org-name.vdc-name.vapp-name.vm-name` then the function lists all internal disks and their IDs in that VM
// 3. The functions splits the dot-formatted path and tries to lookup the object
// 4. If the lookup succeeds it sets the ID field for `_resource_name_` resource in statefile
// (the resource must be already defined in .tf config otherwise `terraform import` will complain)
// 5. `terraform refresh` is being implicitly launched. The Read method looks up all other fields
// based on the known ID of object.
//
// Example resource name (_resource_name_): vcd_vm_internal_disk.my-disk
// Example import path (_the_id_string_): org-name.vdc-name.vapp-name.vm-name.my-internal-disk-id
// Example list path (_the_id_string_): list@org-name.vdc-name.vapp-name.vm-name
func resourceVcdVmInternalDiskImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	var commandOrgName, orgName, vdcName, vappName, vmName, diskId string

	resourceURI := strings.Split(d.Id(), ImportSeparator)

	log.Printf("[DEBUG] importing vcd_vm_internal_disk resource with provided id %s", d.Id())

	if len(resourceURI) != 4 && len(resourceURI) != 5 {
		return nil, errHelpInternalDiskImport
	}

	if strings.Contains(d.Id(), "list@") {
		commandOrgName, vdcName, vappName, vmName = resourceURI[0], resourceURI[1], resourceURI[2], resourceURI[3]
		commandOrgNameSplit := strings.Split(commandOrgName, "@")
		if len(commandOrgNameSplit) != 2 {
			return nil, errHelpDiskImport
		}
		orgName = commandOrgNameSplit[1]
		return listInternalDisksForImport(meta, orgName, vdcName, vappName, vmName)
	} else {
		orgName, vdcName, vappName, vmName, diskId = resourceURI[0], resourceURI[1], resourceURI[2], resourceURI[3], resourceURI[4]
		return getInternalDiskForImport(d, meta, orgName, vdcName, vappName, vmName, diskId)
	}
}

func listInternalDisksForImport(meta interface{}, orgName, vdcName, vappName, vmName string) ([]*schema.ResourceData, error) {

	vcdClient := meta.(*VCDClient)
	_, vdc, err := vcdClient.GetOrgAndVdc(orgName, vdcName)
	if err != nil {
		return nil, fmt.Errorf(errorRetrievingOrgAndVdc, err)
	}
	vapp, err := vdc.GetVAppByName(vappName, false)
	if err != nil {
		return nil, fmt.Errorf("[Error] failed to get vApp: %s", err)
	}
	vm, err := vapp.GetVMByName(vmName, false)
	if err != nil {
		return nil, fmt.Errorf("[Error] failed to get VM: %s", err)
	}

	_, _ = fmt.Fprintln(getTerraformStdout(), "Retrieving all disks")
	if vm.VM.VmSpecSection.DiskSection == nil || vm.VM.VmSpecSection.DiskSection.DiskSettings == nil ||
		len(vm.VM.VmSpecSection.DiskSection.DiskSettings) == 0 {
		return nil, fmt.Errorf("no internal disks found on VM: %s", vmName)
	}

	writer := tabwriter.NewWriter(getTerraformStdout(), 0, 8, 1, '\t', tabwriter.AlignRight)

	fmt.Fprintln(writer, "No\tID\tBusType\tBusNumber\tUnitNumber\tSize\tStorageProfile\tIops\tThinProvisioned")
	fmt.Fprintln(writer, "--\t--\t-------\t---------\t----------\t----\t-------------\t----\t---------------")
	for index, disk := range vm.VM.VmSpecSection.DiskSection.DiskSettings {
		// API shows internal disk and independent disks in one list. If disk.Disk != nil then it's independent disk
		if disk.Disk == nil {
			fmt.Fprintf(writer, "%d\t%s\t%s\t%d\t%d\t%d\t%s\t%d\t%t\n", (index + 1), disk.DiskId, internalDiskBusTypesFromValues[disk.AdapterType], disk.BusNumber, disk.UnitNumber, disk.SizeMb,
				disk.StorageProfile.Name, *disk.Iops, *disk.ThinProvisioned)
		}
	}
	writer.Flush()

	return nil, fmt.Errorf("resource was not imported! %s", errHelpInternalDiskImport)
}

func getInternalDiskForImport(d *schema.ResourceData, meta interface{}, orgName, vdcName, vappName, vmName, diskId string) ([]*schema.ResourceData, error) {
	vcdClient := meta.(*VCDClient)
	_, vdc, err := vcdClient.GetOrgAndVdc(orgName, vdcName)
	if err != nil {
		return nil, fmt.Errorf(errorRetrievingOrgAndVdc, err)
	}
	vapp, err := vdc.GetVAppByName(vappName, false)
	if err != nil {
		return nil, fmt.Errorf("[Error] failed to get vApp: %s", err)
	}
	vm, err := vapp.GetVMByName(vmName, false)
	if err != nil {
		return nil, fmt.Errorf("[Error] failed to get VM: %s", err)
	}

	disk, err := vm.GetInternalDiskById(diskId, false)
	if err != nil {
		return []*schema.ResourceData{}, fmt.Errorf("unable to find internal disk with id %s: %s",
			d.Id(), err)
	}

	d.SetId(disk.DiskId)
	if vcdClient.Org != orgName {
		d.Set("org", orgName)
	}
	if vcdClient.Vdc != vdcName {
		d.Set("vdc", vdcName)
	}
	d.Set("vapp_name", vappName)
	d.Set("vm_name", vmName)
	d.Set("allow_vm_reboot", false)
	return []*schema.ResourceData{d}, nil
}
