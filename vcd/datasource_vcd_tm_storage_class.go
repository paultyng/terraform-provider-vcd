package vcd

import (
	"context"
	"fmt"
	"github.com/vmware/go-vcloud-director/v3/types/v56"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const labelTmStorageClass = "Storage Class"

func datasourceVcdTmStorageClass() *schema.Resource {
	return &schema.Resource{
		ReadContext: datasourceVcdTmStorageClassRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: fmt.Sprintf("%s name", labelTmStorageClass),
			},
			"region_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: fmt.Sprintf("The Region that this %s belongs to", labelTmStorageClass),
			},
			"storage_capacity_mib": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: fmt.Sprintf("The total storage capacity of the %s in mebibytes", labelTmStorageClass),
			},
			"storage_consumed_mib": {
				Type:     schema.TypeInt,
				Computed: true,
				Description: fmt.Sprintf("For tenants, this represents the total storage given to all namespaces consuming from this %s in mebibytes. "+
					"For providers, this represents the total storage given to tenants from this %s in mebibytes.", labelTmStorageClass, labelTmStorageClass),
			},
			"zone_ids": {
				Type:        schema.TypeSet,
				Computed:    true,
				Description: fmt.Sprintf("A set with all the IDs of the zones available to the %s", labelTmStorageClass),
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func datasourceVcdTmStorageClassRead(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vcdClient := meta.(*VCDClient)
	regionId := d.Get("region_id").(string)
	region, err := vcdClient.GetRegionById(regionId)
	if err != nil {
		return diag.Errorf("error retrieving Region with ID '%s': %s", regionId, err)
	}

	scName := d.Get("name").(string)
	sc, err := region.GetStorageClassByName(scName)
	if err != nil {
		return diag.Errorf("error retrieving Storage Class '%s': %s", scName, err)
	}

	err = setStorageClassData(d, sc.StorageClass)
	if err != nil {
		return diag.Errorf("error saving Storage Class data into state: %s", err)
	}

	d.SetId(sc.StorageClass.ID)
	return nil
}

func setStorageClassData(d *schema.ResourceData, sc *types.StorageClass) error {
	dSet(d, "name", sc.Name)
	dSet(d, "storage_capacity_mib", sc.StorageCapacityMiB)
	dSet(d, "storage_consumed_mib", sc.StorageConsumedMiB)
	regionId := ""
	if sc.Region != nil {
		regionId = sc.Region.ID
	}
	dSet(d, "region_id", regionId)

	var zoneIds []string
	if len(sc.Zones) > 0 {
		zoneIds = extractIdsFromOpenApiReferences(sc.Zones)
	}
	err := d.Set("zone_ids", zoneIds)
	if err != nil {
		return err
	}

	return nil
}
