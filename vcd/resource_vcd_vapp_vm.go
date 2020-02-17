package vcd

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"

	"github.com/hashicorp/terraform-plugin-sdk/helper/hashcode"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/vmware/go-vcloud-director/v2/govcd"
	"github.com/vmware/go-vcloud-director/v2/types/v56"
)

func resourceVcdVAppVm() *schema.Resource {

	return &schema.Resource{
		Create: resourceVcdVAppVmCreate,
		Update: resourceVcdVAppVmUpdate,
		Read:   resourceVcdVAppVmRead,
		Delete: resourceVcdVAppVmDelete,
		Importer: &schema.ResourceImporter{
			State: resourceVcdVappVmImport,
		},

		Schema: map[string]*schema.Schema{
			"vapp_name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The vApp this VM belongs to",
			},
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "A name for the VM, unique within the vApp",
			},
			"computer_name": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Computer name to assign to this virtual machine",
			},
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
			"template_name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of the vApp Template to use",
			},
			"catalog_name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The catalog name in which to find the given vApp Template",
			},
			"description": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The VM description",
				// Note: description is read only, as we lack the needed fields to set it at creation.
				// Currently, this field has the description of the OVA used to create the VM
			},
			"memory": &schema.Schema{
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "The amount of RAM (in MB) to allocate to the VM",
				ValidateFunc: validateMultipleOf4(),
			},
			"cpus": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     1,
				Description: "The number of virtual CPUs to allocate to the VM",
			},
			"cpu_cores": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     1,
				Description: "The number of cores per socket",
			},
			"ip": &schema.Schema{
				Computed:         true,
				ConflictsWith:    []string{"network"},
				Deprecated:       "In favor of network",
				DiffSuppressFunc: suppressIfIPIsOneOf(),
				ForceNew:         true,
				Optional:         true,
				Type:             schema.TypeString,
			},
			"mac": {
				Computed:      true,
				ConflictsWith: []string{"network"},
				Deprecated:    "In favor of network",
				Optional:      true,
				Type:          schema.TypeString,
			},
			"initscript": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Script to run on initial boot or with customization.force=true set",
			},
			"metadata": {
				Type:     schema.TypeMap,
				Optional: true,
				// For now underlying go-vcloud-director repo only supports
				// a value of type String in this map.
				Description: "Key value map of metadata to assign to this VM",
			},
			"href": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "VM Hyper Reference",
			},
			"accept_all_eulas": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Automatically accept EULA if OVA has it",
			},
			"power_on": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "A boolean value stating if this VM should be powered on",
			},
			"storage_profile": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Storage profile to override the default one",
			},
			"network_href": &schema.Schema{
				ConflictsWith: []string{"network"},
				Deprecated:    "In favor of network",
				Type:          schema.TypeString,
				Optional:      true,
			},
			"network": {
				ConflictsWith: []string{"ip", "network_name", "vapp_network_name", "network_href"},
				Optional:      true,
				Type:          schema.TypeList,
				Description:   " A block to define network interface. Multiple can be used.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Required:     true,
							Type:         schema.TypeString,
							ValidateFunc: validation.StringInSlice([]string{"vapp", "org", "none"}, false),
							Description:  "Network type to use: 'vapp', 'org' or 'none'. Use 'vapp' for vApp network, 'org' to attach Org VDC network. 'none' for empty NIC.",
						},
						"ip_allocation_mode": {
							Optional:     true,
							Type:         schema.TypeString,
							ValidateFunc: validation.StringInSlice([]string{"POOL", "DHCP", "MANUAL", "NONE"}, false),
							Description:  "IP address allocation mode. One of POOL, DHCP, MANUAL, NONE",
						},
						"name": {
							ForceNew:    false,
							Optional:    true, // In case of type = none it is not required
							Type:        schema.TypeString,
							Description: "Name of the network this VM should connect to. Always required except for `type` `NONE`",
						},
						"ip": {
							Computed:     true,
							Optional:     true,
							Type:         schema.TypeString,
							ValidateFunc: checkEmptyOrSingleIP(), // Must accept empty string to ease using HCL interpolation
							Description:  "IP of the VM. Settings depend on `ip_allocation_mode`. Omitted or empty for DHCP, POOL, NONE. Required for MANUAL",
						},
						"is_primary": {
							Default:  false,
							Optional: true,
							// By default if the value is omitted it will report schema change
							// on every terraform operation. The below function
							// suppresses such cases "" => "false" when applying.
							DiffSuppressFunc: falseBoolSuppress(),
							Type:             schema.TypeBool,
							Description:      "Set to true if network interface should be primary. First network card in the list will be primary by default",
						},
						"mac": {
							Computed:    true,
							Optional:    true,
							Type:        schema.TypeString,
							Description: "Mac address of network interface",
						},
						"adapter_type": {
							Type:             schema.TypeString,
							Computed:         true,
							Optional:         true,
							DiffSuppressFunc: suppressCase,
							Description:      "Network card adapter type. (e.g. 'E1000', 'E1000E', 'SRIOVETHERNETCARD', 'VMXNET3', 'PCNet32')",
						},
					},
				},
			},
			"network_name": &schema.Schema{
				ConflictsWith: []string{"network"},
				Deprecated:    "In favor of network",
				ForceNew:      true,
				Optional:      true,
				Type:          schema.TypeString,
			},
			"vapp_network_name": &schema.Schema{
				ConflictsWith: []string{"network"},
				Deprecated:    "In favor of network",
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
			},
			"disk": {
				Type: schema.TypeSet,
				Elem: &schema.Resource{Schema: map[string]*schema.Schema{
					"name": {
						Type:        schema.TypeString,
						Required:    true,
						Description: "Independent disk name",
					},
					"bus_number": {
						Type:        schema.TypeString,
						Required:    true,
						Description: "Bus number on which to place the disk controller",
					},
					"unit_number": {
						Type:        schema.TypeString,
						Required:    true,
						Description: "Unit number (slot) on the bus specified by BusNumber",
					},
					"size_in_mb": {
						Type:        schema.TypeInt,
						Computed:    true,
						Description: "The size of the disk in MB.",
					},
				}},
				Optional: true,
				Set:      resourceVcdVmIndependentDiskHash,
			},
			"override_template_disk": {
				Type:        schema.TypeSet,
				Optional:    true,
				ForceNew:    true,
				Description: "A block to match internal_disk interface in template. Multiple can be used. Disk will be matched by bus_type, bus_number and unit_number.",
				Elem: &schema.Resource{Schema: map[string]*schema.Schema{
					"bus_type": {
						Type:         schema.TypeString,
						Required:     true,
						ForceNew:     true,
						ValidateFunc: validation.StringInSlice([]string{"ide", "parallel", "sas", "paravirtual", "sata"}, false),
						Description:  "The type of disk controller. Possible values: ide, parallel( LSI Logic Parallel SCSI), sas(LSI Logic SAS (SCSI)), paravirtual(Paravirtual (SCSI)), sata",
					},
					"size_in_mb": {
						Type:        schema.TypeInt,
						ForceNew:    true,
						Required:    true,
						Description: "The size of the disk in MB.",
					},
					"bus_number": {
						Type:        schema.TypeInt,
						ForceNew:    true,
						Required:    true,
						Description: "The number of the SCSI or IDE controller itself.",
					},
					"unit_number": {
						Type:        schema.TypeInt,
						ForceNew:    true,
						Required:    true,
						Description: "The device number on the SCSI or IDE controller of the disk.",
					},
					"iops": {
						Type:        schema.TypeInt,
						ForceNew:    true,
						Optional:    true,
						Description: "Specifies the IOPS for the disk. Default - 0.",
					},
					"storage_profile": &schema.Schema{
						Type:        schema.TypeString,
						ForceNew:    true,
						Optional:    true,
						Description: "Storage profile to override the VM default one",
					},
				}},
			},
			"internal_disk": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "A block will show internal disk details",
				Elem: &schema.Resource{Schema: map[string]*schema.Schema{
					"disk_id": {
						Type:        schema.TypeString,
						Computed:    true,
						Description: "The disk ID.",
					},
					"bus_type": {
						Type:        schema.TypeString,
						Computed:    true,
						Description: "The type of disk controller. Possible values: ide, parallel( LSI Logic Parallel SCSI), sas(LSI Logic SAS (SCSI)), paravirtual(Paravirtual (SCSI)), sata",
					},
					"size_in_mb": {
						Type:        schema.TypeInt,
						Computed:    true,
						Description: "The size of the disk in MB.",
					},
					"bus_number": {
						Type:        schema.TypeInt,
						Computed:    true,
						Description: "The number of the SCSI or IDE controller itself.",
					},
					"unit_number": {
						Type:        schema.TypeInt,
						Computed:    true,
						Description: "The device number on the SCSI or IDE controller of the disk.",
					},
					"thin_provisioned": {
						Type:        schema.TypeBool,
						Computed:    true,
						Description: "Specifies whether the disk storage is pre-allocated or allocated on demand.",
					},
					"iops": {
						Type:        schema.TypeInt,
						Computed:    true,
						Description: "Specifies the IOPS for the disk. Default - 0.",
					},
					"storage_profile": &schema.Schema{
						Type:        schema.TypeString,
						Computed:    true,
						Description: "Storage profile to override the VM default one",
					},
				}},
			},
			"expose_hardware_virtualization": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Expose hardware-assisted CPU virtualization to guest OS.",
			},
			"guest_properties": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "Key/value settings for guest properties",
			},
			"customization": &schema.Schema{
				Optional:    true,
				MinItems:    1,
				MaxItems:    1,
				Type:        schema.TypeList,
				Description: "Guest customization block",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"force": {
							ValidateFunc: noopValueWarningValidator(true,
								"Using 'true' value for field 'vcd_vapp_vm.customization.force' will reboot VM on every 'terraform apply' operation"),
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
							// This settings is used as a 'flag' and it does not matter what is set in the
							// state. If it is 'true' - then it means that 'update' procedure must set the
							// VM for customization at next boot and reboot it.
							DiffSuppressFunc: suppressFalse(),
							Description:      "'true' value will cause the VM to reboot on every 'apply' operation",
						},
					},
				},
			},
		},
	}
}

