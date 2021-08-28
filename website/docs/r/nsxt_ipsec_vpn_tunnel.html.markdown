---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_nsxt_ipsec_vpn_tunnel"
sidebar_current: "docs-vcd-resource-nsxt-ipsec-vpn-tunnel"
description: |-
  Provides a resource to manage NSX-T IPsec VPN Tunnel. You can configure site-to-site connectivity between an NSX-T Data
  Center Edge Gateway and remote sites. The remote sites must use NSX-T Data Center, have third-party hardware routers, 
  or VPN gateways that support IPSec.
---

# vcd\_nsxt\_ipsec\_vpn\_tunnel

Provides a resource to manage NSX-T IPsec VPN Tunnel. You can configure site-to-site connectivity between an NSX-T Data
Center Edge Gateway and remote sites. The remote sites must use NSX-T Data Center, have third-party hardware routers,
or VPN gateways that support IPSec.

## Example Usage (IPsec VPN Tunnel with default Security Profile)

```hcl
resource "vcd_nsxt_ipsec_vpn_tunnel" "tunnel1" {
  org = "my-org"
  vdc = "my-nsxt-vdc"

  edge_gateway_id = data.vcd_nsxt_edgegateway.existing.id

  name        = "First"
  description = "testing tunnel"

  pre_shared_key = "my-presharaed-key"
  # Primary IP address of Edge Gateway pulled from data source
  local_ip_address = tolist(data.vcd_nsxt_edgegateway.existing_gw.subnet)[0].primary_ip
  local_networks   = ["10.10.10.0/24", "30.30.30.0/28", "40.40.40.1/32"]
  # That is a fake remote IP address
  remote_ip_address = "1.2.3.4"
  remote_networks   = ["192.168.1.0/24", "192.168.10.0/24", "192.168.20.0/28"]
}
```

## Example Usage (IPsec VPN Tunnel with customized Security Profile)

```hcl
resource "vcd_nsxt_ipsec_vpn_tunnel" "tunnel1" {
  org = "my-org"
  vdc = "my-nsxt-vdc"

  edge_gateway_id = data.vcd_nsxt_edgegateway.existing.id

  name        = "customized-sec-profile"
  description = "IPsec VPN Tunnel with customized security profile"

  pre_shared_key = "test-psk"
  # Primary IP address of Edge Gateway
  local_ip_address = tolist(data.vcd_nsxt_edgegateway.existing_gw.subnet)[0].primary_ip
  local_networks   = ["10.10.10.0/24", "30.30.30.0/28", "40.40.40.1/32"]
  # That is a fake remote IP address as there is nothing else to peer to
  remote_ip_address = "1.2.3.4"
  remote_networks   = ["192.168.1.0/24", "192.168.10.0/24", "192.168.20.0/28"]

  security_profile_customization {
    ike_version               = "IKE_V2"
    ike_encryption_algorithms = ["AES_128"]
    ike_digest_algorithms     = ["SHA2_256"]
    ike_dh_groups             = ["GROUP14"]
    ike_sa_lifetime           = 86400

    tunnel_pfs_enabled           = true
    tunnel_df_policy             = "COPY"
    tunnel_encryption_algorithms = ["AES_256"]
    tunnel_digest_algorithms     = ["SHA2_256"]
    tunnel_dh_groups             = ["GROUP14"]
    tunnel_sa_lifetime           = 3600

    dpd_probe_internal = "30"
  }
}
```

## Argument Reference

The following arguments are supported:

* `org` - (Optional) The name of organization to use, optional if defined at provider level. Useful
  when connected as sysadmin working across different organisations.
* `vdc` - (Optional) The name of VDC to use, optional if defined at provider level.
* `edge_gateway_id` - (Required) The ID of the edge gateway (NSX-T only). Can be looked up using
  `vcd_nsxt_edgegateway` data source
