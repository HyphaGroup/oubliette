#!/usr/bin/env bash
set -euo pipefail

# Update reference repositories used for container builds
# Repos are listed in repos.json (gitignored)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPOS_FILE="$SCRIPT_DIR/repos.json"

RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

info() { echo -e "${BLUE}$1${NC}"; }
success() { echo -e "${GREEN}$1${NC}"; }
error() { echo -e "${RED}Error: $1${NC}" >&2; exit 1; }

# Create repos.json template if it doesn't exist
if [[ ! -f "$REPOS_FILE" ]]; then
    cat > "$REPOS_FILE" <<'EOF'
{
  "repos": [
    {
      "name": "example-repo",
      "url": "https://github.com/org/example-repo.git",
      "branch": "main"
    }
  ]
}
EOF
    info "Created $REPOS_FILE template - edit it to add your repositories"
    exit 0
fi

# Check for jq
command -v jq &>/dev/null || error "jq is required but not installed"

# Read and process repos
repos=$(jq -c '.repos[]' "$REPOS_FILE")

while IFS= read -r repo; do
    name=$(echo "$repo" | jq -r '.name')
    url=$(echo "$repo" | jq -r '.url')
    branch=$(echo "$repo" | jq -r '.branch // "main"')
    
    repo_dir="$SCRIPT_DIR/$name"
    
    if [[ -d "$repo_dir/.git" ]]; then
        info "Updating $name..."
        cd "$repo_dir"
        git fetch origin
        git checkout "$branch" 2>/dev/null || git checkout -b "$branch" "origin/$branch"
        git reset --hard "origin/$branch"
        cd "$SCRIPT_DIR"
        success "  Updated to origin/$branch"
    else
        info "Cloning $name..."
        git clone --branch "$branch" "$url" "$repo_dir"
        success "  Cloned $branch"
    fi
done <<< "$repos"

success "All repositories updated"