func resourceVcdVAppVmCreate(d *schema.ResourceData, meta interface{}) error {
	vcdClient := meta.(*VCDClient)

	vcdClient.lockParentVapp(d)
	defer vcdClient.unLockParentVapp(d)

	org, vdc, err := vcdClient.GetOrgAndVdcFromResource(d)
	if err != nil {
		return fmt.Errorf(errorRetrievingOrgAndVdc, err)
	}

	catalog, err := org.GetCatalogByName(d.Get("catalog_name").(string), false)
	if err != nil {
		return fmt.Errorf("error finding catalog %s: %s", d.Get("catalog_name").(string), err)
	}

	catalogItem, err := catalog.GetCatalogItemByName(d.Get("template_name").(string), false)
	if err != nil {
		return fmt.Errorf("error finding catalog item: %s", err)
	}

	vappTemplate, err := catalogItem.GetVAppTemplate()
	if err != nil {
		return fmt.Errorf("error finding VAppTemplate: %s", err)
	}

	acceptEulas := d.Get("accept_all_eulas").(bool)

	vapp, err := vdc.GetVAppByName(d.Get("vapp_name").(string), false)
	if err != nil {
		return fmt.Errorf("error finding vApp: %s", err)
	}

	// Determine whether we use new 'networks' or deprecated network configuration and process inputs based on it.
	// TODO v3.0 remove else branch once 'network_name', 'vapp_network_name', 'ip' are deprecated
	networkConnectionSection := types.NetworkConnectionSection{}
	if len(d.Get("network").([]interface{})) > 0 {
		networkConnectionSection, err = networksToConfig(d.Get("network").([]interface{}), vdc, *vapp, vcdClient)
	} else {
		networkConnectionSection, err = deprecatedNetworksToConfig(d.Get("network_name").(string),
			d.Get("vapp_network_name").(string), d.Get("ip").(string), vdc, *vapp, vcdClient)
	}
	if err != nil {
		return fmt.Errorf("unable to process network configuration: %s", err)
	}

	log.Printf("[TRACE] Creating VM: %s", d.Get("name").(string))
	var storageProfile types.Reference
	storageProfilePtr := &storageProfile
	storageProfileName := d.Get("storage_profile").(string)
	if storageProfileName != "" {
		storageProfile, err = vdc.FindStorageProfileReference(storageProfileName)
		if err != nil {
			return fmt.Errorf("[vm creation] error retrieving storage profile %s : %s", storageProfileName, err)
		}
	} else {
		storageProfilePtr = nil
	}
	task, err := vapp.AddNewVMWithStorageProfile(d.Get("name").(string), vappTemplate, &networkConnectionSection, storageProfilePtr, acceptEulas)
	if err != nil {
		return fmt.Errorf("[VM creation] error adding VM: %s", err)
	}
	err = task.WaitTaskCompletion()
	if err != nil {
		return fmt.Errorf(errorCompletingTask, err)
	}

	vmName := d.Get("name").(string)
	vm, err := vapp.GetVMByName(vmName, true)

	if err != nil {
		d.SetId("")
		return fmt.Errorf("[VM creation] error getting VM %s : %s", vmName, err)
	}

	// The below operation assumes VM is powered off and does not check for it because VM is being
	// powered on in the last stage of create/update cycle
	if d.Get("expose_hardware_virtualization").(bool) {

		task, err := vm.ToggleHardwareVirtualization(true)
		if err != nil {
			return fmt.Errorf("error enabling hardware assisted virtualization: %s", err)
		}
		err = task.WaitTaskCompletion()

		if err != nil {
			return fmt.Errorf(errorCompletingTask, err)
		}
	}

	// for back compatibility we allow to set computer name from `name` if computer_name isn't provided
	var computerName string
	if cName, ok := d.GetOk("computer_name"); ok {
		computerName = cName.(string)
	} else {
		computerName = d.Get("name").(string)
	}

	if initScript, ok := d.GetOk("initscript"); ok {
		if _, ok := d.GetOk("computer_name"); !ok {
			_, _ = fmt.Fprint(getTerraformStdout(), "WARNING of DEPRECATED behavior: when `initscript` is set,"+
				" VM `name` is used as a computer name - this behavior will be removed in future versions, hence please use the new `computer_name` field instead\n")
		}
		task, err := vm.RunCustomizationScript(computerName, initScript.(string))
		if err != nil {
			return fmt.Errorf("error with init script setting: %s", err)
		}
		err = task.WaitTaskCompletion()
		if err != nil {
			return fmt.Errorf(errorCompletingTask, err)
		}
	} else if newComputerName, ok := d.GetOk("computer_name"); ok {
		customizationSection, err := vm.GetGuestCustomizationSection()
		if err != nil {
			return fmt.Errorf("error get customization section before applying computer name: %s", err)
		}
		customizationSection.ComputerName = newComputerName.(string)
		_, err = vm.SetGuestCustomizationSection(customizationSection)
		if err != nil {
			return fmt.Errorf("error with applying computer name: %s", err)
		}
	}

	if _, ok := d.GetOk("guest_properties"); ok {
		vmProperties, err := getGuestProperties(d)
		if err != nil {
			return fmt.Errorf("unable to convert guest properties to data structure")
		}

		log.Printf("[TRACE] Setting VM guest properties")
		_, err = vm.SetProductSectionList(vmProperties)
		if err != nil {
			return fmt.Errorf("error setting guest properties: %s", err)
		}
	}

	d.SetId(vm.VM.ID)

	// update existing internal disks in template
	err = updateTemplateInternalDisks(d, meta, *vm)
	if err != nil {
		d.Set("override_template_disk", nil)
		return fmt.Errorf("error managing internal disks : %s", err)
	} else {
		// add details of internal disk to state
		errReadInternalDisk := updateStateOfInternalDisks(d, *vm)
		if errReadInternalDisk != nil {
			d.Set("internal_disk", nil)
			log.Printf("error reading interal disks : %s", errReadInternalDisk)
		}
	}

	// TODO do not trigger resourceVcdVAppVmUpdate from create. These must be separate actions.
	err = resourceVcdVAppVmUpdateExecute(d, meta)
	if err != nil {
		errAttachedDisk := updateStateOfAttachedDisks(d, *vm, vdc)
		if errAttachedDisk != nil {
			d.Set("disk", nil)
			return fmt.Errorf("error reading attached disks : %s and internal error : %s", errAttachedDisk, err)
		}
		return err
	}

	return nil
}

