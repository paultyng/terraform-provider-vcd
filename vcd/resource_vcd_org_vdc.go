package vcd

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"github.com/vmware/go-vcloud-director/v2/govcd"
	"github.com/vmware/go-vcloud-director/v2/types/v56"
)

func resourceVcdOrgVdc() *schema.Resource {
	capacityWithUsage := schema.Schema{
		Type:     schema.TypeList,
		Required: true,
		MinItems: 1,
		MaxItems: 1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"allocated": {
					Type:        schema.TypeInt,
					Optional:    true,
					Computed:    true,
					Description: "Capacity that is committed to be available. Value in MB or MHz. Used with AllocationPool (Allocation pool) and ReservationPool (Reservation pool).",
				},
				"limit": {
					Type:        schema.TypeInt,
					Optional:    true,
					Computed:    true,
					Description: "Capacity limit relative to the value specified for Allocation. It must not be less than that value. If it is greater than that value, it implies over provisioning. A value of 0 specifies unlimited units. Value in MB or MHz. Used with AllocationVApp (Pay as you go).",
				},
				"reserved": {
					Type:     schema.TypeInt,
					Computed: true,
				},
				"used": {
					Type:     schema.TypeInt,
					Computed: true,
				},
				"overhead": {
					Type:     schema.TypeInt,
					Computed: true,
				},
			},
		},
	}

	return &schema.Resource{
		Create: resourceVcdVdcCreate,
		Delete: resourceVcdVdcDelete,
		Read:   resourceVcdVdcRead,
		Update: resourceVcdVdcUpdate,
		Importer: &schema.ResourceImporter{
			State: resourceVcdOrgVdcImport,
		},
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
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
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
				MinItems: 1,
				MaxItems: 1,
				Type:     schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"cpu":    &capacityWithUsage,
						"memory": &capacityWithUsage,
					},
				},
				Description: "The compute capacity allocated to this VDC.",
				//Set:         hashMapStringForCapacity,
			},
			"nic_quota": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Maximum number of virtual NICs allowed in this VDC. Defaults to 0, which specifies an unlimited number.",
			},
			"network_quota": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Maximum number of network objects that can be deployed in this VDC. Defaults to 0, which means no networks can be deployed.",
			},
			"vm_quota": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The maximum number of VMs that can be created in this VDC. Includes deployed and undeployed VMs in vApps and vApp templates. Defaults to 0, which specifies an unlimited number.",
			},
			"enabled": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "True if this VDC is enabled for use by the organization VDCs. Default is true.",
			},
			"storage_profile": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				ForceNew: true,
				MinItems: 1,
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
							Default:     true,
							Description: "True if this storage profile is enabled for use in the VDC.",
						},
						"limit": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "Maximum number of MB allocated for this storage profile. A value of 0 specifies unlimited MB.",
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
				Type:     schema.TypeFloat,
				Computed: true,
				Optional: true,
				Description: "Percentage of allocated memory resources guaranteed to vApps deployed in this VDC. " +
					"For example, if this value is 0.75, then 75% of allocated resources are guaranteed. " +
					"Required when AllocationModel is AllocationVApp or AllocationPool. When Allocation model is AllocationPool minimum value is 0.2. If the element is empty, vCD sets a value.",
			},
			"cpu_guaranteed": &schema.Schema{
				Type:     schema.TypeFloat,
				Optional: true,
				Computed: true,
				Description: "Percentage of allocated CPU resources guaranteed to vApps deployed in this VDC. " +
					"For example, if this value is 0.75, then 75% of allocated resources are guaranteed. " +
					"Required when AllocationModel is AllocationVApp or AllocationPool. If the element is empty, vCD sets a value",
			},
			"cpu_speed": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				Computed:    true,
				Description: "Specifies the clock frequency, in Megahertz, for any virtual CPU that is allocated to a VM. A VM with 2 vCPUs will consume twice as much of this value. Ignored for ReservationPool. Required when AllocationModel is AllocationVApp or AllocationPool, and may not be less than 256 MHz. Defaults to 1000 MHz if the element is empty or missing.",
			},
			"enable_thin_provisioning": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Boolean to request thin provisioning. Request will be honored only if the underlying datastore supports it. Thin provisioning saves storage space by committing it on demand. This allows over-allocation of storage.",
			},
			"network_pool_name": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The name of a network pool in the Provider VDC. Required if this VDC will contain routed or isolated networks.",
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
				Description: "Request for fast provisioning. Request will be honored only if the underlying datas tore supports it. Fast provisioning can reduce the time it takes to create virtual machines by using vSphere linked clones. If you disable fast provisioning, all provisioning operations will result in full clones.",
			},
			//  Always null in the response to a GET request. On update, set to false to disallow the update if the AllocationModel is AllocationPool or ReservationPool
			//  and the ComputeCapacity you specified is greater than what the backing Provider VDC can supply. Defaults to true if empty or missing.
			"allow_over_commit": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
				Description: "Set to false to disallow creation of the VDC if the AllocationModel is AllocationPool or ReservationPool and the ComputeCapacity you specified is greater than what the backing Provider VDC can supply. Default is true.",
			},
			"enable_vm_discovery": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "True if discovery of vCenter VMs is enabled for resource pools backing this VDC. If left unspecified, the actual behaviour depends on enablement at the organization level and at the system level.",
			},

			"delete_force": &schema.Schema{
				Type:        schema.TypeBool,
				Required:    true,
				Description: "When destroying use delete_force=True to remove a vdc and any objects it contains, regardless of their state.",
			},
			"delete_recursive": &schema.Schema{
				Type:        schema.TypeBool,
				Required:    true,
				Description: "When destroying use delete_recursive=True to remove the vdc and any objects it contains that are in a state that normally allows removal.",
			},
			"metadata": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "Key and value pairs for Org VDC metadata",
				// For now underlying go-vcloud-director repo only supports
				// a value of type String in this map.
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

	orgVdcName := d.Get("name").(string)
	orgVdc, _ := adminOrg.GetVDCByName(orgVdcName, false)
	if orgVdc != nil {
		return fmt.Errorf("org VDC with such name already exists: %s", orgVdcName)
	}

	params, err := getVcdVdcInput(d, vcdClient)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Creating vdc: %#v", params)

	task, err := adminOrg.CreateVdc(params)
	if err != nil {
		log.Printf("[DEBUG] Error creating vdc: %s", err)
		return fmt.Errorf("error creating vdc: %s", err)
	}

	err = task.WaitTaskCompletion()
	if err != nil {
		log.Printf("[DEBUG] Error waiting for vdc to finish: %s", err)
		return fmt.Errorf("error waiting for vdc to finish: %s", err)
	}

	err = adminOrg.Refresh()
	if err != nil {
		log.Printf("[DEBUG] Unable to refresh org.")
		return fmt.Errorf("unable to refresh org. %s", err)
	}

	adminVdc, err := adminOrg.GetAdminVDCByName(d.Get("name").(string), false)
	if err != nil {
		log.Printf("[DEBUG] Unable to find vdc.")
		return fmt.Errorf("unable to find VDC. %s", err)
	}

	d.SetId(adminVdc.AdminVdc.ID)
	log.Printf("[TRACE] vdc created: %#v", task)

	err = createOrUpdateMetadata(d, meta)
	if err != nil {
		return fmt.Errorf("error adding metadata to VDC: %s", err)
	}

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

	adminVdc, err := adminOrg.GetAdminVDCByName(d.Get("name").(string), false)
	if err != nil {
		log.Printf("[DEBUG] Unable to find VDC")
		return fmt.Errorf("unable to find VDC %s", err)
	}

	return setOrgVdcData(d, vcdClient, adminOrg, adminVdc)
}

