## Requirements

| Name | Version |
| ---- | ------- |
| <a name="requirement_terraform"></a> [terraform](#requirement\_terraform) | >= 1.5.7 |
| <a name="requirement_sumologic"></a> [sumologic](#requirement\_sumologic) | >= 3.3.0, < 4.0.0 |
| <a name="requirement_time"></a> [time](#requirement\_time) | >= 0.11.1 |

## Providers

| Name | Version |
| ---- | ------- |
| <a name="provider_sumologic"></a> [sumologic](#provider\_sumologic) | >= 3.3.0, < 4.0.0 |

## Modules

No modules.

## Resources

| Name | Type |
| ---- | ---- |
| [sumologic_app.amazon_overview](https://registry.terraform.io/providers/SumoLogic/sumologic/latest/docs/resources/app) | resource |
| [sumologic_app.apps](https://registry.terraform.io/providers/SumoLogic/sumologic/latest/docs/resources/app) | resource |
| [sumologic_hierarchy.awso_hierarchy](https://registry.terraform.io/providers/SumoLogic/sumologic/latest/docs/resources/hierarchy) | resource |

## Inputs

| Name | Description | Type | Default | Required |
| ---- | ----------- | ---- | ------- | :------: |
| <a name="input_installation_apps_list"></a> [installation\_apps\_list](#input\_installation\_apps\_list) | List of Sumo Logic apps to be installed. Each app can have custom parameters specific to that app. | <pre>list(object({<br/>    uuid       = string<br/>    name       = string<br/>    version    = string<br/>    parameters = optional(map(string), {})<br/>  }))</pre> | `[]` | no |
| <a name="input_sumologic_access_id"></a> [sumologic\_access\_id](#input\_sumologic\_access\_id) | Sumo Logic Access ID. Visit https://help.sumologic.com/Manage/Security/Access-Keys#Create_an_access_key | `string` | n/a | yes |
| <a name="input_sumologic_access_key"></a> [sumologic\_access\_key](#input\_sumologic\_access\_key) | Sumo Logic Access Key. Visit https://help.sumologic.com/Manage/Security/Access-Keys#Create_an_access_key | `string` | n/a | yes |
| <a name="input_sumologic_environment"></a> [sumologic\_environment](#input\_sumologic\_environment) | Enter au, ca, de, eu, jp, us2, kr, fed ch or us1. For more information on Sumo Logic deployments visit https://help.sumologic.com/APIs/General-API-Information/Sumo-Logic-Endpoints-and-Firewall-Security | `string` | n/a | yes |
| <a name="input_sumologic_environment_base_url"></a> [sumologic\_environment\_base\_url](#input\_sumologic\_environment\_base\_url) | Base URL for custom Sumo Logic environments (e.g., 'https://api.ch.sumologic.com/api/' for Switzerland). If provided, this takes precedence over the sumologic\_environment parameter. Leave empty for standard deployments. | `string` | `null` | no |

## Outputs

| Name | Description |
| ---- | ----------- |
| <a name="output_hierarchy_id"></a> [hierarchy\_id](#output\_hierarchy\_id) | ID of the AWS Observability Entity Inspector hierarchy |
| <a name="output_installed_apps"></a> [installed\_apps](#output\_installed\_apps) | Information about installed Sumo Logic apps |