// Adds existing org VDC network to VM network configuration
// Returns configured OrgVDCNetwork for Vm, networkName, error if any occur
func addVdcNetwork(networkNameToAdd string, vdc *govcd.Vdc, vapp govcd.VApp, vcdClient *VCDClient) (*types.OrgVDCNetwork, error) {
	if networkNameToAdd == "" {
		return &types.OrgVDCNetwork{}, fmt.Errorf("'network_name' must be valid when adding VM to raw vApp")
	}

	network, err := vdc.GetOrgVdcNetworkByName(networkNameToAdd, false)
	if err != nil {
		return &types.OrgVDCNetwork{}, fmt.Errorf("network %s wasn't found as VDC network", networkNameToAdd)
	}
	vdcNetwork := network.OrgVDCNetwork

	vAppNetworkConfig, err := vapp.GetNetworkConfig()
	if err != nil {
		return &types.OrgVDCNetwork{}, fmt.Errorf("could not get network config: %s", err)
	}

	isAlreadyVappNetwork := false
	for _, networkConfig := range vAppNetworkConfig.NetworkConfig {
		if networkConfig.NetworkName == networkNameToAdd {
			log.Printf("[TRACE] VDC network found as vApp network: %s", networkNameToAdd)
			isAlreadyVappNetwork = true
		}
	}

	if !isAlreadyVappNetwork {
		task, err := vapp.AddRAWNetworkConfig([]*types.OrgVDCNetwork{vdcNetwork})
		if err != nil {
			return &types.OrgVDCNetwork{}, fmt.Errorf("error assigning network to vApp: %s", err)
		}
		err = task.WaitTaskCompletion()
		if err != nil {
			return &types.OrgVDCNetwork{}, fmt.Errorf("error assigning network to vApp:: %s", err)
		}
	}

	return vdcNetwork, nil
}

// isItVappNetwork checks if it is a genuine vApp network (not only attached to vApp)
func isItVappNetwork(vAppNetworkName string, vapp govcd.VApp) (bool, error) {

	vAppNetworkConfig, err := vapp.GetNetworkConfig()
	if err != nil {
		return false, fmt.Errorf("error getting vApp networks: %s", err)
	}
	// If vApp network is "isolated" and has no ParentNetwork - it is a vApp network.
	// https://code.vmware.com/apis/72/vcloud/doc/doc/types/NetworkConfigurationType.html
	for _, networkConfig := range vAppNetworkConfig.NetworkConfig {
		if networkConfig.NetworkName == vAppNetworkName &&
			networkConfig.Configuration.ParentNetwork == nil &&
			networkConfig.Configuration.FenceMode == "isolated" {
			log.Printf("[TRACE] vApp network found: %s", vAppNetworkName)
			return true, nil
		}
	}

	return false, fmt.Errorf("configured vApp network isn't found: %s", err)
}

type diskParams struct {
	name       string
	busNumber  *int
	unitNumber *int
}

func expandDisksProperties(v interface{}) ([]diskParams, error) {
	v = v.(*schema.Set).List()
	l := v.([]interface{})
	diskParamsArray := make([]diskParams, 0, len(l))

	for _, raw := range l {
		if raw == nil {
			continue
		}
		original := raw.(map[string]interface{})
		addParams := diskParams{name: original["name"].(string)}

		busNumber := original["bus_number"].(string)
		if busNumber != "" {
			convertedBusNumber, err := strconv.Atoi(busNumber)
			if err != nil {
				return nil, fmt.Errorf("value `%s` bus_number is not number. err: %s", busNumber, err)
			}
			addParams.busNumber = &convertedBusNumber
		}

		unitNumber := original["unit_number"].(string)
		if unitNumber != "" {
			convertedUnitNumber, err := strconv.Atoi(unitNumber)
			if err != nil {
				return nil, fmt.Errorf("value `%s` unit_number is not number. err: %s", unitNumber, err)
			}
			addParams.unitNumber = &convertedUnitNumber
		}

		diskParamsArray = append(diskParamsArray, addParams)
	}
	return diskParamsArray, nil
}

func getVmIndependentDisks(vm govcd.VM) []string {

	var disks []string
	// We use VirtualHardwareSection because in time of implementation we didn't have access to VmSpecSection which we used for internal disks.
	for _, item := range vm.VM.VirtualHardwareSection.Item {
		// disk resource type is 17
		if item.ResourceType == 17 && item.HostResource[0].Disk != "" {
			disks = append(disks, item.HostResource[0].Disk)
		}
	}
	return disks
}

func resourceVcdVAppVmUpdate(d *schema.ResourceData, meta interface{}) error {
	vcdClient := meta.(*VCDClient)

	// When there is more then one VM in a vApp Terraform will try to parallelise their creation.
	// However, vApp throws errors when simultaneous requests are executed.
	// To avoid them, below block is using mutex as a workaround,
	// so that the one vApp VMs are created not in parallelisation.

	vcdClient.lockParentVapp(d)
	defer vcdClient.unLockParentVapp(d)

	return resourceVcdVAppVmUpdateExecute(d, meta)
}

