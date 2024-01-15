package vcd

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/vmware/go-vcloud-director/v2/govcd"
	"github.com/vmware/go-vcloud-director/v2/types/v56"
	"strconv"
	"strings"
	"text/template"
	"time"
)

// TODO: Split per CSE version: 4.1, 4.2...
//
//go:embed cse/rde.tmpl
var cseRdeJsonTemplate string

//go:embed cse/capi-yaml/cluster.tmpl
var cseClusterYamlTemplate string

//go:embed cse/capi-yaml/node_pool.tmpl
var cseNodePoolTemplate string

// Map of CSE version -> [VCDKEConfig RDE Type version, CAPVCD RDE Type version, CAPVCD Behavior version]
var cseVersions = map[string][]string{
	"4.2": {"1.1.0", "1.2.0", "1.0.0"},
}

func resourceVcdCseKubernetesCluster() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceVcdCseKubernetesClusterCreate,
		ReadContext:   resourceVcdCseKubernetesRead,
		UpdateContext: resourceVcdCseKubernetesUpdate,
		DeleteContext: resourceVcdCseKubernetesDelete,
		Schema: map[string]*schema.Schema{
			"cse_version": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice(getKeys(cseVersions), false),
				Description:  "The CSE version to use",
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
				Required:    true,
				ForceNew:    true,
				Description: "A file generated by 'vcd_api_token' resource, that stores the API token used to create and manage the cluster, owned by the user specified in 'owner'. Be careful about this file and its contents, as it contains sensitive information",
			},
			"ssh_public_key": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "The SSH public key used to login into the cluster nodes",
			},
			"control_plane": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Required: true,
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
							ValidateDiagFunc: minimumValue(20, "disk size in Gibibytes must be at least 20"),
							Description:      "Disk size, in Gibibytes, for the control plane nodes. Must be at least 20",
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
							ForceNew:     true,
							Description:  "IP for the control plane",
							ValidateFunc: checkEmptyOrSingleIP(),
						},
					},
				},
			},
			"node_pool": {
				Type:     schema.TypeSet,
				Required: true,
				MinItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The name of this node pool",
							ValidateDiagFunc: matchRegex(`^[a-z](?:[a-z0-9-]{0,29}[a-z0-9])?$`, "name must contain only lowercase alphanumeric characters or '-',"+
								"start with an alphabetic character, end with an alphanumeric, and contain at most 31 characters"),
						},
						"machine_count": {
							Type:             schema.TypeInt,
							Optional:         true,
							Default:          1, // As suggested in UI
							Description:      "The number of nodes that this node pool has. Must be higher than 0",
							ValidateDiagFunc: minimumValue(1, "number of nodes must be higher than 0"),
						},
						"disk_size_gi": {
							Type:             schema.TypeInt,
							Optional:         true,
							Default:          20, // As suggested in UI
							ForceNew:         true,
							Description:      "Disk size, in Gibibytes, for the control plane nodes",
							ValidateDiagFunc: minimumValue(20, "disk size in Gibibytes must be at least 20"),
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
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"storage_profile_id": {
							Required:    true,
							Type:        schema.TypeString,
							Description: "ID of the storage profile to use for the storage class",
						},
						"name": {
							Required:    true,
							Type:        schema.TypeString,
							Description: "Name to give to this storage class",
							ValidateDiagFunc: matchRegex(`^[a-z](?:[a-z0-9-]{0,29}[a-z0-9])?$`, "name must contain only lowercase alphanumeric characters or '-',"+
								"start with an alphabetic character, end with an alphanumeric, and contain at most 31 characters"),
						},
						"reclaim_policy": {
							Required:     true,
							Type:         schema.TypeString,
							ValidateFunc: validation.StringInSlice([]string{"delete", "retain"}, false),
							Description:  "'delete' deletes the volume when the PersistentVolumeClaim is deleted. 'retain' does not, and the volume can be manually reclaimed",
						},
						"filesystem": {
							Required:     true,
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
				Default:     false,
				Description: "If errors occur before the Kubernetes cluster becomes available, and this argument is 'true', CSE Server will automatically attempt to repair the cluster",
			},
			"node_health_check": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "After the Kubernetes cluster becomes available, nodes that become unhealthy will be remediated according to unhealthy node conditions and remediation rules",
			},
			"delete_timeout_seconds": {
				Type:             schema.TypeInt,
				Optional:         true,
				Default:          120,
				Description:      "The time, in seconds, to wait for the cluster to be deleted when it is marked for deletion. 0 means wait indefinitely",
				ValidateDiagFunc: minimumValue(0, "timeout must be at least 0 (unlimited)"),
			},
			"state": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The state of the cluster, can be 'provisioning', 'provisioned' or 'error'. Useful to check whether the Kubernetes cluster is in a stable status",
			},
			"kubeconfig": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The contents of the kubeconfig of the Kubernetes cluster, only available when 'state=provisioned'",
			},
			"raw_cluster_rde_json": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The raw JSON that describes the cluster configuration inside the Runtime Defined Entity",
			},
		},
	}
}

