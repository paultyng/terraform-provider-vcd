package vcd

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/vmware/go-vcloud-director/v2/govcd"
	"github.com/vmware/go-vcloud-director/v2/types/v56"
	"github.com/vmware/go-vcloud-director/v2/util"
	"io"
	"log"
	"net/http"
	"strings"
)

func resourceVcdRdeType() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceVcdRdeTypeCreate,
		ReadContext:   resourceVcdRdeTypeRead,
		UpdateContext: resourceVcdRdeTypeUpdate,
		DeleteContext: resourceVcdRdeTypeDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceVcdRdeTypeImport,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the Runtime Defined Entity type",
			},
			"vendor": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The vendor name for the Runtime Defined Entity type",
			},
			"namespace": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "A unique namespace associated with the Runtime Defined Entity type",
			},
			"version": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The version of the Runtime Defined Entity type. The version string must follow semantic versioning rules",
			},
			"interface_ids": {
				Type: schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Required:    true,
				Description: "Set of Defined Interface URNs that this Runtime Defined Entity type is referenced by",
			},
			"schema_url": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "URL that should point to a JSON-Schema valid definition file of the Runtime Defined Entity type",
				AtLeastOneOf: []string{"schema_url", "schema"},
			},
			"schema": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  "The JSON-Schema valid definition of the Runtime Defined Entity type",
				AtLeastOneOf: []string{"schema_url", "schema"},
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The description of the Runtime Defined Entity type",
			},
			"external_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "An external entity's id that this definition may apply to",
			},
			"inherited_version": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Description: "To be used when creating a new version of a Runtime Defined Entity type. Specifies the version of the type that will be the template for the authorization configuration of the new version." +
					"The Type ACLs and the access requirements of the Type Behaviors of the new version will be copied from those of the inherited version." +
					"If not set, then the new type version will not inherit another version and will have the default authorization settings, just like the first version of a new type",
			},
			// FIXME: It seems only accepts false???? Doesnt support Update neither
			"readonly": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "True if the Runtime Defined Entity type cannot be modified",
			},
		},
	}
}

func resourceVcdRdeTypeCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vcdClient := meta.(*VCDClient)

	jsonSchema, err := getRdeTypeSchema(d)
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = vcdClient.VCDClient.CreateRDEType(&types.DefinedEntityType{
		Name:             d.Get("name").(string),
		Namespace:        d.Get("namespace").(string),
		Version:          d.Get("version").(string),
		Description:      d.Get("description").(string),
		ExternalId:       d.Get("external_id").(string),
		InheritedVersion: d.Get("inherited_version").(string),
		Interfaces:       convertSchemaSetToSliceOfStrings(d.Get("interface_ids").(*schema.Set)),
		IsReadOnly:       d.Get("readonly").(bool),
		Schema:           jsonSchema,
		Vendor:           d.Get("vendor").(string),
	})
	if err != nil {
		return diag.Errorf("could not create the Runtime Defined Entity type: %s", err)
	}
	return resourceVcdRdeTypeRead(ctx, d, meta)
}

// getRdeTypeSchema gets the schema as string from the Terraform configuration
func getRdeTypeSchema(d *schema.ResourceData) (map[string]interface{}, error) {
	var jsonSchema string
	var err error
	if url, isUrlSet := d.GetOk("schema_url"); isUrlSet {
		jsonSchema, err = fileFromUrlToString(url.(string), ".json")
		if err != nil {
			return nil, fmt.Errorf("could not download JSON schema from url %s: %s", url, err)
		}
	} else {
		jsonSchema = d.Get("schema").(string)
	}

	var unmarshalledJson map[string]interface{}
	err = json.Unmarshal([]byte(jsonSchema), &unmarshalledJson)
	if err != nil {
		return nil, err
	}

	return unmarshalledJson, err
}

// fileFromUrlToString checks that the given url is correct and points to a given file type,
// if so it downloads its contents and returns it as string.
func fileFromUrlToString(url, fileType string) (string, error) {
	if !strings.HasSuffix(url, fileType) {
		return "", fmt.Errorf("it was expecting the URL to point to a %s file but it was %s", fileType, url)
	}
	buf := new(strings.Builder)

	// #nosec G107 -- The URL needs to come from a variable for this purpose
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			util.Logger.Printf("[ERROR] Could not close body: %s", err)
		}
	}()
	_, err = io.Copy(buf, resp.Body)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func resourceVcdRdeTypeRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return genericVcdRdeTypeRead(ctx, d, meta, "resource")
}

