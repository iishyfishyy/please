# Release Guide

This document explains how to create a new release of `please`.

## Prerequisites

1. Make sure all changes are committed and pushed to `main`
2. Ensure tests pass (if you have any): `go test ./...`
3. Update CHANGELOG if you're maintaining one

## Creating a Release

The release process is fully automated using GoReleaser and GitHub Actions. To create a new release:

### 1. Create and push a version tag

```bash
# Create a new tag (use semantic versioning: v1.0.0, v1.0.1, v2.0.0, etc.)
git tag -a v1.0.0 -m "Release v1.0.0"

# Push the tag to GitHub
git push origin v1.0.0
```

### 2. Automated process

Once you push the tag, GitHub Actions will automatically:

1. Build binaries for:
   - macOS (Intel and Apple Silicon)
   - Linux (amd64 and arm64)
   - Windows (amd64)

2. Create a GitHub release with:
   - Release notes (generated from commits)
   - Downloadable binaries for all platforms
   - Checksums file

3. Update the Homebrew tap (requires setup - see below)

### 3. Verify the release

1. Go to https://github.com/iishyfishyy/please/releases
2. Check that the release appears with all binaries
3. Test installation with one of the methods

## First-Time Setup

### Setting up Homebrew tap (optional but recommended)

For Homebrew installation to work, you need to:

1. Create a new repository called `homebrew-tap`:
   ```bash
   # On GitHub, create a new repo: iishyfishyy/homebrew-tap
   ```

2. Create a personal access token for the Homebrew tap:
   - Go to GitHub Settings → Developer settings → Personal access tokens → Tokens (classic)
   - Click "Generate new token (classic)"
   - Give it a name like "GoReleaser Homebrew Tap"
   - Select scopes: `repo` (all repo permissions)
   - Generate token and copy it

3. Add the token as a repository secret:
   - Go to your `please` repository settings
   - Secrets and variables → Actions
   - New repository secret
   - Name: `HOMEBREW_TAP_GITHUB_TOKEN`
   - Value: [paste your token]

After this one-time setup, GoReleaser will automatically update the Homebrew formula with each release.

## Version Numbering

Follow semantic versioning:

- `v1.0.0` - Major release (breaking changes)
- `v1.1.0` - Minor release (new features, backward compatible)
- `v1.0.1` - Patch release (bug fixes)

## Testing a Release Locally

Before pushing a tag, you can test the release process locally:

```bash
# Install GoReleaser
go install github.com/goreleaser/goreleaser@latest

# Test the build (doesn't publish anything)
goreleaser release --snapshot --clean

# Check the dist/ folder for built binaries
ls -la dist/
```

## Troubleshooting

### Release failed

1. Check the Actions tab on GitHub for error details
2. Common issues:
   - Invalid version tag format (must start with 'v')
   - Missing HOMEBREW_TAP_GITHUB_TOKEN secret
   - Build errors in code

### Homebrew formula not updating

1. Verify HOMEBREW_TAP_GITHUB_TOKEN is set correctly
2. Check that homebrew-tap repository exists
3. Look at the GitHub Actions logs for Homebrew-related errors

### Need to delete a bad release

```bash
# Delete the tag locally
git tag -d v1.0.0

# Delete the tag from GitHub
git push origin :refs/tags/v1.0.0

# Then delete the release from GitHub UI
# Go to releases page, find the release, click Delete
```

## Release Checklist

Before creating a release:

- [ ] All tests pass
- [ ] Documentation is up to date
- [ ] CHANGELOG is updated (if maintaining one)
- [ ] Version number follows semantic versioning
- [ ] All changes are committed and pushed

After creating a release:

- [ ] GitHub release appears with all binaries
- [ ] Test installation via `brew install` (if tap is set up)
- [ ] Test installation via install script
- [ ] Test installation via `go install`
- [ ] Announce the release (Twitter, Discord, etc.)
