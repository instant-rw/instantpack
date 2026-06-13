# Instantpack

Instantpack is a fork of [Railpack](https://github.com/railwayapp/railpack) maintained for Instant deployments. It keeps upstream compatibility for config files and build plans while branding output and images as Instantpack.

## Images

Release images are published to GitHub Container Registry:

- `ghcr.io/instant-rw/instantpack-builder`
- `ghcr.io/instant-rw/instantpack-runtime`
- `ghcr.io/instant-rw/instantpack-frontend`

## Local development

```bash
# Install mise & instantpack
mise trust
mise install

# Build the CLI
mise run build

# Start BuildKit and build a project
docker run --rm --privileged -d --name buildkit moby/buildkit
export BUILDKIT_HOST='docker-container://buildkit'

./bin/instantpack build .
```

## Configuration

Instantpack keeps upstream `railpack.json` compatibility. Environment variables accept both `INSTANTPACK_*` and legacy `RAILPACK_*` prefixes.

To bypass Bun's `--frozen-lockfile` during install (useful when lockfiles are out of sync):

```bash
export INSTANTPACK_BUN_NO_FROZEN_LOCKFILE=true
```

## Release

Push a version tag to trigger the GitHub Actions release workflow:

```bash
./scripts/release-instantpack.sh v0.1.0
```

This publishes CLI binaries and the GHCR images above.

## Upstream

Instantpack tracks [railwayapp/railpack](https://github.com/railwayapp/railpack). Upstream docs remain useful for build plan behavior: [railpack.com](https://railpack.com).