// setOrgVdcData sets object state from *govcd.AdminVdc
func setOrgVdcData(d *schema.ResourceData, vcdClient *VCDClient, adminOrg *govcd.AdminOrg, adminVdc *govcd.AdminVdc) error {

	d.Set("allocation_model", adminVdc.AdminVdc.AllocationModel)
	d.Set("cpu_guaranteed", *adminVdc.AdminVdc.ResourceGuaranteedCpu)
	d.Set("cpu_speed", adminVdc.AdminVdc.VCpuInMhz)
	d.Set("description", adminVdc.AdminVdc.Description)
	d.Set("enable_fast_provisioning", adminVdc.AdminVdc.UsesFastProvisioning)
	d.Set("enable_thin_provisioning", adminVdc.AdminVdc.IsThinProvision)
	d.Set("enable_vm_discovery", adminVdc.AdminVdc.VmDiscoveryEnabled)
	d.Set("enabled", adminVdc.AdminVdc.IsEnabled)
	d.Set("memory_guaranteed", *adminVdc.AdminVdc.ResourceGuaranteedMemory)
	d.Set("name", adminVdc.AdminVdc.Name)

	networkPool, err := govcd.GetNetworkPoolByHREF(vcdClient.VCDClient, adminVdc.AdminVdc.NetworkPoolReference.HREF)
	if err != nil {
		return fmt.Errorf("error retrieving network pool: %s", err)
	}

	d.Set("network_pool_name", networkPool.Name)
	d.Set("network_quota", adminVdc.AdminVdc.NetworkQuota)
	d.Set("nic_quota", adminVdc.AdminVdc.Vdc.NicQuota)
	d.Set("provider_vdc_name", adminVdc.AdminVdc.ProviderVdcReference.Name)
	d.Set("vm_quota", adminVdc.AdminVdc.Vdc.VMQuota)

	if err := d.Set("compute_capacity", getComputeCapacities(adminVdc.AdminVdc.ComputeCapacity)); err != nil {
		return fmt.Errorf("error setting compute_capacity: %s", err)
	}

	storageProfileStateData, err := getComputeStorageProfiles(vcdClient, adminVdc.AdminVdc.VdcStorageProfiles)
	if err != nil {
		return fmt.Errorf("error preparing storage profile data: %s", err)
	}

	if err := d.Set("storage_profile", storageProfileStateData); err != nil {
		return fmt.Errorf("error setting compute_capacity: %s", err)
	}

	vdc, err := adminOrg.GetVDCByName(d.Get("name").(string), false)
	if err != nil {
		log.Printf("[DEBUG] Unable to find VDC")
		return fmt.Errorf("unable to find VDC %s", err)
	}
	metadata, err := vdc.GetMetadata()
	if err != nil {
		log.Printf("[DEBUG] Unable to get VDC metadata")
		return fmt.Errorf("unable to get VDC metadata %s", err)
	}

	d.Set("metadata", getMetadataStruct(metadata.MetadataEntry))

	log.Printf("[TRACE] vdc read completed: %#v", adminVdc.AdminVdc)
	return nil
}