// getCseRdeTypeVersions gets the RDE Type versions. First returned parameter is VCDKEConfig, second is CAPVCDCluster
func getCseRdeTypeVersions(d *schema.ResourceData) (string, string, string) {
	versions := cseVersions[d.Get("cse_version").(string)]
	return versions[0], versions[1], versions[2]
}

func resourceVcdCseKubernetesClusterCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vcdClient := meta.(*VCDClient)
	vcdKeConfigRdeTypeVersion, capvcdClusterRdeTypeVersion, _ := getCseRdeTypeVersions(d)

	clusterDetails, err := createClusterInfoDto(d, vcdClient, vcdKeConfigRdeTypeVersion, capvcdClusterRdeTypeVersion)
	if err != nil {
		return diag.Errorf("could not create Kubernetes cluster: %s", err)
	}

	entityMap, err := getCseKubernetesClusterEntityMap(d, clusterDetails)
	if err != nil {
		return diag.Errorf("could not create Kubernetes cluster with name '%s': %s", clusterDetails.Name, err)
	}

	rde, err := clusterDetails.RdeType.CreateRde(types.DefinedEntity{
		EntityType: clusterDetails.RdeType.DefinedEntityType.ID,
		Name:       clusterDetails.Name,
		Entity:     entityMap,
	}, &govcd.TenantContext{
		OrgId:   clusterDetails.Org.AdminOrg.ID,
		OrgName: clusterDetails.Org.AdminOrg.Name,
	})
	if err != nil {
		return diag.Errorf("could not create Kubernetes cluster with name '%s': %s", clusterDetails.Name, err)
	}

	// We need to set the ID here to be able to distinguish this cluster from all the others that may have the same name and RDE Type.
	// We could use some other ways of filtering, but ID is the best and most accurate.
	d.SetId(rde.DefinedEntity.ID)
	return resourceVcdCseKubernetesRead(ctx, d, meta)
}

func resourceVcdCseKubernetesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vcdClient := meta.(*VCDClient)
	_, _, capvcdBehaviorVersion := getCseRdeTypeVersions(d)

	var status interface{}
	var rde *govcd.DefinedEntity

	// TODO: Add timeout
	for status == nil {
		// The ID must be already set for the read to be successful. We can't rely on GetRdesByName as there can be
		// many clusters with the same name and RDE Type.
		var err error
		rde, err = vcdClient.GetRdeById(d.Id())
		if err != nil {
			return diag.Errorf("could not read Kubernetes cluster with ID '%s': %s", d.Id(), err)
		}

		status = rde.DefinedEntity.Entity["status"]
		time.Sleep(10 * time.Second)
	}
	if rde == nil {
		return diag.Errorf("could not read Kubernetes cluster with ID '%s': object is nil", d.Id())
	}
	vcdKe, ok := status.(map[string]interface{})["vcdKe"]
	if !ok {
		return diag.Errorf("could not read the 'status.vcdKe' JSON object of the Kubernetes cluster with ID '%s'", d.Id())
	}

	// TODO: Add timeout
	for vcdKe.(map[string]interface{})["state"] != nil && vcdKe.(map[string]interface{})["state"].(string) != "provisioned" {
		if d.Get("auto_repair_on_errors").(bool) && vcdKe.(map[string]interface{})["state"].(string) == "error" {
			return diag.Errorf("cluster creation finished with errors")
		}
		time.Sleep(30 * time.Second)
	}
	dSet(d, "state", vcdKe.(map[string]interface{})["state"])

	_, err := rde.InvokeBehavior(fmt.Sprintf("urn:vcloud:behavior-interface:getFullEntity:cse:capvcd:%s", capvcdBehaviorVersion), types.BehaviorInvocation{})
	if err != nil {
		return diag.Errorf("could not retrieve Kubeconfig: %s", err)
	}

	// This must be the last step, so it has the most possible elements
	jsonEntity, err := jsonToCompactString(rde.DefinedEntity.Entity)
	if err != nil {
		return diag.Errorf("could not save the cluster '%s' raw RDE contents into state: %s", rde.DefinedEntity.ID, err)
	}
	dSet(d, "raw_cluster_rde_json", jsonEntity)

	d.SetId(rde.DefinedEntity.ID) // ID is already there, but just for completeness/readability
	return nil
}

func resourceVcdCseKubernetesUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// TODO
	return diag.Errorf("not implemented")
}

// resourceVcdCseKubernetesDelete deletes a CSE Kubernetes cluster. To delete a Kubernetes cluster, one must send
// the flags "markForDelete" and "forceDelete" back to true, so the CSE Server is able to delete all cluster elements
// and perform a cleanup. Hence, this function sends these properties and waits for deletion.
func resourceVcdCseKubernetesDelete(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vcdClient := meta.(*VCDClient)

	// We need to do this operation with retries due to the mutex mechanism VCD has (ETags).
	// We may hit an error if CSE Server is doing any operation in the background and we attempt to mark the cluster for deletion,
	// so we need to insist several times.
	_, err := runWithRetry(
		fmt.Sprintf("marking the cluster %s for deletion", d.Get("name").(string)),
		"error marking the cluster for deletion",
		30*time.Second,
		nil,
		func() (any, error) {
			rde, err := vcdClient.GetRdeById(d.Id())
			if err != nil {
				return nil, fmt.Errorf("could not retrieve the Kubernetes cluster with ID '%s': %s", d.Id(), err)
			}

			vcdKe, err := navigateMap[map[string]interface{}](rde.DefinedEntity.Entity, "spec.vcdKe")
			if err != nil {
				return nil, fmt.Errorf("JSON object 'spec.vcdKe' is not correct in the RDE")
			}
			vcdKe["markForDelete"] = true
			vcdKe["forceDelete"] = true
			rde.DefinedEntity.Entity["spec"].(map[string]interface{})["vcdKe"] = vcdKe

			err = rde.Update(*rde.DefinedEntity)
			if err != nil {
				return nil, err
			}
			return nil, nil
		},
	)
	if err != nil {
		return diag.FromErr(err)
	}
	_, err = runWithRetry(
		fmt.Sprintf("checking the cluster %s is correctly marked for deletion", d.Get("name").(string)),
		"error completing the deletion of the cluster",
		time.Duration(d.Get("delete_timeout_seconds").(int))*time.Second,
		nil,
		func() (any, error) {
			_, err := vcdClient.GetRdeById(d.Id())
			if err != nil {
				if govcd.IsNotFound(err) {
					return nil, nil // All is correct, the cluster RDE is gone, so it is deleted
				}
				return nil, fmt.Errorf("the cluster with ID '%s' is still present in VCD but it is unreadable: %s", d.Id(), err)
			}

			return nil, fmt.Errorf("the cluster with ID '%s' is marked for deletion but still present in VCD", d.Id())
		},
	)
	if err != nil {
		return diag.FromErr(err)
	}
	return nil
}

// getCseKubernetesClusterEntityMap gets the payload for the RDE that manages the Kubernetes cluster, so it
// can be created or updated.
func getCseKubernetesClusterEntityMap(d *schema.ResourceData, clusterDetails *clusterInfoDto) (StringMap, error) {
	capiYaml, err := generateCapiYaml(d, clusterDetails)
	if err != nil {
		return nil, err
	}

	args := map[string]string{
		"Name":               clusterDetails.Name,
		"Org":                clusterDetails.Org.AdminOrg.Name,
		"VcdUrl":             clusterDetails.VcdUrl,
		"Vdc":                clusterDetails.VdcName,
		"Delete":             "false",
		"ForceDelete":        "false",
		"AutoRepairOnErrors": strconv.FormatBool(d.Get("auto_repair_on_errors").(bool)),
		"ApiToken":           clusterDetails.ApiToken,
		"CapiYaml":           capiYaml,
	}

	if _, isStorageClassSet := d.GetOk("default_storage_class"); isStorageClassSet {
		args["DefaultStorageClassStorageProfile"] = clusterDetails.UrnToNamesCache[d.Get("default_storage_class.0.storage_profile_id").(string)]
		args["DefaultStorageClassName"] = d.Get("default_storage_class.0.name").(string)
		if d.Get("default_storage_class.0.reclaim_policy").(string) == "delete" {
			args["DefaultStorageClassUseDeleteReclaimPolicy"] = "true"
		} else {
			args["DefaultStorageClassUseDeleteReclaimPolicy"] = "false"
		}
		args["DefaultStorageClassFileSystem"] = d.Get("default_storage_class.0.filesystem").(string)
	}

	capvcdEmpty := template.Must(template.New(clusterDetails.Name).Parse(cseRdeJsonTemplate))
	buf := &bytes.Buffer{}
	if err := capvcdEmpty.Execute(buf, args); err != nil {
		return nil, fmt.Errorf("could not render the Go template with the CAPVCD JSON: %s", err)
	}

	var result interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	if err != nil {
		return nil, fmt.Errorf("could not generate a correct CAPVCD JSON: %s", err)
	}

	return result.(map[string]interface{}), nil
}

