#! /bin/bash

# ----------------------------------------------------------------------------------------------------------------------------------------------------------
# This script imports the existing app installations (required by aws observability solution) if app(s) are already installed in the user's Sumo Logic account.
# For SUMOLOGIC_ENV, provide one from the list : au, ca, ch, de, eu, esc, jp, us2, kr, fed or us1. For more information on Sumo Logic deployments visit https://help.sumologic.com/APIs/General-API-Information/Sumo-Logic-Endpoints-and-Firewall-Security"
# Before using this script, set following environment variables using below commands:
# export SUMOLOGIC_ENV=""
# export SUMOLOGIC_ACCESSID=""
# export SUMOLOGIC_ACCESSKEY=""
#-----------------------------------------------------------------------------------------------------------------------------------------------------------

# Validate Sumo Logic environment/deployment.

if ! [[ "$SUMOLOGIC_ENV" =~ ^(au|ca|ch|de|eu|esc|jp|us2|fed|kr|us1|stag)$ ]]; then
    echo "$SUMOLOGIC_ENV is invalid Sumo Logic deployment. For SUMOLOGIC_ENV, provide one from list : au, ca, ch, de, eu, esc, fed, jp, kr, us1, us2 or stag. For more information on Sumo Logic deployments visit https://help.sumologic.com/APIs/General-API-Information/Sumo-Logic-Endpoints-and-Firewall-Security"
    exit 1
fi

# Get Sumo Logic api endpoint based on SUMOLOGIC_ENV
if [ "${SUMOLOGIC_ENV}" == "us1" ]; then
    SUMOLOGIC_BASE_URL="https://api.sumologic.com/api/"
elif [ "${SUMOLOGIC_ENV}" == "stag" ]; then
    SUMOLOGIC_BASE_URL="https://stag-api.sumologic.net/api/"
else
    SUMOLOGIC_BASE_URL="https://api.${SUMOLOGIC_ENV}.sumologic.com/api/"
fi

# awso_apps_list contains apps required for AWS Observability Solution.
# Each entry is "uuid|name" matching the installation_apps_list in local.tf.
# Update the list if new apps are added to the solution.
declare -ra awso_apps_list=(
    "c26c3149-6131-4b97-b250-adf53ad7cd16|Amazon ECS(Without Container Insights and Traces)"
    "69c8db21-9d42-4210-8fa8-365155d90074|Amazon ECS(With Container Insights and Traces)"
    "1354f831-1745-447e-91b9-0dd6b32eebec|Amazon ElastiCache"
    "32c8b96c-161c-46d4-b81d-235cc0b56b87|Amazon Overview"
    "30519e43-1a92-482c-be08-deaaf09c88b6|Amazon RDS"
    "7590a039-5935-47ff-87ea-bc0970e6e96a|Amazon SNS"
    "dea50b70-dc1f-406d-884d-3945214861ae|Amazon SQS"
    "9b952ef9-7a20-44e4-861a-25f9124846ff|AWS API Gateway"
    "8ae2e0f4-cb4f-476b-ba9c-ee84bbab471b|AWS Application Load Balancer"
    "f113016e-4d15-4f4e-a254-4510a995f525|AWS Classic Load Balancer"
    "7d9b9d0b-0a1f-4af9-9edd-889c2190024b|AWS DynamoDB"
    "3dcaacb4-a5de-4e57-a477-fccd04f9e40f|AWS EC2"
    "a542409f-c491-404f-9a63-7078fcc945e2|AWS Lambda"
    "3f59d805-e2a8-4e0d-ae90-d40f0eb63671|AWS Network Load Balancer"
    "5ebe888f-43f0-4021-9225-fd5360e58ca4|Host Metrics (EC2)"
)

function get_app_instances() {
    local RESPONSE
    readonly RESPONSE="$(curl -XGET -s \
        -u "${SUMOLOGIC_ACCESSID}:${SUMOLOGIC_ACCESSKEY}" \
        "${SUMOLOGIC_BASE_URL}"v2/apps/instances)"

   echo "${RESPONSE}"
}

get_app_instances
INSTANCES_RESPONSE=$(get_app_instances)
outputVal=$?

if ! jq -e <<< "${INSTANCES_RESPONSE}" > /dev/null 2>&1; then
    printf "Failed requesting Apps instances API:\n%s\n" "${INSTANCES_RESPONSE}"
    # Credential Issue
    outputVal=2
elif ! jq -e '.data' <<< "${INSTANCES_RESPONSE}" > /dev/null 2>&1; then
    printf "Failed requesting Apps instances API:\n%s\n" "${INSTANCES_RESPONSE}"
    # Permissions/credential issues
    outputVal=3
fi

if [ $outputVal == 0 ]; then
    for ENTRY in "${awso_apps_list[@]}"; do
        APP_UUID="${ENTRY%%|*}"
        APP_NAME="${ENTRY##*|}"
        echo "$APP_NAME - $APP_UUID"

        INSTALLATION_ID=$(echo "${INSTANCES_RESPONSE}" | jq -r ".data[] | select(.uuid == \"${APP_UUID}\") | .id" | head -1)

        if [[ -z "${INSTALLATION_ID}" ]]; then
            # App not installed in Sumo org, skip importing
            continue
        fi

        # App installation exists in Sumo org, hence import
        terraform import \
            "module.app-module.sumologic_app.apps[\"${APP_NAME}\"]" "${INSTALLATION_ID}"
    done
elif [ $outputVal == 2 ]; then
    echo "Error in calling Sumo Logic Apps API."
    echo "User's credentials (SUMOLOGIC_ACCESSID and SUMOLOGIC_ACCESSKEY) are not valid."
elif [ $outputVal == 3 ]; then
    echo "Error in calling Sumo Logic Apps API. The reasons can be:"
    echo "1. Credentials could not be verified. Cross check SUMOLOGIC_ACCESSID and SUMOLOGIC_ACCESSKEY."
    echo "2. You do not have the role capabilities to manage Sumo Logic apps. Please see the Sumo Logic docs on role capabilities https://help.sumologic.com/Manage/Users-and-Roles/Manage-Roles/05-Role-Capabilities"
else
    echo "Error in calling Sumo Logic Apps API. The reasons can be:"
    echo "1. User's credentials (SUMOLOGIC_ACCESSID and SUMOLOGIC_ACCESSKEY) are not associated with SUMOLOGIC_ENV"
    echo "2. You do not have the role capabilities to manage Sumo Logic apps. Please see the Sumo Logic docs on role capabilities https://help.sumologic.com/Manage/Users-and-Roles/Manage-Roles/05-Role-Capabilities"
fi
