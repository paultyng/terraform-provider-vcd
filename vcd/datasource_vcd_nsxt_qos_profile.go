package vcd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func datasourceVcdNsxtQosProfile() *schema.Resource {
	return &schema.Resource{
		ReadContext: datasourceNsxtQosProfileRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Description of Segment QoS Profile",
			},
			"context_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "ID of VDC, VDC Group, or NSX-T Manager. Required if the VCD instance has more than one NSX-T manager",
			},

			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Description of Segment QoS Profile",
			},
			"class_of_service": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Groups similar types of traffic in the network and each type of traffic is treated as a class with its own level of service priority",
			},
			"dscp_priority": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Differentiated Services Code Point priority",
			},
			"dscp_trust_mode": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Differentiated Services Code Point trust mode",
			},
			"egress_rate_limiter_avg_bandwidth": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Average bandwidth in Mb/s",
			},
			"egress_rate_limiter_burst_size": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Burst size in bytes",
			},
			"egress_rate_limiter_peak_bandwidth": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Peak bandwidth in Mb/s",
			},
			"ingress_broadcast_rate_limiter_avg_bandwidth": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Average bandwidth in Mb/s",
			},
			"ingress_broadcast_rate_limiter_burst_size": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Burst size in bytes",
			},
			"ingress_broadcast_rate_limiter_peak_bandwidth": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Peak bandwidth in Mb/s",
			},
			"ingress_rate_limiter_avg_bandwidth": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Average bandwidth in Mb/s",
			},
			"ingress_rate_limiter_burst_size": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Burst size in bytes",
			},
			"ingress_rate_limiter_peak_bandwidth": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Peak bandwidth in Mb/s",
			},
		},
	}
}

func datasourceNsxtQosProfileRead(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	vcdClient := meta.(*VCDClient)
	profileName := d.Get("name").(string)
	contextUrn := d.Get("context_id").(string)

	contextFilterField, err := getContextFilterField(contextUrn)
	if err != nil {
		return diag.FromErr(err)
	}

	queryFilter := url.Values{}
	queryFilter.Add("filter", fmt.Sprintf("%s==%s", contextFilterField, contextUrn))

	qosProfile, err := vcdClient.GetQoSProfileByName(profileName, queryFilter)
	if err != nil {
		return diag.Errorf("could not find QoS Profile by name '%s': %s", profileName, err)
	}

	dSet(d, "description", qosProfile.Description)
	dSet(d, "class_of_service", qosProfile.ClassOfService)
	dSet(d, "dscp_priority", qosProfile.DscpConfig.Priority)
	dSet(d, "dscp_trust_mode", qosProfile.DscpConfig.TrustMode)
	dSet(d, "egress_rate_limiter_avg_bandwidth", qosProfile.EgressRateLimiter.AvgBandwidth)
	dSet(d, "egress_rate_limiter_burst_size", qosProfile.EgressRateLimiter.BurstSize)
	dSet(d, "egress_rate_limiter_peak_bandwidth", qosProfile.EgressRateLimiter.PeakBandwidth)
	dSet(d, "ingress_broadcast_rate_limiter_avg_bandwidth", qosProfile.IngressBroadcastRateLimiter.AvgBandwidth)
	dSet(d, "ingress_broadcast_rate_limiter_burst_size", qosProfile.IngressBroadcastRateLimiter.BurstSize)
	dSet(d, "ingress_broadcast_rate_limiter_peak_bandwidth", qosProfile.IngressBroadcastRateLimiter.PeakBandwidth)
	dSet(d, "ingress_rate_limiter_avg_bandwidth", qosProfile.IngressRateLimiter.AvgBandwidth)
	dSet(d, "ingress_rate_limiter_burst_size", qosProfile.IngressRateLimiter.BurstSize)
	dSet(d, "ingress_rate_limiter_peak_bandwidth", qosProfile.IngressRateLimiter.PeakBandwidth)

	d.SetId(qosProfile.ID)

	return nil
}