// getComputeStorageProfiles constructs specific struct to be saved in Terraform state file.
// Expected E.g.
func getComputeStorageProfiles(vcdClient *VCDClient, profile *types.VdcStorageProfiles) ([]map[string]interface{}, error) {
	root := make([]map[string]interface{}, 0)

	for _, vdcStorageProfile := range profile.VdcStorageProfile {
		vdcStorageProfileDetails, err := govcd.GetStorageProfileByHref(vcdClient.VCDClient, vdcStorageProfile.HREF)
		if err != nil {
			return nil, err
		}
		storageProfileData := make(map[string]interface{})
		storageProfileData["limit"] = vdcStorageProfileDetails.Limit
		storageProfileData["default"] = vdcStorageProfileDetails.Default
		storageProfileData["enabled"] = vdcStorageProfileDetails.Enabled
		storageProfileData["name"] = vdcStorageProfileDetails.ProviderVdcStorageProfile.Name
		root = append(root, storageProfileData)
	}

	return root, nil
}

// getComputeCapacities constructs specific struct to be saved in Terraform state file.
// Expected E.g. &[]map[string]interface {}
// {map[string]interface {}{"cpu":(*[]map[string]interface {})
// ({"allocated":8000, "limit":8000, "overhead":0, "reserved":4000, "used":0}),
// "memory":(*[]map[string]interface {})
// ({"allocated":7168, "limit":7168, "overhead":0, "reserved":3584, "used":0})},
func getComputeCapacities(capacities []*types.ComputeCapacity) *[]map[string]interface{} {
	rootInternal := map[string]interface{}{}
	var root []map[string]interface{}

	for _, capacity := range capacities {
		cpuValueMap := map[string]interface{}{}
		cpuValueMap["limit"] = int(capacity.CPU.Limit)
		cpuValueMap["allocated"] = int(capacity.CPU.Allocated)
		cpuValueMap["reserved"] = int(capacity.CPU.Reserved)
		cpuValueMap["used"] = int(capacity.CPU.Used)
		cpuValueMap["overhead"] = int(capacity.CPU.Overhead)

		memoryValueMap := map[string]interface{}{}
		memoryValueMap["limit"] = int(capacity.Memory.Limit)
		memoryValueMap["allocated"] = int(capacity.Memory.Allocated)
		memoryValueMap["reserved"] = int(capacity.Memory.Reserved)
		memoryValueMap["used"] = int(capacity.Memory.Used)
		memoryValueMap["overhead"] = int(capacity.Memory.Overhead)

		var memoryCapacityArray []map[string]interface{}
		memoryCapacityArray = append(memoryCapacityArray, memoryValueMap)
		var cpuCapacityArray []map[string]interface{}
		cpuCapacityArray = append(cpuCapacityArray, cpuValueMap)

		rootInternal["cpu"] = &cpuCapacityArray
		rootInternal["memory"] = &memoryCapacityArray

		root = append(root, rootInternal)
	}

	return &root
}

