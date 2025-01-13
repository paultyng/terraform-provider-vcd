package vcd

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/vmware/go-vcloud-director/v3/govcd"
	"github.com/vmware/go-vcloud-director/v3/types/v56"
)

const labelTmContentLibraryItem = "Content Library Item"

func resourceVcdTmContentLibraryItem() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceVcdTmContentLibraryItemCreate,
		ReadContext:   resourceVcdTmContentLibraryItemRead,
		// TODO: TM: Update not supported yet
		// UpdateContext: resourceVcdTmContentLibraryItemUpdate,
		DeleteContext: resourceVcdTmContentLibraryItemDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceVcdTmContentLibraryItemImport,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true, // TODO: TM: Update not supported yet
				Description: fmt.Sprintf("Name of the %s", labelTmContentLibraryItem),
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true, // TODO: TM: Update not supported yet
				Description: fmt.Sprintf("The description of the %s", labelTmContentLibraryItem),
			},
			"content_library_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: fmt.Sprintf("ID of the Content Library that this %s belongs to", labelTmContentLibraryItem),
			},
			"file_path": {
				Type:        schema.TypeString,
				Optional:    true, // Not needed when Importing
				ForceNew:    true, // TODO: TM: Update not supported yet
				Description: fmt.Sprintf("Path to the OVA/ISO to create the %s", labelTmContentLibraryItem),
			},
			"upload_piece_size": {
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    true, // TODO: TM: Update not supported yet
				Default:     1,
				Description: fmt.Sprintf("When uploading the %s, this argument defines the size of the file chunks in which it is split on every upload request. It can possibly impact upload performance. Default 1 MB", labelTmContentLibraryItem),
			},
			"creation_date": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: fmt.Sprintf("The ISO-8601 timestamp representing when this %s was created", labelTmContentLibraryItem),
			},
			"image_identifier": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: fmt.Sprintf("Virtual Machine Identifier (VMI) of the %s. This is a ReadOnly field", labelTmContentLibraryItem),
			},
			"is_published": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: fmt.Sprintf("Whether this %s is published", labelTmContentLibraryItem),
			},
			"is_subscribed": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: fmt.Sprintf("Whether this %s is subscribed", labelTmContentLibraryItem),
			},
			"last_successful_sync": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: fmt.Sprintf("The ISO-8601 timestamp representing when this %s was last synced if subscribed", labelTmContentLibraryItem),
			},
			"owner_org_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: fmt.Sprintf("The reference to the organization that the %s belongs to", labelTmContentLibraryItem),
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: fmt.Sprintf("Status of this %s", labelTmContentLibraryItem),
			},
			"version": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: fmt.Sprintf("The version of this %s. For a subscribed library, this version is same as in publisher library", labelTmContentLibraryItem),
			},
		},
	}
}

func resourceVcdTmContentLibraryItemCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vcdClient := meta.(*VCDClient)

	clId := d.Get("content_library_id").(string)
	// TODO: TM: Tenant Context should not be nil and depend on the configured owner_org_id
	cl, err := vcdClient.GetContentLibraryById(clId, nil)
	if err != nil {
		return diag.Errorf("could not retrieve Content Library with ID '%s': %s", clId, err)
	}

	filePath := d.Get("file_path").(string)
	uploadPieceSize := d.Get("upload_piece_size").(int)

	c := crudConfig[*govcd.ContentLibraryItem, types.ContentLibraryItem]{
		entityLabel:    labelTmContentLibraryItem,
		getTypeFunc:    getContentLibraryItemType,
		stateStoreFunc: setContentLibraryItemData,
		createFunc: func(config *types.ContentLibraryItem) (*govcd.ContentLibraryItem, error) {
			return cl.CreateContentLibraryItem(config, govcd.ContentLibraryItemUploadArguments{
				FilePath:        filePath,
				UploadPieceSize: int64(uploadPieceSize) * 1024 * 1024,
			})
		},
		resourceReadFunc: resourceVcdTmContentLibraryItemRead,
	}
	return createResource(ctx, d, meta, c)
}

