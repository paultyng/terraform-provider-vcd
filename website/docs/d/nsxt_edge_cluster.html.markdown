---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_nsxt_edge_cluster"
sidebar_current: "docs-vcd-data-source-nsxt-edge-cluster"
description: |-
  Provides a data source for available NSX-T Edge Clusters.
---

# vcd\_nsxt\_edge\_cluster

Provides a data source for available NSX-T Edge Clusters.

Supported in provider *v3.1+*

-> **Note:** This data source uses new VMware Cloud Director
[OpenAPI](https://code.vmware.com/docs/11982/getting-started-with-vmware-cloud-director-openapi) and
requires at least VCD *10.1.1+* and NSX-T *3.0+*.

## Example Usage 

```hcl
data "vcd_nsxt_edge_cluster" "first" {
  org  = "my-org"
  vdc  = "my-vdc"
  name = "edge-cluster-one"
}
```


## Argument Reference

The following arguments are supported:

* `org` - (Optional) The name of organization to which edge cluster belongs. Optional if defined at provider level.
* `vdc` - (Optional) The name of VDC that owns the edge cluster. Optional if defined at provider level.
* `name` - (Required) NSX-T Edge Cluster name. **Note.** NSX-T does allow to have duplicate names therefore to be able
to correctly use this data source there should not be multiple NSX-T Edge Clusters with the same name defined.

## Attribute reference

* `description` - Edge Cluster description in NSX-T manager.
* `node_count` - Number of nodes in Edge Cluster.
* `node_type` - Type of nodes in Edge Cluster.
* `deployment_type` - Deployment type of Edge Cluster.