// Converts to terraform understandable structure
func getMetadataStruct(metadata []*types.MetadataEntry) StringMap {
	metadataMap := make(StringMap, len(metadata))
	for _, metadataEntry := range metadata {
		metadataMap[metadataEntry.Key] = metadataEntry.TypedValue.Value
	}
	return metadataMap
}

//resourceVcdVdcUpdate function updates resource with found configurations changes
func resourceVcdVdcUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[TRACE] vdc update initiated")

	vcdClient := meta.(*VCDClient)

	adminOrg, err := vcdClient.GetAdminOrgFromResource(d)
	if err != nil {
		return fmt.Errorf(errorRetrievingOrg, err)
	}

	vdcName := d.Get("name").(string)
	if d.HasChange("name") {
		oldValue, _ := d.GetChange("name")
		vdcName = oldValue.(string)
	}

	adminVdc, err := adminOrg.GetAdminVDCByName(vdcName, false)
	if err != nil {
		log.Printf("[DEBUG] Unable to find VDC.")
		return fmt.Errorf("unable to find VDC %s", err)
	}

	changedAdminVdc, err := getUpdatedVdcInput(d, vcdClient, adminVdc)
	if err != nil {
		log.Printf("[DEBUG] Error updating VDC %s", err)
		return fmt.Errorf("error updating VDC %s", err)
	}

	_, err = changedAdminVdc.Update()
	if err != nil {
		log.Printf("[DEBUG] Error updating VDC %s", err)
		return fmt.Errorf("error updating VDC %s", err)
	}

	err = createOrUpdateMetadata(d, meta)
	if err != nil {
		return fmt.Errorf("error updating VDC metadata: %s", err)
	}

	log.Printf("[TRACE] vdc update completed: %s", adminVdc.AdminVdc.Name)
	return resourceVcdVdcRead(d, meta)
}

// Deletes a vdc, optionally removing all objects in it as well
func resourceVcdVdcDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[TRACE] vdc delete started")

	vcdClient := meta.(*VCDClient)

	vdcName := d.Get("name").(string)
	if !vcdClient.Client.IsSysAdmin {
		return fmt.Errorf("functionality requires system administrator privileges")
	}

	adminOrg, err := vcdClient.GetAdminOrgFromResource(d)
	if err != nil {
		return fmt.Errorf(errorRetrievingOrg, err)
	}

	vdc, err := adminOrg.GetVDCByName(vdcName, false)
	if err != nil {
		log.Printf("[DEBUG] Unable to find vdc. Removing from tfstate")
		d.SetId("")
		return nil
	}

	err = vdc.DeleteWait(d.Get("delete_force").(bool), d.Get("delete_recursive").(bool))
	if err != nil {
		log.Printf("[DEBUG] Error removing vdc %s", err)
		return fmt.Errorf("error removing vdc %s", err)
	}

	vdc, err = adminOrg.GetVDCByName(vdcName, true)
	if err == nil {
		return fmt.Errorf("vdc %s still found after deletion", vdcName)
	}
	log.Printf("[TRACE] vdc delete completed: %s", vdcName)
	return nil
}

func createOrUpdateMetadata(d *schema.ResourceData, meta interface{}) error {

	log.Printf("[TRACE] adding/updating metadata to VDC")

	vcdClient := meta.(*VCDClient)

	adminOrg, err := vcdClient.GetAdminOrgFromResource(d)
	if err != nil {
		return fmt.Errorf(errorRetrievingOrg, err)
	}

	vdc, err := adminOrg.GetVDCByName(d.Get("name").(string), false)
	if err != nil {
		return fmt.Errorf(errorRetrievingVdcFromOrg, d.Get("org").(string), d.Get("name").(string), err)
	}

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
			_, err := vdc.DeleteMetadata(k)
			if err != nil {
				return fmt.Errorf("error deleting metadata: %s", err)
			}
		}
		// Add new metadata
		for k, v := range newMetadata {
			_, err := vdc.AddMetadata(k, v.(string))
			if err != nil {
				return fmt.Errorf("error adding metadata: %s", err)
			}
		}
	}
	return nil
}

