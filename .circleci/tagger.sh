#!/bin/bash
# This script fetches the latest tag from the GitHub repo (via git fetch),
# checks a local file (.circleci/tag.dat) for a proposed new tag (single line value, e.g., v0.0.5).
# If the file is missing, empty, matches the current tag, or is older, this script will only
# increment the patch version & use that value as the next tag.
# Assumes SemVer tags like vX.Y.Z. If no tags exist, starts from v0.0.1.
# Creates the new tag locally and pushes it to origin.
# Works in bash/zsh (sh-compatible).

FILE=".circleci/tag.dat"

# Fetch tags silently
git fetch --tags >/dev/null 2>&1

# Get the latest tag (sorted descending)
CURRENT_TAG=$(git tag --sort=-v:refname | head -n1)
echo "Current tag is: $CURRENT_TAG"

# If no tags, default to v0.0.0
if [ -z "$CURRENT_TAG" ]; then
  CURRENT_TAG="v0.0.0"
fi

# Function to strip 'v' prefix
strip_v() {
  echo "$1" | sed 's/^v//'
}

# Function to compare if version a > b (using sort -V for SemVer)
version_gt() {
  a=$(strip_v "$1")
  b=$(strip_v "$2")
  if [ "$(printf '%s\n%s' "$b" "$a" | sort -V | head -n1)" = "$b" ] && [ "$a" != "$b" ]; then
    return 0  # a > b
  else
    return 1
  fi
}

# Function to check if versions equal
version_eq() {
  [ "$(strip_v "$1")" = "$(strip_v "$2")" ]
}

# Function to bump patch version
bump_patch() {
  ver=$(strip_v "$1")
  major=$(echo "$ver" | cut -d. -f1)
  minor=$(echo "$ver" | cut -d. -f2)
  patch=$(echo "$ver" | cut -d. -f3)
  new_patch=$((patch + 1))
  echo "v${major}.${minor}.${new_patch}"
}

# Read proposed tag from file (trim whitespace)
if [ -f "$FILE" ]; then
  PROPOSED=$(cat "$FILE" | tr -d '[:space:]')
else
  PROPOSED=""
fi

echo "Proposed tag: $PROPOSED"

# Decide new tag
if [ -z "$PROPOSED" ] || version_eq "$PROPOSED" "$CURRENT_TAG" || ! version_gt "$PROPOSED" "$CURRENT_TAG"; then
  NEW_TAG=$(bump_patch "$CURRENT_TAG")
else
  NEW_TAG="$PROPOSED"
  # Normalize to start with 'v' if missing
  if ! echo "$NEW_TAG" | grep -q '^v'; then
    NEW_TAG="v$NEW_TAG"
  fi
fi

# Create and push the tag
echo "Tagging as $NEW_TAG"
git tag "$NEW_TAG"
git push origin "$NEW_TAG"

# Optionally clean up the file
# rm -f "$FILE"

echo "Created and pushed new tag: $NEW_TAG"
