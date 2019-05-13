package vcd

import (
	"errors"
	"fmt"
	"github.com/hashicorp/terraform/helper/validation"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/vmware/go-vcloud-director/v2/govcd"
	"github.com/vmware/go-vcloud-director/v2/types/v56"
)

func resourceVcdOrgVdc() *schema.Resource {
	capacityWithUsage := schema.Schema{
		Type:     schema.TypeSet,
		Required: true,
		ForceNew: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"units": {
					Type:         schema.TypeString,
					Required:     true,
					ForceNew:     true,
					ValidateFunc: validation.StringInSlice([]string{"MHz", "GHz", "MB", "GB"}, false),
					Description:  "Units in which capacity is allocated. For CPU capacity, one of: {MHz, GHz}.  For memory capacity, one of: {MB, GB}.",
				},
				"allocated": {
					Type:        schema.TypeInt,
					Optional:    true,
					ForceNew:    true,
					Description: "Capacity that is committed to be available.",
				},
				"limit": {
					Type:        schema.TypeInt,
					Required:    true,
					ForceNew:    true,
					Description: "Capacity limit relative to the value specified for Allocation. It must not be less than that value. If it is greater than that value, it implies over provisioning.",
				},
				"reserved": {
					Type:        schema.TypeInt,
					Optional:    true,
					ForceNew:    true,
					Description: "Capacity reserved",
				},
				"used": {
					Type:        schema.TypeInt,
					Optional:    true,
					ForceNew:    true,
					Description: "Capacity used. If the VDC AllocationModel is ReservationPool, this number represents the percentage of the reservation that is in use. For all other allocation models, it represents the percentage of the allocation that is in use.",
				},
				"overhead": {
					Type:        schema.TypeInt,
					Optional:    true,
					ForceNew:    true,
					Description: "Number of Units allocated to system resources such as vShield Manager virtual machines and shadow virtual machines provisioned from this Provider VDC.",
				},
			},
		},
	}

	return &schema.Resource{
		Create: resourceVcdVdcCreate,
		Delete: resourceVcdVdcDelete,
		Read:   resourceVcdVdcRead,

		Schema: map[string]*schema.Schema{
			"org": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Organization to create the VDC in",
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"allocation_model": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{"AllocationVApp", "AllocationPool", "ReservationPool"}, false),
				Description:  "The allocation model used by this VDC; must be one of {AllocationVApp, AllocationPool, ReservationPool}",
			},
			"compute_capacity": &schema.Schema{
				Required: true,
				ForceNew: true,
				Type:     schema.TypeSet,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"cpu":    &capacityWithUsage,
						"memory": &capacityWithUsage,
					},
				},
				Description: "The compute capacity allocated to this VDC.",
			},
			"nic_quota": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    true,
				Description: "Maximum number of virtual NICs allowed in this VDC. Defaults to 0, which specifies an unlimited number.",
			},
			"network_quota": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    true,
				Description: "Maximum number of network objects that can be deployed in this VDC. Defaults to 0, which means no networks can be deployed.",
			},
			"vm_quota": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    true,
				Description: "The maximum number of VMs that can be created in this VDC. Includes deployed and undeployed VMs in vApps and vApp templates. Defaults to 0, which specifies an unlimited number.",
			},
			"enabled": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				ForceNew:    true,
				Default:     true,
				Description: "True if this VDC is enabled for use by the organization VDCs. Default is true",
			},
			"storage_profile": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Name of Provider VDC storage profile.",
						},
						"enabled": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "True if this storage profile is enabled for use in the VDC.",
						},
						"limit": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "Maximum number of Units allocated for this storage profile. A value of 0 specifies unlimited Units.",
						},
						"default": {
							Type:        schema.TypeBool,
							Required:    true,
							Description: "True if this is default storage profile for this VDC. The default storage profile is used when an object that can specify a storage profile is created with no storage profile specified.",
						},
					},
				},
				Description: "Storage profiles supported by this VDC.",
			},
			"memory_guaranteed": &schema.Schema{
				Type:        schema.TypeFloat,
				Optional:    true,
				ForceNew:    true,
				Description: "Percentage of allocated memory resources guaranteed to vApps deployed in this VDC. For example, if this value is 0.75, then 75% of allocated resources are guaranteed. Required when AllocationModel is AllocationVApp or AllocationPool. Value defaults to 1.0 if the element is empty.",
			},
			"cpu_guaranteed": &schema.Schema{
				Type:        schema.TypeFloat,
				Optional:    true,
				ForceNew:    true,
				Description: "Percentage of allocated CPU resources guaranteed to vApps deployed in this VDC. For example, if this value is 0.75, then 75% of allocated resources are guaranteed. Required when AllocationModel is AllocationVApp or AllocationPool. Value defaults to 1.0 if the element is empty.",
			},
			"cpu_frequency": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    true,
				Description: "Specifies the clock frequency, in Megahertz, for any virtual CPU that is allocated to a VM. A VM with 2 vCPUs will consume twice as much of this value. Ignored for ReservationPool. Required when AllocationModel is AllocationVApp or AllocationPool, and may not be less than 256 MHz. Defaults to 1000 MHz if the element is empty or missing.",
			},
			"enable_thin_provisioning": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				ForceNew:    true,
				Description: "Boolean to request thin provisioning. Request will be honored only if the underlying datastore supports it. Thin provisioning saves storage space by committing it on demand. This allows over-allocation of storage.",
			},
			"network_pool_name": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "A name of a network pool in the Provider VDC. Required if this VDC will contain routed or isolated networks.",
			},
			"provider_vdc_name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "A reference to the Provider VDC from which this organization VDC is provisioned.",
			},
			"enable_fast_provisioning": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				ForceNew:    true,
				Description: "Boolean to request fast provisioning. Request will be honored only if the underlying datas tore supports it. Fast provisioning can reduce the time it takes to create virtual machines by using vSphere linked clones. If you disable fast provisioning, all provisioning operations will result in full clones.",
			},
			"over_commit_allowed": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				ForceNew:    true,
				Default:     true,
				Description: "Set to false to disallow creation of the VDC if the AllocationModel is AllocationPool or ReservationPool and the ComputeCapacity you specified is greater than what the backing Provider VDC can supply. Default is true.",
			},
			"enable_vm_discovery": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				ForceNew:    true,
				Description: "True if discovery of vCenter VMs is enabled for resource pools backing this VDC. If left unspecified, the actual behaviour depends on enablement at the organization level and at the system level.",
			},

			"delete_force": &schema.Schema{
				Type:        schema.TypeBool,
				Required:    true,
				ForceNew:    true,
				Description: "When destroying use delete_force=True to remove a vdc and any objects it contains, regardless of their state.",
			},
			"delete_recursive": &schema.Schema{
				Type:        schema.TypeBool,
				Required:    true,
				ForceNew:    true,
				Description: "When destroying use delete_recursive=True to remove the vdc and any objects it contains that are in a state that normally allows removal.",
			},
		},
	}
}

