#!/bin/bash
# Release script for asimonim
# Creates git tag and GitHub release
#
# Usage: ./scripts/release.sh <version|patch|minor|major>
# Example: ./scripts/release.sh v0.1.0
# Example: ./scripts/release.sh patch

set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=scripts/lib/release-version.sh
source "$SCRIPT_DIR/lib/release-version.sh"
# shellcheck source=scripts/lib/release-notes.sh
source "$SCRIPT_DIR/lib/release-notes.sh"

if [ -z "${1:-}" ]; then
  echo "Error: VERSION or bump type is required"
  echo "Usage: $0 <vX.Y.Z|patch|minor|major>"
  echo "  $0 v0.1.0   - Release explicit version"
  echo "  $0 patch    - Bump patch version (0.0.x)"
  echo "  $0 minor    - Bump minor version (0.x.0)"
  echo "  $0 major    - Bump major version (x.0.0)"
  echo "  Versions must be v-prefixed."
  exit 1
fi

INPUT="$1"
REPO_ROOT=$(git rev-parse --show-toplevel)
cd "$REPO_ROOT"

if [[ "$INPUT" =~ ^(patch|minor|major)$ ]]; then
  LATEST_TAG=$(latest_release_tag)
  echo "Latest tag: $LATEST_TAG"
  VERSION=$(resolve_release_version "$INPUT" "$LATEST_TAG") || exit 1
  echo "Bumping $INPUT: $LATEST_TAG → $VERSION"
  echo ""
else
  VERSION=$(resolve_release_version "$INPUT") || exit 1
fi

echo "Checking if tag $VERSION already exists..."
if git rev-parse "$VERSION" >/dev/null 2>&1; then
  echo "Error: Tag $VERSION already exists"
  echo "Use 'git tag -d $VERSION' to delete locally if needed"
  exit 1
fi
if git remote get-url origin >/dev/null 2>&1; then
  if git ls-remote --tags origin "refs/tags/$VERSION" 2>/dev/null | grep -q .; then
    echo "Error: Tag $VERSION already exists on origin"
    exit 1
  fi
fi
echo "✓ Tag $VERSION does not exist"
echo ""

NOTES_FILE=$(release_notes_file "$REPO_ROOT" "$VERSION")
META_FILE=$(release_notes_meta_file "$REPO_ROOT" "$VERSION")
if [ ! -f "$NOTES_FILE" ]; then
  echo "Error: release notes not found: $NOTES_FILE"
  echo "Run 'make notes $VERSION' first, then review the draft."
  exit 1
fi
if [ ! -f "$META_FILE" ]; then
  echo "Error: release notes metadata not found: $META_FILE"
  echo "Run 'make notes $VERSION' to regenerate the notes and metadata."
  exit 1
fi
"$SCRIPT_DIR/release-notes/verify.sh" "$NOTES_FILE" "$VERSION"

BASE_SHA=$(read_release_notes_base "$META_FILE") || BASE_SHA=""
if [ -z "$BASE_SHA" ]; then
  echo "Error: $META_FILE has no valid base commit"
  echo "Run 'make notes $VERSION' to regenerate the notes and metadata."
  exit 1
fi

set +e
check_release_notes_freshness "$BASE_SHA"
freshness=$?
set -e
if [ "$freshness" -eq 1 ]; then
  echo "Error: release notes are stale"
  if git merge-base --is-ancestor "$BASE_SHA" HEAD 2>/dev/null; then
    echo "Commits landed after the notes were generated:"
    git log --oneline "$BASE_SHA..HEAD" -- . ':(exclude)docs/releases'
  else
    echo "The notes were generated from a commit that is not an ancestor of HEAD."
  fi
  echo ""
  echo "Remove $NOTES_FILE and $META_FILE, then rerun 'make notes $VERSION'."
  exit 1
elif [ "$freshness" -ne 0 ]; then
  echo "Error: cannot verify release notes base $BASE_SHA"
  exit 1
fi

echo "✓ Release notes found: $NOTES_FILE"
echo "✓ Release notes are current"
echo ""

# Update extension versions
"$SCRIPT_DIR/version.sh" "$VERSION"
write_release_notes_meta "$META_FILE" "$VERSION" "$(git rev-parse HEAD)"

# Check for uncommitted changes
if [[ $(git status --porcelain 2>/dev/null) ]]; then
  echo "Error: Working directory has uncommitted changes"
  echo "Please commit or stash your changes before releasing"
  git status --short
  exit 1
fi
echo "✓ Working directory is clean"
echo ""

# Ensure we're up to date with remote
echo "Checking remote status..."
git fetch origin
LOCAL=$(git rev-parse HEAD)
REMOTE=$(git rev-parse origin/main 2>/dev/null || git rev-parse origin/master 2>/dev/null || echo "")
if [ -n "$REMOTE" ] && [ "$LOCAL" != "$REMOTE" ]; then
  echo "Warning: Local branch differs from remote"
  if [[ -t 0 ]]; then
    read -p "Continue anyway? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
      exit 1
    fi
  else
    echo "Non-interactive mode: aborting due to remote divergence"
    exit 1
  fi
fi
echo ""

echo "Creating release $VERSION..."
echo ""

# Push any unpushed commits first
echo "Step 1: Pushing commits..."
git push
echo ""

echo "Step 2: Creating GitHub release (gh will tag and push)..."
gh release create "$VERSION" \
  --target "$(git rev-parse HEAD)" \
  --notes-file "$NOTES_FILE"
