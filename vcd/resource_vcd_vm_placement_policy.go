package vcd

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/vmware/go-vcloud-director/v2/util"
	"log"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/vmware/go-vcloud-director/v2/govcd"
	"github.com/vmware/go-vcloud-director/v2/types/v56"
)

func resourceVcdVmPlacementPolicy() *schema.Resource {

	return &schema.Resource{
		CreateContext: resourceVmPlacementPolicyCreate,
		ReadContext:   resourceVmPlacementPolicyRead,
		UpdateContext: resourceVmPlacementPolicyUpdate,
		DeleteContext: resourceVmPlacementPolicyDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceVmPlacementPolicyImport,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the VM Placement Policy",
			},
			"provider_vdc_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "ID of the Provider VDC to which the VM Placement Policy belongs",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Description of the VM Placement Policy",
			},
			"vm_group_ids": {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "IDs of the collection of VMs with similar host requirements",
			},
			"logical_vm_group_ids": {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "IDs of one or more Logical VM Groups to define this VM Placement policy. There is an AND relationship among all the entries set in this attribute",
			},
		},
	}
}

func resourceVmPlacementPolicyCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	policyName := d.Get("name").(string)
	providerVdcId := d.Get("provider_vdc_id").(string)
	log.Printf("[TRACE] VM Placement Policy creation initiated: %s in pVDC %s", policyName, providerVdcId)

	vcdClient := meta.(*VCDClient)

	if !vcdClient.Client.IsSysAdmin {
		return diag.Errorf("functionality requires System administrator privileges")
	}
	computePolicy := &types.VdcComputePolicy{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
	}

	vmGroups, err := getVmGroups(d, vcdClient)
	if err != nil {
		return diag.FromErr(err)
	}
	computePolicy.NamedVMGroups = vmGroups

	logicalVmGroups, err := getLogicalVmGroups(d, vcdClient)
	if err != nil {
		return diag.FromErr(err)
	}
	computePolicy.LogicalVMGroupReferences = logicalVmGroups

	log.Printf("[DEBUG] Creating VM Placement policy: %#v", computePolicy)

	createdVmSizingPolicy, err := vcdClient.Client.CreateVdcComputePolicy(computePolicy)
	if err != nil {
		log.Printf("[DEBUG] Error creating VM Placement policy: %s", err)
		return diag.Errorf("error creating VM Placement policy: %s", err)
	}

	d.SetId(createdVmSizingPolicy.VdcComputePolicy.ID)
	log.Printf("[TRACE] VM Placement policy created: %#v", createdVmSizingPolicy.VdcComputePolicy)

	return resourceVmPlacementPolicyRead(ctx, d, meta)
}

func getVmGroups(d *schema.ResourceData, vcdClient *VCDClient) ([]types.OpenApiReferences, error) {
	vmGroupIdsSet := d.Get("vm_group_ids").(*schema.Set)
	if vmGroupIdsSet != nil {
		vmGroupIdsList := vmGroupIdsSet.List()
		var vmGroupReferences types.OpenApiReferences
		for _, vmGroupId := range vmGroupIdsList {
			vmGroup, err := vcdClient.GetVmGroupByNamedVmGroupId(vmGroupId.(string))
			if err != nil {
				return nil, fmt.Errorf("error retrieving the associated name of VM Group %s: %s", vmGroupId, err)
			}
			vmGroupReferences = append(vmGroupReferences, types.OpenApiReference{
				ID:   vmGroup.VmGroup.NamedVmGroupId,
				Name: vmGroup.VmGroup.Name,
			})
		}
		var vmGroupReferencesSlice []types.OpenApiReferences
		return append(vmGroupReferencesSlice, vmGroupReferences), nil
	}
	return []types.OpenApiReferences{}, nil
}