// generateCapiYaml generates the YAML string that is required during Kubernetes cluster creation, to be embedded
// in the CAPVCD cluster JSON payload. This function picks data from the Terraform schema and the clusterInfoDto to
// populate several Go templates and build a final YAML.
func generateCapiYaml(d *schema.ResourceData, clusterDetails *clusterInfoDto) (string, error) {
	// This YAML snippet contains special strings, such as "%,", that render wrong using the Go template engine
	sanitizedTemplate := strings.NewReplacer("%", "%%").Replace(cseClusterYamlTemplate)
	capiYamlEmpty := template.Must(template.New(clusterDetails.Name + "_CapiYaml").Parse(sanitizedTemplate))

	nodePoolYaml, err := generateNodePoolYaml(d, clusterDetails)
	if err != nil {
		return "", err
	}

	buf := &bytes.Buffer{}
	args := map[string]string{
		"ClusterName":                 clusterDetails.Name,
		"TargetNamespace":             clusterDetails.Name + "-ns",
		"TkrVersion":                  clusterDetails.TkgVersion.Tkr,
		"TkgVersion":                  clusterDetails.TkgVersion.Tkg[0],
		"UsernameB64":                 base64.StdEncoding.EncodeToString([]byte(clusterDetails.Owner)),
		"ApiTokenB64":                 base64.StdEncoding.EncodeToString([]byte(clusterDetails.ApiToken)),
		"PodCidr":                     d.Get("pods_cidr").(string),
		"ServiceCidr":                 d.Get("services_cidr").(string),
		"VcdSite":                     clusterDetails.VcdUrl,
		"Org":                         clusterDetails.Org.AdminOrg.Name,
		"OrgVdc":                      clusterDetails.VdcName,
		"OrgVdcNetwork":               clusterDetails.NetworkName,
		"Catalog":                     clusterDetails.CatalogName,
		"VAppTemplate":                clusterDetails.OvaName,
		"ControlPlaneSizingPolicy":    clusterDetails.UrnToNamesCache[d.Get("control_plane.0.sizing_policy_id").(string)],
		"ControlPlanePlacementPolicy": clusterDetails.UrnToNamesCache[d.Get("control_plane.0.placement_policy_id").(string)],
		"ControlPlaneStorageProfile":  clusterDetails.UrnToNamesCache[d.Get("control_plane.0.storage_profile_id").(string)],
		"ControlPlaneDiskSize":        fmt.Sprintf("%dGi", d.Get("control_plane.0.disk_size_gi").(int)),
		"ControlPlaneMachineCount":    strconv.Itoa(d.Get("control_plane.0.machine_count").(int)),
		"DnsVersion":                  clusterDetails.TkgVersion.CoreDns,
		"EtcdVersion":                 clusterDetails.TkgVersion.Etcd,
		"ContainerRegistryUrl":        clusterDetails.VCDKEConfig.ContainerRegistryUrl,
		"KubernetesVersion":           clusterDetails.TkgVersion.KubernetesVersion,
		"SshPublicKey":                d.Get("ssh_public_key").(string),
	}

	if _, ok := d.GetOk("control_plane.0.ip"); ok {
		args["ControlPlaneEndpoint"] = d.Get("control_plane.0.ip").(string)
	}
	if _, ok := d.GetOk("virtual_ip_subnet"); ok {
		args["VirtualIpSubnet"] = d.Get("virtual_ip_subnet").(string)
	}

	if d.Get("node_health_check").(bool) {
		args["MaxUnhealthyNodePercentage"] = fmt.Sprintf("%s%%", clusterDetails.VCDKEConfig.MaxUnhealthyNodesPercentage) // With the 'percentage' suffix, it is doubled to render the template correctly
		args["NodeStartupTimeout"] = fmt.Sprintf("%ss", clusterDetails.VCDKEConfig.NodeStartupTimeout)                   // With the 'second' suffix
		args["NodeUnknownTimeout"] = fmt.Sprintf("%ss", clusterDetails.VCDKEConfig.NodeUnknownTimeout)                   // With the 'second' suffix
		args["NodeNotReadyTimeout"] = fmt.Sprintf("%ss", clusterDetails.VCDKEConfig.NodeNotReadyTimeout)                 // With the 'second' suffix
	}

	if err := capiYamlEmpty.Execute(buf, args); err != nil {
		return "", fmt.Errorf("could not generate a correct CAPI YAML: %s", err)
	}

	prettyYaml := fmt.Sprintf("%s\n%s", nodePoolYaml, buf.String())

	// This encoder is used instead of a standard json.Marshal as the YAML contains special
	// characters that are not encoded properly, such as '<'.
	buf.Reset()
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	err = enc.Encode(prettyYaml)
	if err != nil {
		return "", fmt.Errorf("could not encode the CAPI YAML into JSON: %s", err)
	}

	return strings.Trim(strings.TrimSpace(buf.String()), "\""), nil
}

