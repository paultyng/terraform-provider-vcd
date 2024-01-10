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
	"net/url"
	"text/template"
	"time"
)

//go:embed cse/4.2/capvcd.tmpl
var capvcdTemplate string

//go:embed cse/4.2/default_storage_class.tmpl
var defaultStorageClass string

//go:embed cse/4.2/capi_yaml.tmpl
var capiYaml string

func resourceVcdCseKubernetesCluster() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceVcdCseKubernetesClusterCreate,
		ReadContext:   resourceVcdCseKubernetesRead,
		UpdateContext: resourceVcdCseKubernetesUpdate,
		DeleteContext: resourceVcdCseKubernetesDelete,
		Schema: map[string]*schema.Schema{
			"runtime": {
				Type:         schema.TypeString,
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
			},
			"ova_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of the vApp Template that corresponds to a Kubernetes template OVA",
			},
			"capvcd_rde_type_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The CAPVCD RDE Type ID",
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
				Required:    true,
				ForceNew:    true,
				Description: "The user that creates the cluster and owns the API token specified in 'api_token'. It must have the 'Kubernetes Cluster Author' role",
			},
			"api_token_file": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "A file that stores the API token used to create and manage the cluster, owned by the user specified in 'owner'",
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
						"disk_size": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      20, // As suggested in UI
							ForceNew:     true,
							ValidateFunc: IsIntAndAtLeast(20),
							Description:  "Disk size for the control plane nodes",
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
						"storage_profile": {
							Type:        schema.TypeString,
							Optional:    true,
							ForceNew:    true,
							Description: "Storage profile for the control plane nodes",
						},
						"ip": {
							Type:        schema.TypeString,
							Optional:    true,
							ForceNew:    true,
							Description: "IP for the control plane",
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
						"machine_count": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      1, // As suggested in UI
							Description:  "The number of nodes that this node pool has. Must be higher than 0",
							ValidateFunc: IsIntAndAtLeast(1),
						},
						"disk_size": {
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     20, // As suggested in UI
							ForceNew:    true,
							Description: "Disk size for the control plane nodes",
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
			"storage_class": {
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
				Required:    true,
				Description: "Virtual IP subnet for the cluster",
			},
			"auto_repair_on_errors": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "If errors occur before the Kubernetes cluster becomes available, and this argument is 'true', CSE Server will automatically attempt to repair the cluster",
			},
			"node_health_check": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "After the Kubernetes cluster becomes available, nodes that become unhealthy will be remediated according to unhealthy node conditions and remediation rules",
			},
			"state": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The state of the cluster, can be 'provisioning', 'provisioned' or 'error'. Useful to check whether the Kubernetes cluster is in a stable status",
			},
			"raw_cluster_rde_json": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The raw JSON that describes the cluster configuration inside the Runtime Defined Entity",
			},
		},
	}
}

func resourceVcdCseKubernetesClusterCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vcdClient := meta.(*VCDClient)

	clusterDetails, err := createClusterInfoDto(d, vcdClient, "1.1.0")
	if err != nil {
		return diag.Errorf("could not create Kubernetes cluster with name '%s': %s", clusterDetails.Name, err)
	}

	entityMap, err := getCseKubernetesClusterEntityMap(d, clusterDetails)
	if err != nil {
		return diag.Errorf("could not create Kubernetes cluster with name '%s': %s", clusterDetails.Name, err)
	}

	_, err = clusterDetails.RdeType.CreateRde(types.DefinedEntity{
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

	return resourceVcdCseKubernetesRead(ctx, d, meta)
}

func getCseKubernetesClusterEntityMap(d *schema.ResourceData, clusterDetails *clusterInfoDto) (StringMap, error) {
	capiYamlRendered, err := getCapiYamlPlaintext(d, clusterDetails)
	if err != nil {
		return nil, err
	}
	storageClass := "{}"
	if clusterDetails.StorageProfileName != "" {
		storageClassEmpty := template.Must(template.New(clusterDetails.Name + "_StorageClass").Parse(defaultStorageClass))
		storageClassName := d.Get("storage_class.0.name").(string)
		reclaimPolicy := d.Get("storage_class.0.reclaim_policy").(string)
		filesystem := d.Get("storage_class.0.filesystem").(string)

		buf := &bytes.Buffer{}
		if err := storageClassEmpty.Execute(buf, map[string]string{
			"FileSystem":     filesystem,
			"Name":           storageClassName,
			"StorageProfile": clusterDetails.StorageProfileName,
			"ReclaimPolicy":  reclaimPolicy,
		}); err != nil {
			return nil, fmt.Errorf("could not generate a correct storage class JSON block: %s", err)
		}
		storageClass = buf.String()
	}

	capvcdEmpty := template.Must(template.New(clusterDetails.Name).Parse(capvcdTemplate))
	buf := &bytes.Buffer{}
	if err := capvcdEmpty.Execute(buf, map[string]string{
		"Name":                       clusterDetails.Name,
		"Org":                        clusterDetails.Org.AdminOrg.Name,
		"VcdUrl":                     clusterDetails.VcdUrl.String(),
		"Vdc":                        clusterDetails.VdcName,
		"Delete":                     "false",
		"ForceDelete":                "false",
		"AutoRepairOnErrors":         d.Get("auto_repair_on_errors").(string),
		"DefaultStorageClassOptions": storageClass,
		"ApiToken":                   d.Get("api_token").(string),
		"CapiYaml":                   capiYamlRendered,
	}); err != nil {
		return nil, fmt.Errorf("could not generate a correct CAPVCD JSON: %s", err)
	}

	result := map[string]interface{}{}
	err = json.Unmarshal(buf.Bytes(), &result)
	if err != nil {
		return nil, fmt.Errorf("could not generate a correct CAPVCD JSON: %s", err)
	}

	return result, nil
}

func getCapiYamlPlaintext(d *schema.ResourceData, clusterDetails *clusterInfoDto) (string, error) {
	capiYamlEmpty := template.Must(template.New(clusterDetails.Name + "_CapiYaml").Parse(capiYaml))

	buf := &bytes.Buffer{}
	args := map[string]string{
		"ClusterName":                 clusterDetails.Name,
		"TargetNamespace":             clusterDetails.Name + "-ns",
		"MaxUnhealthyNodePercentage":  clusterDetails.VCDKEConfig.MaxUnhealthyNodesPercentage,
		"NodeStartupTimeout":          clusterDetails.VCDKEConfig.NodeStartupTimeout,
		"NodeNotReadyTimeout":         clusterDetails.VCDKEConfig.NodeNotReadyTimeout,
		"TkrVersion":                  clusterDetails.TkrVersion,
		"TkgVersion":                  clusterDetails.TkgVersion,
		"PodCidr":                     d.Get("pods_cidr").(string),
		"ServiceCidr":                 d.Get("service_cidr").(string),
		"UsernameB64":                 base64.StdEncoding.EncodeToString([]byte(d.Get("owner").(string))),
		"ApiTokenB64":                 base64.StdEncoding.EncodeToString([]byte(d.Get("api_token").(string))),
		"VcdSite":                     clusterDetails.VcdUrl.String(),
		"Org":                         clusterDetails.Org.AdminOrg.Name,
		"OrgVdc":                      clusterDetails.VdcName,
		"OrgVdcNetwork":               clusterDetails.NetworkName,
		"CatalogName":                 clusterDetails.CatalogName,
		"VAppTemplateName":            clusterDetails.OvaName,
		"ControlPlaneSizingPolicy":    d.Get("control_plane.0.sizing_policy").(string),
		"ControlPlanePlacementPolicy": d.Get("control_plane.0.placement_policy").(string),
		"ControlPlaneStorageProfile":  d.Get("control_plane.0.storage_profile").(string),
		"ControlPlaneDiskSize":        d.Get("control_plane.0.sizing_policy").(string),
		"ControlPlaneMachineCount":    d.Get("control_plane.0.machine_count").(string),
		"ContainerRegistryUrl":        clusterDetails.VCDKEConfig.ContainerRegistryUrl,
		"SshPublicKey":                d.Get("ssh_public_key").(string),
	}
	for i, nodePoolRaw := range d.Get("node_pool").(*schema.Set).List() {
		nodePool := nodePoolRaw.(map[string]interface{})
		args[fmt.Sprintf("NodePool%dDiskSize", i)] = nodePool["disk_size"].(string)
		args[fmt.Sprintf("NodePool%dMachineCount", i)] = nodePool["machine_count"].(string)
		args[fmt.Sprintf("NodePool%dStorageProfile", i)] = nodePool["storage_profile"].(string)
		args[fmt.Sprintf("NodePool%dPlacementPolicy", i)] = nodePool["placement_policy"].(string)
		args[fmt.Sprintf("NodePool%dSizingPolicy", i)] = nodePool["sizing_policy"].(string)
		args[fmt.Sprintf("NodePool%dMachineCount", i)] = nodePool["machine_count"].(string)
	}

	if err := capiYamlEmpty.Execute(buf, args); err != nil {
		return "", fmt.Errorf("could not generate a correct CAPI YAML: %s", err)
	}
	return buf.String(), nil
}

func resourceVcdCseKubernetesRead(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vcdClient := meta.(*VCDClient)

	rde, err := vcdClient.GetRdeById(d.Id())
	if err != nil {
		return diag.Errorf("could not read Kubernetes cluster with ID '%s': %s", d.Id(), err)
	}
	jsonEntity, err := jsonToCompactString(rde.DefinedEntity.Entity)
	if err != nil {
		return diag.Errorf("could not save the cluster '%s' raw RDE contents into state: %s", rde.DefinedEntity.ID, err)
	}
	dSet(d, "raw_cluster_rde_json", jsonEntity)

	status, ok := rde.DefinedEntity.Entity["status"].(StringMap)
	if !ok {
		return diag.Errorf("could not read the 'status' JSON object of the Kubernetes cluster with ID '%s'", d.Id())
	}

	vcdKe, ok := status["vcdKe"].(StringMap)
	if !ok {
		return diag.Errorf("could not read the 'status.vcdKe' JSON object of the Kubernetes cluster with ID '%s'", d.Id())
	}

	dSet(d, "state", vcdKe["state"])
	d.SetId(rde.DefinedEntity.ID)
	return nil
}

func resourceVcdCseKubernetesUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}