// genericVcdRdeTypeRead reads a Runtime Defined Entity type from VCD and sets the Terraform state accordingly.
// If origin == "datasource", if the referenced RDE type doesn't exist, it errors.
// If origin == "resource", if the referenced RDE type doesn't exist, it removes it from tfstate and exits normally.
func genericVcdRdeTypeRead(_ context.Context, d *schema.ResourceData, meta interface{}, origin string) diag.Diagnostics {
	rdeType, err := getRDEType(d, meta)
	if origin == "resource" && govcd.ContainsNotFound(err) {
		log.Printf("[DEBUG] Runtime Defined Entity type no longer exists. Removing from tfstate")
		d.SetId("")
		return nil
	}
	if err != nil {
		return diag.FromErr(err)
	}

	dSet(d, "vendor", rdeType.DefinedEntityType.Vendor)
	dSet(d, "namespace", rdeType.DefinedEntityType.Namespace)
	dSet(d, "version", rdeType.DefinedEntityType.Version)
	dSet(d, "name", rdeType.DefinedEntityType.Name)
	dSet(d, "readonly", rdeType.DefinedEntityType.IsReadOnly)
	dSet(d, "description", rdeType.DefinedEntityType.Description)
	dSet(d, "external_id", rdeType.DefinedEntityType.ExternalId)
	dSet(d, "inherited_version", rdeType.DefinedEntityType.InheritedVersion)
	err = d.Set("interface_ids", rdeType.DefinedEntityType.Interfaces)
	if err != nil {
		return diag.FromErr(err)
	}
	jsonSchema, err := jsonToCompactString(rdeType.DefinedEntityType.Schema)
	if err != nil {
		return diag.Errorf("could not save the Runtime Defined Entity type schema into state: %s", err)
	}
	err = d.Set("schema", jsonSchema)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(rdeType.DefinedEntityType.ID)

	return nil
}

// getRDEType retrieves a Runtime Defined Entity type from VCD with the required attributes from the Terraform config.
func getRDEType(d *schema.ResourceData, meta interface{}) (*govcd.DefinedEntityType, error) {
	vcdClient := meta.(*VCDClient)

	if d.Id() != "" {
		return vcdClient.VCDClient.GetRDETypeById(d.Id())
	}

	vendor := d.Get("vendor").(string)
	nss := d.Get("namespace").(string)
	version := d.Get("version").(string)

	return vcdClient.VCDClient.GetRDEType(vendor, nss, version)
}

func resourceVcdRdeTypeUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	rdeType, err := getRDEType(d, meta)
	if govcd.ContainsNotFound(err) {
		log.Printf("[DEBUG] Runtime Defined Entity type no longer exists. Removing from tfstate")
		return nil
	}
	if err != nil {
		return diag.FromErr(err)
	}
	jsonSchema, err := getRdeTypeSchema(d)
	if err != nil {
		return diag.FromErr(err)
	}

	err = rdeType.Update(types.DefinedEntityType{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		ExternalId:  d.Get("external_id").(string),
		Interfaces:  convertSchemaSetToSliceOfStrings(d.Get("interface_ids").(*schema.Set)),
		Schema:      jsonSchema,
	})
	if err != nil {
		return diag.Errorf("could not update the Runtime Defined Entity type: %s", err)
	}
	return resourceVcdRdeTypeRead(ctx, d, meta)
}

func resourceVcdRdeTypeDelete(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	rdeType, err := getRDEType(d, meta)
	if govcd.ContainsNotFound(err) {
		log.Printf("[DEBUG] Runtime Defined Entity type no longer exists. Removing from tfstate")
		return nil
	}
	if err != nil {
		return diag.FromErr(err)
	}
	err = rdeType.Delete()
	if err != nil {
		return diag.Errorf("could not delete the Runtime Defined Entity type: %s", err)
	}
	return nil
}

// resourceVcdRdeTypeImport is responsible for importing the resource.
// The following steps happen as part of import
// 1. The user supplies `terraform import _resource_name_ _the_id_string_` command
// 2. `_the_id_string_` contains a dot formatted path to resource as in the example below
// 3. The functions splits the dot-formatted path and tries to lookup the object
// 4. If the lookup succeeds it set's the ID field for `_resource_name_` resource in state file
// (the resource must be already defined in .tf config otherwise `terraform import` will complain)
// 5. `terraform refresh` is being implicitly launched. The Read method looks up all other fields
// based on the known ID of object.
//
// Example resource name (_resource_name_): vcd_rde_type.outer-type
// Example import path (_the_id_string_): vmware.kubernetes.1.0.0
// Note: the separator can be changed using Provider.import_separator or variable VCD_IMPORT_SEPARATOR
func resourceVcdRdeTypeImport(_ context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	resourceURI := strings.Split(d.Id(), ImportSeparator)
	if len(resourceURI) < 3 {
		return nil, fmt.Errorf("resource identifier must be specified as vendor.namespace.version")
	}
	vendor, namespace, version := resourceURI[0], resourceURI[1], strings.Join(resourceURI[2:], ".")

	vcdClient := meta.(*VCDClient)
	rdeType, err := vcdClient.GetRDEType(vendor, namespace, version)
	if err != nil {
		return nil, fmt.Errorf("error finding Runtime Defined Entity type with vendor %s, namespace %s and version %s: %s", vendor, namespace, version, err)
	}

	d.SetId(rdeType.DefinedEntityType.ID)
	return []*schema.ResourceData{d}, nil
}
