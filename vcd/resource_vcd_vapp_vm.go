package vcd

//lint:file-ignore SA1019 ignore deprecated functions
import (
	"bytes"
	"fmt"
	"log"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/kr/pretty"
	"github.com/vmware/go-vcloud-director/v2/govcd"
	"github.com/vmware/go-vcloud-director/v2/types/v56"
	"github.com/vmware/go-vcloud-director/v2/util"
)

type typeOfVm string

const (
	standaloneVmType typeOfVm = "vcd_vm"
	vappVmType       typeOfVm = "vcd_vapp_vm"
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
		Schema: vmSchemaFunc(vappVmType),
	}
}

func resourceVcdVAppVmCreate(d *schema.ResourceData, meta interface{}) error {
	return genericResourceVmCreate(d, meta, vappVmType)
}

func resourceVcdVAppVmRead(d *schema.ResourceData, meta interface{}) error {
	return genericVcdVmRead(d, meta, "resource", vappVmType)
}

func resourceVcdVAppVmUpdate(d *schema.ResourceData, meta interface{}) error {
	return genericResourceVcdVmUpdate(d, meta, vappVmType)
}

// VM Schema is defined as global so that it can be directly accessible in other places
func vmSchemaFunc(vmType typeOfVm) map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"vapp_name": &schema.Schema{
			Type:        schema.TypeString,
			Required:    vmType == vappVmType,
			Optional:    vmType == standaloneVmType,
			Computed:    vmType == standaloneVmType,
			ForceNew:    vmType == vappVmType,
			Description: "The vApp this VM belongs to - Required, unless it is a standalone VM",
		},
		"vm_type": &schema.Schema{
			Type:        schema.TypeString,
			Computed:    true,
			Description: fmt.Sprintf("Type of VM: either '%s' or '%s'", vappVmType, standaloneVmType),
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
			Optional:    true,
			ForceNew:    true,
			Description: "The name of the vApp Template to use",
		},
		"vm_name_in_template": &schema.Schema{
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    true,
			Description: "The name of the VM in vApp Template to use. In cases when vApp template has more than one VM",
		},
		"catalog_name": &schema.Schema{
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The catalog name in which to find the given vApp Template or media for boot_image",
		},
		"description": &schema.Schema{
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "The VM description",
		},
		"memory": &schema.Schema{
			Type:         schema.TypeInt,
			Optional:     true,
			Computed:     true,
			Description:  "The amount of RAM (in MB) to allocate to the VM",
			ValidateFunc: validateMultipleOf4(),
		},
		"cpus": &schema.Schema{
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
			Description: "The number of virtual CPUs to allocate to the VM",
		},
		"cpu_cores": &schema.Schema{
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
			Description: "The number of cores per socket",
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
		"os_type": &schema.Schema{
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "Operating System type. Possible values can be found in documentation.",
		},
		"hardware_version": &schema.Schema{
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "Virtual Hardware Version (e.g.`vmx-14`, `vmx-13`, `vmx-12`, etc.)",
		},
		"boot_image": &schema.Schema{
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Media name to add as boot image.",
		},
		"network_dhcp_wait_seconds": {
			Optional:     true,
			Type:         schema.TypeInt,
			ValidateFunc: validation.IntAtLeast(0),
			Description: "Optional number of seconds to try and wait for DHCP IP (valid for " +
				"'network' block only)",
		},
		"network": {
			Optional:    true,
			Type:        schema.TypeList,
			Description: " A block to define network interface. Multiple can be used.",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"type": {
						Required:     true,
						Type:         schema.TypeString,
						ValidateFunc: vmNetworkTypeValidator(vmType),
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
						Optional: true,
						Computed: true,
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
					"connected": {
						Type:        schema.TypeBool,
						Optional:    true,
						Default:     true,
						Description: "It defines if NIC is connected or not.",
					},
				},
			},
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
					ValidateFunc: validation.StringInSlice([]string{"ide", "parallel", "sas", "paravirtual", "sata", "nvme"}, false),
					Description:  "The type of disk controller. Possible values: ide, parallel( LSI Logic Parallel SCSI), sas(LSI Logic SAS (SCSI)), paravirtual(Paravirtual (SCSI)), sata, nvme",
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
					Description: "Specifies the IOPS for the disk. Default is 0.",
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
					Description: "The type of disk controller. Possible values: ide, parallel( LSI Logic Parallel SCSI), sas(LSI Logic SAS (SCSI)), paravirtual(Paravirtual (SCSI)), sata, nvme",
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
					Description: "Specifies the IOPS for the disk. Default is 0.",
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
			Computed:    true,
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
					"enabled": {
						Type:        schema.TypeBool,
						Optional:    true,
						Computed:    true,
						Description: "'true' value will enable guest customization. It may occur on first boot or when 'force' is used",
					},
					"change_sid": {
						Type:        schema.TypeBool,
						Optional:    true,
						Computed:    true,
						Description: "'true' value will change SID. Applicable only for Windows VMs",
					},
					"allow_local_admin_password": {
						Type:        schema.TypeBool,
						Optional:    true,
						Computed:    true,
						Description: "Allow local administrator password",
					},
					"must_change_password_on_first_login": {
						Type:        schema.TypeBool,
						Optional:    true,
						Computed:    true,
						Description: "Require Administrator to change password on first login",
					},
					"auto_generate_password": {
						Type:        schema.TypeBool,
						Optional:    true,
						Computed:    true,
						Description: "Auto generate password",
					},
					"admin_password": {
						Type:        schema.TypeString,
						Optional:    true,
						Computed:    true,
						Sensitive:   true,
						Description: "Manually specify admin password",
					},
					"number_of_auto_logons": {
						Type:         schema.TypeInt,
						Optional:     true,
						Computed:     true,
						Description:  "Number of times to log on automatically. '0' - disabled.",
						ValidateFunc: validation.IntAtLeast(0),
					},
					"join_domain": {
						Type:        schema.TypeBool,
						Optional:    true,
						Computed:    true,
						Description: "Enable this VM to join a domain",
					},
					"join_org_domain": {
						Type:        schema.TypeBool,
						Optional:    true,
						Computed:    true,
						Description: "Use organization's domain for joining",
					},
					"join_domain_name": {
						Type:        schema.TypeString,
						Optional:    true,
						Computed:    true,
						Description: "Custom domain name for join",
					},
					"join_domain_user": {
						Type:        schema.TypeString,
						Optional:    true,
						Computed:    true,
						Description: "Username for custom domain name join",
					},
					"join_domain_password": {
						Type:        schema.TypeString,
						Optional:    true,
						Computed:    true,
						Sensitive:   true,
						Description: "Password for custom domain name join",
					},
					"join_domain_account_ou": {
						Type:        schema.TypeString,
						Optional:    true,
						Computed:    true,
						Description: "Account organizational unit for domain name join",
					},
					"initscript": &schema.Schema{
						Type:        schema.TypeString,
						Optional:    true,
						Computed:    true,
						Description: "Script to run on initial boot or with customization.force=true set",
					},
				},
			},
		},
		"cpu_hot_add_enabled": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "True if the virtual machine supports addition of virtual CPUs while powered on.",
		},
		"memory_hot_add_enabled": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "True if the virtual machine supports addition of memory while powered on.",
		},
		"prevent_update_power_off": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "True if the update of resource should fail when virtual machine power off needed.",
		},
		"sizing_policy_id": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "VM sizing policy ID. Has to be assigned to Org VDC.",
		},
	}
}

// vmTemplatefromVappTemplate returns a given VM from a vApp template
// If no name is provided, it returns the first VM from the template
func vmTemplatefromVappTemplate(name string, vappTemplate *types.VAppTemplate) *types.VAppTemplate {
	if vappTemplate.Children == nil {
		return nil
	}
	for _, vm := range vappTemplate.Children.VM {
		if name == vm.Name || name == "" {
			return vm
		}
	}
	return nil
}