func resourceVcdVAppVmUpdateExecute(d *schema.ResourceData, meta interface{}) error {

	vcdClient := meta.(*VCDClient)

	_, vdc, err := vcdClient.GetOrgAndVdcFromResource(d)
	if err != nil {
		return fmt.Errorf(errorRetrievingOrgAndVdc, err)
	}

	vapp, err := vdc.GetVAppByName(d.Get("vapp_name").(string), false)

	if err != nil {
		return fmt.Errorf("error finding vApp: %s", err)
	}

	identifier := d.Id()
	if identifier == "" {
		identifier = d.Get("name").(string)
	}
	if identifier == "" {
		return fmt.Errorf("[VM update] neither name or ID was set")
	}

	vm, err := vapp.GetVMByNameOrId(identifier, false)

	if err != nil {
		d.SetId("")
		return fmt.Errorf("[VM update] error getting VM %s: %s", identifier, err)
	}

	vmStatusBeforeUpdate, err := vm.GetStatus()
	if err != nil {
		return fmt.Errorf("[VM update] error getting VM (%s) status before update: %s", identifier, err)
	}

	if d.HasChange("guest_properties") {
		vmProperties, err := getGuestProperties(d)
		if err != nil {
			return fmt.Errorf("unable to convert guest properties to data structure")
		}

		log.Printf("[TRACE] Updating VM guest properties")
		_, err = vm.SetProductSectionList(vmProperties)
		if err != nil {
			return fmt.Errorf("error setting guest properties: %s", err)
		}
	}
	// Check if the user requested for forced customization of VM
	customizationNeeded := isForcedCustomization(d.Get("customization"))

	// VM does not have to be in POWERED_OFF state for metadata operations
	if d.HasChange("metadata") {
		oldRaw, newRaw := d.GetChange("metadata")
		oldMetadata := oldRaw.(map[string]interface{})
		newMetadata := newRaw.(map[string]interface{})
		var toBeRemovedMetadata []string
		// Check if any key in old metadata was removed in new metadata.
		// Creates a list of keys to be removed.
		for k := range oldMetadata {
			if _, ok := newMetadata[k]; !ok {
				toBeRemovedMetadata = append(toBeRemovedMetadata, k)
			}
		}
		for _, k := range toBeRemovedMetadata {
			task, err := vm.DeleteMetadata(k)
			if err != nil {
				return fmt.Errorf("error deleting metadata: %s", err)
			}
			err = task.WaitTaskCompletion()
			if err != nil {
				return fmt.Errorf(errorCompletingTask, err)
			}
		}
		for k, v := range newMetadata {
			task, err := vm.AddMetadata(k, v.(string))
			if err != nil {
				return fmt.Errorf("error adding metadata: %s", err)
			}
			err = task.WaitTaskCompletion()
			if err != nil {
				return fmt.Errorf(errorCompletingTask, err)
			}
		}
	}

	if d.HasChange("memory") || d.HasChange("cpus") || d.HasChange("cpu_cores") || d.HasChange("power_on") || d.HasChange("disk") ||
		d.HasChange("expose_hardware_virtualization") || d.HasChange("network") || d.HasChange("computer_name") {

		log.Printf("[TRACE] VM %s has changes: memory(%t), cpus(%t), cpu_cores(%t), power_on(%t), disk(%t), expose_hardware_virtualization(%t),"+
			" network(%t), computer_name(%t)",
			vm.VM.Name, d.HasChange("memory"), d.HasChange("cpus"), d.HasChange("cpu_cores"), d.HasChange("power_on"), d.HasChange("disk"),
			d.HasChange("expose_hardware_virtualization"), d.HasChange("network"), d.HasChange("computer_name"))

		// If customization is not requested then a simple shutdown is enough
		if vmStatusBeforeUpdate != "POWERED_OFF" && !customizationNeeded {
			log.Printf("[DEBUG] Powering off VM %s for offline update. Previous state %s",
				vm.VM.Name, vmStatusBeforeUpdate)
			task, err := vm.PowerOff()
			if err != nil {
				return fmt.Errorf("error Powering Off: %s", err)
			}
			err = task.WaitTaskCompletion()
			if err != nil {
				return fmt.Errorf(errorCompletingTask, err)
			}
		}

		// If customization was requested then a shutdown with undeploy is needed
		if vmStatusBeforeUpdate != "POWERED_OFF" && customizationNeeded {
			log.Printf("[DEBUG] Un-deploying VM %s for offline update. Previous state %s",
				vm.VM.Name, vmStatusBeforeUpdate)
			task, err := vm.Undeploy()
			if err != nil {
				return fmt.Errorf("error triggering undeploy for VM %s: %s", vm.VM.Name, err)
			}
			err = task.WaitTaskCompletion()
			if err != nil {
				return fmt.Errorf("error waiting for undeploy task for VM %s: %s", vm.VM.Name, err)
			}
		}

		// detaching independent disks - only possible when VM power off
		if d.HasChange("disk") {
			err = attachDetachDisks(d, *vm, vdc)
			if err != nil {
				errAttachedDisk := updateStateOfAttachedDisks(d, *vm, vdc)
				if errAttachedDisk != nil {
					d.Set("disk", nil)
					return fmt.Errorf("error reading attached disks : %s and internal error : %s", errAttachedDisk, err)
				}
				return fmt.Errorf("error attaching-detaching  disks when updating resource : %s", err)
			}
		}

		if d.HasChange("memory") {

			task, err := vm.ChangeMemorySize(d.Get("memory").(int))
			if err != nil {
				return fmt.Errorf("error changing memory size: %s", err)
			}

			err = task.WaitTaskCompletion()
			if err != nil {
				return err
			}
		}

		if d.HasChange("cpus") || d.HasChange("cpu_cores") {
			var task govcd.Task
			var err error
			if d.Get("cpu_cores") != nil {
				coreCounts := d.Get("cpu_cores").(int)
				task, err = vm.ChangeCPUCountWithCore(d.Get("cpus").(int), &coreCounts)
			} else {
				task, err = vm.ChangeCPUCount(d.Get("cpus").(int))
			}
			if err != nil {
				return fmt.Errorf("error changing cpu count: %s", err)
			}

			err = task.WaitTaskCompletion()
			if err != nil {
				return fmt.Errorf(errorCompletingTask, err)
			}
		}

		if d.HasChange("expose_hardware_virtualization") {

			task, err := vm.ToggleHardwareVirtualization(d.Get("expose_hardware_virtualization").(bool))
			if err != nil {
				return fmt.Errorf("error changing hardware assisted virtualization: %s", err)
			}

			err = task.WaitTaskCompletion()
			if err != nil {
				return err
			}
		}

		if d.HasChange("network") {
			networkConnectionSection, err := networksToConfig(d.Get("network").([]interface{}), vdc, *vapp, vcdClient)
			if err != nil {
				return fmt.Errorf("unable to setup network configuration for update: %s", err)
			}
			err = vm.UpdateNetworkConnectionSection(&networkConnectionSection)
			if err != nil {
				return fmt.Errorf("unable to update network configuration: %s", err)
			}
		}

		// we pass init script, to not override with empty one
		if d.HasChange("computer_name") {
			customizationSection, err := vm.GetGuestCustomizationSection()
			if err != nil {
				return fmt.Errorf("error get customization section before applying computer name: %s", err)
			}
			customizationSection.ComputerName = d.Get("computer_name").(string)
			_, err = vm.SetGuestCustomizationSection(customizationSection)
			if err != nil {
				return fmt.Errorf("error with applying computer name: %s", err)
			}
		}

	}

	// If the VM was powered off during update but it has to be powered off
	if d.Get("power_on").(bool) {
		vmStatus, err := vm.GetStatus()
		if err != nil {
			return fmt.Errorf("error getting VM status before ensuring it is powered on: %s", err)
		}
		log.Printf("[DEBUG] Powering on VM %s after update. Previous state %s", vm.VM.Name, vmStatus)

		// Simply power on if customization is not requested
		if !customizationNeeded && vmStatus != "POWERED_ON" {
			task, err := vm.PowerOn()
			if err != nil {
				return fmt.Errorf("error powering on: %s", err)
			}
			err = task.WaitTaskCompletion()
			if err != nil {
				return fmt.Errorf(errorCompletingTask, err)
			}
		}

		// When customization is requested VM must be un-deployed before starting it
		if customizationNeeded {
			log.Printf("[TRACE] forced customization for VM %s was requested. Current state %s",
				vm.VM.Name, vmStatus)

			if vmStatus != "POWERED_OFF" {
				log.Printf("[TRACE] VM %s is in state %s. Un-deploying", vm.VM.Name, vmStatus)
				task, err := vm.Undeploy()
				if err != nil {
					return fmt.Errorf("error triggering undeploy for VM %s: %s", vm.VM.Name, err)
				}
				err = task.WaitTaskCompletion()
				if err != nil {
					return fmt.Errorf("error waiting for undeploy task for VM %s: %s", vm.VM.Name, err)
				}
			}

			log.Printf("[TRACE] Powering on VM %s with forced customization", vm.VM.Name)
			err = vm.PowerOnAndForceCustomization()
			if err != nil {
				return fmt.Errorf("failed powering on with customization: %s", err)
			}
		}
	}

	return resourceVcdVAppVmRead(d, meta)
}

