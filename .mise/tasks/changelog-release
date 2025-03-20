#!/usr/bin/env bash
# mise description="Creates a CHANGELOG.md file for the current git tag"
set -euo pipefail

# Get the current tag that we're on.
CURRENT_TAG=$(git describe --tags --abbrev=0)

args=("mise" "run" "changelog" "--")

# If we're on a non-rc version, use the current tag, otherwise use
# unreleased.
if [[ $CURRENT_TAG == *"-rc"* ]]; then
	# Get the previous rc version.
	# shellcheck disable=SC2001
	PREVIOUS_RC_TAG=$(git tag --list --sort=-v:refname |
		grep -E "$(sed 's/-rc\.[0-9]*//' <<<"$CURRENT_TAG")" | grep -v "$CURRENT_TAG" |
		head -n 1 || true)

	if [[ -z $PREVIOUS_RC_TAG ]]; then
		args+=("--unreleased")
	else
		echo "Previous rc tag: $PREVIOUS_RC_TAG" >&2
		args+=("--" "$PREVIOUS_RC_TAG..$CURRENT_TAG")
	fi
else
	args+=("--current")
fi

# Run mise to generate the changelog.
"${args[@]}"

# If we're on a rc version, fix the header.
if [[ $CURRENT_TAG == *"-rc"* ]]; then
	sed -i.bak "s/^## \[unreleased\]/## $CURRENT_TAG/" CHANGELOG.md
	rm CHANGELOG.md.bak
fi