// generateNodePoolYaml generates YAML blocks corresponding to the Kubernetes node pools.
func generateNodePoolYaml(d *schema.ResourceData, clusterDetails *clusterInfoDto) (string, error) {
	nodePoolEmptyTmpl := template.Must(template.New(clusterDetails.Name + "_NodePool").Parse(cseNodePoolTemplate))
	resultYaml := ""
	buf := &bytes.Buffer{}

	// We can have many node pool blocks, we build a YAML object for each one of them.
	for _, nodePoolRaw := range d.Get("node_pool").(*schema.Set).List() {
		nodePool := nodePoolRaw.(map[string]interface{})
		name := nodePool["name"].(string)

		// Check the correctness of the compute policies in the node pool block
		placementPolicyId := nodePool["placement_policy_id"]
		vpguPolicyId := nodePool["vgpu_policy_id"]
		if placementPolicyId != "" && vpguPolicyId != "" {
			return "", fmt.Errorf("the node pool '%s' should have either a Placement Policy or a vGPU Policy, not both", name)
		}
		if vpguPolicyId != "" {
			placementPolicyId = vpguPolicyId // For convenience, we just use one of them as both cannot be set at same time
		}

		if err := nodePoolEmptyTmpl.Execute(buf, map[string]string{
			"ClusterName":             clusterDetails.Name,
			"NodePoolName":            name,
			"TargetNamespace":         clusterDetails.Name + "-ns",
			"Catalog":                 clusterDetails.CatalogName,
			"VAppTemplate":            clusterDetails.OvaName,
			"NodePoolSizingPolicy":    clusterDetails.UrnToNamesCache[nodePool["sizing_policy_id"].(string)],
			"NodePoolPlacementPolicy": clusterDetails.UrnToNamesCache[placementPolicyId.(string)],
			"NodePoolStorageProfile":  clusterDetails.UrnToNamesCache[nodePool["storage_profile_id"].(string)],
			"NodePoolDiskSize":        fmt.Sprintf("%dGi", nodePool["disk_size_gi"].(int)),
			"NodePoolEnableGpu":       strconv.FormatBool(vpguPolicyId != ""),
			"NodePoolMachineCount":    strconv.Itoa(nodePool["machine_count"].(int)),
			"KubernetesVersion":       clusterDetails.TkgVersion.KubernetesVersion,
		}); err != nil {
			return "", fmt.Errorf("could not generate a correct Node Pool YAML: %s", err)
		}
		resultYaml += fmt.Sprintf("%s\n---\n", buf.String())
		buf.Reset()
	}
	return resultYaml, nil
}

// clusterInfoDto is a helper struct that contains all the required elements to successfully create and manage
// a Kubernetes cluster using CSE.
type clusterInfoDto struct {
	Name            string
	VcdUrl          string
	Org             *govcd.AdminOrg
	VdcName         string
	OvaName         string
	CatalogName     string
	NetworkName     string
	RdeType         *govcd.DefinedEntityType
	UrnToNamesCache map[string]string // Maps unique IDs with their resource names (example: Compute policy ID with its name)
	VCDKEConfig     struct {
		MaxUnhealthyNodesPercentage string
		NodeStartupTimeout          string
		NodeNotReadyTimeout         string
		NodeUnknownTimeout          string
		ContainerRegistryUrl        string
	}
	TkgVersion *tkgVersion
	Owner      string
	ApiToken   string
}

// tkgVersion is an auxiliary structure used by the tkgMap variable to map
// a Kubernetes template OVA to some specific TKG components versions.
type tkgVersion struct {
	Tkg               []string
	Tkr               string
	Etcd              string
	CoreDns           string
	KubernetesVersion string
}

