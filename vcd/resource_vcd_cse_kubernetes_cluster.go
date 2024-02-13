package vcd

import (
	"context"
	_ "embed"
	"fmt"
	"github.com/hashicorp/go-cty/cty"
	semver "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/vmware/go-vcloud-director/v2/govcd"
	"time"
)

func resourceVcdCseKubernetesCluster() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceVcdCseKubernetesClusterCreate,
		ReadContext:   resourceVcdCseKubernetesRead,
		UpdateContext: resourceVcdCseKubernetesUpdate,
		DeleteContext: resourceVcdCseKubernetesDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceVcdCseKubernetesImport,
		},
		Schema: map[string]*schema.Schema{
			"cse_version": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{"4.1.0", "4.1.1", "4.2.0"}, false),
				Description:  "The CSE version to use",
				DiffSuppressFunc: func(k, oldValue, newValue string, d *schema.ResourceData) bool {
					// This custom diff function allows to correctly compare versions
					oldVersion, err := semver.NewVersion(oldValue)
					if err != nil {
						return false
					}
					newVersion, err := semver.NewVersion(newValue)
					if err != nil {
						return false
					}
					return oldVersion.Equal(newVersion)
				},
				DiffSuppressOnRefresh: true,
			},
			"runtime": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "tkg",
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{"tkg"}, false), // May add others in future releases of CSE
				Description:  "The Kubernetes runtime for the cluster. Only 'tkg' (Tanzu Kubernetes Grid) is supported",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of the Kubernetes cluster",
				ValidateDiagFunc: matchRegex(`^[a-z](?:[a-z0-9-]{0,29}[a-z0-9])?$`, "name must contain only lowercase alphanumeric characters or '-',"+
					"start with an alphabetic character, end with an alphanumeric, and contain at most 31 characters"),
			},
			"ova_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of the vApp Template that corresponds to a Kubernetes template OVA",
			},
			"org": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Description: "The name of organization that will own this Kubernetes cluster, optional if defined at provider " +
					"level. Useful when connected as sysadmin working across different organizations",
			},
			"vdc_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of the VDC that hosts the Kubernetes cluster",
			},
			"network_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of the network that the Kubernetes cluster will use",
			},
			"owner": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "The user that creates the cluster and owns the API token specified in 'api_token'. It must have the 'Kubernetes Cluster Author' role. If not specified, it assumes it's the user from the provider configuration",
			},
			"api_token_file": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "A file generated by 'vcd_api_token' resource, that stores the API token used to create and manage the cluster, owned by the user specified in 'owner'. Be careful about this file, as it contains sensitive information",
			},
			"ssh_public_key": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "The SSH public key used to login into the cluster nodes",
			},
			"control_plane": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Required:    true,
				Description: "Defines the control plane for the cluster",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"machine_count": {
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     3, // As suggested in UI
							Description: "The number of nodes that the control plane has. Must be an odd number and higher than 0",
							ValidateDiagFunc: func(v interface{}, path cty.Path) diag.Diagnostics {
								value, ok := v.(int)
								if !ok {
									return diag.Errorf("could not parse int value '%v' for control plane nodes", v)
								}
								if value < 1 || value%2 == 0 {
									return diag.Errorf("number of control plane nodes must be odd and higher than 0, but it was '%d'", value)
								}
								return nil
							},
						},
						"disk_size_gi": {
							Type:             schema.TypeInt,
							Optional:         true,
							Default:          20, // As suggested in UI
							ForceNew:         true,
							ValidateDiagFunc: minimumValue(20, "disk size in Gibibytes (Gi) must be at least 20"),
							Description:      "Disk size, in Gibibytes (Gi), for the control plane nodes. Must be at least 20",
						},
						"sizing_policy_id": {
							Type:        schema.TypeString,
							Optional:    true,
							ForceNew:    true,
							Description: "VM Sizing policy for the control plane nodes",
						},
						"placement_policy_id": {
							Type:        schema.TypeString,
							Optional:    true,
							ForceNew:    true,
							Description: "VM Placement policy for the control plane nodes",
						},
						"storage_profile_id": {
							Type:        schema.TypeString,
							Optional:    true,
							ForceNew:    true,
							Description: "Storage profile for the control plane nodes",
						},
						"ip": {
							Type:         schema.TypeString,
							Optional:     true,
							Computed:     true,
							ForceNew:     true,
							Description:  "IP for the control plane. It will be automatically assigned during cluster creation if left empty",
							ValidateFunc: checkEmptyOrSingleIP(),
						},
					},
				},
			},
			"worker_pool": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "Defines a node pool for the cluster",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							ForceNew:    true,
							Description: "The name of this worker pool",
							ValidateDiagFunc: matchRegex(`^[a-z](?:[a-z0-9-]{0,29}[a-z0-9])?$`, "name must contain only lowercase alphanumeric characters or '-',"+
								"start with an alphabetic character, end with an alphanumeric, and contain at most 31 characters"),
						},
						"machine_count": {
							Type:             schema.TypeInt,
							Optional:         true,
							Default:          1, // As suggested in UI
							Description:      "The number of nodes that this worker pool has. Must be higher than 0",
							ValidateDiagFunc: minimumValue(0, "number of nodes must be higher than or equal to 0"),
						},
						"disk_size_gi": {
							Type:             schema.TypeInt,
							Optional:         true,
							Default:          20, // As suggested in UI
							ForceNew:         true,
							Description:      "Disk size, in Gibibytes (Gi), for the control plane nodes",
							ValidateDiagFunc: minimumValue(20, "disk size in Gibibytes (Gi) must be at least 20"),
						},
						"sizing_policy_id": {
							Type:        schema.TypeString,
							Optional:    true,
							ForceNew:    true,
							Description: "VM Sizing policy for the control plane nodes",
						},
						"placement_policy_id": {
							Type:        schema.TypeString,
							Optional:    true,
							ForceNew:    true,
							Description: "VM Placement policy for the control plane nodes",
						},
						"vgpu_policy_id": {
							Type:        schema.TypeString,
							Optional:    true,
							ForceNew:    true,
							Description: "vGPU policy for the control plane nodes",
						},
						"storage_profile_id": {
							Type:        schema.TypeString,
							Optional:    true,
							ForceNew:    true,
							Description: "Storage profile for the control plane nodes",
						},
					},
				},
			},
			"default_storage_class": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "Defines the default storage class for the cluster",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"storage_profile_id": {
							Required:    true,
							ForceNew:    true,
							Type:        schema.TypeString,
							Description: "ID of the storage profile to use for the storage class",
						},
						"name": {
							Required:    true,
							ForceNew:    true,
							Type:        schema.TypeString,
							Description: "Name to give to this storage class",
							ValidateDiagFunc: matchRegex(`^[a-z](?:[a-z0-9-]{0,29}[a-z0-9])?$`, "name must contain only lowercase alphanumeric characters or '-',"+
								"start with an alphabetic character, end with an alphanumeric, and contain at most 31 characters"),
						},
						"reclaim_policy": {
							Required:     true,
							ForceNew:     true,
							Type:         schema.TypeString,
							ValidateFunc: validation.StringInSlice([]string{"delete", "retain"}, false),
							Description:  "'delete' deletes the volume when the PersistentVolumeClaim is deleted. 'retain' does not, and the volume can be manually reclaimed",
						},
						"filesystem": {
							Required:     true,
							ForceNew:     true,
							Type:         schema.TypeString,
							ValidateFunc: validation.StringInSlice([]string{"ext4", "xfs"}, false),
							Description:  "Filesystem of the storage class, can be either 'ext4' or 'xfs'",
						},
					},
				},
			},
			"pods_cidr": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "100.96.0.0/11", // As suggested in UI
				Description: "CIDR that the Kubernetes pods will use",
			},
			"services_cidr": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "100.64.0.0/13", // As suggested in UI
				Description: "CIDR that the Kubernetes services will use",
			},
			"virtual_ip_subnet": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Virtual IP subnet for the cluster",
			},
			"auto_repair_on_errors": {
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true, // CSE Server turns this off when the cluster is created
				Description: "If errors occur before the Kubernetes cluster becomes available, and this argument is 'true', CSE Server will automatically attempt to repair the cluster",
			},
			"node_health_check": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "After the Kubernetes cluster becomes available, nodes that become unhealthy will be remediated according to unhealthy node conditions and remediation rules",
			},
			"operations_timeout_minutes": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  60,
				Description: "The time, in minutes, to wait for the cluster operations to be successfully completed. For example, during cluster creation, it should be in `provisioned`" +
					"state before the timeout is reached, otherwise the operation will return an error. For cluster deletion, this timeout" +
					"specifies the time to wait until the cluster is completely deleted. Setting this argument to `0` means to wait indefinitely",
				ValidateDiagFunc: minimumValue(0, "timeout must be at least 0 (no timeout)"),
			},
			"kubernetes_version": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The version of Kubernetes installed in this cluster",
			},
			"tkg_product_version": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The version of TKG installed in this cluster",
			},
			"capvcd_version": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The version of CAPVCD used by this cluster",
			},
			"cluster_resource_set_bindings": {
				Type:        schema.TypeSet,
				Computed:    true,
				Description: "The cluster resource set bindings of this cluster",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"cpi_version": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The version of the Cloud Provider Interface used by this cluster",
			},
			"csi_version": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The version of the Container Storage Interface used by this cluster",
			},
			"state": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The state of the cluster, can be 'provisioning', 'provisioned', 'deleting' or 'error'. Useful to check whether the Kubernetes cluster is in a stable status",
			},
			"kubeconfig": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The contents of the kubeconfig of the Kubernetes cluster, only available when 'state=provisioned'",
			},
		},
	}
}

func resourceVcdCseKubernetesClusterCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	cseVersion, err := semver.NewSemver(d.Get("cse_version").(string))
	if err != nil {
		return diag.Errorf("the introduced 'cse_version=%s' is not valid: %s", d.Get("cse_version"), err)
	}

	vcdClient := meta.(*VCDClient)
	org, err := vcdClient.GetOrgFromResource(d)
	if err != nil {
		return diag.Errorf("could not create a Kubernetes cluster in the target Organization: %s", err)
	}

	apiTokenFile := d.Get("api_token_file").(string)
	apiToken, err := govcd.GetTokenFromFile(apiTokenFile)
	if err != nil {
		return diag.Errorf("could not read the API token from the file '%s': %s", apiTokenFile, err)
	}
	owner := d.Get("owner").(string)
	if owner == "" {
		session, err := vcdClient.Client.GetSessionInfo()
		if err != nil {
			return diag.Errorf("could not get an Owner for the Kubernetes cluster. 'owner' is not set and cannot get one from the Provider configuration: %s", err)
		}
		owner = session.User.Name
		if owner == "" {
			return diag.Errorf("could not get an Owner for the Kubernetes cluster. 'owner' is not set and cannot get one from the Provider configuration")
		}
	}

	creationData := govcd.CseClusterSettings{
		CseVersion:              *cseVersion,
		Name:                    d.Get("name").(string),
		OrganizationId:          org.Org.ID,
		VdcId:                   d.Get("vdc_id").(string),
		NetworkId:               d.Get("network_id").(string),
		KubernetesTemplateOvaId: d.Get("ova_id").(string),
		ControlPlane: govcd.CseControlPlaneSettings{
			MachineCount:      d.Get("control_plane.0.machine_count").(int),
			DiskSizeGi:        d.Get("control_plane.0.disk_size_gi").(int),
			SizingPolicyId:    d.Get("control_plane.0.sizing_policy_id").(string),
			PlacementPolicyId: d.Get("control_plane.0.placement_policy_id").(string),
			StorageProfileId:  d.Get("control_plane.0.storage_profile_id").(string),
			Ip:                d.Get("control_plane.0.ip").(string),
		},
		Owner:              owner,
		ApiToken:           apiToken.RefreshToken,
		NodeHealthCheck:    d.Get("node_health_check").(bool),
		PodCidr:            d.Get("pods_cidr").(string),
		ServiceCidr:        d.Get("services_cidr").(string),
		SshPublicKey:       d.Get("ssh_public_key").(string),
		VirtualIpSubnet:    d.Get("virtual_ip_subnet").(string),
		AutoRepairOnErrors: d.Get("auto_repair_on_errors").(bool),
	}

	workerPoolsAttr := d.Get("worker_pool").([]interface{})
	workerPools := make([]govcd.CseWorkerPoolSettings, len(workerPoolsAttr))
	for i, w := range workerPoolsAttr {
		workerPool := w.(map[string]interface{})
		workerPools[i] = govcd.CseWorkerPoolSettings{
			Name:              workerPool["name"].(string),
			MachineCount:      workerPool["machine_count"].(int),
			DiskSizeGi:        workerPool["disk_size_gi"].(int),
			SizingPolicyId:    workerPool["sizing_policy_id"].(string),
			PlacementPolicyId: workerPool["placement_policy_id"].(string),
			VGpuPolicyId:      workerPool["vgpu_policy_id"].(string),
			StorageProfileId:  workerPool["storage_profile_id"].(string),
		}
	}
	creationData.WorkerPools = workerPools

	if _, ok := d.GetOk("default_storage_class"); ok {
		creationData.DefaultStorageClass = &govcd.CseDefaultStorageClassSettings{
			StorageProfileId: d.Get("default_storage_class.0.storage_profile_id").(string),
			Name:             d.Get("default_storage_class.0.name").(string),
			ReclaimPolicy:    d.Get("default_storage_class.0.reclaim_policy").(string),
			Filesystem:       d.Get("default_storage_class.0.filesystem").(string),
		}
	}

	cluster, err := org.CseCreateKubernetesCluster(creationData, time.Duration(d.Get("operations_timeout_minutes").(int))*time.Minute)
	if err != nil {
		if cluster != nil {
			if cluster.State != "provisioned" {
				return diag.Errorf("Kubernetes cluster creation finished, but it is in '%s' state, not 'provisioned': '%s'", cluster.State, err)
			}
		}
		return diag.Errorf("Kubernetes cluster creation failed: %s", err)
	}
	// We need to set the ID here to be able to distinguish this cluster from all the others that may have the same name and RDE Type.
	// We could use some other ways of filtering, but ID is the only accurate.
	// Also, the RDE is created at this point, so Terraform should trigger an update/delete next.
	// If the cluster can't be created due to errors, users should delete it and retry, like in UI.
	d.SetId(cluster.ID)

	return resourceVcdCseKubernetesRead(ctx, d, meta)
}