func getLogicalVmGroups(d *schema.ResourceData, vcdClient *VCDClient) (types.OpenApiReferences, error) {
	vmGroupIdsSet := d.Get("logical_vm_group_ids").(*schema.Set)
	if vmGroupIdsSet != nil {
		vmGroupIdsList := vmGroupIdsSet.List()
		var logicalVmGroupReferences types.OpenApiReferences
		for _, vmGroupId := range vmGroupIdsList {
			logicalVmGroup, err := vcdClient.GetLogicalVmGroupById(vmGroupId.(string))
			if err != nil {
				return nil, fmt.Errorf("error retrieving the associated name of Logical VM Group %s: %s", vmGroupId, err)
			}
			logicalVmGroupReferences = append(logicalVmGroupReferences, types.OpenApiReference{
				ID:   logicalVmGroup.LogicalVmGroup.ID,
				Name: logicalVmGroup.LogicalVmGroup.Name,
			})
		}
		return logicalVmGroupReferences, nil
	}
	return types.OpenApiReferences{}, nil
}

// resourceVmPlacementPolicyRead reads a resource VM Placement Policy
func resourceVmPlacementPolicyRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return genericVcdVmPlacementPolicyRead(ctx, d, meta)
}

func genericVcdVmPlacementPolicyRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	policyName := d.Get("name").(string)
	pVdcId := d.Get("provider_vdc_id").(string)
	log.Printf("[TRACE] VM Placement Policy read initiated: %s in pVDC with ID %s", policyName, pVdcId)

	vcdClient := meta.(*VCDClient)

	// The method variable stores the information about how we found the rule, for logging purposes
	method := "id"

	var policy *govcd.VdcComputePolicy
	var err error
	if d.Id() != "" {
		policy, err = vcdClient.Client.GetVdcComputePolicyById(d.Id())
		if err != nil {
			log.Printf("[DEBUG] Unable to find VM Placement Policy %s. Removing from tfstate.", policyName)
			d.SetId("")
			return diag.Errorf("unable to find VM Placement Policy %s, err: %s. Removing from tfstate", policyName, err)
		}
	}

	// The secondary method of retrieval is from name
	if d.Id() == "" {
		if policyName == "" {
			return diag.Errorf("both Placement Policy name and ID are empty")
		}
		if pVdcId == "" {
			return diag.Errorf("both Provider VDC ID and Placement Policy ID are empty")
		}

		method = "name"
		queryParams := url.Values{}
		queryParams.Add("filter", fmt.Sprintf("name==%s;pvdcId==%s", policyName, pVdcId))
		filteredPoliciesByName, err := vcdClient.Client.GetAllVdcComputePolicies(queryParams)
		if err != nil {
			log.Printf("[DEBUG] Unable to find VM Placement policy %s of Provider VDC %s. Removing from tfstate.", policyName, pVdcId)
			d.SetId("")
			return diag.Errorf("unable to find VM Placement policy %s of Provider VDC %s, err: %s. Removing from tfstate", policyName, pVdcId, err)
		}
		if len(filteredPoliciesByName) != 1 {
			log.Printf("[DEBUG] Unable to find VM Placement policy %s of Provider VDC %s. Found Policies by name: %d. Removing from tfstate.", policyName, pVdcId, len(filteredPoliciesByName))
			d.SetId("")
			return diag.Errorf("[DEBUG] Unable to find VM Placement policy %s of Provider VDC %s, err: %s. Found Policies by name: %d. Removing from tfstate", policyName, pVdcId, govcd.ErrorEntityNotFound, len(filteredPoliciesByName))
		}
		policy = filteredPoliciesByName[0]
		d.SetId(policy.VdcComputePolicy.ID)
	}

	// Fix coverity warning
	if policy == nil {
		return diag.Errorf("[datasourceVcdVmPlacementPolicyRead] error defining VM Placement Policy")
	}
	util.Logger.Printf("[TRACE] [get VM Placement Policy] Retrieved by %s\n", method)
	return setVmPlacementPolicy(ctx, d, *policy.VdcComputePolicy)
}

