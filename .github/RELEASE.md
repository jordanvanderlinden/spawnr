# Release Process

This document describes how releases work in this repository.

## Automated Releases

Releases are **fully automated** using semantic-release. No manual steps required!

### How to Trigger a Release

Simply push commits to `master` using [Conventional Commits](https://www.conventionalcommits.org/) format:

```bash
# This will trigger a MINOR version bump (e.g., 1.0.0 → 1.1.0)
git commit -m "feat: add new cluster management UI"

# This will trigger a PATCH version bump (e.g., 1.0.0 → 1.0.1)
git commit -m "fix: resolve pod cleanup issue"

# This will trigger a MAJOR version bump (e.g., 1.0.0 → 2.0.0)
git commit -m "feat!: redesign API endpoints

BREAKING CHANGE: The API structure has changed completely"

# These will NOT trigger a release
git commit -m "docs: update README"
git commit -m "chore: update dependencies"
git commit -m "ci: fix workflow"
```

### What Happens During Release

When you push to `master`:

1. **semantic-release** analyzes your commit messages
2. Determines the next version number
3. Generates/updates `CHANGELOG.md`
4. Creates a Git tag (e.g., `v1.2.3`)
5. Publishes a GitHub Release with auto-generated notes
6. Triggers the Docker build job
7. Triggers the Helm chart release job

### Docker Images

Multi-architecture images are automatically built and pushed to GitHub Container Registry:

```bash
# Pull by version
docker pull ghcr.io/jordanvanderlinden/spawnr:1.2.3
docker pull ghcr.io/jordanvanderlinden/spawnr:1.2
docker pull ghcr.io/jordanvanderlinden/spawnr:1

# Or pull latest
docker pull ghcr.io/jordanvanderlinden/spawnr:latest
```

Supported architectures:
- `linux/amd64`
- `linux/arm64`

### Helm Chart Releases

The Helm chart is automatically:
- Version updated to match the release
- Packaged
- Published to GitHub Pages

Users can install it via:

```bash
helm repo add spawnr https://jordanvanderlinden.github.io/spawnr
helm repo update
helm install my-spawnr spawnr/spawnr
```

## Commit Message Format

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types

| Type | Description | Version Bump |
|------|-------------|--------------|
| `feat` | New feature | MINOR |
| `fix` | Bug fix | PATCH |
| `perf` | Performance improvement | PATCH |
| `docs` | Documentation only | PATCH |
| `style` | Code style changes | PATCH |
| `refactor` | Code refactoring | PATCH |
| `test` | Test updates | PATCH |
| `build` | Build system changes | PATCH |
| `ci` | CI/CD changes | PATCH |
| `chore` | Other changes | NO RELEASE |

### Breaking Changes

Add `!` after type or `BREAKING CHANGE:` in footer for major version bump:

```bash
feat!: change API structure
# or
feat: change API structure

BREAKING CHANGE: API endpoints have been restructured
```

## Examples

### Multiple Changes in One Commit

```bash
git commit -m "feat(clusters): add dark mode and improve UI

- Added dark mode toggle with persistent preference
- Improved cluster card layout
- Added connectivity testing button

Closes #123"
```

### Fixing a Bug

```bash
git commit -m "fix(jobs): prevent orphaned pods on job deletion

Previously, pods could remain after job deletion. Now we explicitly
delete all associated pods before removing the job.

Fixes #456"
```

### Documentation Update (No Release)

```bash
git commit -m "docs: update installation instructions

Added troubleshooting section for RBAC issues"
```

## Checking Release Status

After pushing to `master`:

1. Go to **Actions** tab in GitHub
2. Find the workflow run for your commit
3. Monitor the progress of:
   - Semantic Release
   - Docker Build
   - Helm Chart Release

## Manual Release (Not Recommended)

If you absolutely need to create a release manually:

```bash
# This is NOT recommended - use semantic-release instead
git tag v1.2.3
git push origin v1.2.3
```

However, this won't:
- Update the CHANGELOG
- Create a GitHub Release
- Trigger Docker/Helm builds

**Always prefer the automated process!**

## Troubleshooting

### Release didn't trigger

- Check commit message format
- Ensure you pushed to `master` branch
- Check GitHub Actions workflow status
- Verify no syntax errors in commit message

### Docker push failed

- Check GitHub Container Registry permissions
- Verify `GITHUB_TOKEN` has package write permissions
- Check for image size limits

### Helm chart not published

- Ensure GitHub Pages is enabled
- Check chart-releaser action logs
- Verify `helm/spawnr/Chart.yaml` is valid

## Version History

View all releases: https://github.com/jordanvanderlinden/spawnr/releases

View changelog: [CHANGELOG.md](../CHANGELOG.md)
