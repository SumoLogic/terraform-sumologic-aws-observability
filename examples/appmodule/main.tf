#
# The below module installs Sumo Logic apps into the "Installed Apps" catalog,
# creates the AWS Observability Entity Inspector hierarchy, and manages FERs and fields.
# NOTE - The "app-modules" should be installed per Sumo Logic organization.
#
module "sumo-module" {
  source                 = "../../modules/apps"
  sumologic_access_id    = var.sumologic_access_id
  sumologic_access_key   = var.sumologic_access_key
  sumologic_environment  = var.sumologic_environment
}