// tkgMap maps specific Kubernetes template OVAs to specific TKG components versions.
var tkgMap = map[string]tkgVersion{
	"v1.25.7+vmware.2-tkg.1-8a74b9f12e488c54605b3537acb683bc": {
		Tkg:               []string{"v2.2.0"},
		Tkr:               "v1.25.7---vmware.2-tkg.1",
		Etcd:              "v3.5.6_vmware.9",
		CoreDns:           "v1.9.3_vmware.8",
		KubernetesVersion: "v1.25.7+vmware.2",
	},
	"v1.27.5+vmware.1-tkg.1-0eb96d2f9f4f705ac87c40633d4b69st": {
		Tkg:               []string{"v2.4.0"},
		Tkr:               "v1.27.5---vmware.1-tkg.1",
		Etcd:              "v3.5.7_vmware.6",
		CoreDns:           "v1.10.1_vmware.7",
		KubernetesVersion: "v1.25.7+vmware.2",
	},
	"v1.26.8+vmware.1-tkg.1-b8c57a6c8c98d227f74e7b1a9eef27st": {
		Tkg:               []string{"v2.4.0"},
		Tkr:               "v1.26.8---vmware.1-tkg.1",
		Etcd:              "v3.5.6_vmware.20",
		CoreDns:           "v1.10.1_vmware.7",
		KubernetesVersion: "v1.25.7+vmware.2",
	},
	"v1.26.8+vmware.1-tkg.1-0edd4dafbefbdb503f64d5472e500cf8": {
		Tkg:               []string{"v2.3.1"},
		Tkr:               "v1.26.8---vmware.1-tkg.2",
		Etcd:              "v3.5.6_vmware.20",
		CoreDns:           "v1.9.3_vmware.16",
		KubernetesVersion: "v1.25.7+vmware.2",
	},
}

