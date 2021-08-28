---
layout: "vcd"
page_title: "VMware Cloud Director: data source filters"
sidebar_current: "docs-vcd-guides-filters"
description: |-
  Provides guidance on filters.
---


## Retrieving a data source by filter

Supported in provider *v2.9+*

Several data sources can be retrieved using a filter.
The criteria should be specific enough to retrieve a single item. The retrieval will fail if the criteria match more than
one item.

When you don't know the name, you may get the data source using the `filter` section, and using one or more of the following criteria:

* `name_regex` (Optional) matches the name using a regular expression.
* `date` (Optional) is an expression starting with an operator (`>`, `<`, `>=`, `<=`, `==`), followed by a date, with
  optional spaces in between. For example: `> 2020-02-01 12:35:00.523Z`
  The filter recognizes several formats, but one of `yyyy-mm-dd [hh:mm[:ss[.nnnZ]]]` or `dd-MMM-yyyy [hh:mm[:ss[.nnnZ]]]`
  is recommended. The time stamp can be defined down to microsecond level. One of the formats used in creation dates
  for vApp templates, catalogs, etc, is  `"YYYY-MM-DDThh:mm:ss.µµµZ"` (RFC3339)
  Comparison with equality operator (`==`) needs the date to be defined with microseconds precision.
* `latest` (Optional) If `true`, retrieve the latest item among the ones matching other parameters. If no other parameters
  are set, it retrieves the newest item.
* `earliest` (Optional) If `true`, retrieve the earliest item among the ones matching other parameters. If no other parameters
  are set, it retrieves the oldest item.
* `ip` (Optional) matches the IP of the resource using a regular expression.
* `metadata` (Optional) One or more parameters that will match metadata contents, as defined below

### Metadata filter arguments

* `key` (Required) The name of the metadata field
* `value` (Required) The value to look for. It is treated as a regular expression.
* `is_system` (Optional) If `true`, the metadata fields will be passed as `metadata@SYSTEM:fieldName`. This parameter
is needed when searching for metadata that was set by the system, such as the annotations in metadata when a vApp is
saved into a catalog. See Example 7 below.

### Multiple filter expressions

When a filter contains multiple clauses, you achieve the overall match only if all the clauses match. For example:

```hcl
# TF-UPGRADE-TODO: Block type was not recognized, so this block and its contents were not automatically upgraded.
filter {
  name_regex = "^p.*11$"
  date       = "> 2020-02-10"

  metadata {
    key   = "key1"
    value = "value1"
  }

  metadata {
    key   = "keyABC"
    value = "valueXYZ"
  }
}
```

This filter will retrieve the entity ONLY if ALL the conditions are true:

* The name starts with `p` and ends with `11`
* The entity was created after the 10th of February
* Both metadata fields were found with the requested values

### Availability of filters

Not all the data sources support filters, and when they do, they may not support all the search fields. For example,
an edge gateway only supports `name_regex`, while the `vcd_network_*` support name, IP, and metadata, and catalog related
objects support name, date, and metadata.

### Empty filter

An empty filter will retrieve all existing entities for the given parent, without restrictions. This idiom is **useful when
you know that there is only one such entity**, and you want to access it without knowing the name.
Like populated filters, when the search returns more than one item, the data source retrieval fails.

### About metadata search

Metadata can be searched even for those data sources that don't expose metadata in their interface. If the `filter`
section lists `metadata` among the available criteria, you can search the metadata and get the results accordingly,
although the vcd provider may not show the metadata for the found item.
Note that the names of the metadata fields are case-sensitive.

## Example filter 1

```hcl
data "vcd_catalog_item" "unknown" {

  org     = "datacloud"
  catalog = "cat-datacloud"

  filter {
    name_regex = "^p.*11$"
  }
}

output "filtered_item" {
  value = data.vcd_catalog_item.unknown
}
```

Will find a catalog item with name starting with `p` and ending with `11`.
It fails if there are several items named as requested, such as `photon-v11`, `platform911`, or `poorName211`
Note that regular expressions are case-sensitive: `photon-v11` and `Photon-v11` are two different entities.

## Example filter 2

```hcl
data "vcd_catalog_item" "unknown" {
  org     = "datacloud"
  catalog = "cat-datacloud"

  filter {
    name_regex = "^CentOS"
    latest     = true
  }
}
```