// Creates a new vdc from a resource definition
func resourceVcdVdcCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[TRACE] vdc creation initiated")

	vcdClient := meta.(*VCDClient)

	if !vcdClient.Client.IsSysAdmin {
		return fmt.Errorf("functionality requires system administrator privileges")
	}

	// vdc creation is accessible only in administrator API part
	adminOrg, err := vcdClient.GetAdminOrgFromResource(d)
	if err != nil {
		return fmt.Errorf(errorRetrievingOrg, err)
	}

	params, err := getVcdVdcInput(d, vcdClient)
	if err != nil {
		return err
	}

	task, err := adminOrg.CreateVdc(params)
	if err != nil {
		log.Printf("[DEBUG] Error creating vdc: %#v", err)
		return fmt.Errorf("error creating vdc: %#v", err)
	}

	err = task.WaitTaskCompletion()
	if err != nil {
		log.Printf("[DEBUG] Error waiting for vdc to finish: %#v", err)
		return fmt.Errorf("error waiting for vdc to finish: %#v", err)
	}

	d.SetId(d.Get("name").(string))
	log.Printf("[TRACE] vdc created: %#v", task)
	return resourceVcdVdcRead(d, meta)
}

// Fetches information about an existing vdc for a data definition
func resourceVcdVdcRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[TRACE] vdc read initiated")

	vcdClient := meta.(*VCDClient)

	adminOrg, err := vcdClient.GetAdminOrgFromResource(d)
	if err != nil {
		return fmt.Errorf(errorRetrievingOrg, err)
	}

	vdc, err := adminOrg.GetVdcByName(d.Id())
	if err != nil || vdc == (govcd.Vdc{}) {
		log.Printf("[DEBUG] Unable to find vdc. Removing from tfstate")
		d.SetId("")
		return nil
	}

	log.Printf("[TRACE] vdc read completed: %#v", vdc.Vdc)
	return nil
}