// resourceVcdCseKubernetesDelete deletes a CSE Kubernetes cluster. To delete a Kubernetes cluster, one must send
// the flags "markForDelete" and "forceDelete" back to true, so the CSE Server is able to delete all cluster elements
// and perform a cleanup. Hence, this function sends these properties and waits for deletion.
func resourceVcdCseKubernetesDelete(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vcdClient := meta.(*VCDClient)

	rde, err := vcdClient.GetRdeById(d.Id())
	if err != nil {
		return diag.Errorf("could not retrieve the Kubernetes cluster with ID '%s': %s", d.Id(), err)
	}

	spec, ok := rde.DefinedEntity.Entity["spec"].(StringMap)
	if !ok {
		return diag.Errorf("could not delete the cluster, JSON object 'spec' is not correct in the RDE")
	}

	spec["markForDelete"] = true
	spec["forceDelete"] = true
	rde.DefinedEntity.Entity["spec"] = spec

	err = rde.Update(*rde.DefinedEntity)
	if err != nil {
		return diag.Errorf("could not delete the cluster '%s': %s", rde.DefinedEntity.ID, err)
	}

	// TODO: Add a timeout
	deletionComplete := false
	for !deletionComplete {
		_, err = vcdClient.GetRdeById(d.Id())
		if err != nil {
			if govcd.IsNotFound(err) {
				deletionComplete = true
			}
			return diag.Errorf("could not check whether the cluster '%s' is deleted: %s", d.Id(), err)
		}
		time.Sleep(30 * time.Second)
	}
	return nil
}

// clusterInfoDto is a helper struct that contains all the required elements to successfully create and manage
// a Kubernetes cluster using CSE.
type clusterInfoDto struct {
	Name               string
	VcdUrl             url.URL
	Org                *govcd.AdminOrg
	VdcName            string
	OvaName            string
	CatalogName        string
	NetworkName        string
	RdeType            *govcd.DefinedEntityType
	StorageProfileName string
	VCDKEConfig        struct {
		MaxUnhealthyNodesPercentage string
		NodeStartupTimeout          string
		NodeNotReadyTimeout         string
		NodeUnknownTimeout          string
		ContainerRegistryUrl        string
	}
	TkgVersion string
	TkrVersion string
}

