package vcd

//lint:file-ignore SA1019 ignore deprecated functions
import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func vcdVmDS(vmType typeOfVm) map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"vapp_name": &schema.Schema{
			Type:        schema.TypeString,
			Required:    vmType == vappVmType,
			Optional:    vmType == standaloneVmType,
			Computed:    vmType == standaloneVmType,
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
			Description: "A name for the VM, unique within the vApp",
		},
		"org": {
			Type:     schema.TypeString,
			Optional: true,
			Description: "The name of organization to use, optional if defined at provider " +
				"level. Useful when connected as sysadmin working across different organizations",
		},
		"vdc": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The name of VDC to use, optional if defined at provider level",
		},
		"computer_name": &schema.Schema{
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Computer name assigned to this virtual machine",
		},
		"description": &schema.Schema{
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The VM description",
			// Currently, this field has the description of the OVA used to create the VM
		},
		"memory": &schema.Schema{
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "The amount of RAM (in MB) to allocate to the VM",
		},
		"cpus": &schema.Schema{
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "The number of virtual CPUs to allocate to the VM",
		},
		"cpu_cores": &schema.Schema{
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "The number of cores per socket",
		},
		"metadata": {
			Type:        schema.TypeMap,
			Computed:    true,
			Description: "Key value map of metadata to assign to this VM",
		},
		"href": &schema.Schema{
			Type:        schema.TypeString,
			Computed:    true,
			Description: "VM Hyper Reference",
		},
		"storage_profile": &schema.Schema{
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Storage profile used with the VM",
		},
		"os_type": &schema.Schema{
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Operating System type.",
		},
		"hardware_version": &schema.Schema{
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Virtual Hardware Version.",
		},
		"network_dhcp_wait_seconds": {
			Optional:     true,
			Type:         schema.TypeInt,
			ValidateFunc: validation.IntAtLeast(0),
			Description: "Optional number of seconds to try and wait for DHCP IP (valid for " +
				"'network' block only)",
		},
		"network": {
			Computed:    true,
			Type:        schema.TypeList,
			Description: "",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"type": {
						Computed:    true,
						Type:        schema.TypeString,
						Description: "Network type",
					},
					"ip_allocation_mode": {
						Computed:    true,
						Type:        schema.TypeString,
						Description: "IP address allocation mode.",
					},
					"name": {
						Computed:    true,
						Type:        schema.TypeString,
						Description: "Name of the network this VM should connect to.",
					},
					"ip": {
						Computed:    true,
						Type:        schema.TypeString,
						Description: "IP of the VM. Settings depend on `ip_allocation_mode`",
					},
					"is_primary": {
						Computed:    true,
						Type:        schema.TypeBool,
						Description: "Set to true if network interface should be primary. First network card in the list will be primary by default",
					},
					"mac": {
						Computed:    true,
						Type:        schema.TypeString,
						Description: "Mac address of network interface",
					},
					"adapter_type": {
						Type:        schema.TypeString,
						Computed:    true,
						Description: "Network card adapter type. (e.g. 'E1000', 'E1000E', 'SRIOVETHERNETCARD', 'VMXNET3', 'PCNet32')",
					},
					"connected": {
						Type:        schema.TypeBool,
						Computed:    true,
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
					Computed:    true,
					Description: "Independent disk name",
				},
				"bus_number": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Bus number on which to place the disk controller",
				},
				"unit_number": {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Unit number (slot) on the bus specified by BusNumber",
				},
				"size_in_mb": {
					Type:        schema.TypeInt,
					Computed:    true,
					Description: "The size of the disk in MB.",
				},
			}},
			Computed: true,
			Set:      resourceVcdVmIndependentDiskHash,
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
			Computed:    true,
			Description: "Expose hardware-assisted CPU virtualization to guest OS.",
		},
		"guest_properties": {
			Type:        schema.TypeMap,
			Computed:    true,
			Description: "Key/value settings for guest properties",
		},

		"customization": &schema.Schema{
			Computed:    true,
			Type:        schema.TypeList,
			Description: "Guest customization block",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"force": {
						Type:        schema.TypeBool,
						Computed:    true,
						Description: "'true' value will cause the VM to reboot on every 'apply' operation",
					},
					"enabled": {
						Type:        schema.TypeBool,
						Computed:    true,
						Description: "'true' value will enable guest customization. It may occur on first boot or when 'force' is used",
					},
					"change_sid": {
						Type:        schema.TypeBool,
						Computed:    true,
						Description: "'true' value will change SID. Applicable only for Windows VMs",
					},
					"allow_local_admin_password": {
						Type:        schema.TypeBool,
						Computed:    true,
						Description: "Allow local administrator password",
					},
					"must_change_password_on_first_login": {
						Type:        schema.TypeBool,
						Computed:    true,
						Description: "Require Administrator to change password on first login",
					},
					"auto_generate_password": {
						Type:        schema.TypeBool,
						Computed:    true,
						Description: "Auto generate password",
					},
					"admin_password": {
						Type:        schema.TypeString,
						Computed:    true,
						Sensitive:   true,
						Description: "Manually specify admin password",
					},
					"number_of_auto_logons": {
						Type:        schema.TypeInt,
						Computed:    true,
						Description: "Number of times to log on automatically",
					},
					"join_domain": {
						Type:        schema.TypeBool,
						Computed:    true,
						Description: "Enable this VM to join a domain",
					},
					"join_org_domain": {
						Type:        schema.TypeBool,
						Computed:    true,
						Description: "Use organization's domain for joining",
					},
					"join_domain_name": {
						Type:        schema.TypeString,
						Computed:    true,
						Description: "Custom domain name for join",
					},
					"join_domain_user": {
						Type:        schema.TypeString,
						Computed:    true,
						Description: "Username for custom domain name join",
					},
					"join_domain_password": {
						Type:        schema.TypeString,
						Computed:    true,
						Sensitive:   true,
						Description: "Password for custom domain name join",
					},
					"join_domain_account_ou": {
						Type:        schema.TypeString,
						Computed:    true,
						Description: "Account organizational unit for domain name join",
					},
					"initscript": &schema.Schema{
						Type:        schema.TypeString,
						Computed:    true,
						Description: "Script to run on initial boot or with customization.force=true set",
					},
				},
			},
		},
		"cpu_hot_add_enabled": {
			Type:        schema.TypeBool,
			Computed:    true,
			Description: "True if the virtual machine supports addition of virtual CPUs while powered on.",
		},
		"memory_hot_add_enabled": {
			Type:        schema.TypeBool,
			Computed:    true,
			Description: "True if the virtual machine supports addition of memory while powered on.",
		},
		"sizing_policy_id": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "VM sizing policy ID.",
		},
	}
}

func datasourceVcdVAppVm() *schema.Resource {
	return &schema.Resource{
		Read:   datasourceVcdVAppVmRead,
		Schema: vcdVmDS(vappVmType),
	}
}

func datasourceVcdVAppVmRead(d *schema.ResourceData, meta interface{}) error {
	return genericVcdVmRead(d, meta, "datasource", vappVmType)
}
