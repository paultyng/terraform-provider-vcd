---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_external_network_v2"
sidebar_current: "docs-vcd-data-source-external-network-v2"
description: |-
  Provides a VMware Cloud Director External Network data source (version 2). New version of this data source
  uses new VCD API and is capable of creating NSX-T backed external networks as well as port group
  backed ones.
---

# vcd\_external\_network\_v2

Provides a VMware Cloud Director External Network data source (version 2). New version of this data source uses new VCD
API and is capable of handling NSX-T backed external networks as well as port group backed ones.

-> **Note:** This resource uses new VMware Cloud Director
[OpenAPI](https://code.vmware.com/docs/11982/getting-started-with-vmware-cloud-director-openapi) and
requires at least VCD *10.0+*. It supports both NSX-T and NSX-V backed networks (NSX-T *3.0+* requires VCD *10.1.1+*)

Supported in provider *v3.0+*.

## Example Usage

```hcl
data "vcd_external_network_v2" "ext_net" {
  name = "my-nsxt-net"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) external network name

## Attribute Reference

All properties defined in [vcd_external_network_v2](/docs/providers/vcd/r/external_network_v2.html)
resource are available.
