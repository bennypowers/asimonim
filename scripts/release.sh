#!/bin/bash
# Release script for asimonim
# Creates git tag and GitHub release
#
# Usage: ./scripts/release.sh <version|patch|minor|major>
# Example: ./scripts/release.sh v0.1.0
# Example: ./scripts/release.sh patch

set -e

if [ -z "$1" ]; then
  echo "Error: VERSION or bump type is required"
  echo "Usage: $0 <version|patch|minor|major>"
  echo "  $0 v0.1.0   - Release explicit version"
  echo "  $0 patch    - Bump patch version (0.0.x)"
  echo "  $0 minor    - Bump minor version (0.x.0)"
  echo "  $0 major    - Bump major version (x.0.0)"
  exit 1
fi

INPUT="$1"

# Check if input is a bump type (patch/minor/major)
if [[ "$INPUT" =~ ^(patch|minor|major)$ ]]; then
  BUMP_TYPE="$INPUT"

  # Get the latest tag
  LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
  echo "Latest tag: $LATEST_TAG"

  # Remove 'v' prefix if present
  CURRENT_VERSION="${LATEST_TAG#v}"

  # Parse version components
  if [[ ! "$CURRENT_VERSION" =~ ^([0-9]+)\.([0-9]+)\.([0-9]+)$ ]]; then
    echo "Error: Latest tag '$LATEST_TAG' is not a valid semver version"
    echo "Expected format: v0.0.0"
    exit 1
  fi

  MAJOR="${BASH_REMATCH[1]}"
  MINOR="${BASH_REMATCH[2]}"
  PATCH="${BASH_REMATCH[3]}"

  # Bump the appropriate component
  case "$BUMP_TYPE" in
    patch)
      PATCH=$((PATCH + 1))
      ;;
    minor)
      MINOR=$((MINOR + 1))
      PATCH=0
      ;;
    major)
      MAJOR=$((MAJOR + 1))
      MINOR=0
      PATCH=0
      ;;
  esac

  VERSION="v${MAJOR}.${MINOR}.${PATCH}"
  echo "Bumping $BUMP_TYPE: $LATEST_TAG → $VERSION"
  echo ""
else
  # Use explicit version
  VERSION="$INPUT"
  # Ensure v prefix
  if [[ ! "$VERSION" =~ ^v ]]; then
    VERSION="v$VERSION"
  fi
fi

echo "Checking if tag $VERSION already exists..."
if git rev-parse "$VERSION" >/dev/null 2>&1; then
  echo "Error: Tag $VERSION already exists"
  echo "Use 'git tag -d $VERSION' to delete locally if needed"
  exit 1
fi
echo "✓ Tag $VERSION does not exist"
echo ""

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
  read -p "Continue anyway? (y/n) " -n 1 -r
  echo
  if [[ ! $REPLY =~ ^[Yy]$ ]]; then
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
gh release create "$VERSION" --generate-notes