func genericResourceVmCreate(d *schema.ResourceData, meta interface{}, vmType typeOfVm) error {
	util.Logger.Printf("[DEBUG] [VM create] started")
	vcdClient := meta.(*VCDClient)

	vappName := d.Get("vapp_name").(string)
	if vappName == "" && vmType == vappVmType {
		return fmt.Errorf("vApp name is mandatory for this VM type")
	}
	if vappName != "" && vmType == standaloneVmType {
		return fmt.Errorf("vApp name must not be set for a standalone VM")
	}
	if vappName != "" && vmType == vappVmType {
		vcdClient.lockParentVapp(d)
		defer vcdClient.unLockParentVapp(d)
	}

	org, vdc, err := vcdClient.GetOrgAndVdcFromResource(d)
	if err != nil {
		return fmt.Errorf(errorRetrievingOrgAndVdc, err)
	}

	catalogName := d.Get("catalog_name").(string)
	templateName := d.Get("template_name").(string)
	vmName := d.Get("name").(string)
	description := d.Get("description").(string)
	powerOn := d.Get("power_on").(bool)

	var vapp *govcd.VApp

	//create not empty VM - use provided template
	if catalogName != "" && templateName != "" {

		catalog, err := org.GetCatalogByName(catalogName, false)
		if err != nil {
			return fmt.Errorf("error finding catalog %s: %s", catalogName, err)
		}

		var vappTemplate govcd.VAppTemplate
		if vmNameInTemplate, ok := d.GetOk("vm_name_in_template"); ok {
			vmInTemplateRecord, err := vdc.QueryVappVmTemplate(catalogName, templateName, vmNameInTemplate.(string))
			if err != nil {
				return fmt.Errorf("error quering VM template %s: %s", vmNameInTemplate, err)
			}
			util.Logger.Printf("[VM create] vmInTemplateRecord %# v", pretty.Formatter(vmInTemplateRecord))
			returnedVappTemplate, err := catalog.GetVappTemplateByHref(vmInTemplateRecord.HREF)
			if err != nil {
				return fmt.Errorf("error quering VM template %s: %s", vmNameInTemplate, err)
			}
			util.Logger.Printf("[VM create] returnedVappTemplate %#v", pretty.Formatter(returnedVappTemplate))
			vappTemplate = *returnedVappTemplate
		} else {
			catalogItem, err := catalog.GetCatalogItemByName(templateName, false)
			if err != nil {
				return fmt.Errorf("error finding catalog item %s: %s", templateName, err)
			}
			vappTemplate, err = catalogItem.GetVAppTemplate()
			if err != nil {
				return fmt.Errorf("[VM create] error finding VAppTemplate %s: %s", templateName, err)
			}

		}
		acceptEulas := d.Get("accept_all_eulas").(bool)

		if vappName != "" {
			vapp, err = vdc.GetVAppByName(vappName, false)
			if err != nil {
				return fmt.Errorf("[VM create] error finding vApp %s: %s", vappName, err)
			}
		}

		networkConnectionSection := types.NetworkConnectionSection{}
		if len(d.Get("network").([]interface{})) > 0 {
			networkConnectionSection, err = networksToConfig(d, vdc, vapp, vcdClient)
			if err != nil {
				return fmt.Errorf("unable to process network configuration: %s", err)
			}
			util.Logger.Printf("[VM create] networkConnectionSection %# v", pretty.Formatter(networkConnectionSection))
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

		var sizingPolicy *types.VdcComputePolicy
		var vmComputePolicy *types.ComputePolicy
		if value, ok := d.GetOk("sizing_policy_id"); ok {
			vdcComputePolicy, err := org.GetVdcComputePolicyById(value.(string))
			if err != nil {
				return fmt.Errorf("error getting sizing policy %s: %s", value.(string), err)
			}
			sizingPolicy = vdcComputePolicy.VdcComputePolicy
			if vdcComputePolicy.Href == "" {
				return fmt.Errorf("empty sizing policy HREF detected")
			}
			vmComputePolicy = &types.ComputePolicy{
				VmSizingPolicy: &types.Reference{HREF: vdcComputePolicy.Href},
			}
			util.Logger.Printf("[VM create] sizingPolicy (%s) %# v", vdcComputePolicy.Href, pretty.Formatter(sizingPolicy))
		}

		var vm *govcd.VM

		if vappName == "" {
			// Build a standalone VM
			if vmComputePolicy != nil && vcdClient.Client.APIVCDMaxVersionIs("< 33.0") {
				util.Logger.Printf("[Warning] compute policy is ignored because VCD version doesn't support it")
				vmComputePolicy = nil
			}
			vmTemplate := vmTemplatefromVappTemplate(d.Get("vm_name_in_template").(string), vappTemplate.VAppTemplate)
			vmParams := types.InstantiateVmTemplateParams{
				Xmlns:            types.XMLNamespaceVCloud,
				Name:             vmName,
				PowerOn:          powerOn,
				AllEULAsAccepted: acceptEulas,
				ComputePolicy:    vmComputePolicy,
				Description:      description,
				SourcedVmTemplateItem: &types.SourcedVmTemplateParams{
					Source: &types.Reference{
						HREF: vmTemplate.HREF,
						ID:   vmTemplate.ID,
						Type: vmTemplate.Type,
						Name: vmTemplate.Name,
					},
					StorageProfile: storageProfilePtr,
					VmTemplateInstantiationParams: &types.InstantiationParams{
						NetworkConnectionSection: &networkConnectionSection,
					},
				},
			}
			util.Logger.Printf("%# v", pretty.Formatter(vmParams))
			vm, err = vdc.CreateStandaloneVMFromTemplate(&vmParams)
			if err != nil {
				d.SetId("")
				return fmt.Errorf("[VM creation] error creating standalone VM %s : %s", vmName, err)
			}
			util.Logger.Printf("[VM create] VM after creation %# v", pretty.Formatter(vm.VM))
			vapp, err = vm.GetParentVApp()
			if err != nil {
				d.SetId("")
				return fmt.Errorf("[VM creation] error retrieving vApp from standalone VM %s : %s", vmName, err)
			}
			util.Logger.Printf("[VM create] vApp after creation %# v", pretty.Formatter(vapp.VApp))
			_ = d.Set("vapp_name", vapp.VApp.Name)
		} else {
			task, err := vapp.AddNewVMWithComputePolicy(vmName, vappTemplate, &networkConnectionSection, storageProfilePtr, sizingPolicy, acceptEulas)
			if err != nil {
				return fmt.Errorf("[VM creation] error adding VM: %s", err)
			}
			err = task.WaitTaskCompletion()
			if err != nil {
				return fmt.Errorf(errorCompletingTask, err)
			}

			vm, err = vapp.GetVMByName(vmName, true)

			if err != nil {
				d.SetId("")
				return fmt.Errorf("[VM creation] error getting VM %s : %s", vmName, err)
			}
		}

		var computedVmType string
		if vapp.VApp.IsAutoNature {
			computedVmType = string(standaloneVmType)
		} else {
			computedVmType = string(vappVmType)
		}
		// VM creation already succeeded so ID must be set
		d.SetId(vm.VM.ID)
		_ = d.Set("vm_type", computedVmType)

		err = handleExposeHardwareVirtualization(d, vm)
		if err != nil {
			return err
		}

		if err := updateGuestCustomizationSetting(d, vm); err != nil {
			return fmt.Errorf("error setting guest customization during creation: %s", err)
		}

		err = addRemoveGuestProperties(d, vm)
		if err != nil {
			return err
		}

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

		err = addRemoveMetaData(d, vm)
		if err != nil {
			return err
		}

		// TODO do not trigger resourceVcdVAppVmUpdate from create. These must be separate actions.
		err = resourceVcdVAppVmUpdateExecute(d, meta, "create", vmType)
		if err != nil {
			errAttachedDisk := updateStateOfAttachedDisks(d, *vm, vdc)
			if errAttachedDisk != nil {
				d.Set("disk", nil)
				return fmt.Errorf("error reading attached disks : %s and internal error : %s", errAttachedDisk, err)
			}
			return err
		}
	} else {
		//create empty VM
		vm, err := addEmptyVm(d, vcdClient, org, vdc, vappName)
		if err != nil {
			return err
		}
		if err != nil {
			d.SetId("")
			return fmt.Errorf("[VM creation] error creating standalone VM %s : %s", vmName, err)
		}
		util.Logger.Printf("[VM create] VM after creation %# v", pretty.Formatter(vm.VM))
		vapp, err = vm.GetParentVApp()
		if err != nil {
			d.SetId("")
			return fmt.Errorf("[VM creation] error retrieving vApp from standalone VM %s : %s", vmName, err)
		}
		util.Logger.Printf("[VM create] vApp after creation %# v", pretty.Formatter(vapp.VApp))
		_ = d.Set("vapp_name", vapp.VApp.Name)
		return genericVcdVmRead(d, meta, "create", vmType)
	}

	log.Printf("[DEBUG] [VM create] finished")
	return nil
}

// isItVappNetwork checks if it is an vApp network (not vApp Org Network)
func isItVappNetwork(vAppNetworkName string, vapp govcd.VApp) (bool, error) {
	vAppNetworkConfig, err := vapp.GetNetworkConfig()
	if err != nil {
		return false, fmt.Errorf("error getting vApp networks: %s", err)
	}

	for _, networkConfig := range vAppNetworkConfig.NetworkConfig {
		if networkConfig.NetworkName == vAppNetworkName &&
			govcd.IsVappNetwork(networkConfig.Configuration) {
			log.Printf("[TRACE] vApp network found: %s", vAppNetworkName)
			return true, nil
		}
	}

	return false, fmt.Errorf("configured vApp network isn't found: %s", vAppNetworkName)
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

func genericResourceVcdVmUpdate(d *schema.ResourceData, meta interface{}, vmType typeOfVm) error {
	log.Printf("[DEBUG] [VM update] started with lock")
	vcdClient := meta.(*VCDClient)

	// When there is more then one VM in a vApp Terraform will try to parallelise their creation.
	// However, vApp throws errors when simultaneous requests are executed.
	// To avoid them, below block is using mutex as a workaround,
	// so that the one vApp VMs are created not in parallelisation.

	if vmType == vappVmType {
		vcdClient.lockParentVapp(d)
		defer vcdClient.unLockParentVapp(d)
	}

	// Exit early only if "network_dhcp_wait_seconds" is changed because this field only supports
	// update so that its value can be written into statefile and be accessible in read function
	if onlyHasChange("network_dhcp_wait_seconds", vmSchemaFunc(vmType), d) {
		log.Printf("[DEBUG] [VM update] exiting early because only 'network_dhcp_wait_seconds' has change")
		return genericVcdVmRead(d, meta, "update", vmType)
	}

	err := resourceVmHotUpdate(d, meta, vmType)
	if err != nil {
		return err
	}

	return resourceVcdVAppVmUpdateExecute(d, meta, "update", vmType)
}

func resourceVmHotUpdate(d *schema.ResourceData, meta interface{}, vmType typeOfVm) error {
	vcdClient, _, vdc, vapp, _, vm, err := getVmFromResource(d, meta, vmType)
	if err != nil {
		return err
	}
	if d.Get("memory_hot_add_enabled").(bool) && d.HasChange("memory") {
		err = changeMemorySize(d, vm)
		if err != nil {
			return err
		}
	}

	if d.Get("cpu_hot_add_enabled").(bool) && d.HasChange("cpus") {
		err = changeCpuCount(d, vm)
		if err != nil {
			return err
		}
	}

	// due the bug in VCD 10.1 hot update possible for adding new or update existing network, removing of network has to be done with cold update
	if d.HasChange("network") && !isNetworkRemovedInVcd101(d, meta) {
		networkConnectionSection, err := networksToConfig(d, vdc, vapp, vcdClient)
		if err != nil {
			return fmt.Errorf("unable to setup network configuration for update: %s", err)
		}
		err = vm.UpdateNetworkConnectionSection(&networkConnectionSection)
		if err != nil {
			return fmt.Errorf("unable to update network configuration: %s", err)
		}
	}

	err = addRemoveMetaData(d, vm)
	if err != nil {
		return err
	}

	err = addRemoveGuestProperties(d, vm)
	if err != nil {
		return err
	}

	if d.HasChange("sizing_policy_id") {
		var sizingPolicy *types.VdcComputePolicy
		org, _, err := vcdClient.GetOrgAndVdcFromResource(d)
		if err != nil {
			return fmt.Errorf(errorRetrievingOrg, err)
		}
		value := d.Get("sizing_policy_id")
		vdcComputePolicy, err := org.GetVdcComputePolicyById(value.(string))
		if err != nil {
			return fmt.Errorf("error getting sizing policy %s: %s", value.(string), err)
		}
		sizingPolicy = vdcComputePolicy.VdcComputePolicy
		_, err = vm.UpdateComputePolicy(sizingPolicy)
		if err != nil {
			return fmt.Errorf("error updating sizing policy %s: %s", value.(string), err)
		}
	}

	storageProfileName := d.Get("storage_profile").(string)
	if d.HasChange("storage_profile") && storageProfileName != "" {
		storageProfile, err := vdc.FindStorageProfileReference(storageProfileName)
		if err != nil {
			return fmt.Errorf("[vm update] error retrieving storage profile %s : %s", storageProfileName, err)
		}
		_, err = vm.UpdateStorageProfile(storageProfile.HREF)
		if err != nil {
			return fmt.Errorf("error updating changing storage profile to %s: %s", storageProfileName, err)
		}
	}

	return nil
}

func addRemoveGuestProperties(d *schema.ResourceData, vm *govcd.VM) error {
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
	return nil
}

func addRemoveMetaData(d *schema.ResourceData, vm *govcd.VM) error {
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
	return nil
}

// isNetworkRemovedInVcd101 returns true only if network removed and VCD version 10.1
func isNetworkRemovedInVcd101(d *schema.ResourceData, meta interface{}) bool {
	vcdClient := meta.(*VCDClient)
	oldNetworksRaw, newNetworkRaw := d.GetChange("network")
	oldNetworks := oldNetworksRaw.([]interface{})
	newNetworks := newNetworkRaw.([]interface{})
	// return true only VCD version 10.1
	return len(oldNetworks) > len(newNetworks) && vcdClient.Client.APIVCDMaxVersionIs("= 34.0")
}

func changeCpuCount(d *schema.ResourceData, vm *govcd.VM) error {
	task, err := vm.ChangeCPUCount(d.Get("cpus").(int))
	if err != nil {
		return fmt.Errorf("error setting CPU count: %s", err)
	}
	err = task.WaitTaskCompletion()
	if err != nil {
		return err
	}
	return nil
}

func changeMemorySize(d *schema.ResourceData, vm *govcd.VM) error {
	task, err := vm.ChangeMemorySize(d.Get("memory").(int))
	if err != nil {
		return fmt.Errorf("error changing memory size: %s", err)
	}
	err = task.WaitTaskCompletion()
	if err != nil {
		return err
	}
	return nil
}

func resourceVcdVAppVmUpdateExecute(d *schema.ResourceData, meta interface{}, executionType string, vmType typeOfVm) error {
	log.Printf("[DEBUG] [VM update] started without lock")

	vcdClient, org, vdc, vapp, identifier, vm, err := getVmFromResource(d, meta, vmType)
	if err != nil {
		return err
	}

	if vcdClient.Client.APIVCDMaxVersionIs("< 33.0") {
		if _, ok := d.GetOk("sizing_policy_id"); ok {
			return fmt.Errorf("'sizing_policy_id' only available for VCD 10.0+")
		}
	}

	vmStatusBeforeUpdate, err := vm.GetStatus()
	if err != nil {
		return fmt.Errorf("[VM update] error getting VM (%s) status before update: %s", identifier, err)
	}

	// Check if the user requested for forced customization of VM
	customizationNeeded := isForcedCustomization(d.Get("customization"))

	// Update guest customization if any of the customization related fields have changed
	if d.HasChanges("customization", "computer_name", "name") {
		log.Printf("[TRACE] VM %s customization has changes: customization(%t), computer_name(%t), name(%t)",
			vm.VM.Name, d.HasChange("customization"), d.HasChange("computer_name"), d.HasChange("name"))
		err = updateGuestCustomizationSetting(d, vm)
		if err != nil {
			return fmt.Errorf("errors updating guest customization: %s", err)
		}

	}

	memoryNeedsColdChange := false
	cpusNeedsColdChange := false
	networksNeedsColdChange := false
	if executionType == "update" {
		if !d.Get("memory_hot_add_enabled").(bool) && d.HasChange("memory") {
			memoryNeedsColdChange = true
		}
		if !d.Get("cpu_hot_add_enabled").(bool) && d.HasChange("cpus") {
			cpusNeedsColdChange = true
		}
		if d.HasChange("network") && isNetworkRemovedInVcd101(d, meta) {
			networksNeedsColdChange = true
		}
	} else if len(d.Get("network").([]interface{})) > 0 {
		networksNeedsColdChange = true
	}

	// this represent fields which has to be changed in cold (with VM power off)
	if d.HasChanges("cpu_cores", "power_on", "disk", "expose_hardware_virtualization", "boot_image",
		"hardware_version", "os_type", "description", "cpu_hot_add_enabled",
		"memory_hot_add_enabled") || memoryNeedsColdChange || cpusNeedsColdChange || networksNeedsColdChange {

		log.Printf("[TRACE] VM %s has changes: memory(%t), cpus(%t), cpu_cores(%t), power_on(%t), disk(%t), expose_hardware_virtualization(%t),"+
			" boot_image(%t), hardware_version(%t), os_type(%t), description(%t), cpu_hot_add_enabled(%t), memory_hot_add_enabled(%t), network(%t)",
			vm.VM.Name, d.HasChange("memory"), d.HasChange("cpus"), d.HasChange("cpu_cores"), d.HasChange("power_on"), d.HasChange("disk"),
			d.HasChange("expose_hardware_virtualization"), d.HasChange("boot_image"), d.HasChange("hardware_version"),
			d.HasChange("os_type"), d.HasChange("description"), d.HasChange("cpu_hot_add_enabled"), d.HasChange("memory_hot_add_enabled"), d.HasChange("network"))

		if vmStatusBeforeUpdate != "POWERED_OFF" {
			if d.Get("prevent_update_power_off").(bool) && executionType == "update" {
				return fmt.Errorf("update stopped: VM needs to power off to change properties, but `prevent_update_power_off` is `true`")
			}
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

		if memoryNeedsColdChange || executionType == "create" {
			err = changeMemorySize(d, vm)
			if err != nil {
				return err
			}
		}

		if d.HasChange("cpu_cores") {
			coreCounts := d.Get("cpu_cores").(int)
			task, err := vm.ChangeCPUCountWithCore(d.Get("cpus").(int), &coreCounts)
			if err != nil {
				return fmt.Errorf("error changing cpu count: %s", err)
			}

			err = task.WaitTaskCompletion()
			if err != nil {
				return fmt.Errorf(errorCompletingTask, err)
			}
		}

		if cpusNeedsColdChange || executionType == "create" {
			err = changeCpuCount(d, vm)
			if err != nil {
				return err
			}
		}

		if networksNeedsColdChange {
			networkConnectionSection, err := networksToConfig(d, vdc, vapp, vcdClient)
			if err != nil {
				return fmt.Errorf("unable to setup network configuration for update: %s", err)
			}
			err = vm.UpdateNetworkConnectionSection(&networkConnectionSection)
			if err != nil {
				return fmt.Errorf("unable to update network configuration: %s", err)
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

		// updating fields of VM spec section
		if d.HasChange("hardware_version") || d.HasChange("os_type") || d.HasChange("description") {
			vmSpecSection := vm.VM.VmSpecSection
			description := vm.VM.Description
			if d.HasChange("hardware_version") {
				vmSpecSection.HardwareVersion = &types.HardwareVersion{Value: d.Get("hardware_version").(string)}
			}
			if d.HasChange("os_type") {
				vmSpecSection.OsType = d.Get("os_type").(string)
			}

			if d.HasChange("description") {
				description = d.Get("description").(string)
			}

			_, err := vm.UpdateVmSpecSection(vmSpecSection, description)
			if err != nil {
				return fmt.Errorf("error changing VM spec section: %s", err)
			}
		}

		if d.HasChange("cpu_hot_add_enabled") || d.HasChange("memory_hot_add_enabled") {
			_, err := vm.UpdateVmCpuAndMemoryHotAdd(d.Get("cpu_hot_add_enabled").(bool), d.Get("memory_hot_add_enabled").(bool))
			if err != nil {
				return fmt.Errorf("error changing VM capabilities: %s", err)
			}
		}

		// we detach boot image if it's value change to empty.
		bootImage := d.Get("boot_image")
		if d.HasChange("boot_image") && bootImage.(string) == "" {
			previousBootImageValue, _ := d.GetChange("boot_image")
			previousCatalogName, _ := d.GetChange("catalog_name")
			catalog, err := org.GetCatalogByName(previousCatalogName.(string), false)
			if err != nil {
				return fmt.Errorf("[VM Update] error finding catalog %s: %s", previousCatalogName, err)
			}
			result, err := catalog.GetMediaByName(previousBootImageValue.(string), false)
			if err != nil {
				return fmt.Errorf("[VM Update] error getting boot image %s : %s", previousBootImageValue, err)
			}

			task, err := vm.HandleEjectMedia(org, previousCatalogName.(string), result.Media.Name)
			if err != nil {
				return fmt.Errorf("error: %#v", err)
			}

			err = task.WaitTaskCompletion(true)
			if err != nil {
				return fmt.Errorf("error: %#v", err)
			}
		}
	}

	// If the VM was powered off during update but it has to be powered on
	if d.Get("power_on").(bool) {
		vmStatus, err := vm.GetStatus()
		if err != nil {
			return fmt.Errorf("error getting VM status before ensuring it is powered on: %s", err)
		}

		// Simply power on if customization is not requested
		if !customizationNeeded && vmStatus != "POWERED_ON" {
			log.Printf("[DEBUG] Powering on VM %s after update. Previous state %s", vm.VM.Name, vmStatus)
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
	log.Printf("[DEBUG] [VM update] finished")
	return genericVcdVmRead(d, meta, "update", vmType)
}

func getVmFromResource(d *schema.ResourceData, meta interface{}, vmType typeOfVm) (*VCDClient, *govcd.Org, *govcd.Vdc, *govcd.VApp, string, *govcd.VM, error) {
	vcdClient := meta.(*VCDClient)

	org, vdc, err := vcdClient.GetOrgAndVdcFromResource(d)
	if err != nil {
		return nil, nil, nil, nil, "", nil, fmt.Errorf(errorRetrievingOrgAndVdc, err)
	}

	vappName := d.Get("vapp_name").(string)
	vapp, err := vdc.GetVAppByName(vappName, false)

	if err != nil {
		additionalMessage := ""
		if vmType == standaloneVmType {
			additionalMessage = fmt.Sprintf("\nAdding a vApp name to a standalone VM is not allowed." +
				"Please use 'vcd_vapp_vm' resource to specify vApp")
			_ = d.Set("vapp_name", "")
		}

		return nil, nil, nil, nil, "", nil, fmt.Errorf("[getVmFromResource] error finding vApp '%s': %s%s", vappName, err, additionalMessage)
	}

	identifier := d.Id()
	if identifier == "" {
		identifier = d.Get("name").(string)
	}
	if identifier == "" {
		return nil, nil, nil, nil, "", nil, fmt.Errorf("[VM update] neither name or ID was set")
	}

	vm, err := vapp.GetVMByNameOrId(identifier, false)

	if err != nil {
		d.SetId("")
		return nil, nil, nil, nil, "", nil, fmt.Errorf("[VM update] error getting VM %s: %s", identifier, err)
	}
	return vcdClient, org, vdc, vapp, identifier, vm, nil
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

func genericVcdVmRead(d *schema.ResourceData, meta interface{}, origin string, vmType typeOfVm) error {
	log.Printf("[DEBUG] [VM read] started with origin %s", origin)
	vcdClient := meta.(*VCDClient)

	_, vdc, err := vcdClient.GetOrgAndVdcFromResource(d)
	if err != nil {
		return fmt.Errorf("[VM read ]"+errorRetrievingOrgAndVdc, err)
	}

	isStandalone := false
	setVmType, ok := d.GetOk("vm_type")
	if ok {
		isStandalone = setVmType.(string) == string(standaloneVmType)
	}
	var vapp *govcd.VApp
	var vm *govcd.VM
	identifier := d.Id()
	if identifier == "" {
		identifier = d.Get("name").(string)
	}
	if identifier == "" {
		return fmt.Errorf("[VM read] neither name or ID were set for this VM")
	}
	vappName := d.Get("vapp_name").(string)
	if vappName == "" {
		vm, err = vdc.QueryVmById(identifier)
		if govcd.IsNotFound(err) {
			vmByName, listStr, errByName := getVmByName(vcdClient, vdc, identifier)
			if errByName != nil && listStr != "" {
				return fmt.Errorf("[VM read] error retrieving VM %s by name: %s\n%s\n%s", identifier, errByName, listStr, err)
			}
			vm = vmByName
			err = errByName
		}
	} else {
		vapp, err = vdc.GetVAppByName(vappName, false)
		if err != nil {
			additionalMessage := ""
			if isStandalone {
				additionalMessage = fmt.Sprintf("\nAdding a vApp name to a standalone VM is not allowed." +
					"Please use 'vcd_vapp_vm' resource to specify vApp")
				_ = d.Set("vapp_name", "")
			}
			return fmt.Errorf("[VM read] error finding vApp '%s': %s%s", vappName, err, additionalMessage)
		}
		vm, err = vapp.GetVMByNameOrId(identifier, false)
	}
	if err != nil {
		if origin == "resource" {
			log.Printf("[DEBUG] Unable to find VM. Removing from tfstate")
			d.SetId("")
			return nil
		}
		return fmt.Errorf("[VM read] error getting VM %s : %s", identifier, err)
	}
	if vapp == nil {
		vapp, err = vm.GetParentVApp()
		if err != nil {
			return fmt.Errorf("[VM read] error retrieving parent vApp for VM %s: %s", vm.VM.Name, err)
		}
	}

	var computedVmType string
	if vapp.VApp.IsAutoNature {
		computedVmType = string(standaloneVmType)
	} else {
		computedVmType = string(vappVmType)
	}
	// org, vdc, and vapp_name are already implicitly set
	_ = d.Set("name", vm.VM.Name)
	_ = d.Set("vapp_name", vapp.VApp.Name)
	_ = d.Set("description", vm.VM.Description)
	d.SetId(vm.VM.ID)
	_ = d.Set("vm_type", computedVmType)

	networks, err := readNetworks(d, *vm, *vapp, vdc)
	if err != nil {
		return fmt.Errorf("[VM read] failed reading network details: %s", err)
	}

	err = d.Set("network", networks)
	if err != nil {
		return err
	}

	_ = d.Set("href", vm.VM.HREF)
	_ = d.Set("expose_hardware_virtualization", vm.VM.NestedHypervisorEnabled)
	_ = d.Set("cpu_hot_add_enabled", vm.VM.VMCapabilities.CPUHotAddEnabled)
	_ = d.Set("memory_hot_add_enabled", vm.VM.VMCapabilities.MemoryHotAddEnabled)

	cpus := int64(0)
	coresPerSocket := 0
	memory := int64(0)
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

	if err := setGuestCustomizationData(d, vm); err != nil {
		return fmt.Errorf("error storing customzation block: %s", err)
	}

	if vm.VM.VmSpecSection != nil && vm.VM.VmSpecSection.HardwareVersion != nil && vm.VM.VmSpecSection.HardwareVersion.Value != "" {
		_ = d.Set("hardware_version", vm.VM.VmSpecSection.HardwareVersion.Value)
	}
	if vm.VM.VmSpecSection != nil && vm.VM.VmSpecSection.OsType != "" {
		_ = d.Set("os_type", vm.VM.VmSpecSection.OsType)
	}

	if vm.VM.ComputePolicy != nil && vm.VM.ComputePolicy.VmSizingPolicy != nil {
		_ = d.Set("sizing_policy_id", vm.VM.ComputePolicy.VmSizingPolicy.ID)
	}

	log.Printf("[DEBUG] [VM read] finished with origin %s", origin)
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
			return fmt.Errorf("[vm creation] disk with bus type %s, bus number %d and unit number %d not found",
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
	log.Printf("[DEBUG] [VM delete] started")

	vcdClient := meta.(*VCDClient)

	vcdClient.lockParentVapp(d)
	defer vcdClient.unLockParentVapp(d)

	_, vdc, err := vcdClient.GetOrgAndVdcFromResource(d)
	if err != nil {
		return fmt.Errorf(errorRetrievingOrgAndVdc, err)
	}

	vappName := d.Get("vapp_name").(string)
	vapp, err := vdc.GetVAppByName(vappName, false)

	if err != nil {
		return fmt.Errorf("[VM delete] error finding vApp '%s': %s", vappName, err)
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

	// If it is a standalone VM, we remove it in one go
	if vapp.VApp.IsAutoNature {
		return vm.Delete()
	}
	util.Logger.Printf("[VM delete] vApp before deletion %# v", pretty.Formatter(vapp.VApp))
	util.Logger.Printf("[VM delete] VM before deletion %# v", pretty.Formatter(vm.VM))
	deployed, err := vm.IsDeployed()
	if err != nil {
		return fmt.Errorf("error getting VM deploy status: %s", err)
	}

	log.Printf("[TRACE] VM deploy Status: %t", deployed)
	if deployed {
		log.Printf("[TRACE] Undeploying VM: %s", vm.VM.Name)
		task, err := vm.Undeploy()
		if err != nil {
			return fmt.Errorf("error Undeploying: %s", err)
		}

		err = task.WaitTaskCompletion()
		if err != nil {
			return fmt.Errorf("error Undeploying VM: %s", err)
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
	log.Printf("[DEBUG] [VM delete] finished")
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
	return hashcodeString(buf.String())
}

// networksToConfig converts terraform schema for 'network' and converts to types.NetworkConnectionSection
// which is used for creating new VM
func networksToConfig(d *schema.ResourceData, vdc *govcd.Vdc, vapp *govcd.VApp, vcdClient *VCDClient) (types.NetworkConnectionSection, error) {
	networks := d.Get("network").([]interface{})

	isStandaloneVm := vapp == nil || (vapp != nil && vapp.VApp.IsAutoNature)
	networkConnectionSection := types.NetworkConnectionSection{}

	// sets existing primary network connection index. Further changes index only if change is found
	for index, singleNetwork := range networks {
		nic := singleNetwork.(map[string]interface{})
		isPrimary := nic["is_primary"].(bool)
		if isPrimary {
			networkConnectionSection.PrimaryNetworkConnectionIndex = index
		}
	}

	for index, singleNetwork := range networks {

		nic := singleNetwork.(map[string]interface{})
		netConn := &types.NetworkConnection{}

		networkName := nic["name"].(string)
		if networkName == "" {
			networkName = "none"
		}
		ipAllocationMode := nic["ip_allocation_mode"].(string)
		ip := nic["ip"].(string)
		macAddress, macIsSet := nic["mac"].(string)

		isPrimary := nic["is_primary"].(bool)
		nicHasPrimaryChange := d.HasChange("network." + strconv.Itoa(index) + ".is_primary")
		if nicHasPrimaryChange && isPrimary {
			networkConnectionSection.PrimaryNetworkConnectionIndex = index
		}

		networkType := nic["type"].(string)
		if networkType == "org" && !isStandaloneVm {
			isVappOrgNetwork, err := isItVappOrgNetwork(networkName, *vapp)
			if err != nil {
				return types.NetworkConnectionSection{}, err
			}
			if !isVappOrgNetwork {
				return types.NetworkConnectionSection{}, fmt.Errorf("vApp Org network : %s is not found", networkName)
			}
		}
		if networkType == "vapp" && !isStandaloneVm {
			isVappNetwork, err := isItVappNetwork(networkName, *vapp)
			if err != nil {
				return types.NetworkConnectionSection{}, fmt.Errorf("unable to find vApp network %s: %s", networkName, err)
			}
			if !isVappNetwork {
				return types.NetworkConnectionSection{}, fmt.Errorf("vApp network : %s is not found", networkName)
			}
		}

		netConn.IsConnected = nic["connected"].(bool)
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

// isItVappOrgNetwork checks if it is an vApp Org network (not vApp Network)
func isItVappOrgNetwork(vAppNetworkName string, vapp govcd.VApp) (bool, error) {
	vAppNetworkConfig, err := vapp.GetNetworkConfig()
	if err != nil {
		return false, fmt.Errorf("error getting vApp networks: %s", err)
	}

	for _, networkConfig := range vAppNetworkConfig.NetworkConfig {
		if networkConfig.NetworkName == vAppNetworkName &&
			!govcd.IsVappNetwork(networkConfig.Configuration) {
			log.Printf("[TRACE] vApp Org network found: %s", vAppNetworkName)
			return true, nil
		}
	}

	return false, fmt.Errorf("configured vApp Org network isn't found: %s", vAppNetworkName)
}

// getVmNicIndexesWithDhcpEnabled loops over VMs NICs and returns list of indexes for the ones using
// DHCP
func getVmNicIndexesWithDhcpEnabled(networkConnectionSection *types.NetworkConnectionSection) []int {

	var nicIndexes []int

	// Sort NIC cards by their virtual slot numbers as the API returns them in random order
	sort.SliceStable(networkConnectionSection.NetworkConnection, func(i, j int) bool {
		return networkConnectionSection.NetworkConnection[i].NetworkConnectionIndex <
			networkConnectionSection.NetworkConnection[j].NetworkConnectionIndex
	})

	for nicIndex, singleNic := range networkConnectionSection.NetworkConnection {

		// validate if the NIC is suitable for DHCP waiting (has DHCP interface)
		if singleNic.IPAddressAllocationMode != types.IPAllocationModeDHCP {
			log.Printf("[DEBUG] [VM read] [DHCP IP Lookup] NIC '%d' is not using DHCP in 'ip_allocation_mode'. Skipping IP wait", nicIndex)
			continue
		}
		log.Printf("[DEBUG] [VM read] [DHCP IP Lookup] NIC '%d' is using DHCP in 'ip_allocation_mode'.", nicIndex)
		nicIndexes = append(nicIndexes, singleNic.NetworkConnectionIndex)

	}
	return nicIndexes
}

// readNetworks returns network configuration for saving into statefile
func readNetworks(d *schema.ResourceData, vm govcd.VM, vapp govcd.VApp, vdc *govcd.Vdc) ([]map[string]interface{}, error) {
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
		case govcd.IsVappNetwork(netConfig.Configuration):
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

	for _, vmNet := range vm.VM.NetworkConnectionSection.NetworkConnection {
		singleNIC := make(map[string]interface{})
		singleNIC["ip_allocation_mode"] = vmNet.IPAddressAllocationMode
		singleNIC["ip"] = vmNet.IPAddress
		singleNIC["mac"] = vmNet.MACAddress
		singleNIC["adapter_type"] = vmNet.NetworkAdapterType
		singleNIC["connected"] = vmNet.IsConnected
		if vmNet.Network != types.NoneNetwork {
			singleNIC["name"] = vmNet.Network
		}

		singleNIC["is_primary"] = false
		if vmNet.NetworkConnectionIndex == vm.VM.NetworkConnectionSection.PrimaryNetworkConnectionIndex {
			singleNIC["is_primary"] = true
		}

		var ok bool
		if singleNIC["type"], ok = vAppNetworkTypes[vmNet.Network]; !ok {
			// Prior vCD 10.1 used to return a placeholder for none networks. It allowed to identify
			// NIC type for types.NoneNetwork. This was removed in 10.1 therefore when vApp network
			// type has no details - the NIC network type is types.NoneNetwork
			singleNIC["type"] = types.NoneNetwork
		}

		nets = append(nets, singleNIC)
	}

	vmStatus, err := vm.GetStatus()
	if err != nil {
		return nil, fmt.Errorf("unablet to check if VM is powered on: %s", err)
	}

	// If at least one`network_dhcp_wait_seconds` was defined
	if maxDhcpWaitSeconds, ok := d.GetOk("network_dhcp_wait_seconds"); ok && vmStatus == "POWERED_ON" {
		maxDhcpWaitSecondsInt := maxDhcpWaitSeconds.(int)

		// lookup NIC indexes which have DHCP enabled
		dhcpNicIndexes := getVmNicIndexesWithDhcpEnabled(vm.VM.NetworkConnectionSection)
		log.Printf("[DEBUG] [VM read] [DHCP IP Lookup] '%s' DHCP is used on NICs %v with wait time '%d seconds'",
			vm.VM.Name, dhcpNicIndexes, maxDhcpWaitSecondsInt)
		if len(dhcpNicIndexes) == 0 {
			_, _ = fmt.Fprint(getTerraformStdout(), "INFO: Using 'network_dhcp_wait_seconds' only "+
				"makes sense if at least one NIC is using 'ip_allocation_mode=DHCP'\n")
		}

		if len(dhcpNicIndexes) > 0 { // at least one NIC uses DHCP for IP allocation mode
			log.Printf("[DEBUG] [VM read] [DHCP IP Lookup] '%s' waiting for DHCP IPs up to '%d' seconds on NICs %v",
				vm.VM.Name, maxDhcpWaitSeconds, dhcpNicIndexes)

			start := time.Now()

			// Only use DHCP lease check if it is NSX-V as NSX-T Edge Gateway does not expose it and errors on such query
			useNsxvDhcpLeaseCheck := vdc.IsNsxv()
			nicIps, timeout, err := vm.WaitForDhcpIpByNicIndexes(dhcpNicIndexes, maxDhcpWaitSecondsInt, useNsxvDhcpLeaseCheck)
			if err != nil {
				return nil, fmt.Errorf("unable to to lookup DHCP IPs for VM NICs '%v': %s", dhcpNicIndexes, err)
			}

			if timeout {
				log.Printf("[DEBUG] [VM read] [DHCP IP Lookup] VM %s timed out waiting %d seconds "+
					"to report DHCP IPs. You may want to increase 'network_dhcp_wait_seconds' or ensure "+
					"your DHCP settings are correct.\n", vm.VM.Name, maxDhcpWaitSeconds)
				_, _ = fmt.Fprintf(getTerraformStdout(), "WARNING: VM %s timed out waiting %d seconds "+
					"to report DHCP IPs. You may want to increase 'network_dhcp_wait_seconds' or ensure "+
					"your DHCP settings are correct.\n", vm.VM.Name, maxDhcpWaitSeconds)
			}

			log.Printf("[DEBUG] [VM read] [DHCP IP Lookup] VM '%s' waiting for DHCP IPs took '%s' (of '%ds')",
				vm.VM.Name, time.Since(start), maxDhcpWaitSeconds)

			for sliceIndex, nicIndex := range dhcpNicIndexes {
				log.Printf("[DEBUG] [VM read] [DHCP IP Lookup] VM '%s' NIC %d reported IP %s",
					vm.VM.Name, nicIndex, nicIps[sliceIndex])
				nets[nicIndex]["ip"] = nicIps[sliceIndex]
			}
		}
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

	// if properties object does not have actual properties - do not set it at all (leave Terraform 'null')
	log.Printf("[TRACE] Setting empty properties into statefile because no properties were specified")
	if properties == nil || properties.ProductSection == nil || len(properties.ProductSection.Property) == 0 {
		return nil
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
// Example import path for VM within vApp (_the_id_string_): org-name.vdc-name.vapp-name.vm-name
// Example import path for standalone VM (_the_id_string_): org-name.vdc-name.vm-name
// or
// Example import path for standalone VM (_the_id_string_): org-name.vdc-name.vm-ID
//
// The VM identifier can be either the VM name or its ID
// If we are dealing with standalone VMs, the name can retrieve duplicates. When that happens, the import fails
// and a list of VM information (ID, guest OS, network, IP) is returned
func resourceVcdVappVmImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	resourceURI := strings.Split(d.Id(), ImportSeparator)
	vcdClient := meta.(*VCDClient)

	var vapp *govcd.VApp
	var vm *govcd.VM
	var orgName string
	var vdcName string
	var vappName string
	var vmIdentifier string
	standaloneVm := false

	// With three arguments, we expect a standalone VM
	if len(resourceURI) == 3 {
		// standalone VM
		orgName, vdcName, vmIdentifier = resourceURI[0], resourceURI[1], resourceURI[2]
		standaloneVm = true
	} else {
		// With 4 arguments, it's a VM within a vApp
		if len(resourceURI) != 4 {
			return nil, fmt.Errorf("[VM import] resource name must be specified as org-name.vdc-name.vapp-name.vm-name-or-ID")
		}
		orgName, vdcName, vappName, vmIdentifier = resourceURI[0], resourceURI[1], resourceURI[2], resourceURI[3]
	}

	isUuid := extractUuid(vmIdentifier) != ""
	_, vdc, err := vcdClient.GetOrgAndVdc(orgName, vdcName)
	if err != nil {
		return nil, fmt.Errorf("[VM import] unable to find VDC %s: %s ", vdcName, err)
	}

	if standaloneVm {
		if isUuid {
			vm, err = vdc.QueryVmById(vmIdentifier)
			if err != nil {
				return nil, fmt.Errorf("[VM import] error retrieving VM %s by ID: %s", vmIdentifier, err)
			}
		} else {
			vmByName, listStr, err := getVmByName(vcdClient, vdc, vmIdentifier)
			if err != nil {
				return nil, fmt.Errorf("[VM import] error retrieving VM %s by name: %s\n%s", vmIdentifier, err, listStr)
			}
			vm = vmByName
		}

	} else {
		vapp, err = vdc.GetVAppByName(vappName, false)
		if err != nil {
			return nil, fmt.Errorf("[VM import] error retrieving vApp %s: %s", vappName, err)
		}
		vm, err = vapp.GetVMByNameOrId(vmIdentifier, false)
		if err != nil {
			return nil, fmt.Errorf("[VM import] error retrieving VM %s: %s", vmIdentifier, err)
		}
	}

	_ = d.Set("name", vmIdentifier)
	_ = d.Set("org", orgName)
	_ = d.Set("vdc", vdcName)
	_ = d.Set("vapp_name", vappName)
	d.SetId(vm.VM.ID)
	return []*schema.ResourceData{d}, nil
}

// getVmByName returns a VM by the given name if found unequivocally
// If there are more than one instance by the wanted name, it also returns a list of
// matching VMs with sample information (ID, guest OS, network name, IP address)
func getVmByName(client *VCDClient, vdc *govcd.Vdc, name string) (*govcd.VM, string, error) {

	vmList, err := vdc.QueryVmList(types.VmQueryFilterOnlyDeployed)
	if err != nil {
		return nil, "", err
	}

	var foundList []*types.QueryResultVMRecordType
	for _, vm := range vmList {
		if vm.Name == name {
			foundList = append(foundList, vm)
		}
	}

	if len(foundList) == 0 {
		return nil, "", govcd.ErrorEntityNotFound
	}
	if len(foundList) == 1 {
		vm, err := client.Client.GetVMByHref(foundList[0].HREF)
		if err != nil {
			return nil, "", err
		}
		return vm, "", nil
	}
	// More than one element found for the given name. Returning the list
	listStr := fmt.Sprintf("%-50s %-30s %s\n", "ID", "Guest OS", "Network")
	listStr += fmt.Sprintf("%-50s %-30s %s\n", strings.Repeat("-", 50), strings.Repeat("-", 30), strings.Repeat("-", 20))
	for _, vm := range foundList {
		id := extractUuid(vm.HREF)
		networkInfo := ""
		if vm.NetworkName != "" || vm.IpAddress != "" {
			networkInfo = fmt.Sprintf("(%s - %s)", vm.NetworkName, vm.IpAddress)
		}
		listStr += fmt.Sprintf("urn:vcloud:vm:%s %-30s %s\n", id, vm.GuestOS, networkInfo)
	}
	return nil, listStr, fmt.Errorf("more than one VM found with name %s", name)
}

// updateGuestCustomizationSetting is responsible for setting all the data related to VM customization
func updateGuestCustomizationSetting(d *schema.ResourceData, vm *govcd.VM) error {
	// Retrieve existing customization section to only customize what was throughout this function
	customizationSection, err := vm.GetGuestCustomizationSection()
	if err != nil {
		return fmt.Errorf("error getting existing customization section before changing: %s", err)
	}

	// for back compatibility we allow to set computer name from `name` if computer_name isn't provided
	var computerName string
	if cName, ok := d.GetOk("computer_name"); ok {
		computerName = cName.(string)
	} else {
		computerName = d.Get("name").(string)
	}

	if _, isSetComputerName := d.GetOk("computer_name"); isSetComputerName {
		customizationSection.ComputerName = computerName
	}

	// Process parameters from 'customization' block
	updateCustomizationSection(d.Get("customization"), d, customizationSection)

	// Apply any of the settings we have set
	if _, err = vm.SetGuestCustomizationSection(customizationSection); err != nil {
		return fmt.Errorf("error applying guest customization details: %s", err)
	}

	return nil
}

func updateCustomizationSection(customizationInterface interface{}, d *schema.ResourceData, customizationSection *types.GuestCustomizationSection) {
	customizationSlice := customizationInterface.([]interface{})
	if len(customizationSlice) == 1 {
		cust := customizationSlice[0]
		if cust != nil {

			if enabled, isSetEnabled := d.GetOkExists("customization.0.enabled"); isSetEnabled {
				customizationSection.Enabled = takeBoolPointer(enabled.(bool))
			}
			if initScript, isSetInitScript := d.GetOkExists("customization.0.initscript"); isSetInitScript {
				customizationSection.CustomizationScript = initScript.(string)
			}

			if changeSid, isSetChangeSid := d.GetOkExists("customization.0.change_sid"); isSetChangeSid {
				customizationSection.ChangeSid = takeBoolPointer(changeSid.(bool))
			}

			if allowLocalAdminPasswd, isSetAllowLocalAdminPasswd := d.GetOkExists("customization.0.allow_local_admin_password"); isSetAllowLocalAdminPasswd {
				customizationSection.AdminPasswordEnabled = takeBoolPointer(allowLocalAdminPasswd.(bool))

			}

			if mustChangeOnFirstLogin, isSetMustChangeOnFirstLogin := d.GetOkExists("customization.0.must_change_password_on_first_login"); isSetMustChangeOnFirstLogin {
				customizationSection.ResetPasswordRequired = takeBoolPointer(mustChangeOnFirstLogin.(bool))
			}

			if autoGeneratePasswd, isSetAutoGeneratePasswd := d.GetOkExists("customization.0.auto_generate_password"); isSetAutoGeneratePasswd {
				customizationSection.AdminPasswordAuto = takeBoolPointer(autoGeneratePasswd.(bool))
			}

			if adminPasswd, isSetAdminPasswd := d.GetOkExists("customization.0.admin_password"); isSetAdminPasswd {
				customizationSection.AdminPassword = adminPasswd.(string)
				// customizationSection.AdminPasswordEnabled = takeBoolPointer(true)
			}

			if nrTimesForLogin, isSetNrTimesForLogin := d.GetOkExists("customization.0.number_of_auto_logons"); isSetNrTimesForLogin {
				// The AdminAutoLogonEnabled is "hidden" from direct user input to behave exactly like UI does. UI sets
				// the value of this field behind the scenes based on number_of_auto_logons count.
				// AdminAutoLogonEnabled=false if number_of_auto_logons == 0
				// AdminAutoLogonEnabled=true if number_of_auto_logons > 0
				isMoreThanZero := nrTimesForLogin.(int) > 0
				customizationSection.AdminAutoLogonEnabled = takeBoolPointer(isMoreThanZero)

				customizationSection.AdminAutoLogonCount = nrTimesForLogin.(int)
			}

			if joinDomain, isSetJoinDomain := d.GetOkExists("customization.0.join_domain"); isSetJoinDomain {
				customizationSection.JoinDomainEnabled = takeBoolPointer(joinDomain.(bool))
			}

			if joinOrgDomain, isSetJoinOrgDomain := d.GetOkExists("customization.0.join_org_domain"); isSetJoinOrgDomain {
				customizationSection.UseOrgSettings = takeBoolPointer(joinOrgDomain.(bool))
			}

			if joinDomainName, isSetJoinDomainName := d.GetOkExists("customization.0.join_domain_name"); isSetJoinDomainName {
				customizationSection.DomainName = joinDomainName.(string)
			}

			if joinDomainUser, isSetJoinDomainUser := d.GetOkExists("customization.0.join_domain_user"); isSetJoinDomainUser {
				customizationSection.DomainUserName = joinDomainUser.(string)
			}

			if joinDomainPasswd, isSetJoinDomainPasswd := d.GetOkExists("customization.0.join_domain_password"); isSetJoinDomainPasswd {
				customizationSection.DomainUserPassword = joinDomainPasswd.(string)
			}

			if joinDomainOu, isSetJoinDomainOu := d.GetOkExists("customization.0.join_domain_account_ou"); isSetJoinDomainOu {
				customizationSection.MachineObjectOU = joinDomainOu.(string)
			}

		}
	}
}

// setGuestCustomizationData is responsible for persisting all guest customization details into statefile
func setGuestCustomizationData(d *schema.ResourceData, vm *govcd.VM) error {
	customizationSection, err := vm.GetGuestCustomizationSection()
	if err != nil {
		return fmt.Errorf("unable to get guest customization section: %s", err)
	}

	_ = d.Set("computer_name", customizationSection.ComputerName)

	customizationBlock := make([]interface{}, 1)
	customizationBlockAttributes := make(map[string]interface{})

	customizationBlockAttributes["enabled"] = customizationSection.Enabled
	customizationBlockAttributes["change_sid"] = customizationSection.ChangeSid
	customizationBlockAttributes["allow_local_admin_password"] = customizationSection.AdminPasswordEnabled
	customizationBlockAttributes["must_change_password_on_first_login"] = customizationSection.ResetPasswordRequired
	customizationBlockAttributes["auto_generate_password"] = customizationSection.AdminPasswordAuto
	customizationBlockAttributes["admin_password"] = customizationSection.AdminPassword
	customizationBlockAttributes["number_of_auto_logons"] = customizationSection.AdminAutoLogonCount
	customizationBlockAttributes["join_domain"] = customizationSection.JoinDomainEnabled
	customizationBlockAttributes["join_org_domain"] = customizationSection.UseOrgSettings
	customizationBlockAttributes["join_domain_name"] = customizationSection.DomainName
	customizationBlockAttributes["join_domain_user"] = customizationSection.DomainUserName
	customizationBlockAttributes["join_domain_password"] = customizationSection.DomainUserPassword
	customizationBlockAttributes["join_domain_account_ou"] = customizationSection.MachineObjectOU
	customizationBlockAttributes["initscript"] = customizationSection.CustomizationScript

	customizationBlock[0] = customizationBlockAttributes

	err = d.Set("customization", customizationBlock)
	if err != nil {
		return fmt.Errorf("")
	}

	return nil
}

func addEmptyVm(d *schema.ResourceData, vcdClient *VCDClient, org *govcd.Org, vdc *govcd.Vdc, vappName string) (*govcd.VM, error) {
	util.Logger.Printf("[TRACE] Creating empty VM: %s", d.Get("name").(string))

	var err error
	var vapp *govcd.VApp

	if vappName != "" {
		vapp, err = vdc.GetVAppByName(vappName, false)
		if err != nil {
			return nil, err
		}
	}
	var ok bool
	var memory interface{}
	_, sizingOk := d.GetOk("sizing_policy_id")
	if memory, ok = d.GetOk("memory"); !ok && !sizingOk {
		return nil, fmt.Errorf("`memory` or `sizing_policy_id` is required when creating empty VM")
	}

	var osType interface{}
	if osType, ok = d.GetOk("os_type"); !ok {
		return nil, fmt.Errorf("`os_type` is required when creating empty VM")
	}

	var hardWareVersion interface{}
	if hardWareVersion, ok = d.GetOk("hardware_version"); !ok {
		return nil, fmt.Errorf("`hardware_version` is required when creating empty VM")
	}

	var computerName interface{}
	if computerName, ok = d.GetOk("computer_name"); !ok {
		return nil, fmt.Errorf("`computer_name` is required when creating empty VM")
	}

	var bootImage *types.Media
	if bootImageName, ok := d.GetOk("boot_image"); ok {
		var catalogName interface{}
		if catalogName, ok = d.GetOk("catalog_name"); !ok {
			return nil, fmt.Errorf("`catalog_name` is required when creating empty VM with boot_image")
		}
		catalog, err := org.GetCatalogByName(catalogName.(string), false)
		if err != nil {
			return nil, fmt.Errorf("error finding catalog %s: %s", catalogName, err)
		}
		result, err := catalog.GetMediaByName(bootImageName.(string), false)
		if err != nil {
			return nil, fmt.Errorf("[VM creation] error getting boot image %s : %s", bootImageName, err)
		}

		bootImage = &types.Media{HREF: result.Media.HREF, Name: result.Media.Name, ID: result.Media.ID}
	} else {
		bootImage = nil
	}

	customizationSection := &types.GuestCustomizationSection{}
	customizationSection.ComputerName = computerName.(string)

	// Process parameters from 'customization' block
	updateCustomizationSection(d.Get("customization"), d, customizationSection)

	isVirtualCpuType64 := strings.Contains(d.Get("os_type").(string), "64")
	virtualCpuType := "VM32"
	if isVirtualCpuType64 {
		virtualCpuType = "VM64"
	}
	vmName := d.Get("name").(string)
	recomposeVAppParamsForEmptyVm := &types.RecomposeVAppParamsForEmptyVm{
		XmlnsVcloud: types.XMLNamespaceVCloud,
		XmlnsOvf:    types.XMLNamespaceOVF,
		CreateItem: &types.CreateItem{
			Name: vmName,
			// Bug in vCD - accepts only org VDC networks and automatically add them. We add them with update
			// BUG in vCD 9.5 version, do not allow empty NetworkConnectionSection, so we pass simplest network configuration
			// and after VM created update with real config
			NetworkConnectionSection: &types.NetworkConnectionSection{
				PrimaryNetworkConnectionIndex: 0,
				NetworkConnection: []*types.NetworkConnection{
					&types.NetworkConnection{Network: "none", NetworkConnectionIndex: 0, IPAddress: "any", IsConnected: false, IPAddressAllocationMode: "NONE"}},
			},
			Description:               d.Get("description").(string),
			GuestCustomizationSection: customizationSection,
			VmSpecSection: &types.VmSpecSection{
				Modified:          takeBoolPointer(true),
				Info:              "Virtual Machine specification",
				OsType:            osType.(string),
				NumCpus:           takeIntPointer(d.Get("cpus").(int)),
				NumCoresPerSocket: takeIntPointer(d.Get("cpu_cores").(int)),
				CpuResourceMhz:    &types.CpuResourceMhz{Configured: 0},
				MemoryResourceMb:  &types.MemoryResourceMb{Configured: int64(memory.(int))},
				MediaSection:      nil,
				// can be created with resource internal_disk
				DiskSection:      &types.DiskSection{DiskSettings: []*types.DiskSettings{}},
				HardwareVersion:  &types.HardwareVersion{Value: hardWareVersion.(string)}, // need support older version vCD
				VmToolsVersion:   "",
				VirtualCpuType:   virtualCpuType,
				TimeSyncWithHost: nil,
			},
			BootImage: bootImage,
		},
		AllEULAsAccepted: true,
	}

	err = addSizingPolicy(d, vcdClient, org, recomposeVAppParamsForEmptyVm)
	if err != nil {
		return nil, err
	}

	var newVm *govcd.VM
	var mediaReference *types.Reference
	if bootImage != nil {
		mediaReference = &types.Reference{
			HREF: bootImage.HREF,
			ID:   bootImage.ID,
			Type: bootImage.Type,
			Name: bootImage.Name,
		}
	}
	if vappName == "" {
		params := types.CreateVmParams{
			Name:        vmName,
			PowerOn:     false,
			Description: recomposeVAppParamsForEmptyVm.CreateItem.Description,
			CreateVm: &types.Vm{
				Name:                      vmName,
				ComputePolicy:             recomposeVAppParamsForEmptyVm.CreateItem.ComputePolicy,
				NetworkConnectionSection:  recomposeVAppParamsForEmptyVm.CreateItem.NetworkConnectionSection,
				VmSpecSection:             recomposeVAppParamsForEmptyVm.CreateItem.VmSpecSection,
				GuestCustomizationSection: recomposeVAppParamsForEmptyVm.CreateItem.GuestCustomizationSection,
				Media:                     mediaReference,
			},
			Xmlns: types.XMLNamespaceVCloud,
		}
		newVm, err = vdc.CreateStandaloneVm(&params)
		if err != nil {
			return nil, err
		}
	} else {
		util.Logger.Printf("[VM create - add empty VM] recomposeVAppParamsForEmptyVm %# v", pretty.Formatter(recomposeVAppParamsForEmptyVm))
		newVm, err = vapp.AddEmptyVm(recomposeVAppParamsForEmptyVm)
		if err != nil {
			d.SetId("")
			return nil, fmt.Errorf("[VM creation] error creating VM %s : %s", vmName, err)
		}
	}

	d.SetId(newVm.VM.ID)

	// Due the Bug in vCD VM creation(works only with org VDC networks, not vapp) - we setup network configuration with update. Fixed only 10.1 version.
	networkConnectionSection, err := networksToConfig(d, vdc, vapp, vcdClient)
	if err != nil {
		return nil, fmt.Errorf("unable to setup network configuration for empty VM: %s", err)
	}
	// firstly cleanup dummy network as network adapter type can't be changed
	err = newVm.UpdateNetworkConnectionSection(&types.NetworkConnectionSection{})
	if err != nil {
		return nil, fmt.Errorf("unable to setup network configuration for empty VM %s", err)
	}
	// add real network configuration
	err = newVm.UpdateNetworkConnectionSection(&networkConnectionSection)
	if err != nil {
		return nil, fmt.Errorf("unable to setup network configuration for empty VM %s", err)
	}

	err = handleExposeHardwareVirtualization(d, newVm)
	if err != nil {
		return nil, err
	}

	err = addRemoveGuestProperties(d, newVm)
	if err != nil {
		return nil, err
	}

	if d.HasChange("cpu_hot_add_enabled") || d.HasChange("memory_hot_add_enabled") {
		_, err := newVm.UpdateVmCpuAndMemoryHotAdd(d.Get("cpu_hot_add_enabled").(bool), d.Get("memory_hot_add_enabled").(bool))
		if err != nil {
			return nil, fmt.Errorf("error changing VM capabilities: %s", err)
		}
	}

	err = addRemoveMetaData(d, newVm)
	if err != nil {
		return nil, err
	}

	if d.Get("power_on").(bool) {
		log.Printf("[DEBUG] Powering on VM %s", newVm.VM.Name)
		task, err := newVm.PowerOn()
		if err != nil {
			return nil, fmt.Errorf("error powering on: %s", err)
		}
		err = task.WaitTaskCompletion()
		if err != nil {
			return nil, fmt.Errorf(errorCompletingTask, err)
		}
	}
	return newVm, nil
}

func addSizingPolicy(d *schema.ResourceData, vcdClient *VCDClient, org *govcd.Org, recomposeVAppParamsForEmptyVm *types.RecomposeVAppParamsForEmptyVm) error {
	vcdComputePolicyHref, err := vcdClient.Client.OpenApiBuildEndpoint(types.OpenApiPathVersion1_0_0, types.OpenApiEndpointVdcComputePolicies)
	if err != nil {
		return fmt.Errorf("error constructing HREF for compute policy")
	}

	if value, ok := d.GetOk("sizing_policy_id"); ok {
		recomposeVAppParamsForEmptyVm.CreateItem.ComputePolicy = &types.ComputePolicy{VmSizingPolicy: &types.Reference{HREF: vcdComputePolicyHref.String() + value.(string)}}
		sizingPolicy, err := org.GetVdcComputePolicyById(value.(string))
		if err != nil {
			return fmt.Errorf("error getting sizing policy %s: %s", value.(string), err)
		}
		if _, ok = d.GetOk("cpus"); !ok && sizingPolicy.VdcComputePolicy.CPUCount == nil {
			return fmt.Errorf("`cpus` has to be defined as provided sizing policy `sizing_policy_id` cpu count isn't configured")
		}
		if sizingPolicy.VdcComputePolicy.CPUCount != nil {
			recomposeVAppParamsForEmptyVm.CreateItem.VmSpecSection.NumCpus = sizingPolicy.VdcComputePolicy.CPUCount
		}
		if _, ok = d.GetOk("cpu_cores"); !ok && sizingPolicy.VdcComputePolicy.CoresPerSocket == nil {
			return fmt.Errorf("`cpu_cores` has to be defined as provided sizing policy `sizing_policy_id` cpu cores per socket isn't configured")
		}
		if sizingPolicy.VdcComputePolicy.CoresPerSocket != nil {
			recomposeVAppParamsForEmptyVm.CreateItem.VmSpecSection.NumCoresPerSocket = sizingPolicy.VdcComputePolicy.CoresPerSocket
		}
		if _, ok = d.GetOk("memory"); !ok && sizingPolicy.VdcComputePolicy.Memory == nil {
			return fmt.Errorf("`memory` has to be defined as provided sizing policy `sizing_policy_id` memory isn't configured")
		}
		if sizingPolicy.VdcComputePolicy.Memory != nil {
			recomposeVAppParamsForEmptyVm.CreateItem.VmSpecSection.MemoryResourceMb = &types.MemoryResourceMb{Configured: int64(*sizingPolicy.VdcComputePolicy.Memory)}
		}
	}
	return nil
}

// handleExposeHardwareVirtualization toggles hardware virtualization according `expose_hardware_virtualization` field value.
func handleExposeHardwareVirtualization(d *schema.ResourceData, newVm *govcd.VM) error {
	// The below operation assumes VM is powered off and does not check for it because VM is being
	// powered on in the last stage of create/update cycle
	if d.Get("expose_hardware_virtualization").(bool) {

		task, err := newVm.ToggleHardwareVirtualization(true)
		if err != nil {
			return fmt.Errorf("error enabling hardware assisted virtualization: %s", err)
		}
		err = task.WaitTaskCompletion()

		if err != nil {
			return fmt.Errorf(errorCompletingTask, err)
		}
	}
	return nil
}