* `name` - (Required) A name for NSX-T IPsec VPN Tunnel
* `description` - (Optional) An optional description of the NSX-T IPsec VPN Tunnel
* `enabled` - (Optional) Enables or disables IPsec VPN Tunnel (default `true`)
* `pre_shared_key` - (Required) Pre-shared key for negotiation. **Note** the pre-shared key must be the same on the 
other end of the IPSec VPN tunnel.
* `local_ip_address` - (Required) IPv4 Address for the endpoint. This has to be a suballocated IP on the Edge Gateway.
* `local_networks` - (Required) A set of local networks in CIDR format. At least one value required
* `remote_ip_address` - (Required) Public IPv4 Address of the remote device terminating the VPN connection
* `remote_networks` - (Optional) Set of remote networks in CIDR format. Leaving it empty is interpreted as 0.0.0.0/0
* `logging` - (Optional) Sets whether logging for the tunnel is enabled or not. (default - `false`)
* `security_profile_customization` - (Optional) a block allowing to
[customize default security profile](#security-profile) parameters

<a id="security-profile"></a>
## Security Profile customization
* `ike_version` - (Required) One of `IKE_V1`, `IKE_V2`, `IKE_FLEX`
* `ike_encryption_algorithms` - (Required) Encryption algorithms One of `AES_128`, `AES_256`, `AES_GCM_128`, `AES_GCM_192`, 
  `AES_GCM_256`
* `ike_digest_algorithms` - (Required) Secure hashing algorithms to use during the IKE negotiation. One of `SHA1`,
  `SHA2_256`, `SHA2_384`, `SHA2_512`
* `ike_dh_groups` - (Required) Diffie-Hellman groups to be used if Perfect Forward Secrecy is enabled. One of
  `GROUP2`, `GROUP5`, `GROUP14`, `GROUP15`, `GROUP16`, `GROUP19`, `GROUP20`, `GROUP21`
* `ike_sa_lifetime` - (Required) Security association lifetime in seconds. It is number of seconds before the IPsec 
  tunnel needs to reestablish
* `tunnel_pfs_enabled` - (Required) PFS (Perfect Forward Secrecy) enabled or disabled.
* `tunnel_df_policy` - (Required) Policy for handling defragmentation bit. One of COPY, CLEAR
* `tunnel_encryption_algorithms` - (Required) Encryption algorithms to use in IPSec tunnel establishment. 
  One of `AES_128`, `AES_256`, `AES_GCM_128`, `AES_GCM_192`, `AES_GCM_256`, `NO_ENCRYPTION_AUTH_AES_GMAC_128`,
  `NO_ENCRYPTION_AUTH_AES_GMAC_192`, `NO_ENCRYPTION_AUTH_AES_GMAC_256`, `NO_ENCRYPTION`
* `tunnel_digest_algorithms` - (Required) Digest algorithms to be used for message digest. 
  One of `SHA1`, `SHA2_256`, `SHA2_384`, `SHA2_512`
* `tunnel_dh_groups` - (Required) Diffie-Hellman groups to be used is PFS is enabled. 
  One of `GROUP2`, `GROUP5`, `GROUP14`, `GROUP15`, `GROUP16`, `GROUP19`, `GROUP20`, `GROUP21`
* `tunnel_sa_lifetime` - (Required) Security Association life time in seconds 
* `dpd_probe_internal` - (Required) Value in seconds of dead probe detection interval. Minimum is 3 seconds and the
  maximum is 60 seconds


## Attribute Reference
* `security_profile` - `DEFAULT` for system provided configuration or `CUSTOM` if `security_profile_customization` is set
* `status` - Overall IPsec VPN Tunnel Status
* `ike_service_status` - Status for the actual IKE Session for the given tunnel
* `ike_fail_reason` - Provides more details of failure if the IKE service is not UP


-> Status related fields might not immediatelly show up. It depends on when NSX-T updates its status

## Importing

~> The current implementation of Terraform import can only import resources into the state.
It does not generate configuration. [More information.](https://www.terraform.io/docs/import/)

An existing IPSec VPN Tunnel configuration can be [imported][docs-import] into this resource
via supplying the full dot separated path for your IPsec VPN Tunnel name or ID. An example is
below:

[docs-import]: https://www.terraform.io/docs/import/

Supplying Name
```
terraform import vcd_nsxt_ipsec_vpn_tunnel.imported my-org.my-org-vdc.my-nsxt-edge-gateway.my-ipsec-vpn-tunnel-name
```



-> When there are multiple IPsec VPN Tunnels with the same name they will all be listed so that one can pick
it by ID

```
$ terraform import vcd_nsxt_ipsec_vpn_tunnel.first my-org.nsxt-vdc.nsxt-gw.tunnel1
vcd_nsxt_nat_rule.dnat: Importing from ID "my-org.nsxt-vdc.nsxt-gw.dnat1"...
# The following IPsec VPN Tunnels with Name 'tunnel1' are available
# Please use ID instead of Name in import path to pick exact rule
ID                                   Name    Local IP     Remote IP
04fde766-2cbd-4986-93bb-7f57e59c6b19 tunnel1 1.1.1.1      2.2.2.2
f40e3d68-cfa6-42ea-83ed-5571659b3e7b tunnel1 4.4.4.4      8.8.8.8
$ terraform import vcd_nsxt_ipsec_vpn_tunnel.imported my-org.my-org-vdc.my-nsxt-edge-gateway.04fde766-2cbd-4986-93bb-7f57e59c6b19
```

The above would import the `my-ipsec-vpn-tunnel-name` IPsec VPN Tunne config settings that are defined
on NSX-T Edge Gateway `my-nsxt-edge-gateway` which is configured in organization named `my-org` and
VDC named `my-org-vdc`.
