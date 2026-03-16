#!/bin/bash
# Version management script for asimonim
# Updates version across VSCode extension, Zed extension, and creates git commit
#
# Usage: ./scripts/version.sh <version>
# Example: ./scripts/version.sh 0.2.0

set -e

if [ -z "$1" ]; then
  echo "Usage: $0 <version>"
  echo "Example: $0 0.2.0"
  exit 1
fi

VERSION="$1"
# Remove 'v' prefix if present
VERSION="${VERSION#v}"

echo "Updating version to: $VERSION"

# Helper: portable sed -i
sed_inplace() {
  if [[ "$OSTYPE" == "darwin"* ]]; then
    sed -i '' "$@"
  else
    sed -i "$@"
  fi
}

# Update VSCode extension version
echo "Updating extensions/vscode/package.json..."
if command -v jq &> /dev/null; then
  jq ".version = \"$VERSION\"" extensions/vscode/package.json > extensions/vscode/package.json.tmp
  mv extensions/vscode/package.json.tmp extensions/vscode/package.json
elif command -v node &> /dev/null; then
  node -e "
    const fs = require('fs');
    const pkg = JSON.parse(fs.readFileSync('extensions/vscode/package.json', 'utf8'));
    pkg.version = '$VERSION';
    fs.writeFileSync('extensions/vscode/package.json', JSON.stringify(pkg, null, 2) + '\n');
  "
else
  sed_inplace "s/\"version\": \".*\"/\"version\": \"$VERSION\"/" extensions/vscode/package.json
fi

# Update npm package version
echo "Updating npm/package.json..."
if command -v jq &> /dev/null; then
  jq ".version = \"$VERSION\"" npm/package.json > npm/package.json.tmp
  mv npm/package.json.tmp npm/package.json
elif command -v node &> /dev/null; then
  node -e "
    const fs = require('fs');
    const pkg = JSON.parse(fs.readFileSync('npm/package.json', 'utf8'));
    pkg.version = '$VERSION';
    fs.writeFileSync('npm/package.json', JSON.stringify(pkg, null, 2) + '\n');
  "
else
  sed -i "s/\"version\": \".*\"/\"version\": \"$VERSION\"/" npm/package.json
fi

# Update Zed extension version
echo "Updating extensions/zed/extension.toml..."
sed_inplace "s/^version = \".*\"/version = \"$VERSION\"/" extensions/zed/extension.toml

# Update Claude Code plugin version
echo "Updating .claude-plugin/marketplace.json..."
if command -v jq &> /dev/null; then
  jq ".plugins[0].version = \"$VERSION\"" .claude-plugin/marketplace.json > .claude-plugin/marketplace.json.tmp
  mv .claude-plugin/marketplace.json.tmp .claude-plugin/marketplace.json
elif command -v node &> /dev/null; then
  node -e "
    const fs = require('fs');
    const m = JSON.parse(fs.readFileSync('.claude-plugin/marketplace.json', 'utf8'));
    m.plugins[0].version = '$VERSION';
    fs.writeFileSync('.claude-plugin/marketplace.json', JSON.stringify(m, null, 2) + '\n');
  "
fi

VERSION_FILES="extensions/vscode/package.json extensions/zed/extension.toml .claude-plugin/marketplace.json npm/package.json"

# Show changes
echo ""
echo "Version updated in:"
echo "  - extensions/vscode/package.json"
echo "  - extensions/zed/extension.toml"
echo "  - .claude-plugin/marketplace.json"
echo "  - npm/package.json"
echo ""
echo "Changes:"
git diff $VERSION_FILES

# Check if there are changes
if ! git diff --quiet $VERSION_FILES; then
  # Auto-commit in non-interactive mode (CI or piped stdin)
  if [[ ! -t 0 ]]; then
    git add $VERSION_FILES
    git commit -m "chore: prepare version $VERSION"
    echo "✓ Version changes committed (non-interactive)"
  else
    echo ""
    read -p "Commit version changes? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
      git add $VERSION_FILES
      git commit -m "chore: prepare version $VERSION"
      echo "✓ Version changes committed"
      echo ""
      echo "Next steps:"
      echo "  make release v$VERSION  (to tag, push, and create GitHub release)"
    else
      echo "Version changes rejected by user."
      git checkout -- $VERSION_FILES
      exit 1
    fi
  fi
else
  echo ""
  echo "No changes detected. Version might already be $VERSION"
fi