// updates attached disks to latest state. Removed not needed and add new ones
func attachDetachDisks(d *schema.ResourceData, vm govcd.VM, vdc *govcd.Vdc) error {
	oldValues, newValues := d.GetChange("disk")

	attachDisks := newValues.(*schema.Set).Difference(oldValues.(*schema.Set))
	detachDisks := oldValues.(*schema.Set).Difference(newValues.(*schema.Set))

	removeDiskProperties, err := expandDisksProperties(detachDisks)
	if err != nil {
		return err
	}

	for _, diskData := range removeDiskProperties {
		disk, err := vdc.QueryDisk(diskData.name)
		if err != nil {
			return fmt.Errorf("did not find disk `%s`: %s", diskData.name, err)
		}

		attachParams := &types.DiskAttachOrDetachParams{Disk: &types.Reference{HREF: disk.Disk.HREF}}
		if diskData.unitNumber != nil {
			attachParams.UnitNumber = diskData.unitNumber
		}
		if diskData.busNumber != nil {
			attachParams.BusNumber = diskData.busNumber
		}

		task, err := vm.DetachDisk(attachParams)
		if err != nil {
			return fmt.Errorf("error detaching disk `%s` to vm %s", diskData.name, err)
		}
		err = task.WaitTaskCompletion()
		if err != nil {
			return fmt.Errorf("error waiting for task to complete detaching disk `%s` to vm %s", diskData.name, err)
		}
	}

	// attach new independent disks
	newDiskProperties, err := expandDisksProperties(attachDisks)
	if err != nil {
		return err
	}

	sort.SliceStable(newDiskProperties, func(i, j int) bool {
		if newDiskProperties[i].busNumber == newDiskProperties[j].busNumber {
			return *newDiskProperties[i].unitNumber > *newDiskProperties[j].unitNumber
		}
		return *newDiskProperties[i].busNumber > *newDiskProperties[j].busNumber
	})

	for _, diskData := range newDiskProperties {
		disk, err := vdc.QueryDisk(diskData.name)
		if err != nil {
			return fmt.Errorf("did not find disk `%s`: %s", diskData.name, err)
		}

		attachParams := &types.DiskAttachOrDetachParams{Disk: &types.Reference{HREF: disk.Disk.HREF}}
		if diskData.unitNumber != nil {
			attachParams.UnitNumber = diskData.unitNumber
		}
		if diskData.busNumber != nil {
			attachParams.BusNumber = diskData.busNumber
		}

		task, err := vm.AttachDisk(attachParams)
		if err != nil {
			return fmt.Errorf("error attaching disk `%s` to vm %s", diskData.name, err)
		}
		err = task.WaitTaskCompletion()
		if err != nil {
			return fmt.Errorf("error waiting for task to complete attaching disk `%s` to vm %s", diskData.name, err)
		}
	}
	return nil
}

func resourceVcdVAppVmRead(d *schema.ResourceData, meta interface{}) error {
	return genericVcdVAppVmRead(d, meta, "resource")
}

func genericVcdVAppVmRead(d *schema.ResourceData, meta interface{}, origin string) error {
	vcdClient := meta.(*VCDClient)

	_, vdc, err := vcdClient.GetOrgAndVdcFromResource(d)
	if err != nil {
		return fmt.Errorf("[VM read ]"+errorRetrievingOrgAndVdc, err)
	}

	vappName := d.Get("vapp_name").(string)
	vapp, err := vdc.GetVAppByName(vappName, false)

	if err != nil {
		return fmt.Errorf("[VM read] error finding vApp: %s", err)
	}

	identifier := d.Id()
	if identifier == "" {
		identifier = d.Get("name").(string)
	}
	if identifier == "" {
		return fmt.Errorf("[VM read] neither name or ID were set for this VM")
	}
	vm, err := vapp.GetVMByNameOrId(identifier, false)

	if err != nil {
		if origin == "resource" {
			log.Printf("[DEBUG] Unable to find VM. Removing from tfstate")
			d.SetId("")
			return nil
		}
		return fmt.Errorf("[VM read] error getting VM : %s", err)
	}

	// org, vdc, and vapp_name are already implicitly set
	_ = d.Set("name", vm.VM.Name)
	_ = d.Set("description", vm.VM.Description)
	d.SetId(vm.VM.ID)

	var networkName string
	var vappNetworkName string
	networkNameRaw, ok := d.GetOk("network_name")
	if ok {
		networkName = networkNameRaw.(string)
	}
	vappNetworkNameRaw, ok := d.GetOk("vapp_network_name")
	if ok {
		vappNetworkName = vappNetworkNameRaw.(string)
	}
	// Read either new or deprecated networks configuration based on which are used
	switch {
	// TODO v3.0 remove this case block when we cleanup deprecated 'ip' and 'network_name' attributes
	case networkName != "" || vappNetworkName != "":
		ip, mac, err := deprecatedReadNetworks(networkName, vappNetworkName, *vm)
		if err != nil {
			return fmt.Errorf("[ VM read] failed reading network details: %s", err)
		}
		_ = d.Set("ip", ip)
		_ = d.Set("mac", mac)
		// TODO v3.0 EO remove this case block when we cleanup deprecated 'ip' and 'network_name' attributes
	default:
		networks, err := readNetworks(*vm, *vapp)
		if err != nil {
			return fmt.Errorf("[ VM read] failed reading network details: %s", err)
		}

		err = d.Set("network", networks)
		if err != nil {
			return err
		}
	}

	_ = d.Set("href", vm.VM.HREF)
	_ = d.Set("expose_hardware_virtualization", vm.VM.NestedHypervisorEnabled)

	cpus := 0
	coresPerSocket := 0
	memory := 0
	for _, item := range vm.VM.VirtualHardwareSection.Item {
		if item.ResourceType == 3 {
			cpus += item.VirtualQuantity
			coresPerSocket = item.CoresPerSocket
		}
		if item.ResourceType == 4 {
			memory = item.VirtualQuantity
		}
	}
	_ = d.Set("memory", memory)
	_ = d.Set("cpus", cpus)
	_ = d.Set("cpu_cores", coresPerSocket)

	metadata, err := vm.GetMetadata()
	if err != nil {
		return fmt.Errorf("[vm read] get metadata: %s", err)
	}
	err = d.Set("metadata", getMetadataStruct(metadata.MetadataEntry))
	if err != nil {
		return fmt.Errorf("[VM read] set metadata: %s", err)
	}

	if vm.VM.StorageProfile != nil {
		_ = d.Set("storage_profile", vm.VM.StorageProfile.Name)
	}

	// update guest properties
	guestProperties, err := vm.GetProductSectionList()
	if err != nil {
		return fmt.Errorf("[VM read] unable to read guest properties: %s", err)
	}

	err = setGuestProperties(d, guestProperties)
	if err != nil {
		return fmt.Errorf("[VM read] unable to set guest properties in state: %s", err)
	}

	err = updateStateOfInternalDisks(d, *vm)
	if err != nil {
		d.Set("internal_disk", nil)
		return fmt.Errorf("[VM read] error reading internal disks : %s", err)
	}

	err = updateStateOfAttachedDisks(d, *vm, vdc)
	if err != nil {
		d.Set("disk", nil)
		return fmt.Errorf("[VM read] error reading attached disks : %s", err)
	}

	guestCustomizationSection, err := vm.GetGuestCustomizationSection()
	if err != nil {
		return fmt.Errorf("[VM read] error reading guest customization : %s", err)
	}
	_ = d.Set("computer_name", guestCustomizationSection.ComputerName)

	return nil
}

