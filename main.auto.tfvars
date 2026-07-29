####### BELOW ARE REQUIRED PARAMETERS FOR TERRAFORM SCRIPT #######
# Visit - https://help.sumologic.com/Solutions/AWS_Observability_Solution/03_Set_Up_the_AWS_Observability_Solution#sumo-logic-access-configuration-required
sumologic_environment     = ""                                                             # Please replace <YOUR SUMO DEPLOYMENT> (including brackets) with au, ca, ch, de, eu, esc, jp, us2, kr, fed or us1.
sumologic_access_id       = ""                                                   # Please replace <YOUR SUMO ACCESS ID> (including brackets) with your Sumo Logic Access ID.
sumologic_access_key      = "" # Please replace <YOUR SUMO ACCESS KEY> (including brackets) with your Sumo Logic Access KEY.
sumologic_organization_id = ""                                                 # Please replace <YOUR SUMO ORG ID> (including brackets) with your Sumo Logic Organization ID.
aws_account_alias         = ""
# sumologic_environment_base_url = "" # Uncomment and set only for non-standard deployments (e.g. stag). Leave commented for standard environments (eu, us2, etc.).