#!/bin/bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# Regenerate README.md for every Terraform directory in the repo.
# Run this from anywhere; it always anchors to the repository root.
find "$BASE_DIR" \
  \( -path "*/.git" -o -path "*/.git/*" -o -path "*/.terraform" -o -path "*/.terraform/*" -o -path "*/.terragrunt-cache" -o -path "*/.terragrunt-cache/*" \) -prune \
  -o -type f -name "README.md" -print0 |
while IFS= read -r -d '' readme; do
  readme_dir="$(dirname "$readme")"

  if [[ "$readme" == "$BASE_DIR/README.md" ]]; then
    continue
  fi

  if compgen -G "$readme_dir/*.tf" >/dev/null || compgen -G "$readme_dir/*.tf.json" >/dev/null; then
    echo "Updating $readme_dir/README.md"
    (cd "$readme_dir" && terraform-docs markdown . > README.md)
  fi
done