// helper for transforming the compute capacity section of the resource input into the VdcConfiguration structure
func capacityWithUsage(d map[string]interface{}, units string) *types.CapacityWithUsage {
	capacity := &types.CapacityWithUsage{
		Units: units,
	}

	if allocated, ok := d["allocated"]; ok {
		capacity.Allocated = int64(allocated.(int))
	}

	if limit, ok := d["limit"]; ok {
		capacity.Limit = int64(limit.(int))
	}

	return capacity
}

// helper for transforming the resource input into the AdminVdc structure
func getUpdatedVdcInput(d *schema.ResourceData, vcdClient *VCDClient, vdc *govcd.AdminVdc) (*govcd.AdminVdc, error) {

	if d.HasChange("compute_capacity") {
		computeCapacityList := d.Get("compute_capacity").(*schema.Set).List()
		if len(computeCapacityList) == 0 {
			return &govcd.AdminVdc{}, errors.New("no compute_capacity field")
		}
		computeCapacity := computeCapacityList[0].(map[string]interface{})

		cpuCapacityList := computeCapacity["cpu"].(*schema.Set).List()
		if len(cpuCapacityList) == 0 {
			return &govcd.AdminVdc{}, errors.New("no cpu field in compute_capacity")
		}
		memoryCapacityList := computeCapacity["memory"].(*schema.Set).List()
		if len(memoryCapacityList) == 0 {
			return &govcd.AdminVdc{}, errors.New("no memory field in compute_capacity")
		}
		vdc.AdminVdc.ComputeCapacity[0].Memory = capacityWithUsage(memoryCapacityList[0].(map[string]interface{}), "MB")
		vdc.AdminVdc.ComputeCapacity[0].CPU = capacityWithUsage(cpuCapacityList[0].(map[string]interface{}), "MHz")
	}

	if d.HasChange("allocation_model") {
		vdc.AdminVdc.AllocationModel = d.Get("allocation_model").(string)
	}

	if d.HasChange("name") {
		vdc.AdminVdc.Name = d.Get("name").(string)
	}

	if d.HasChange("description") {
		vdc.AdminVdc.Description = d.Get("description").(string)
	}

	if d.HasChange("nic_quota") {
		vdc.AdminVdc.NicQuota = d.Get("nic_quota").(int)
	}

	if d.HasChange("network_quota") {
		vdc.AdminVdc.NetworkQuota = d.Get("network_quota").(int)
	}

	if d.HasChange("vm_quota") {
		vdc.AdminVdc.VMQuota = d.Get("vm_quota").(int)
	}

	if d.HasChange("enabled") {
		vdc.AdminVdc.IsEnabled = d.Get("enabled").(bool)
	}

	if d.HasChange("memory_guaranteed") {
		// only set 0 if value configured
		if value, ok := d.GetOkExists("memory_guaranteed"); ok {
			floatValue := value.(float64)
			vdc.AdminVdc.ResourceGuaranteedMemory = &floatValue
		}
	}

	if d.HasChange("cpu_guaranteed") {
		// only set 0 if value configured
		if value, ok := d.GetOkExists("cpu_guaranteed"); ok {
			floatValue := value.(float64)
			vdc.AdminVdc.ResourceGuaranteedCpu = &floatValue
		}
	}

	if d.HasChange("cpu_speed") {
		vdc.AdminVdc.VCpuInMhz = int64(d.Get("cpu_speed").(int))
	}

	if d.HasChange("enable_thin_provisioning") {
		vdc.AdminVdc.IsThinProvision = d.Get("enable_thin_provisioning").(bool)
	}

	if d.HasChange("network_pool_name") {
		networkPoolResults, err := govcd.QueryNetworkPoolByName(vcdClient.VCDClient, d.Get("network_pool_name").(string))
		if err != nil {
			return &govcd.AdminVdc{}, err
		}

		if len(networkPoolResults) == 0 {
			return &govcd.AdminVdc{}, fmt.Errorf("no network pool found with name %s", d.Get("network_pool_name"))
		}
		vdc.AdminVdc.NetworkPoolReference = &types.Reference{
			HREF: networkPoolResults[0].HREF,
		}
	}

	if d.HasChange("enable_fast_provisioning") {
		vdc.AdminVdc.UsesFastProvisioning = d.Get("enable_fast_provisioning").(bool)
	}

	if d.HasChange("allow_over_commit") {
		vdc.AdminVdc.OverCommitAllowed = d.Get("allow_over_commit").(bool)
	}

	if d.HasChange("enable_vm_discovery") {
		vdc.AdminVdc.VmDiscoveryEnabled = d.Get("enable_vm_discovery").(bool)
	}

	//cleanup
	vdc.AdminVdc.Tasks = nil

	return vdc, nil
}