Will find the most recent item where the name starts by `CentOS`.

## Example filter 3

```hcl
data "vcd_catalog_item" "unknown" {
  org     = "datacloud"
  catalog = "cat-datacloud"

  filter {
    date   = ">= 2020-03-20"
    latest = true
  }
}

# Alternative date conditions for the same value:
# date = ">= March 23rd, 2020"
# date = ">= 22-Mar-2020"
# date = ">= 22-Mar-2020 00:00:00"
# date = ">= 2020-03-23 00:00:00"
```

Will find the most recent item created on or after March 23rd, 2020

## Example filter 4

```hcl
data "vcd_catalog_item" "unknown" {
  org     = "datacloud"
  catalog = "cat-datacloud"

  filter {
    metadata {
      key   = "Code"
      value = "BlackEagle"
    }
  }
}
```

Finds an item where the metadata contains a key `Code` with value `BlackEagle`.
Will fail if the criteria match more than one item.

## Example filter 5

```hcl
data "vcd_catalog_item" "unknown" {
  org     = "datacloud"
  catalog = "cat-datacloud"

  filter {
    metadata {
      key   = "ONE"
      value = "FirstValue" # explicit value
      type  = "STRING"
    }
    metadata {
      key   = "TWO"
      value = "SecondValue" # explicit value
      type  = "STRING"
    }
  }
}
```

When you use several `metadata` blocks, they must all match to have a filter match. This example finds an item where the
metadata contains a key `ONE` with value `FirstValue`, AND a key `TWO` with value `SecondValue`.
Will fail if the criteria match more than one item. Will also fail if only one of the two metadata fields was found.

## Example filter 6

```hcl
data "vcd_catalog_item" "unknown" {
  org     = "datacloud"
  catalog = "cat-datacloud"

  filter {
    metadata {
      key   = "ONE"
      value = "^First" # regular expression
    }
    metadata {
      key   = "TWO"
      value = "^S\\w+$" # regular expression
    }
  }
}
```

Will perform the same search of example 5, using regular expressions instead of exact values.

Note that the `value` is treated as a regular expression. For example:
`value = "cloud"` will match a metadata value `cloud`, but also one containing `on clouds` or `cloud9`.
To match only `cloud`, the value should be specified as `"^cloud$"`.

## Example filter 7

```hcl
data "vcd_catalog_item" "unknown" {
  org     = "datacloud"
  catalog = "cat-datacloud"

  filter {
    metadata {
      key       = "vapp.origin.type"
      value     = "com.vmware.vcloud.entity.vapp"
      is_system = true
    }
    metadata {
      key       = "vapp.origin.name"
      value     = "my_vapp_name"
      is_system = true
    }
  }
}
```

Will search a catalog item using SYSTEM metadata. In this example, it was a vApp that was converted to a template, and
got system metadata from vCD.

## Example filter 8

Several data sources with a quick search

```hcl
# Finds the oldest catalog created after April 2nd, 2020
data "vcd_catalog" "unknown_cat" {
  org = "datacloud"

  filter {
    date     = ">= 2020-04-02 10:00"
    earliest = "true"
  }
}
# Finds an isolated network with gateway IP starting with `192.168.3`
data "vcd_network_isolated" "unknown_net" {
  org = "datacloud"
  vdc = "vdc-datacloud"

  filter {
    ip = "^192.168.3"
  }
}

# Finds an edge gateway with name starting with `gw` and ending with `191`
data "vcd_edgegateway" "unknown_egw" {
  org = "datacloud"
  vdc = "vdc-datacloud"

  filter {
    name_regex = "^gw.+191"
  }
}

# Finds the newest media item created after March 1st, 2020
data "vcd_catalog_media" "unknown_cm" {
  org     = "datacloud"
  catalog = "cat-datacloud"

  filter {
    date   = "> 2020-03-01"
    latest = true
  }
}

# Finds an edge gateway when you know for sure that there is only one in your VDC
data "vcd_edgegateway" "only_egw" {
  org = "datacloud"
  vdc = "vdc-datacloud"

  filter {
  }
}
# You can achieve the same result using a generic regular expression
# such as
#   name_regex = ".*"

```
