#!/bin/bash
# Import pre-existing Sumo Logic fields into Terraform state.
# Run this before `terraform apply` on orgs that already have fields
# (e.g., us2 production). Safe to run multiple times — skips fields
# already in state.
#
# Usage: ./scripts/import_existing_fields.sh
#
# Requires: SUMOLOGIC_ACCESS_ID, SUMOLOGIC_ACCESS_KEY, SUMOLOGIC_BASE_URL
#           (or reads from main.auto.tfvars)

set -euo pipefail

# Read credentials from environment or tfvars
if [[ -z "${SUMOLOGIC_ACCESS_ID:-}" ]]; then
  SUMOLOGIC_ACCESS_ID=$(grep 'sumologic_access_id' main.auto.tfvars | sed 's/.*= *"\(.*\)".*/\1/')
fi
if [[ -z "${SUMOLOGIC_ACCESS_KEY:-}" ]]; then
  SUMOLOGIC_ACCESS_KEY=$(grep 'sumologic_access_key' main.auto.tfvars | sed 's/.*= *"\(.*\)".*/\1/')
fi
if [[ -z "${SUMOLOGIC_BASE_URL:-}" ]]; then
  SUMOLOGIC_BASE_URL=$(grep 'sumologic_environment_base_url' main.auto.tfvars | sed 's/.*= *"\(.*\)".*/\1/')
fi

if [[ -z "$SUMOLOGIC_ACCESS_ID" || -z "$SUMOLOGIC_ACCESS_KEY" || -z "$SUMOLOGIC_BASE_URL" ]]; then
  echo "ERROR: Could not determine Sumo Logic credentials."
  echo "Set SUMOLOGIC_ACCESS_ID, SUMOLOGIC_ACCESS_KEY, SUMOLOGIC_BASE_URL or ensure main.auto.tfvars exists."
  exit 1
fi

CREDS="${SUMOLOGIC_ACCESS_ID}:${SUMOLOGIC_ACCESS_KEY}"

FIELD_NAMES=(
  account region accountid namespace loadbalancer loadbalancername
  apiname apiid tablename instanceid clustername cacheclusterid
  functionname networkloadbalancer dbidentifier dbclusteridentifier
  dbinstanceidentifier topicname queuename
)

echo "Fetching existing fields from Sumo Logic..."
FIELDS_JSON=$(curl -s -u "$CREDS" "${SUMOLOGIC_BASE_URL%/}/v1/fields")

imported=0
skipped=0

for field_name in "${FIELD_NAMES[@]}"; do
  tf_addr="module.app-module.sumologic_field.${field_name}"

  # Check if already in state
  if terraform state show "$tf_addr" &>/dev/null; then
    skipped=$((skipped + 1))
    continue
  fi

  # Get field ID from API
  field_id=$(echo "$FIELDS_JSON" | python3 -c "
import json, sys
data = json.load(sys.stdin)
for f in data.get('data', []):
    if f.get('fieldName') == '${field_name}':
        print(f['fieldId'])
        break
" 2>/dev/null)

  if [[ -n "$field_id" ]]; then
    echo "Importing ${field_name} (${field_id})..."
    terraform import "$tf_addr" "$field_id" 2>/dev/null && imported=$((imported + 1)) || echo "  WARN: import failed for ${field_name}"
  fi
done

echo ""
echo "Done. Imported: ${imported}, Already in state: ${skipped}"