// createClusterInfoDto creates and returns a clusterInfoDto object by obtaining all the required information
// from the Terraform resource data and the target VCD.
func createClusterInfoDto(d *schema.ResourceData, vcdClient *VCDClient, vcdKeConfigRdeTypeVersion, capvcdClusterRdeTypeVersion string) (*clusterInfoDto, error) {
	result := &clusterInfoDto{}
	result.UrnToNamesCache = map[string]string{"": ""} // Initialize with a "zero" entry, used when there's no ID set in the Terraform schema

	name := d.Get("name").(string)
	result.Name = name

	org, err := vcdClient.GetAdminOrgFromResource(d)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve the cluster Organization: %s", err)
	}
	result.Org = org

	vdcId := d.Get("vdc_id").(string)
	vdc, err := org.GetVDCById(vdcId, true)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve the VDC with ID '%s': %s", vdcId, err)
	}
	result.VdcName = vdc.Vdc.Name

	vAppTemplateId := d.Get("ova_id").(string)
	vAppTemplate, err := vcdClient.GetVAppTemplateById(vAppTemplateId)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve the Kubernetes OVA with ID '%s': %s", vAppTemplateId, err)
	}
	result.OvaName = vAppTemplate.VAppTemplate.Name

	// Searches for the TKG components versions in the tkgMap with the OVA name details

	ovaCode := strings.ReplaceAll(vAppTemplate.VAppTemplate.Name, ".ova", "")[strings.LastIndex(vAppTemplate.VAppTemplate.Name, "kube-")+len("kube-"):]
	tkgVersion, ok := tkgMap[ovaCode]
	if !ok {
		return nil, fmt.Errorf("could not retrieve the TKG version details from Kubernetes template '%s'. Please check whether the OVA '%s' is compatible", ovaCode, vAppTemplate.VAppTemplate.Name)
	}
	result.TkgVersion = &tkgVersion

	catalogName, err := vAppTemplate.GetCatalogName()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve the CatalogName of the OVA '%s': %s", vAppTemplateId, err)
	}
	result.CatalogName = catalogName

	networkId := d.Get("network_id").(string)
	network, err := vdc.GetOrgVdcNetworkById(networkId, true)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve the Org VDC NetworkName with ID '%s': %s", networkId, err)
	}
	result.NetworkName = network.OrgVDCNetwork.Name

	rdeType, err := vcdClient.GetRdeType("vmware", "capvcdCluster", capvcdClusterRdeTypeVersion)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve RDE Type vmware:capvcdCluster:'%s': %s", capvcdClusterRdeTypeVersion, err)
	}
	result.RdeType = rdeType

	// Builds a map that relates storage profiles IDs (the schema uses them to build a healthy Terraform dependency graph)
	// with their corresponding names (the cluster YAML and CSE in general uses names only).
	// Having this map minimizes the amount of queries to VCD, specially when building the set of node pools,
	// as there can be a lot of them.
	if _, isStorageClassSet := d.GetOk("default_storage_class"); isStorageClassSet {
		storageProfileId := d.Get("default_storage_class.0.storage_profile_id").(string)
		storageProfile, err := vcdClient.GetStorageProfileById(storageProfileId)
		if err != nil {
			return nil, fmt.Errorf("could not get a Storage Profile with ID '%s' for the Storage Class: %s", storageProfileId, err)
		}
		result.UrnToNamesCache[storageProfileId] = storageProfile.Name
	}
	controlPlaneStorageProfileId := d.Get("control_plane.0.storage_profile_id").(string)
	if _, ok := result.UrnToNamesCache[controlPlaneStorageProfileId]; !ok { // Only query if not already present
		storageProfile, err := vcdClient.GetStorageProfileById(controlPlaneStorageProfileId)
		if err != nil {
			return nil, fmt.Errorf("could not get a Storage Profile with ID '%s' for the Control Plane: %s", controlPlaneStorageProfileId, err)
		}
		result.UrnToNamesCache[controlPlaneStorageProfileId] = storageProfile.Name
	}
	for _, nodePoolRaw := range d.Get("node_pool").(*schema.Set).List() {
		nodePool := nodePoolRaw.(map[string]interface{})
		nodePoolStorageProfileId := nodePool["storage_profile_id"].(string)
		if _, ok := result.UrnToNamesCache[nodePoolStorageProfileId]; !ok { // Only query if not already present
			storageProfile, err := vcdClient.GetStorageProfileById(nodePoolStorageProfileId)
			if err != nil {
				return nil, fmt.Errorf("could not get a Storage Profile with ID '%s' for the Node Pool: %s", controlPlaneStorageProfileId, err)
			}
			result.UrnToNamesCache[nodePoolStorageProfileId] = storageProfile.Name
		}
	}

	// Builds a map that relates Compute Policies IDs (the schema uses them to build a healthy Terraform dependency graph)
	// with their corresponding names (the cluster YAML and CSE in general uses names only).
	// Having this map minimizes the amount of queries to VCD, specially when building the set of node pools,
	// as there can be a lot of them.
	if controlPlaneSizingPolicyId, isSet := d.GetOk("control_plane.0.sizing_policy_id"); isSet {
		computePolicy, err := vcdClient.GetVdcComputePolicyV2ById(controlPlaneSizingPolicyId.(string))
		if err != nil {
			return nil, fmt.Errorf("could not get a Sizing Policy with ID '%s' for the Control Plane: %s", controlPlaneStorageProfileId, err)
		}
		result.UrnToNamesCache[controlPlaneSizingPolicyId.(string)] = computePolicy.VdcComputePolicyV2.Name
	}
	if controlPlanePlacementPolicyId, isSet := d.GetOk("control_plane.0.placement_policy_id"); isSet {
		if _, ok := result.UrnToNamesCache[controlPlanePlacementPolicyId.(string)]; !ok { // Only query if not already present
			computePolicy, err := vcdClient.GetVdcComputePolicyV2ById(controlPlanePlacementPolicyId.(string))
			if err != nil {
				return nil, fmt.Errorf("could not get a Placement Policy with ID '%s' for the Control Plane: %s", controlPlaneStorageProfileId, err)
			}
			result.UrnToNamesCache[controlPlanePlacementPolicyId.(string)] = computePolicy.VdcComputePolicyV2.Name
		}
	}
	for _, nodePoolRaw := range d.Get("node_pool").(*schema.Set).List() {
		nodePool := nodePoolRaw.(map[string]interface{})
		if nodePoolSizingPolicyId, isSet := nodePool["sizing_policy_id"]; isSet {
			if _, ok := result.UrnToNamesCache[nodePoolSizingPolicyId.(string)]; !ok { // Only query if not already present
				computePolicy, err := vcdClient.GetVdcComputePolicyV2ById(nodePoolSizingPolicyId.(string))
				if err != nil {
					return nil, fmt.Errorf("could not get a Sizing Policy with ID '%s' for the Node Pool: %s", controlPlaneStorageProfileId, err)
				}
				result.UrnToNamesCache[nodePoolSizingPolicyId.(string)] = computePolicy.VdcComputePolicyV2.Name
			}
		}
		if nodePoolPlacementPolicyId, isSet := nodePool["placement_policy_id"]; isSet {
			if _, ok := result.UrnToNamesCache[nodePoolPlacementPolicyId.(string)]; !ok { // Only query if not already present
				computePolicy, err := vcdClient.GetVdcComputePolicyV2ById(nodePoolPlacementPolicyId.(string))
				if err != nil {
					return nil, fmt.Errorf("could not get a Placement Policy with ID '%s' for the Node Pool: %s", controlPlaneStorageProfileId, err)
				}
				result.UrnToNamesCache[nodePoolPlacementPolicyId.(string)] = computePolicy.VdcComputePolicyV2.Name
			}
		}
		if nodePoolVGpuPolicyId, isSet := nodePool["vgpu_policy_id"]; isSet {
			if _, ok := result.UrnToNamesCache[nodePoolVGpuPolicyId.(string)]; !ok { // Only query if not already present
				computePolicy, err := vcdClient.GetVdcComputePolicyV2ById(nodePoolVGpuPolicyId.(string))
				if err != nil {
					return nil, fmt.Errorf("could not get a Placement Policy with ID '%s' for the Node Pool: %s", controlPlaneStorageProfileId, err)
				}
				result.UrnToNamesCache[nodePoolVGpuPolicyId.(string)] = computePolicy.VdcComputePolicyV2.Name
			}
		}
	}

	rdes, err := vcdClient.GetRdesByName("vmware", "VCDKEConfig", vcdKeConfigRdeTypeVersion, "vcdKeConfig")
	if err != nil {
		return nil, fmt.Errorf("could not retrieve VCDKEConfig RDE with version %s: %s", vcdKeConfigRdeTypeVersion, err)
	}
	if len(rdes) != 1 {
		return nil, fmt.Errorf("expected exactly one VCDKEConfig RDE but got %d", len(rdes))
	}

	// Obtain some required elements from the CSE Server configuration (aka VCDKEConfig), so we don't have
	// to deal with it again.
	type vcdKeConfigType struct {
		Profiles []struct {
			K8Config struct {
				Mhc struct {
					MaxUnhealthyNodes   int `json:"maxUnhealthyNodes:omitempty"`
					NodeStartupTimeout  int `json:"nodeStartupTimeout:omitempty"`
					NodeNotReadyTimeout int `json:"nodeNotReadyTimeout:omitempty"`
					NodeUnknownTimeout  int `json:"nodeUnknownTimeout:omitempty"`
				} `json:"mhc:omitempty"`
			} `json:"K8Config:omitempty"`
			ContainerRegistryUrl string `json:"containerRegistryUrl,omitempty"`
		} `json:"profiles,omitempty"`
	}

	var vcdKeConfig vcdKeConfigType
	rawData, err := json.Marshal(rdes[0].DefinedEntity.Entity)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(rawData, &vcdKeConfig)
	if err != nil {
		return nil, err
	}

	if len(vcdKeConfig.Profiles) != 1 {
		return nil, fmt.Errorf("wrong format of VCDKEConfig, expected a single 'profiles' element, got %d", len(vcdKeConfig.Profiles))
	}

	result.VCDKEConfig.MaxUnhealthyNodesPercentage = strconv.Itoa(vcdKeConfig.Profiles[0].K8Config.Mhc.MaxUnhealthyNodes)
	result.VCDKEConfig.NodeStartupTimeout = strconv.Itoa(vcdKeConfig.Profiles[0].K8Config.Mhc.NodeStartupTimeout)
	result.VCDKEConfig.NodeNotReadyTimeout = strconv.Itoa(vcdKeConfig.Profiles[0].K8Config.Mhc.NodeNotReadyTimeout)
	result.VCDKEConfig.NodeUnknownTimeout = strconv.Itoa(vcdKeConfig.Profiles[0].K8Config.Mhc.NodeUnknownTimeout)
	result.VCDKEConfig.ContainerRegistryUrl = vcdKeConfig.Profiles[0].ContainerRegistryUrl

	owner, ok := d.GetOk("owner")
	if !ok {
		sessionInfo, err := vcdClient.Client.GetSessionInfo()
		if err != nil {
			return nil, fmt.Errorf("error getting the owner of the cluster: %s", err)
		}
		owner = sessionInfo.User.Name
	}
	result.Owner = owner.(string)

	apiToken, err := govcd.GetTokenFromFile(d.Get("api_token_file").(string))
	if err != nil {
		return nil, fmt.Errorf("API token file could not be parsed or found: %s\nPlease check that the format is the one that 'vcd_api_token' resource uses", err)
	}
	result.ApiToken = apiToken.RefreshToken

	result.VcdUrl = strings.Replace(vcdClient.VCDClient.Client.VCDHREF.String(), "/api", "", 1)
	return result, nil
}