func resourceVcdCseKubernetesRead(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	vcdClient := meta.(*VCDClient)
	// The ID must be already set for the read to be successful. We can't rely on the name as there can be
	// many clusters with the same name in the same org.
	cluster, err := vcdClient.CseGetKubernetesClusterById(d.Id())
	if err != nil {
		return diag.Errorf("could not read Kubernetes cluster with ID '%s': %s", d.Id(), err)
	}

	warns, err := saveClusterDataToState(d, cluster)
	if err != nil {
		return diag.Errorf("could not save Kubernetes cluster data into Terraform state: %s", err)
	}
	for _, warning := range warns {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  warning.Error(),
		})
	}

	if len(diags) > 0 {
		return diags
	}
	return nil
}

// resourceVcdCseKubernetesUpdate updates the Kubernetes clusters. Note that re-creating the CAPI YAML and sending it
// back will break everything, so we must patch the YAML piece by piece.
func resourceVcdCseKubernetesUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// Some arguments don't require changes in the backend
	if !d.HasChangesExcept("operations_timeout_minutes") {
		return nil
	}

	vcdClient := meta.(*VCDClient)
	cluster, err := vcdClient.CseGetKubernetesClusterById(d.Id())
	if err != nil {
		return diag.Errorf("could not get Kubernetes cluster with ID '%s': %s", d.Id(), err)
	}
	payload := govcd.CseClusterUpdateInput{}
	if d.HasChange("worker_pool") {
		workerPools := map[string]govcd.CseWorkerPoolUpdateInput{}
		for _, workerPoolAttr := range d.Get("worker_pool").([]interface{}) {
			w := workerPoolAttr.(map[string]interface{})
			workerPools[w["name"].(string)] = govcd.CseWorkerPoolUpdateInput{MachineCount: w["machine_count"].(int)}
		}
		payload.WorkerPools = &workerPools
	}

	err = cluster.Update(payload, true)
	if err != nil {
		if cluster != nil {
			if cluster.State != "provisioned" {
				return diag.Errorf("Kubernetes cluster update finished, but it is in '%s' state, not 'provisioned': '%s'", cluster.State, err)
			}
		}
		return diag.Errorf("Kubernetes cluster update failed: %s", err)
	}

	return resourceVcdCseKubernetesRead(ctx, d, meta)
}

