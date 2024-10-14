package vcd

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/vmware/go-vcloud-director/v2/govcd"
	"github.com/vmware/go-vcloud-director/v2/types/v56"
)

func resourceVcdTmContentLibrary() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceVcdTmContentLibraryCreate,
		ReadContext:   resourceVcdTmContentLibraryRead,
		UpdateContext: resourceVcdTmContentLibraryUpdate,
		DeleteContext: resourceVcdTmContentLibraryDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceVcdTmContentLibraryImport,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true, // TODO: Update not supported
				Description: "The name of the Content Library",
			},
			"storage_policy_ids": {
				Type:        schema.TypeSet,
				Required:    true,
				ForceNew:    true, // TODO: Update not supported
				Description: "A set of Content Library or VDC Storage Policy IDs used by this Content Library",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"auto_attach": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
				ForceNew: true, // Cannot be updated
				Description: "For Tenant Content Libraries this field represents whether this Content Library should be " +
					"automatically attached to all current and future namespaces in the tenant organization. If no value is " +
					"supplied during creation then this field will default to true. If a value of false is supplied, " +
					"then this Tenant Content Library will only be attached to namespaces that explicitly request it. " +
					"For Provider Content Libraries this field is not needed for creation and will always be returned as true. " +
					"This field cannot be updated after Content Library creation",
			},
			"creation_date": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ISO-8601 timestamp representing when this Content Library was created",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true, // TODO: Update not supported
				Description: "The description of the Content Library",
			},
			"is_shared": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether this Content Library is shared with other Organziations",
			},
			"is_subscribed": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether this Content Library is subscribed from an external published library",
			},
			"library_type": {
				Type:     schema.TypeString,
				Computed: true,
				Description: "The type of content library, can be either PROVIDER (Content Library that is scoped to a " +
					"provider) or TENANT (Content Library that is scoped to a tenant organization)",
			},
			"owner_org_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The reference to the Organization that the Content Library belongs to",
			},
			"subscription_config": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Optional:    true,
				Description: "A block representing subscription settings of a Content Library",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"subscription_url": {
							Type:        schema.TypeString,
							Required:    true,
							ForceNew:    true, // TODO: Update not supported
							Description: "Subscription url of this Content Library",
						},
						"password": {
							Type:        schema.TypeString,
							Optional:    true, // Required at Runtime as cannot be Required + Computed in schema. (It is computed as password cannot be recovered)
							Computed:    true,
							Description: "Password to use to authenticate with the publisher",
						},
						"need_local_copy": {
							Type:        schema.TypeBool,
							Optional:    true,
							ForceNew:    true, // TODO: Update not supported
							Description: "Whether to eagerly download content from publisher and store it locally",
						},
					},
				},
			},
			"version_number": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Version number of this Content library",
			},
		},
	}
}

func resourceVcdTmContentLibraryCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vcdClient := meta.(*VCDClient)

	t, err := getContentLibraryType(d)
	if err != nil {
		return diag.Errorf("error getting Content Library type: %s", err)
	}

	cl, err := vcdClient.CreateContentLibrary(t)
	if err != nil {
		return diag.Errorf("error creating Content Library: %s", err)
	}

	d.SetId(cl.ContentLibrary.Id)

	return resourceVcdTmContentLibraryRead(ctx, d, meta)
}

func resourceVcdTmContentLibraryUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vcdClient := meta.(*VCDClient)
	rsp, err := vcdClient.GetContentLibraryById(d.Id())
	if err != nil {
		return diag.Errorf("error retrieving Content Library: %s", err)
	}

	t, err := getContentLibraryType(d)
	if err != nil {
		return diag.Errorf("error getting Content Library type: %s", err)
	}

	_, err = rsp.Update(t)
	if err != nil {
		return diag.Errorf("error updating Content Library Type: %s", err)
	}

	return resourceVcdTmContentLibraryRead(ctx, d, meta)
}

func resourceVcdTmContentLibraryRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return genericVcdTmContentLibraryRead(ctx, d, meta, "resource")
}
func genericVcdTmContentLibraryRead(_ context.Context, d *schema.ResourceData, meta interface{}, origin string) diag.Diagnostics {
	vcdClient := meta.(*VCDClient)

	var cl *govcd.ContentLibrary
	var err error
	if d.Id() != "" {
		cl, err = vcdClient.GetContentLibraryById(d.Id())
	} else {
		cl, err = vcdClient.GetContentLibraryByName(d.Get("name").(string))
	}
	if err != nil {
		if origin == "resource" && govcd.ContainsNotFound(err) {
			d.SetId("")
			return nil
		}
		return diag.Errorf("error retrieving Content Library: %s", err)
	}

	err = setTmContentLibraryData(d, cl.ContentLibrary)
	if err != nil {
		return diag.Errorf("error saving Content Library data into state: %s", err)
	}

	d.SetId(cl.ContentLibrary.Id)
	return nil
}

func resourceVcdTmContentLibraryDelete(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vcdClient := meta.(*VCDClient)
	cl, err := vcdClient.GetContentLibraryById(d.Id())
	if err != nil {
		return diag.Errorf("error retrieving Content Library: %s", err)
	}

	err = cl.Delete()
	if err != nil {
		return diag.Errorf("error deleting Content Library: %s", err)
	}

	return nil
}

func resourceVcdTmContentLibraryImport(_ context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	vcdClient := meta.(*VCDClient)
	rsp, err := vcdClient.GetContentLibraryByName(d.Id())
	if err != nil {
		return nil, fmt.Errorf("error retrieving Content Library with name '%s': %s", d.Id(), err)
	}

	d.SetId(rsp.ContentLibrary.Id)
	dSet(d, "name", rsp.ContentLibrary.Name)
	return []*schema.ResourceData{d}, nil
}

func getContentLibraryType(d *schema.ResourceData) (*types.ContentLibrary, error) {
	t := &types.ContentLibrary{
		Name:            d.Get("name").(string),
		Description:     d.Get("description").(string),
		AutoAttach:      d.Get("auto_attach").(bool),
		StoragePolicies: convertSliceOfStringsToOpenApiReferenceIds(convertTypeListToSliceOfStrings(d.Get("storage_policy_ids").(*schema.Set).List())),
	}
	if v, ok := d.GetOk("subscription_config"); ok {
		subsConfig := v.([]interface{})[0].(map[string]interface{})
		t.SubscriptionConfig = &types.ContentLibrarySubscriptionConfig{
			SubscriptionUrl: subsConfig["subscription_url"].(string),
			NeedLocalCopy:   subsConfig["need_local_copy"].(bool),
			Password:        subsConfig["password"].(string),
		}
	}
	return t, nil
}

func setTmContentLibraryData(d *schema.ResourceData, cl *types.ContentLibrary) error {
	dSet(d, "name", cl.Name)
	dSet(d, "auto_attach", cl.AutoAttach)
	dSet(d, "creation_date", cl.CreationDate)
	dSet(d, "description", cl.Description)
	dSet(d, "is_shared", cl.IsShared)
	dSet(d, "is_subscribed", cl.IsSubscribed)
	dSet(d, "library_type", cl.LibraryType)
	dSet(d, "version_number", cl.VersionNumber)
	if cl.Org != nil {
		dSet(d, "owner_org_id", cl.Org.ID)
	}

	sps := make([]string, len(cl.StoragePolicies))
	for i, sp := range cl.StoragePolicies {
		sps[i] = sp.ID
	}
	err := d.Set("storage_policy_ids", sps)
	if err != nil {
		return err
	}

	subscriptionConfig := make([]interface{}, 0)
	if cl.SubscriptionConfig != nil {
		subscriptionConfig = []interface{}{
			map[string]interface{}{
				"subscription_url": cl.SubscriptionConfig.SubscriptionUrl,
				"password":         cl.SubscriptionConfig.Password,
				"need_local_copy":  cl.SubscriptionConfig.NeedLocalCopy,
			},
		}
	}
	err = d.Set("subscription_config", subscriptionConfig)
	if err != nil {
		return err
	}
	return nil
}