func updateStateOfAttachedDisks(d *schema.ResourceData, vm govcd.VM, vdc *govcd.Vdc) error {

	existingDisks := getVmIndependentDisks(vm)
	transformed := schema.NewSet(resourceVcdVmIndependentDiskHash, []interface{}{})

	for _, existingDiskHref := range existingDisks {
		diskSettings, err := getIndependentDiskFromVmDisks(vm, existingDiskHref)
		if err != nil {
			return fmt.Errorf("did not find disk `%s`: %s", existingDiskHref, err)
		}
		newValues := map[string]interface{}{
			"name":        diskSettings.Disk.Name,
			"bus_number":  strconv.Itoa(diskSettings.BusNumber),
			"unit_number": strconv.Itoa(diskSettings.UnitNumber),
			"size_in_mb":  diskSettings.SizeMb,
		}

		transformed.Add(newValues)
	}

	return d.Set("disk", transformed)
}

// getIndependentDiskFromVmDisks finds independent disk in VM disk list.
func getIndependentDiskFromVmDisks(vm govcd.VM, diskHref string) (*types.DiskSettings, error) {
	if vm.VM.VmSpecSection == nil || vm.VM.VmSpecSection.DiskSection == nil {
		return nil, govcd.ErrorEntityNotFound
	}
	for _, disk := range vm.VM.VmSpecSection.DiskSection.DiskSettings {
		if disk.Disk != nil && disk.Disk.HREF == diskHref {
			return disk, nil
		}
	}
	return nil, govcd.ErrorEntityNotFound
}

func updateStateOfInternalDisks(d *schema.ResourceData, vm govcd.VM) error {
	err := vm.Refresh()
	if err != nil {
		return err
	}

	if vm.VM.VmSpecSection == nil || vm.VM.VmSpecSection.DiskSection == nil {
		return fmt.Errorf("[updateStateOfInternalDisks] VmSpecSection part is missing")
	}
	existingInternalDisks := vm.VM.VmSpecSection.DiskSection.DiskSettings
	var internalDiskList []map[string]interface{}
	for _, internalDisk := range existingInternalDisks {
		// API shows internal disk and independent disks in one list. If disk.Disk != nil then it's independent disk
		// We use VmSpecSection as it is newer type than VirtualHardwareSection. It is used by HTML5 vCD client, has easy understandable structure.
		// VirtualHardwareSection has undocumented relationships between elements and very hard to use without issues for internal disks.
		if internalDisk.Disk == nil {
			newValue := map[string]interface{}{
				"disk_id":          internalDisk.DiskId,
				"bus_type":         internalDiskBusTypesFromValues[internalDisk.AdapterType],
				"size_in_mb":       int(internalDisk.SizeMb),
				"bus_number":       internalDisk.BusNumber,
				"unit_number":      internalDisk.UnitNumber,
				"iops":             int(*internalDisk.Iops),
				"thin_provisioned": *internalDisk.ThinProvisioned,
				"storage_profile":  internalDisk.StorageProfile.Name,
			}
			internalDiskList = append(internalDiskList, newValue)
		}
	}

	return d.Set("internal_disk", internalDiskList)
}

func updateTemplateInternalDisks(d *schema.ResourceData, meta interface{}, vm govcd.VM) error {
	vcdClient := meta.(*VCDClient)

	_, vdc, err := vcdClient.GetOrgAndVdcFromResource(d)
	if err != nil {
		return fmt.Errorf(errorRetrievingOrgAndVdc, err)
	}

	if vm.VM.VmSpecSection == nil || vm.VM.VmSpecSection.DiskSection == nil {
		return fmt.Errorf("[updateTemplateInternalDisks] VmSpecSection part is missing")
	}

	diskSettings := vm.VM.VmSpecSection.DiskSection.DiskSettings

	var storageProfilePrt *types.Reference
	var overrideVmDefault bool

	internalDisksList := d.Get("override_template_disk").(*schema.Set).List()

	if len(internalDisksList) == 0 {
		return nil
	}

	for _, internalDisk := range internalDisksList {
		internalDiskProvidedConfig := internalDisk.(map[string]interface{})
		diskCreatedByTemplate := getMatchedDisk(internalDiskProvidedConfig, diskSettings)

		storageProfileName := internalDiskProvidedConfig["storage_profile"].(string)
		if storageProfileName != "" {
			storageProfile, err := vdc.FindStorageProfileReference(storageProfileName)
			if err != nil {
				return fmt.Errorf("[vm creation] error retrieving storage profile %s : %s", storageProfileName, err)
			}
			storageProfilePrt = &storageProfile
			overrideVmDefault = true
		} else {
			storageProfilePrt = vm.VM.StorageProfile
			overrideVmDefault = false
		}

		if diskCreatedByTemplate == nil {
			return fmt.Errorf("[vm creation] disk with bus type %s, bust number %d and unit number %d not found",
				internalDiskProvidedConfig["bus_type"].(string), internalDiskProvidedConfig["bus_number"].(int), internalDiskProvidedConfig["unit_number"].(int))
		}

		// Update details of internal disk for disk existing in template
		if value, ok := internalDiskProvidedConfig["iops"]; ok {
			iops := int64(value.(int))
			diskCreatedByTemplate.Iops = &iops
		}

		// value is required but not treated.
		isThinProvisioned := true
		diskCreatedByTemplate.ThinProvisioned = &isThinProvisioned

		diskCreatedByTemplate.SizeMb = int64(internalDiskProvidedConfig["size_in_mb"].(int))
		diskCreatedByTemplate.StorageProfile = storageProfilePrt
		diskCreatedByTemplate.OverrideVmDefault = overrideVmDefault
	}

	vmSpecSection := vm.VM.VmSpecSection
	vmSpecSection.DiskSection.DiskSettings = diskSettings
	_, err = vm.UpdateInternalDisks(vmSpecSection)
	if err != nil {
		return fmt.Errorf("error updating VM disks: %s", err)
	}

	return nil
}

// getMatchedDisk returns matched disk by adapter type, bus number and unit number
func getMatchedDisk(internalDiskProvidedConfig map[string]interface{}, diskSettings []*types.DiskSettings) *types.DiskSettings {
	for _, diskSetting := range diskSettings {
		if diskSetting.AdapterType == internalDiskBusTypes[internalDiskProvidedConfig["bus_type"].(string)] &&
			diskSetting.BusNumber == internalDiskProvidedConfig["bus_number"].(int) &&
			diskSetting.UnitNumber == internalDiskProvidedConfig["unit_number"].(int) {
			return diskSetting
		}
	}
	return nil
}

func resourceVcdVAppVmDelete(d *schema.ResourceData, meta interface{}) error {
	vcdClient := meta.(*VCDClient)

	vcdClient.lockParentVapp(d)
	defer vcdClient.unLockParentVapp(d)

	_, vdc, err := vcdClient.GetOrgAndVdcFromResource(d)
	if err != nil {
		return fmt.Errorf(errorRetrievingOrgAndVdc, err)
	}

	vapp, err := vdc.GetVAppByName(d.Get("vapp_name").(string), false)

	if err != nil {
		return fmt.Errorf("error finding vApp: %s", err)
	}

	identifier := d.Id()
	if identifier == "" {
		identifier = d.Get("name").(string)
	}
	if identifier == "" {
		return fmt.Errorf("[VM delete] neither ID or name provided")
	}
	vm, err := vapp.GetVMByNameOrId(identifier, false)

	if err != nil {
		return fmt.Errorf("[VM delete] error getting VM %s : %s", identifier, err)
	}

	status, err := vm.GetStatus()
	if err != nil {
		return fmt.Errorf("error getting VM status: %s", err)
	}

	log.Printf("[TRACE] VM Status: %s", status)
	if status != "POWERED_OFF" {
		log.Printf("[TRACE] Undeploying VM: %s", vm.VM.Name)
		task, err := vm.Undeploy()
		if err != nil {
			return fmt.Errorf("error Undeploying: %s", err)
		}

		err = task.WaitTaskCompletion()
		if err != nil {
			return fmt.Errorf("error Undeploying vApp: %s", err)
		}
	}

	// to avoid race condition for independent disks is attached or not - detach before removing vm
	existingDisks := getVmIndependentDisks(*vm)

	for _, existingDiskHref := range existingDisks {
		disk, err := vdc.GetDiskByHref(existingDiskHref)
		if err != nil {
			return fmt.Errorf("did not find disk `%s`: %s", existingDiskHref, err)
		}

		attachParams := &types.DiskAttachOrDetachParams{Disk: &types.Reference{HREF: disk.Disk.HREF}}
		task, err := vm.DetachDisk(attachParams)
		if err != nil {
			return fmt.Errorf("error detaching disk `%s`: %s", existingDiskHref, err)
		}
		err = task.WaitTaskCompletion()
		if err != nil {
			return fmt.Errorf("error waiting detaching disk task to finish`%s`: %s", existingDiskHref, err)
		}
	}

	log.Printf("[TRACE] Removing VM: %s", vm.VM.Name)

	err = vapp.RemoveVM(*vm)
	if err != nil {
		return fmt.Errorf("error deleting: %s", err)
	}

	return nil
}

func resourceVcdVmIndependentDiskHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-",
		m["name"].(string)))
	// We use the name and no other identifier to calculate the hash
	// With the VM resource, we assume that disks have a unique name.
	// In the event that this is not true, we return an error
	return hashcode.String(buf.String())
}

// networksToConfig converts terraform schema for 'networks' and converts to types.NetworkConnectionSection
// which is used for creating new VM
func networksToConfig(networks []interface{}, vdc *govcd.Vdc, vapp govcd.VApp, vcdClient *VCDClient) (types.NetworkConnectionSection, error) {

	networkConnectionSection := types.NetworkConnectionSection{}
	for index, singleNetwork := range networks {
		nic := singleNetwork.(map[string]interface{})
		netConn := &types.NetworkConnection{}

		networkName := nic["name"].(string)
		ipAllocationMode := nic["ip_allocation_mode"].(string)
		ip := nic["ip"].(string)
		macAddress, macIsSet := nic["mac"].(string)

		isPrimary := nic["is_primary"].(bool)
		if isPrimary {
			networkConnectionSection.PrimaryNetworkConnectionIndex = index
		}

		networkType := nic["type"].(string)
		if networkType == "org" {
			_, err := addVdcNetwork(networkName, vdc, vapp, vcdClient)
			if err != nil {
				return types.NetworkConnectionSection{}, fmt.Errorf("unable to attach org network %s: %s", networkName, err)
			}
		}

		if networkType == "vapp" {
			isVappNetwork, err := isItVappNetwork(networkName, vapp)
			if err != nil {
				return types.NetworkConnectionSection{}, fmt.Errorf("unable to find vApp network %s: %s", networkName, err)
			}
			if !isVappNetwork {
				return types.NetworkConnectionSection{}, fmt.Errorf("vApp network : %s is not found", networkName)
			}
		}

		netConn.IsConnected = true
		netConn.IPAddressAllocationMode = ipAllocationMode
		netConn.NetworkConnectionIndex = index
		netConn.Network = networkName
		if macIsSet {
			netConn.MACAddress = macAddress
		}

		if ipAllocationMode == types.IPAllocationModeNone {
			netConn.Network = types.NoneNetwork
		}

		if net.ParseIP(ip) != nil {
			netConn.IPAddress = ip
		}

		adapterType, isSetAdapterType := nic["adapter_type"]
		if isSetAdapterType {
			netConn.NetworkAdapterType = adapterType.(string)
		}

		networkConnectionSection.NetworkConnection = append(networkConnectionSection.NetworkConnection, netConn)
	}
	return networkConnectionSection, nil
}

// deprecatedNetworksToConfig converts deprecated network configuration in fields
// TODO v3.0 remove this function once 'network_name', 'vapp_network_name', 'ip' are deprecated
func deprecatedNetworksToConfig(network_name, vapp_network_name, ip string, vdc *govcd.Vdc, vapp govcd.VApp, vcdClient *VCDClient) (types.NetworkConnectionSection, error) {
	if vapp_network_name != "" {
		isVappNetwork, err := isItVappNetwork(vapp_network_name, vapp)
		if err != nil {
			return types.NetworkConnectionSection{}, fmt.Errorf("unable to find vApp network %s: %s", vapp_network_name, err)
		}
		if !isVappNetwork {
			return types.NetworkConnectionSection{}, fmt.Errorf("vApp network : %s is not found", vapp_network_name)
		}
	}

	if network_name != "" {
		// Ensure network_name is added to vdc
		_, err := addVdcNetwork(network_name, vdc, vapp, vcdClient)
		if err != nil {
			return types.NetworkConnectionSection{}, fmt.Errorf("unable to attach vdc network %s: %s", network_name, err)
		}
	}

	networkConnectionSection := types.NetworkConnectionSection{}
	var ipAllocationMode string
	var ipAddress string
	var ipFieldString string

	ipIsSet := ip != ""
	ipFieldString = ip

	switch {
	case ipIsSet && ipFieldString == "dhcp": // Deprecated ip="dhcp" mode
		ipAllocationMode = types.IPAllocationModeDHCP
	case ipIsSet && ipFieldString == "allocated": // Deprecated ip="allocated" mode
		ipAllocationMode = types.IPAllocationModePool
	case ipIsSet && ipFieldString == "none": // Deprecated ip="none" mode
		ipAllocationMode = types.IPAllocationModeNone

	// Deprecated ip="valid_ip" mode (currently it is hit by ip_allocation_mode=MANUAL as well)
	case ipIsSet && net.ParseIP(ipFieldString) != nil:
		ipAllocationMode = types.IPAllocationModeManual
		ipAddress = ipFieldString
	case ipIsSet && ipFieldString != "": // Deprecated ip="something_invalid" we default to DHCP. This is odd but backwards compatible.
		ipAllocationMode = types.IPAllocationModeDHCP
	}

	// Construct networkConnectionSection out of this.
	networkConnectionSection.PrimaryNetworkConnectionIndex = 0
	// If we have a network_name specified it will be NIC 0.
	if network_name != "" {
		netConn := &types.NetworkConnection{}
		netConn.IsConnected = true
		netConn.IPAddressAllocationMode = ipAllocationMode
		netConn.NetworkConnectionIndex = 0
		netConn.Network = network_name
		netConn.IPAddress = ipAddress
		networkConnectionSection.NetworkConnection = append(networkConnectionSection.NetworkConnection, netConn)
	}

	// If we have only 'vapp_network_name' specified it will be NIC 0. If we have 'network_name' and 'vapp_network_name'
	// then a second NIC 1 will be added and will use the same IP parameters as the first one. It is completelly odd,
	// but left so for backwards compatibility. Will be removed with v3.0
	if vapp_network_name != "" {
		netConn := &types.NetworkConnection{}
		netConn.IsConnected = true
		netConn.IPAddressAllocationMode = ipAllocationMode
		netConn.NetworkConnectionIndex = len(networkConnectionSection.NetworkConnection)
		netConn.Network = vapp_network_name
		netConn.IPAddress = ipAddress
		networkConnectionSection.NetworkConnection = append(networkConnectionSection.NetworkConnection, netConn)
	}

	return networkConnectionSection, nil
}

