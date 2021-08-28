---
layout: "vcd"
page_title: "VMware Cloud Director: vcd_lb_app_rule"
sidebar_current: "docs-vcd-resource-lb-app-rule"
description: |-
  Provides an NSX edge gateway load balancer application rule resource.
---

# vcd\_lb\_app\_rule

Provides a VMware Cloud Director Edge Gateway Load Balancer Application Rule resource. An application rule
allows to directly manipulate and manage IP application traffic with load balancer.

~> **Note:** To make load balancing work one must ensure that load balancing is enabled on edge
gateway (edge gateway must be advanced).
This depends on NSX version to work properly. Please refer to [VMware Product Interoperability
Matrices](https://www.vmware.com/resources/compatibility/sim/interop_matrix.php#interop&29=&93=) 
to check supported VMware Cloud Director and NSX for vSphere configurations.

~> **Note:** The VMware Cloud Director API for NSX supports a subset of the operations and objects defined
in the NSX vSphere API Guide. The API supports NSX 6.2, 6.3, and 6.4.

Supported in provider *v2.4+*

## Example Usage 1 (Application rule with single line script)

```hcl
resource "vcd_lb_app_rule" "example-one" {
  edge_gateway = "my-edge-gw"
  org          = "my-org"
  vdc          = "my-org-vdc"

  name   = "script1"
  script = "acl vmware_page url_beg / vmware redirect location https://www.vmware.com/ ifvmware_page"
}
```

## Example Usage 2 (Application rule with multi line script)

```hcl
resource "vcd_lb_app_rule" "example-two" {
  edge_gateway = "my-edge-gw"
  org          = "my-org"
  vdc          = "my-org-vdc"
  name         = "script1"
  script       = <<-EOT
    acl vmware_page url_beg / vmware redirect location https://www.vmware.com/ ifvmware_page
    acl other_page2 url_beg / other2 redirect location https://www.other2.com/ ifother_page2
    acl hello payload(0,6) -m bin 48656c6c6f0a
EOT

}
```

## Argument Reference

The following arguments are supported:

* `org` - (Optional) The name of organization to use, optional if defined at provider level. Useful when connected as sysadmin working across different organisations
* `vdc` - (Optional) The name of VDC to use, optional if defined at provider level
* `edge_gateway` - (Required) The name of the edge gateway on which the application rule is to be created
* `name` - (Required) Application rule name
* `script` - (Required) A multiline application rule script.
Terraform's [HEREDOC syntax](https://www.terraform.io/docs/configuration/expressions.html#string-literals)
may be useful for multiline scripts. **Note:** For information on
the application rule syntax, see more in [VMware Cloud Director documentation]
(https://docs.vmware.com/en/vCloud-Director/9.7/com.vmware.vcloud.tenantportal.doc/GUID-AFF9F70F-85C9-4053-BA69-F2B062F34C7F.html)

## Attribute Reference

The following attributes are exported on this resource:

* `id` - The NSX ID of the load balancer application rule

## Importing

~> **Note:** The current implementation of Terraform import can only import resources into the state.
It does not generate configuration. [More information.](https://www.terraform.io/docs/import/)

An existing load balancer application rule can be [imported][docs-import] into this resource
via supplying the full dot separated path for load balancer application rule. An example is
below:

[docs-import]: https://www.terraform.io/docs/import/

```
terraform import vcd_lb_app_rule.imported my-org.my-org-vdc.my-edge-gw.my-lb-app-rule
```

The above would import the application rule named `my-lb-app-rule` that is defined on edge
gateway `my-edge-gw` which is configured in organization named `my-org` and vDC named `my-org-vdc`.
