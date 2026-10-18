---
inclusion: manual
---

# Release Procedure

Use when the user asks to create a release, cut a new version, or publish changes.

CI workflow conventions: `github-actions-workflow.md`. Local checks: `tech.md`.

## Versioning

- Patch (`v1.x.Y+1`): bug fixes, docs, non-breaking additions (new commands, new flags)
- Minor (`v1.X+1.0`): significant new feature sets or API changes
- Never bump manually — determine from commits since last tag

## Step 1 — Gather changes since last release

```bash
# Find latest tag
gh release list --limit 3

# List commits since that tag
git log <last-tag>..HEAD --oneline
```

For each commit, check the diff if needed:
```bash
git show <sha> --stat
```

Before pushing a release tag, wait for the CI run on the exact `main` commit
to pass. Its `security` job scans the built Docker image with Trivy and fails
on `CRITICAL` vulnerabilities. Review any behavior changes for the release
notes before tagging.

```bash
gh run list --workflow=ci.yml --commit <main-sha> --limit 1
gh run watch <ci-run-id> --exit-status
```

## Step 2 — Push the tag

```bash
git tag v<X.Y.Z> && git push origin v<X.Y.Z>
```

The release workflow (`.github/workflows/release.yml`) triggers automatically on `v*` tags.
It uses Go 1.26, builds binaries for 6 platform/arch combinations (`make binaries TARGET=os/arch`),
creates a GitHub release with auto-generated notes, and builds + pushes a multi-arch Docker image
(`linux/amd64` + `linux/arm64`) to Docker Hub.

**Docker tags produced per release:**

| Tag | Example | When |
|-----|---------|------|
| `stable` | `noksa/gokeenapi:stable` | Every non-prerelease tag |
| `X.Y.Z` | `noksa/gokeenapi:1.7.1` | Every tag |
| `X.Y` | `noksa/gokeenapi:1.7` | Every tag |

Prerelease tags (`v1.7.2-beta.1`, `-rc.1`, etc.) produce only the version tag — `stable` is skipped.

## Step 3 — Wait for the workflow to complete

```bash
# Get the run ID
gh run list --workflow=release.yml --limit 1 --json databaseId --jq '.[0].databaseId'

# Watch until done
gh run watch <run-id> --exit-status
```

## Step 4 — Write pretty release notes

Fetch the previous release notes as style reference:
```bash
gh release view <last-tag> --json body --jq '.body'
```

Follow the same structure:
```markdown
## 🚀 New Features

### <Feature name> (`<command>`, `<alias>`)
<Short description. What it does, when to use it.>

\```shell
# Usage example
\```

**Requirements:** <if any>

## 🐛 Bug Fixes

- **<Component>** — <what was fixed>

## 💥 Breaking Changes
None.   ← or describe them

--------
Full Changelog: https://github.com/Noksa/gokeenapi/compare/<prev-tag>...<new-tag>
```

Rules:
- Only include sections that have content — omit empty ones
- New commands always get their own `###` subsection with aliases and a shell example
- Bug fixes are bullet points: `- **Component** — description`
- Breaking changes: list them explicitly or write `None.`

## Step 5 — Update the release

Write notes to a temp file, update via `--notes-file`, then delete the file:

```bash
# Write to temp file (NEVER use heredoc/echo pipe)
# Use fsWrite tool to create the file, then:
gh release edit v<X.Y.Z> --title "v<X.Y.Z>" --notes-file release_notes_vX.Y.Z.md

# Clean up
rm release_notes_vX.Y.Z.md
```

## Checklist

- [ ] `make lint` passes locally
- [ ] `make test` passes locally  
- [ ] All changes pushed to `main`
- [ ] CI on the exact `main` commit passed, including `security`
- [ ] Tag pushed: `git tag vX.Y.Z && git push origin vX.Y.Z`
- [ ] Workflow completed successfully: `gh run watch <id> --exit-status`
- [ ] Docker image verified: `docker buildx imagetools inspect noksa/gokeenapi:X.Y.Z` (should show `linux/amd64` + `linux/arm64`)
- [ ] Release notes updated with pretty format matching v1.7.0/v1.7.1 style