// deprecatedReadNetworks handles read for deprecated network attributes 'ip' and 'mac' and returns
// them for saving in statefile
// TODO v3.0 remove this function once 'network_name', 'vapp_network_name', 'ip' are deprecated
func deprecatedReadNetworks(network_name, vapp_network_name string, vm govcd.VM) (string, string, error) {
	if len(vm.VM.NetworkConnectionSection.NetworkConnection) == 0 {
		return "", "", fmt.Errorf("0 NICs found")
	}

	// The API returns unordered list of NICs therefore we want to be sure and pick 'ip' and 'mac' from primary NIC.
	primaryNicIndex := vm.VM.NetworkConnectionSection.PrimaryNetworkConnectionIndex

	ip := vm.VM.NetworkConnectionSection.NetworkConnection[primaryNicIndex].IPAddress
	// If allocation mode is DHCP and we're not getting the IP - we set this to na (not available)
	if vm.VM.NetworkConnectionSection.NetworkConnection[primaryNicIndex].IPAddressAllocationMode == types.IPAllocationModeDHCP && ip == "" {
		ip = "na"
	}
	mac := vm.VM.NetworkConnectionSection.NetworkConnection[primaryNicIndex].MACAddress

	return ip, mac, nil
}

// readNetworks returns network configuration for saving into statefile
func readNetworks(vm govcd.VM, vapp govcd.VApp) ([]map[string]interface{}, error) {
	// Determine type for all networks in vApp
	vAppNetworkConfig, err := vapp.GetNetworkConfig()
	if err != nil {
		return []map[string]interface{}{}, fmt.Errorf("error getting vApp networks: %s", err)
	}
	// If vApp network is "isolated" and has no ParentNetwork - it is a vApp network.
	// https://code.vmware.com/apis/72/vcloud/doc/doc/types/NetworkConfigurationType.html
	vAppNetworkTypes := make(map[string]string)
	for _, netConfig := range vAppNetworkConfig.NetworkConfig {
		switch {
		case netConfig.NetworkName == types.NoneNetwork:
			vAppNetworkTypes[netConfig.NetworkName] = types.NoneNetwork
		case netConfig.Configuration.ParentNetwork == nil && netConfig.Configuration.FenceMode == "isolated":
			vAppNetworkTypes[netConfig.NetworkName] = "vapp"
		default:
			vAppNetworkTypes[netConfig.NetworkName] = "org"
		}
	}

	var nets []map[string]interface{}
	// Sort NIC cards by their virtual slot numbers as the API returns them in random order
	sort.SliceStable(vm.VM.NetworkConnectionSection.NetworkConnection, func(i, j int) bool {
		return vm.VM.NetworkConnectionSection.NetworkConnection[i].NetworkConnectionIndex <
			vm.VM.NetworkConnectionSection.NetworkConnection[j].NetworkConnectionIndex
	})

	for netIndex, vmNet := range vm.VM.NetworkConnectionSection.NetworkConnection {
		singleNIC := make(map[string]interface{})
		singleNIC["ip_allocation_mode"] = vmNet.IPAddressAllocationMode
		singleNIC["ip"] = vmNet.IPAddress
		singleNIC["mac"] = vmNet.MACAddress
		singleNIC["adapter_type"] = vmNet.NetworkAdapterType
		if vmNet.Network != types.NoneNetwork {
			singleNIC["name"] = vmNet.Network
		}

		singleNIC["is_primary"] = false
		if netIndex == vm.VM.NetworkConnectionSection.PrimaryNetworkConnectionIndex {
			singleNIC["is_primary"] = true
		}

		var ok bool
		if singleNIC["type"], ok = vAppNetworkTypes[vmNet.Network]; !ok {
			return []map[string]interface{}{}, fmt.Errorf("unable to determine vApp network type for: %s", vmNet.Network)
		}

		nets = append(nets, singleNIC)
	}
	return nets, nil
}

// isForcedCustomization checks "customization" block in resource and checks if the value of field "force"
// is set to "true". It returns false if the value is not set or is set to false
func isForcedCustomization(customizationBlock interface{}) bool {
	customizationSlice := customizationBlock.([]interface{})

	if len(customizationSlice) != 1 {
		return false
	}

	cust := customizationSlice[0]
	fc := cust.(map[string]interface{})
	forceCust, ok := fc["force"]
	forceCustBool := forceCust.(bool)

	if !ok || !forceCustBool {
		return false
	}

	return true
}

// getGuestProperties returns a struct for setting guest properties
func getGuestProperties(d *schema.ResourceData) (*types.ProductSectionList, error) {
	guestProperties := d.Get("guest_properties")
	guestProp := convertToStringMap(guestProperties.(map[string]interface{}))
	vmProperties := &types.ProductSectionList{
		ProductSection: &types.ProductSection{
			Info:     "Custom properties",
			Property: []*types.Property{},
		},
	}
	for key, value := range guestProp {
		log.Printf("[TRACE] Adding guest property: key=%s, value=%s to object", key, value)
		oneProp := &types.Property{
			UserConfigurable: true,
			Type:             "string",
			Key:              key,
			Label:            key,
			Value:            &types.Value{Value: value},
		}
		vmProperties.ProductSection.Property = append(vmProperties.ProductSection.Property, oneProp)
	}

	return vmProperties, nil
}

// setGuestProperties sets guest properties into state
func setGuestProperties(d *schema.ResourceData, properties *types.ProductSectionList) error {
	data := make(map[string]string)

	// if properties object does not have actual properties - set state to empty
	log.Printf("[TRACE] Setting empty properties into statefile because no properties were specified")
	if properties == nil || properties.ProductSection == nil || len(properties.ProductSection.Property) == 0 {
		return d.Set("guest_properties", make(map[string]string))
	}

	for _, prop := range properties.ProductSection.Property {
		// if a value was set - use it
		if prop.Value != nil {
			data[prop.Key] = prop.Value.Value
		}
	}

	log.Printf("[TRACE] Setting properties into statefile")
	return d.Set("guest_properties", data)
}

// resourceVcdVappVmImport is responsible for importing the resource.
// The following steps happen as part of import
// 1. The user supplies `terraform import _resource_name_ _the_id_string_` command
// 2. `_the_id_string_` contains a dot formatted path to resource as in the example below
// 3. The functions splits the dot-formatted path and tries to lookup the object
// 4. If the lookup succeeds it sets the ID field for `_resource_name_` resource in statefile
// (the resource must be already defined in .tf config otherwise `terraform import` will complain)
// 5. `terraform refresh` is being implicitly launched. The Read method looks up all other fields
// based on the known ID of object.
//
// Example resource name (_resource_name_): vcd_vapp_vm.VM_name
// Example import path (_the_id_string_): org-name.vdc-name.vapp-name.vm-name
func resourceVcdVappVmImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	resourceURI := strings.Split(d.Id(), ImportSeparator)
	if len(resourceURI) != 4 {
		return nil, fmt.Errorf("[VM import] resource name must be specified as org-name.vdc-name.vapp-name.vm-name")
	}
	orgName, vdcName, vappName, vmName := resourceURI[0], resourceURI[1], resourceURI[2], resourceURI[3]

	vcdClient := meta.(*VCDClient)
	_, vdc, err := vcdClient.GetOrgAndVdc(orgName, vdcName)
	if err != nil {
		return nil, fmt.Errorf("[VM import] unable to find VDC %s: %s ", vdcName, err)
	}

	vapp, err := vdc.GetVAppByName(vappName, false)
	if err != nil {
		return nil, fmt.Errorf("[VM import] error retrieving vapp %s: %s", vappName, err)
	}
	vm, err := vapp.GetVMByName(vmName, false)
	if err != nil {
		return nil, fmt.Errorf("[VM import] error retrieving VM %s: %s", vmName, err)
	}
	_ = d.Set("name", vmName)
	_ = d.Set("org", orgName)
	_ = d.Set("vdc", vdcName)
	_ = d.Set("vapp_name", vappName)
	d.SetId(vm.VM.ID)
	return []*schema.ResourceData{d}, nil
}
