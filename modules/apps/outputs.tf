output "installed_apps" {
  value = { for k, v in sumologic_app.apps : k => {
    uuid = v.uuid
    name = k
    id   = v.id
  } }
  description = "Information about installed Sumo Logic apps"
}

output "hierarchy_id" {
  value       = sumologic_hierarchy.awso_hierarchy.id
  description = "ID of the AWS Observability Entity Inspector hierarchy"
}