// createClusterInfoDto creates and returns a clusterInfoDto object by obtaining all the required information
// from th input Terraform resource data.
func createClusterInfoDto(d *schema.ResourceData, vcdClient *VCDClient, vcdKeConfigVersion string) (*clusterInfoDto, error) {
	result := &clusterInfoDto{}

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
	result.TkgVersion = ""
	result.TkrVersion = ""

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

	rdeTypeId := d.Get("capvcd_rde_type_id").(string)
	rdeType, err := vcdClient.GetRdeTypeById(rdeTypeId)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve RDE Type with ID '%s': %s", rdeTypeId, err)
	}
	result.RdeType = rdeType

	storageProfileName := ""
	if _, isStorageClassSet := d.GetOk("storage_class"); isStorageClassSet {
		storageProfileId := d.Get("storage_class.0.storage_profile_id").(string)
		storageProfile, err := vcdClient.GetStorageProfileById(storageProfileId)
		if err != nil {
			return nil, fmt.Errorf("could not get a Storage Profile with ID '%s': %s", storageProfileId, err)
		}
		storageProfileName = storageProfile.Name
	}
	result.StorageProfileName = storageProfileName

	rdes, err := vcdClient.GetRdesByName("vmware", "VCDKEConfig", vcdKeConfigVersion, "VCDKEConfig")
	if err != nil {
		return nil, fmt.Errorf("could not retrieve VCDKEConfig RDE: %s", err)
	}
	if len(rdes) != 1 {
		return nil, fmt.Errorf("expected exactly one VCDKEConfig RDE but got %d", len(rdes))
	}
	vcdKeConfig := rdes[0].DefinedEntity.Entity
	if _, ok := vcdKeConfig["profiles"]; !ok {
		return nil, fmt.Errorf("expected array 'profiles' in VCDKEConfig, but it is nil")
	}
	if _, ok := vcdKeConfig["profiles"].([]map[string]interface{}); !ok {
		return nil, fmt.Errorf("expected array 'profiles' in VCDKEConfig, but it is not an array")
	}
	if len(vcdKeConfig["profiles"].([]map[string]interface{})) != 1 {
		return nil, fmt.Errorf("expected exactly one 'profiles' item in VCDKEConfig, but it has %d", len(vcdKeConfig["profiles"].([]map[string]interface{})))
	}
	if _, ok := vcdKeConfig["profiles"].([]map[string]interface{})[0]["K8Config"]; !ok {
		return nil, fmt.Errorf("expected item 'profiles[0].K8Config' in VCDKEConfig, but it is nil")
	}
	if _, ok := vcdKeConfig["profiles"].([]map[string]interface{})[0]["K8Config"].(map[string]interface{}); !ok {
		return nil, fmt.Errorf("expected an object 'profiles[0].K8Config' in VCDKEConfig, but it is not an object")
	}
	if _, ok := vcdKeConfig["profiles"].([]map[string]interface{})[0]["K8Config"].(map[string]interface{})["mhc"]; !ok {
		return nil, fmt.Errorf("expected item 'profiles[0].K8Config.mhc' in VCDKEConfig, but it is nil")
	}
	if _, ok := vcdKeConfig["profiles"].([]map[string]interface{})[0]["K8Config"].(map[string]interface{})["mhc"].(map[string]interface{}); !ok {
		return nil, fmt.Errorf("expected an object 'profiles[0].K8Config.mhc' in VCDKEConfig, but it is not an object")
	}
	mhc := vcdKeConfig["profiles"].([]map[string]interface{})[0]["K8Config"].(map[string]interface{})["mhc"].(map[string]interface{})
	result.VCDKEConfig.MaxUnhealthyNodesPercentage = mhc["maxUnhealthyNodes"].(string)
	result.VCDKEConfig.NodeStartupTimeout = mhc["nodeStartupTimeout"].(string)
	result.VCDKEConfig.NodeNotReadyTimeout = mhc["nodeNotReadyTimeout"].(string)
	result.VCDKEConfig.NodeUnknownTimeout = mhc["nodeUnknownTimeout"].(string)

	if _, ok := vcdKeConfig["profiles"].([]map[string]interface{})[0]["containerRegistryUrl"]; !ok {
		return nil, fmt.Errorf("expected item 'profiles[0].containerRegistryUrl' in VCDKEConfig, but it is nil")
	}
	result.VCDKEConfig.ContainerRegistryUrl = vcdKeConfig["profiles"].([]map[string]interface{})[0]["containerRegistryUrl"].(string)

	result.VcdUrl = vcdClient.VCDClient.Client.VCDHREF
	return result, nil
}