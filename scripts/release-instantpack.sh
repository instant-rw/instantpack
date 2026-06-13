#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

usage() {
  cat <<EOF
Usage: $(basename "$0") <version>

Create and push a git tag to trigger the Instantpack release workflow.
The workflow publishes binaries and ghcr.io/instant-rw/instantpack-* images.

Example:
  $(basename "$0") v0.1.0
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

VERSION="${1:-}"
if [[ -z "$VERSION" ]]; then
  usage
  exit 1
fi

if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$ ]]; then
  echo "Version must look like v1.2.3 (optional pre-release suffix allowed)" >&2
  exit 1
fi

if ! git diff --quiet || ! git diff --cached --quiet; then
  echo "Working tree has uncommitted changes. Commit or stash before releasing." >&2
  exit 1
fi

echo "Creating tag $VERSION on $(git rev-parse --short HEAD)"
git tag "$VERSION"
git push origin "$VERSION"

echo
echo "Release tag pushed. Track progress at:"
echo "https://github.com/instant-rw/instantpack/actions"