// helper for transforming the resource input into the VdcConfiguration structure
// any cast operations or default values should be done here so that the create method is simple
func getVcdVdcInput(d *schema.ResourceData, vcdClient *VCDClient) (*types.VdcConfiguration, error) {
	computeCapacityList := d.Get("compute_capacity").([]interface{})
	if len(computeCapacityList) == 0 {
		return &types.VdcConfiguration{}, errors.New("no compute_capacity field")
	}
	computeCapacity := computeCapacityList[0].(map[string]interface{})

	vdcStorageProfilesConfigurations := d.Get("storage_profile").([]interface{})
	if len(vdcStorageProfilesConfigurations) == 0 {
		return &types.VdcConfiguration{}, errors.New("no storage_profile field")
	}

	cpuCapacityList := computeCapacity["cpu"].([]interface{})
	if len(cpuCapacityList) == 0 {
		return &types.VdcConfiguration{}, errors.New("no cpu field in compute_capacity")
	}
	memoryCapacityList := computeCapacity["memory"].([]interface{})
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
				CPU:    capacityWithUsage(cpuCapacityList[0].(map[string]interface{}), "MHz"),
				Memory: capacityWithUsage(memoryCapacityList[0].(map[string]interface{}), "MB"),
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

	// only set 0 if value configured
	if resourceGuaranteedMemory, ok := d.GetOkExists("memory_guaranteed"); ok {
		value := resourceGuaranteedMemory.(float64)
		params.ResourceGuaranteedMemory = &value
	}

	if resourceGuaranteedCpu, ok := d.GetOkExists("cpu_guaranteed"); ok {
		value := resourceGuaranteedCpu.(float64)
		params.ResourceGuaranteedCpu = &value
	}

	if vCpuInMhz, ok := d.GetOk("cpu_speed"); ok {
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

	if overCommitAllowed, ok := d.GetOk("allow_over_commit"); ok {
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

// resourceVcdOrgVdcImport is responsible for importing the resource.
// The following steps happen as part of import
// 1. The user supplies `terraform import _resource_name_ _the_id_string_` command
// 2. `_the_id_string_` contains a dot formatted path to resource as in the example below
// 3. The functions splits the dot-formatted path and tries to lookup the object
// 4. If the lookup succeeds it set's the ID field for `_resource_name_` resource in state file
// (the resource must be already defined in .tf config otherwise `terraform import` will complain)
// 5. `terraform refresh` is being implicitly launched. The Read method looks up all other fields
// based on the known ID of object.
//
// Example resource name (_resource_name_): vcd_org_vdc.my_existing_vdc
// Example import path (_the_id_string_): org.my_existing_vdc
func resourceVcdOrgVdcImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	resourceURI := strings.Split(d.Id(), ".")
	if len(resourceURI) != 2 {
		return nil, fmt.Errorf("resource name must be specified as org.vdc.my_existing_vdc")
	}
	orgName, vdcName := resourceURI[0], resourceURI[1]

	vcdClient := meta.(*VCDClient)

	adminOrg, err := vcdClient.GetAdminOrg(orgName)
	if err != nil {
		return nil, fmt.Errorf(errorRetrievingOrg, err)
	}

	adminVdc, err := adminOrg.GetAdminVDCByName(vdcName, false)
	if err != nil {
		log.Printf("[DEBUG] Unable to find VDC")
		return nil, fmt.Errorf("unable to find VDC %s", err)
	}

	d.Set("org", orgName)
	d.Set("name", vdcName)

	d.SetId(adminVdc.AdminVdc.ID)

	return []*schema.ResourceData{d}, nil
}