// Deletes a vdc, optionally removing all objects in it as well
func resourceVcdVdcDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[TRACE] vdc delete started")

	vcdClient := meta.(*VCDClient)

	if !vcdClient.Client.IsSysAdmin {
		return fmt.Errorf("functionality requires system administrator privileges")
	}

	adminOrg, err := vcdClient.GetAdminOrgFromResource(d)
	if err != nil {
		return fmt.Errorf(errorRetrievingOrg, err)
	}

	vdc, err := adminOrg.GetVdcByName(d.Id())
	if err != nil || vdc == (govcd.Vdc{}) {
		log.Printf("[DEBUG] Unable to find vdc. Removing from tfstate")
		d.SetId("")
		return nil
	}

	err = vdc.DeleteWait(d.Get("delete_force").(bool), d.Get("delete_recursive").(bool))
	if err != nil {
		log.Printf("[DEBUG] Error removing vdc %#v", err)
		return fmt.Errorf("error removing vdc %#v", err)
	}

	log.Printf("[TRACE] vdc delete completed: %#v", vdc.Vdc)
	return nil
}

// helper for tranforming the compute capacity section of the resource input into the VdcConfiguration structure
func capacityWithUsage(d map[string]interface{}) *types.CapacityWithUsage {
	capacity := &types.CapacityWithUsage{
		Units: d["units"].(string),
	}

	if allocated, ok := d["allocated"]; ok {
		capacity.Allocated = int64(allocated.(int))
	}

	if limit, ok := d["limit"]; ok {
		capacity.Limit = int64(limit.(int))
	}

	if reserved, ok := d["reserved"]; ok {
		capacity.Reserved = int64(reserved.(int))
	}

	if used, ok := d["used"]; ok {
		capacity.Used = int64(used.(int))
	}

	if overhead, ok := d["overhead"]; ok {
		capacity.Overhead = int64(overhead.(int))
	}

	return capacity
}