// resourceVcdCseKubernetesDelete deletes a CSE Kubernetes cluster. To delete a Kubernetes cluster, one must send
// the flags "markForDelete" and "forceDelete" back to true, so the CSE Server is able to delete all cluster elements
// and perform a cleanup. Hence, this function sends an update of just these two properties and waits for the cluster RDE
// to be gone.
func resourceVcdCseKubernetesDelete(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vcdClient := meta.(*VCDClient)
	cluster, err := vcdClient.CseGetKubernetesClusterById(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	err = cluster.Delete(time.Duration(d.Get("operations_timeout_minutes").(int)))
	if err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func resourceVcdCseKubernetesImport(_ context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	vcdClient := meta.(*VCDClient)
	cluster, err := vcdClient.CseGetKubernetesClusterById(d.Id())
	if err != nil {
		return nil, fmt.Errorf("error retrieving Kubernetes cluster with ID '%s': %s", d.Id(), err)
	}

	warns, err := saveClusterDataToState(d, cluster)
	if err != nil {
		return nil, fmt.Errorf("failed importing Kubernetes cluster '%s': %s", cluster.ID, err)
	}
	for _, warn := range warns {
		// We can't do much here as Import does not support Diagnostics
		logForScreen(cluster.ID, fmt.Sprintf("got a warning during import: %s", warn))
	}

	return []*schema.ResourceData{d}, nil
}

// saveClusterDataToState reads the received RDE contents and sets the Terraform arguments and attributes.
// Returns a slice of warnings first and an error second.
func saveClusterDataToState(d *schema.ResourceData, cluster *govcd.CseKubernetesCluster) ([]error, error) {
	var warnings []error

	dSet(d, "name", cluster.Name)
	dSet(d, "cse_version", cluster.CseVersion.String())
	dSet(d, "runtime", "tkg") // Only one supported
	dSet(d, "vdc_id", cluster.VdcId)
	dSet(d, "network_id", cluster.NetworkId)
	dSet(d, "cpi_version", cluster.CpiVersion.String())
	dSet(d, "csi_version", cluster.CsiVersion.String())
	dSet(d, "capvcd_version", cluster.CapvcdVersion.String())
	dSet(d, "kubernetes_version", cluster.KubernetesVersion.String())
	dSet(d, "tkg_product_version", cluster.TkgVersion.String())
	dSet(d, "pods_cidr", cluster.PodCidr)
	dSet(d, "services_cidr", cluster.ServiceCidr)
	dSet(d, "ova_id", cluster.KubernetesTemplateOvaId)
	dSet(d, "ssh_public_key", cluster.SshPublicKey)
	dSet(d, "virtual_ip_subnet", cluster.VirtualIpSubnet)
	dSet(d, "auto_repair_on_errors", cluster.AutoRepairOnErrors)
	dSet(d, "node_health_check", cluster.NodeHealthCheck)

	if _, ok := d.GetOk("api_token_file"); !ok {
		// During imports, this field is impossible to get, so we set an artificial value, as this argument
		// is required at runtime
		dSet(d, "api_token_file", "******")
	}
	if _, ok := d.GetOk("owner"); ok {
		// This field is optional, as it can take the value from the VCD client
		dSet(d, "owner", cluster.Owner)
	}

	err := d.Set("cluster_resource_set_bindings", cluster.ClusterResourceSetBindings)
	if err != nil {
		return nil, err
	}

	workerPoolBlocks := make([]map[string]interface{}, len(cluster.WorkerPools))
	for i, workerPool := range cluster.WorkerPools {
		workerPoolBlocks[i] = map[string]interface{}{
			"machine_count":       workerPool.MachineCount,
			"name":                workerPool.Name,
			"vgpu_policy_id":      workerPool.VGpuPolicyId,
			"sizing_policy_id":    workerPool.SizingPolicyId,
			"placement_policy_id": workerPool.PlacementPolicyId,
			"storage_profile_id":  workerPool.StorageProfileId,
			"disk_size_gi":        workerPool.DiskSizeGi,
		}
	}
	err = d.Set("worker_pool", workerPoolBlocks)
	if err != nil {
		return nil, err
	}

	err = d.Set("control_plane", []map[string]interface{}{
		{
			"machine_count":       cluster.ControlPlane.MachineCount,
			"ip":                  cluster.ControlPlane.Ip,
			"sizing_policy_id":    cluster.ControlPlane.SizingPolicyId,
			"placement_policy_id": cluster.ControlPlane.PlacementPolicyId,
			"storage_profile_id":  cluster.ControlPlane.StorageProfileId,
			"disk_size_gi":        cluster.ControlPlane.DiskSizeGi,
		},
	})
	if err != nil {
		return nil, err
	}

	err = d.Set("default_storage_class", []map[string]interface{}{{
		"storage_profile_id": cluster.DefaultStorageClass.StorageProfileId,
		"name":               cluster.DefaultStorageClass.Name,
		"reclaim_policy":     cluster.DefaultStorageClass.ReclaimPolicy,
		"filesystem":         cluster.DefaultStorageClass.Filesystem,
	}})
	if err != nil {
		return nil, err
	}

	dSet(d, "state", cluster.State)

	if cluster.State == "provisioned" {
		kubeconfig, err := cluster.GetKubeconfig()
		if err != nil {
			return nil, fmt.Errorf("error getting Kubeconfig for Kubernetes cluster with ID '%s': %s", cluster.ID, err)
		}
		dSet(d, "kubeconfig", kubeconfig)
	} else {
		warnings = append(warnings, fmt.Errorf("the Kubernetes cluster with ID '%s' is in '%s' state, won't be able to retrieve the Kubeconfig", d.Id(), cluster.State))
	}

	d.SetId(cluster.ID)
	return warnings, nil
}