// resourceVmPlacementPolicyUpdate function updates resource with found configurations changes
func resourceVmPlacementPolicyUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	policyName := d.Get("name").(string)
	log.Printf("[TRACE] VM sizing policy update initiated: %s", policyName)

	vcdClient := meta.(*VCDClient)

	policy, err := vcdClient.Client.GetVdcComputePolicyById(d.Id())
	if err != nil {
		log.Printf("[DEBUG] Unable to find VM Placement Policy %s", policyName)
		return diag.Errorf("unable to find VM Placement Policy %s, error:  %s", policyName, err)
	}

	if d.HasChange("name") {
		policy.VdcComputePolicy.Name = d.Get("name").(string)
	}

	if d.HasChange("description") {
		policy.VdcComputePolicy.Description = d.Get("description").(string)
	}

	if d.HasChange("vm_group_ids") {
		vmGroups, err := getVmGroups(d, vcdClient)
		if err != nil {
			return diag.FromErr(err)
		}
		policy.VdcComputePolicy.NamedVMGroups = vmGroups
	}
	if d.HasChange("logical_vm_group_ids") {
		logicalVmGroups, err := getLogicalVmGroups(d, vcdClient)
		if err != nil {
			return diag.FromErr(err)
		}
		policy.VdcComputePolicy.LogicalVMGroupReferences = logicalVmGroups
	}

	_, err = policy.Update()
	if err != nil {
		log.Printf("[DEBUG] Error updating VM Placement policy %s with error %s", policyName, err)
		return diag.Errorf("error updating VM Placement policy %s, err: %s", policyName, err)
	}

	log.Printf("[TRACE] VM Placement policy update completed: %s", policyName)
	return resourceVmPlacementPolicyRead(ctx, d, meta)
}

// Deletes a VM Placement Policy
func resourceVmPlacementPolicyDelete(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	policyName := d.Get("name").(string)
	log.Printf("[TRACE] VM Placement Policy delete started: %s", policyName)

	vcdClient := meta.(*VCDClient)

	if !vcdClient.Client.IsSysAdmin {
		return diag.Errorf("functionality requires System administrator privileges")
	}

	policy, err := vcdClient.Client.GetVdcComputePolicyById(d.Id())
	if err != nil {
		log.Printf("[DEBUG] Unable to find VM Placement Policy %s. Removing from tfstate", policyName)
		d.SetId("")
		return nil
	}

	err = policy.Delete()
	if err != nil {
		log.Printf("[DEBUG] Error deleting VM Placement Policy %s, err: %s", policyName, err)
		return diag.Errorf("error deleting VM Placement Policy %s, err: %s", policyName, err)
	}

	log.Printf("[TRACE] VM Placement Policy delete completed: %s", policyName)
	return nil
}

var errHelpVmPlacementPolicyImport = fmt.Errorf(`resource id must be specified in one of these formats:
'vm-placement-policy-name', 'vm-placement-policy-id' or 'list@' to get a list of VM placement policies with their IDs`)

// resourceVmPlacementPolicyImport is responsible for importing the resource.
// The following steps happen as part of import
// 1. The user supplies `terraform import _resource_name_ _the_id_string_` command
// 2. `_the_id_string_` contains a dot formatted path to resource as in the example below
// 3. The functions splits the dot-formatted path and tries to lookup the object
// 4. If the lookup succeeds it set's the ID field for `_resource_name_` resource in state file
// (the resource must be already defined in .tf config otherwise `terraform import` will complain)
// 5. `terraform refresh` is being implicitly launched. The Read method looks up all other fields
// based on the known ID of object.
//
// Example resource name (_resource_name_): vcd_vm_placement_policy.my_existing_policy_name
// Example import path (_the_id_string_): my_existing_vm_placement_policy_id
// Example list path (_the_id_string_): list@
// Note: the separator can be changed using Provider.import_separator or variable VCD_IMPORT_SEPARATOR
func resourceVmPlacementPolicyImport(_ context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	resourceURI := strings.Split(d.Id(), ImportSeparator)

	log.Printf("[DEBUG] importing VM Placement Policy resource with provided id %s", d.Id())

	if len(resourceURI) != 1 {
		return nil, errHelpVmPlacementPolicyImport
	}
	if strings.Contains(d.Id(), "list@") {

		return listComputePoliciesForImport(meta, "vcd_vm_placement_policy","placement")
	} else {
		policyId := resourceURI[0]
		return getComputePolicy(d, meta, policyId, "placement")
	}
}