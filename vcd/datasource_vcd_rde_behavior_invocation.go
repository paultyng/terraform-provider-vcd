package vcd

import (
	"context"
	"encoding/json"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/vmware/go-vcloud-director/v2/types/v56"
)

// This data source is unusual, as it represents an imperative call to a function (aka invoking a behavior)
// in the declarative world of Terraform. Despite being a data source, whose goal is to perform read-only operations, invocations
// can mutate RDE contents. The nature of this one is similar to the built-in "http" provider, which has the "http_http" data source
// that can perform "PUT"/"POST"/"DELETE" operations.
func datasourceVcdRdeBehaviorInvocation() *schema.Resource {
	return &schema.Resource{
		ReadContext: datasourceVcdRdeBehaviorInvocationRead,
		Schema: map[string]*schema.Schema{
			"rde_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID of the RDE for which the Behavior will be invoked",
			},
			"behavior_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID of either a RDE Interface Behavior or RDE Type Behavior to be invoked",
			},
			"invoke_on_refresh": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "If 'true', invokes the Behavior on every refresh",
			},
			"arguments": {
				Type:         schema.TypeMap,
				Optional:     true,
				Description:  "The arguments to be passed to the invoked Behavior",
				Deprecated:   "Use 'arguments_json' instead",
				ExactlyOneOf: []string{"arguments", "arguments_json"},
			},
			"arguments_json": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The arguments to be passed to the invoked Behavior, as a JSON string",
				ExactlyOneOf: []string{"arguments", "arguments_json"},
			},
			"metadata": {
				Type:         schema.TypeMap,
				Optional:     true,
				Description:  "Metadata to be passed to the invoked Behavior",
				Deprecated:   "Use 'metadata_json' instead",
				ExactlyOneOf: []string{"metadata", "metadata_json"},
			},
			"metadata_json": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "Metadata to be passed to the invoked Behavior, as a JSON string",
				ExactlyOneOf: []string{"metadata", "metadata_json"},
			},
			"result": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The raw result of the Behavior invocation",
			},
		},
	}
}

func datasourceVcdRdeBehaviorInvocationRead(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vcdClient := meta.(*VCDClient)
	rdeId := d.Get("rde_id").(string)
	behaviorId := d.Get("behavior_id").(string)

	if d.Get("invoke_on_refresh").(bool) {
		rde, err := vcdClient.GetRdeById(rdeId)
		if err != nil {
			return diag.Errorf("[RDE Behavior Invocation] could not retrieve the RDE with ID '%s': %s", rdeId, err)
		}

		var arguments map[string]interface{}
		var metadata map[string]interface{}
		if a, ok := d.GetOk("arguments"); ok {
			arguments = a.(map[string]interface{})
		} else {
			err = json.Unmarshal([]byte(d.Get("arguments_json").(string)), &arguments)
			if err != nil {
				return diag.Errorf("[RDE Behavior Invocation] could not read the arguments JSON: %s", err)
			}
		}
		if m, ok := d.GetOk("metadata"); ok {
			metadata = m.(map[string]interface{})
		} else {
			err = json.Unmarshal([]byte(d.Get("metadata_json").(string)), &metadata)
			if err != nil {
				return diag.Errorf("[RDE Behavior Invocation] could not read the metadata JSON: %s", err)
			}
		}
		result, err := rde.InvokeBehavior(behaviorId, types.BehaviorInvocation{
			Arguments: arguments,
			Metadata:  metadata,
		})
		if err != nil {
			return diag.Errorf("[RDE Behavior Invocation] could not invoke the Behavior of the RDE with ID '%s': %s", rdeId, err)
		}
		dSet(d, "result", result)
	}
	d.SetId(rdeId + "|" + behaviorId) // Invocations are not real entities, so we make an artificial ID.
	return nil
}
