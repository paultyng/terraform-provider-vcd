package vcd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/vmware/go-vcloud-director/v2/types/v56"
)

func expandIPRange(configured []interface{}) types.IPRanges {
	ipRange := make([]*types.IPRange, 0, len(configured))

	for _, ipRaw := range configured {
		data := ipRaw.(map[string]interface{})

		ip := types.IPRange{
			StartAddress: data["start_address"].(string),
			EndAddress:   data["end_address"].(string),
		}

		ipRange = append(ipRange, &ip)
	}

	ipRanges := types.IPRanges{
		IPRange: ipRange,
	}

	return ipRanges
}

func expandFirewallRules(d *schema.ResourceData, gateway *types.EdgeGateway) ([]*types.FirewallRule, error) {
	//firewallRules := make([]*types.FirewallRule, 0, len(configured))
	firewallRules := gateway.Configuration.EdgeGatewayServiceConfiguration.FirewallService.FirewallRule

	rulesCount := d.Get("rule.#").(int)
	for i := 0; i < rulesCount; i++ {
		prefix := fmt.Sprintf("rule.%d", i)

		var protocol *types.FirewallRuleProtocols
		// Allow upper and lower case protocol names
		switch strings.ToLower(d.Get(prefix + ".protocol").(string)) {
		case "tcp":
			protocol = &types.FirewallRuleProtocols{
				TCP: true,
			}
		case "udp":
			protocol = &types.FirewallRuleProtocols{
				UDP: true,
			}
		case "icmp":
			protocol = &types.FirewallRuleProtocols{
				ICMP: true,
			}
		default:
			protocol = &types.FirewallRuleProtocols{
				Any: true,
			}
		}
		rule := &types.FirewallRule{
			//ID: strconv.Itoa(len(configured) - i),
			IsEnabled:            true,
			MatchOnTranslate:     false,
			Description:          d.Get(prefix + ".description").(string),
			Policy:               d.Get(prefix + ".policy").(string),
			Protocols:            protocol,
			Port:                 getNumericPort(d.Get(prefix + ".destination_port")),
			DestinationPortRange: d.Get(prefix + ".destination_port").(string),
			DestinationIP:        d.Get(prefix + ".destination_ip").(string),
			SourcePort:           getNumericPort(d.Get(prefix + ".source_port")),
			SourcePortRange:      d.Get(prefix + ".source_port").(string),
			SourceIP:             d.Get(prefix + ".source_ip").(string),
			EnableLogging:        false,
		}
		firewallRules = append(firewallRules, rule)
	}

	return firewallRules, nil
}

func getProtocol(protocol types.FirewallRuleProtocols) string {
	if protocol.TCP {
		return "tcp"
	}
	if protocol.UDP {
		return "udp"
	}
	if protocol.ICMP {
		return "icmp"
	}
	return "any"
}

func getNumericPort(portrange interface{}) int {
	i, err := strconv.Atoi(portrange.(string))
	if err != nil {
		return -1
	}
	return i
}

func getPortString(port int) string {
	if port == -1 {
		return "any"
	}
	portstring := strconv.Itoa(port)
	return portstring
}

func retryCall(seconds int, f resource.RetryFunc) error {
	return resource.Retry(time.Duration(seconds)*time.Second, f)
}

func convertToStringMap(param map[string]interface{}) map[string]string {
	temp := make(map[string]string)
	for k, v := range param {
		temp[k] = v.(string)
	}
	return temp
}

// convertSchemaSetToSliceOfStrings accepts Terraform's *schema.Set object and converts it to slice
// of strings.
// This is useful for extracting values from a set of strings
func convertSchemaSetToSliceOfStrings(param *schema.Set) []string {
	paramList := param.List()
	result := make([]string, len(paramList))
	for index, value := range paramList {
		result[index] = fmt.Sprint(value)
	}

	return result
}

func convertToTypeSet(param []string) []interface{} {
	slice := make([]interface{}, len(param))
	for index, value := range param {
		slice[index] = value
	}
	return slice
}