//func resourceVcdTmContentLibraryItemUpdate(_ context.Context, _ *schema.ResourceData, _ interface{}) diag.Diagnostics {
//	// TODO: TM: Update is not supported yet
//	return diag.Errorf("update not supported")
//	vcdClient := meta.(*VCDClient)
//
//	clId := d.Get("content_library_id").(string)
//	cl, err := vcdClient.GetContentLibraryById(clId)
//	if err != nil {
//		return diag.Errorf("could not retrieve Content Library with ID '%s': %s", clId, err)
//	}
//
//	c := crudConfig[*govcd.ContentLibraryItem, types.ContentLibraryItem]{
//		entityLabel:      labelTmContentLibraryItem,
//		getTypeFunc:      getContentLibraryItemType,
//		getEntityFunc:    cl.GetContentLibraryItemById,
//		resourceReadFunc: resourceVcdTmContentLibraryItemRead,
//	}
//
//	return updateResource(ctx, d, meta, c)
//}

func resourceVcdTmContentLibraryItemRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vcdClient := meta.(*VCDClient)

	clId := d.Get("content_library_id").(string)
	// TODO: TM: Tenant Context should not be nil and depend on the configured owner_org_id
	cl, err := vcdClient.GetContentLibraryById(clId, nil)
	if err != nil {
		return diag.Errorf("could not retrieve Content Library with ID '%s': %s", clId, err)
	}

	c := crudConfig[*govcd.ContentLibraryItem, types.ContentLibraryItem]{
		entityLabel:    labelTmContentLibraryItem,
		getEntityFunc:  cl.GetContentLibraryItemById,
		stateStoreFunc: setContentLibraryItemData,
	}
	return readResource(ctx, d, meta, c)
}

func resourceVcdTmContentLibraryItemDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vcdClient := meta.(*VCDClient)

	clId := d.Get("content_library_id").(string)
	// TODO: TM: Tenant Context should not be nil and depend on the configured owner_org_id
	cl, err := vcdClient.GetContentLibraryById(clId, nil)
	if err != nil {
		return diag.Errorf("could not retrieve Content Library with ID '%s': %s", clId, err)
	}

	c := crudConfig[*govcd.ContentLibraryItem, types.ContentLibraryItem]{
		entityLabel:   labelTmContentLibraryItem,
		getEntityFunc: cl.GetContentLibraryItemById,
	}

	return deleteResource(ctx, d, meta, c)
}

func resourceVcdTmContentLibraryItemImport(_ context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	vcdClient := meta.(*VCDClient)

	id := strings.Split(d.Id(), ImportSeparator)
	if len(id) != 2 {
		return nil, fmt.Errorf("ID syntax should be \"Content Library name\".\"Content Library Item name\", where '.' is a customisable import separator")
	}

	// TODO: TM: Tenant Context should not be nil and depend on the configured owner_org_id
	cl, err := vcdClient.GetContentLibraryByName(id[0], nil)
	if err != nil {
		return nil, fmt.Errorf("error getting Content Library with name '%s' for import: %s", id[0], err)
	}

	cli, err := cl.GetContentLibraryItemByName(id[1])
	if err != nil {
		return nil, fmt.Errorf("error getting Content Library Item with name '%s': %s", id[1], err)
	}

	d.SetId(cli.ContentLibraryItem.ID)
	dSet(d, "content_library_id", cl.ContentLibrary.ID)
	return []*schema.ResourceData{d}, nil
}

func getContentLibraryItemType(_ *VCDClient, d *schema.ResourceData) (*types.ContentLibraryItem, error) {
	t := &types.ContentLibraryItem{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
	}

	return t, nil
}

func setContentLibraryItemData(_ *VCDClient, d *schema.ResourceData, cli *govcd.ContentLibraryItem) error {
	if cli == nil || cli.ContentLibraryItem == nil {
		return fmt.Errorf("cannot save state for nil Content Library Item")
	}

	dSet(d, "content_library_id", cli.ContentLibraryItem.ContentLibrary.ID)
	dSet(d, "name", cli.ContentLibraryItem.Name)
	dSet(d, "description", cli.ContentLibraryItem.Description)
	dSet(d, "creation_date", cli.ContentLibraryItem.CreationDate)
	dSet(d, "image_identifier", cli.ContentLibraryItem.ImageIdentifier)
	dSet(d, "is_published", cli.ContentLibraryItem.IsPublished)
	dSet(d, "is_subscribed", cli.ContentLibraryItem.IsSubscribed)
	dSet(d, "last_successful_sync", cli.ContentLibraryItem.LastSuccessfulSync)
	if cli.ContentLibraryItem.Org != nil {
		dSet(d, "owner_org_id", cli.ContentLibraryItem.Org.ID)
	}
	dSet(d, "status", cli.ContentLibraryItem.Status)
	dSet(d, "version", cli.ContentLibraryItem.Version)
	d.SetId(cli.ContentLibraryItem.ID)

	return nil
}
