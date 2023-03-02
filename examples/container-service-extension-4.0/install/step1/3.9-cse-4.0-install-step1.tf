# ------------------------------------------------------------------------------------------------------------
# CSE 4.0 installation, step 1:
#
# * Please read the guide present at https://registry.terraform.io/providers/vmware/vcd/latest/docs/guides/container_service_extension_4_0
#   before applying this configuration.
#
# * The installation process is split into two steps as Providers will need to generate an API token for the created
#   CSE administrator user, in order to use it with the CSE Server that will be deployed in the second step.
#
# * This step will only create the required Runtime Defined Entity (RDE) Interfaces, Types, Role and finally
#   the CSE administrator user.
#
# * Rename "terraform.tfvars.example" to "terraform.tfvars" and adapt the values to your needs.
#   Other than that, this snippet should be applied as it is.
#   You can check the comments on each resource/data source for more context.
# ------------------------------------------------------------------------------------------------------------

# VCD Provider configuration. It must be at least v3.9.0 and configured with a System administrator account.
terraform {
  required_providers {
    vcd = {
      source  = "vmware/vcd"
      version = ">= 3.9"
    }
  }
}

provider "vcd" {
  url                  = "${var.vcd_url}/api"
  user                 = var.administrator_user
  password             = var.administrator_password
  auth_type            = "integrated"
  sysorg               = var.administrator_org
  org                  = var.administrator_org
  allow_unverified_ssl = var.insecure_login
  logging              = true
  logging_file         = "cse_install_step1.log"
}

# This is the interface required to create the "VCDKEConfig" Runtime Defined Entity Type.
resource "vcd_rde_interface" "vcdkeconfig_interface" {
  name    = "VCDKEConfig"
  version = "1.0.0"
  vendor  = "vmware"
  nss     = "VCDKEConfig"
}

# This resource will manage the "VCDKEConfig" RDE Type required to instantiate the CSE Server configuration.
# The schema URL points to the JSON schema hosted in the terraform-provider-vcd repository.
resource "vcd_rde_type" "vcdkeconfig_type" {
  name          = "VCD-KE RDE Schema"
  nss           = "VCDKEConfig"
  version       = "1.0.0"
  schema_url    = "https://raw.githubusercontent.com/adambarreiro/terraform-provider-vcd/add-cse40-guide/examples/container-service-extension-4.0/schemas/vcdkeconfig-type-schema.json"
  vendor        = "vmware"
  interface_ids = [vcd_rde_interface.vcdkeconfig_interface.id]
}

# This RDE Interface exists in VCD, so it must be fetched with a RDE Interface data source. This RDE Interface is used to be
# able to create the "capvcdCluster" RDE Type.
data "vcd_rde_interface" "kubernetes_interface" {
  vendor  = "vmware"
  nss     = "k8s"
  version = "1.0.0"
}

# This RDE Interface will create the "capvcdCluster" RDE Type required to create Kubernetes clusters.
# The schema URL points to the JSON schema hosted in the terraform-provider-vcd repository.
resource "vcd_rde_type" "capvcd_cluster_type" {
  name          = "CAPVCD Cluster"
  nss           = "capvcdCluster"
  version       = "1.1.0"
  schema_url    = "https://raw.githubusercontent.com/adambarreiro/terraform-provider-vcd/add-cse40-guide/examples/container-service-extension-4.0/schemas/capvcd-type-schema.json"
  vendor        = "vmware"
  interface_ids = [data.vcd_rde_interface.kubernetes_interface.id]
}

# This role is having only the minimum set of rights required for the CSE Server to function.
# It is created in the "System" provider organization scope.
resource "vcd_role" "cse_admin_role" {
  org         = "System"
  name        = "CSE Admin Role"
  description = "Used for administrative purposes"
  rights = [
    "API Tokens: Manage",
    "${vcd_rde_type.vcdkeconfig_type.vendor}:${vcd_rde_type.vcdkeconfig_type.nss}: Administrator Full access",
    "${vcd_rde_type.vcdkeconfig_type.vendor}:${vcd_rde_type.vcdkeconfig_type.nss}: Administrator View",
    "${vcd_rde_type.vcdkeconfig_type.vendor}:${vcd_rde_type.vcdkeconfig_type.nss}: Full Access",
    "${vcd_rde_type.vcdkeconfig_type.vendor}:${vcd_rde_type.vcdkeconfig_type.nss}: Modify",
    "${vcd_rde_type.vcdkeconfig_type.vendor}:${vcd_rde_type.vcdkeconfig_type.nss}: View",
    "${vcd_rde_type.capvcd_cluster_type.vendor}:${vcd_rde_type.capvcd_cluster_type.nss}: Administrator Full access",
    "${vcd_rde_type.capvcd_cluster_type.vendor}:${vcd_rde_type.capvcd_cluster_type.nss}: Administrator View",
    "${vcd_rde_type.capvcd_cluster_type.vendor}:${vcd_rde_type.capvcd_cluster_type.nss}: Full Access",
    "${vcd_rde_type.capvcd_cluster_type.vendor}:${vcd_rde_type.capvcd_cluster_type.nss}: Modify",
    "${vcd_rde_type.capvcd_cluster_type.vendor}:${vcd_rde_type.capvcd_cluster_type.nss}: View"
  ]
}

# This will allow to have a user with a limited set of rights that can access the Provider area of VCD.
# This user will be used by the CSE Server, with an API token that must be created afterwards.
resource "vcd_org_user" "cse_admin" {
  org      = "System"
  name     = var.cse_admin_user
  password = var.cse_admin_password
  role     = vcd_role.cse_admin_role.name
}

output "cse_admin_username" {
  value = "Please create an API token for ${vcd_org_user.cse_admin.name} as it will be required for step 2"
}