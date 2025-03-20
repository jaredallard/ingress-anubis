#!/usr/bin/env bash
# Determines the next version.
set -euo pipefail

# VERSION_OVERRIDE is an environment variable that allows the next
# version to be overridden. If VERSION_OVERRIDE is set, the next version
# will be the value of VERSION_OVERRIDE.
VERSION_OVERRIDE=${VERSION_OVERRIDE:=}

# BUILD_RC is an environment variable that determines whether the build
# is a release candidate. If BUILD_RC is set to true, the next version
# will be a release candidate version (-rc.X).
BUILD_RC=${BUILD_RC:=false}

# As a special case, if the first argument is --rc, the next version
# will be a release candidate version.
if [[ "${1:-}" == "--rc" ]]; then
	BUILD_RC=true
fi

if [[ -n "$VERSION_OVERRIDE" ]]; then
	echo "Overriding next version with: $VERSION_OVERRIDE" >&2
	echo "$VERSION_OVERRIDE"
	exit 0
fi

# Determine the next version as reported by the next-version command.
next_version=$(get-next-version --prefix v 2>/dev/null | sed 's/-rc.*//' | tr -d '\n')

echo "Next release version: $next_version" >&2

# If the build is a release candidate, determine the last release
# candidate version and increment the release candidate number.
if [[ "$BUILD_RC" == "true" ]]; then
	last_rc_version=$(git tag -l --sort=-v:refname | grep -- "$next_version" | grep -- "-rc." | head -n 1 || true)
	if [[ -z "$last_rc_version" ]]; then
		next_version="${next_version}-rc.1"
	else
		echo "Last release candidate version: $last_rc_version" >&2
		last_rc_version_number=${last_rc_version##*-rc.}
		next_rc_version_number=$((last_rc_version_number + 1))
		next_version="${next_version}-rc.${next_rc_version_number}"
	fi
fi

echo "Next version: $next_version" >&2
echo "$next_version"