// helper for transforming the resource input into the VdcConfiguration structure
// any cast operations or default values should be done here so that the create method is simple
func getVcdVdcInput(d *schema.ResourceData, vcdClient *VCDClient) (*types.VdcConfiguration, error) {
	computeCapacityList := d.Get("compute_capacity").(*schema.Set).List()
	if len(computeCapacityList) == 0 {
		return &types.VdcConfiguration{}, errors.New("no compute_capacity field")
	}
	computeCapacity := computeCapacityList[0].(map[string]interface{})

	vdcStorageProfilesConfigurations := d.Get("storage_profile").([]interface{})
	if len(vdcStorageProfilesConfigurations) == 0 {
		return &types.VdcConfiguration{}, errors.New("no storage_profile field")
	}

	cpuCapacityList := computeCapacity["cpu"].(*schema.Set).List()
	if len(cpuCapacityList) == 0 {
		return &types.VdcConfiguration{}, errors.New("no cpu field in compute_capacity")
	}
	memoryCapacityList := computeCapacity["memory"].(*schema.Set).List()
	if len(memoryCapacityList) == 0 {
		return &types.VdcConfiguration{}, errors.New("no memory field in compute_capacity")
	}

	providerVdcName := d.Get("provider_vdc_name").(string)
	providerVdcResults, err := govcd.QueryProviderVdcByName(vcdClient.VCDClient, providerVdcName)
	if err != nil {
		return &types.VdcConfiguration{}, err
	}
	if len(providerVdcResults) == 0 {
		return &types.VdcConfiguration{}, fmt.Errorf("no provider VDC found with name %s", providerVdcName)
	}

	params := &types.VdcConfiguration{
		Name:            d.Get("name").(string),
		Xmlns:           "http://www.vmware.com/vcloud/v1.5",
		AllocationModel: d.Get("allocation_model").(string),
		ComputeCapacity: []*types.ComputeCapacity{
			&types.ComputeCapacity{
				CPU:    capacityWithUsage(cpuCapacityList[0].(map[string]interface{})),
				Memory: capacityWithUsage(memoryCapacityList[0].(map[string]interface{})),
			},
		},
		ProviderVdcReference: &types.Reference{
			HREF: providerVdcResults[0].HREF,
		},
	}

	var vdcStorageProfiles []*types.VdcStorageProfile
	for _, storageConfigurationValues := range vdcStorageProfilesConfigurations {
		storageConfiguration := storageConfigurationValues.(map[string]interface{})

		href, err := getStorageProfileHREF(vcdClient, storageConfiguration["name"].(string))
		if err != nil {
			return &types.VdcConfiguration{}, err
		}

		vdcStorageProfile := &types.VdcStorageProfile{
			Units:   "MB", // only this value is supported
			Limit:   int64(storageConfiguration["limit"].(int)),
			Default: storageConfiguration["default"].(bool),
			Enabled: storageConfiguration["enabled"].(bool),
			ProviderVdcStorageProfile: &types.Reference{
				HREF: href,
			},
		}
		vdcStorageProfiles = append(vdcStorageProfiles, vdcStorageProfile)
	}

	params.VdcStorageProfile = vdcStorageProfiles

	if description, ok := d.GetOk("description"); ok {
		params.Description = description.(string)
	}

	if nicQuota, ok := d.GetOk("nic_quota"); ok {
		params.NicQuota = nicQuota.(int)
	}

	if networkQuota, ok := d.GetOk("network_quota"); ok {
		params.NetworkQuota = networkQuota.(int)
	}

	if vmQuota, ok := d.GetOk("vm_quota"); ok {
		params.VmQuota = vmQuota.(int)
	}

	if isEnabled, ok := d.GetOk("enabled"); ok {
		params.IsEnabled = isEnabled.(bool)
	}

	if resourceGuaranteedMemory, ok := d.GetOk("memory_guaranteed"); ok {
		params.ResourceGuaranteedMemory = resourceGuaranteedMemory.(float64)
	}

	if resourceGuaranteedCpu, ok := d.GetOk("cpu_guaranteed"); ok {
		params.ResourceGuaranteedCpu = resourceGuaranteedCpu.(float64)
	}

	if vCpuInMhz, ok := d.GetOk("cpu_frequency"); ok {
		params.VCpuInMhz = int64(vCpuInMhz.(int))
	}

	if enableThinProvision, ok := d.GetOk("enable_thin_provisioning"); ok {
		params.IsThinProvision = enableThinProvision.(bool)
	}

	if networkPoolName, ok := d.GetOk("network_pool_name"); ok {
		networkPoolResults, err := govcd.QueryNetworkPoolByName(vcdClient.VCDClient, networkPoolName.(string))
		if err != nil {
			return &types.VdcConfiguration{}, err
		}

		if len(networkPoolResults) == 0 {
			return &types.VdcConfiguration{}, fmt.Errorf("no network pool found with name %s", networkPoolName)
		}
		params.NetworkPoolReference = &types.Reference{
			HREF: networkPoolResults[0].HREF,
		}
	}

	if usesFastProvisioning, ok := d.GetOk("enable_fast_provisioning"); ok {
		params.UsesFastProvisioning = usesFastProvisioning.(bool)
	}

	if overCommitAllowed, ok := d.GetOk("over_commit_allowed"); ok {
		params.OverCommitAllowed = overCommitAllowed.(bool)
	}

	if vmDiscoveryEnabled, ok := d.GetOk("enable_vm_discovery"); ok {
		params.VmDiscoveryEnabled = vmDiscoveryEnabled.(bool)
	}

	return params, nil
}

func getStorageProfileHREF(vcdClient *VCDClient, name string) (string, error) {
	storageProfileRecords, err := govcd.QueryProviderVdcStorageProfileByName(vcdClient.VCDClient, name)
	if err != nil {
		return "", err
	}
	if len(storageProfileRecords) == 0 {
		return "", fmt.Errorf("no provider VDC storage profile found with name %s", name)
	}

	// additional filtering done cause name like `*` returns more value and have to be manually selected
	for _, profileRecord := range storageProfileRecords {
		if profileRecord.Name == name {
			return profileRecord.HREF, nil
		}
	}
	return "", fmt.Errorf("no provider VDC storage profile found with name %s", name)
}
