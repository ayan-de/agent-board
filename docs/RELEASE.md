# Release Guide

This document covers how to ship a new version of AgentBoard.

---

## Prerequisites

- GitHub repo: `ayan-de/agent-board`
- GitHub CLI (`gh`) authenticated: `gh auth login`
- Docker (for cross-platform builds — darwin builds must happen on macOS runners; for now we'll use GitHub Actions which handles all platforms)

---

## Step 1: Prepare the release

Ensure `main` is clean and all tests pass locally:

```bash
go test ./...
git checkout main
git pull origin main
```

---

## Step 2: Bump the version

Pick a version number following [semver](https://semver.org/):

```
v0.1.0  — initial release
v0.2.0  — new features, no breaking changes
v1.0.0  — first stable release
```

Tag the commit:

```bash
git tag -a v0.1.0 -m "Release v0.1.0"
git push origin v0.1.0
```

---

## Step 3: GitHub Actions builds

Pushing the tag triggers `.github/workflows/release.yml` which:

1. Runs on **5 matrix jobs**: linux-amd64, linux-arm64, darwin-amd64, darwin-arm64, android-arm64
2. Builds `agentboard-{OS}-{ARCH}` for each
3. Uses `softprops/action-gh-release` to attach all 5 binaries to a new GitHub Draft Release

**Check the run:**
```bash
gh run list --workflow=release.yml
```

Or watch via GitHub UI: https://github.com/ayan-de/agent-board/actions

---

## Step 4: Publish the release

After CI completes:

1. Go to: https://github.com/ayan-de/agent-board/releases
2. You should see a **Draft** release with all 5 binaries
3. Click **Edit** to add release notes — describe what changed
4. Click **Publish release**

The install script at `https://raw.githubusercontent.com/ayan-de/agent-board/main/install.sh` will automatically point to `releases/latest/download/agentboard-{OS}-{ARCH}`.

---

## How users install

After the release is published, users run:

```bash
curl -sSL https://agentboard.ayande.xyz/install.sh | bash
```

The script:
1. Detects their OS and arch
2. Installs missing dependencies (tmux, Go, npm, node, git)
3. Downloads the correct binary from `releases/latest/download/`
4. Places it in `~/.local/bin/agentboard`

Then:
```bash
export PATH="$HOME/.local/bin:$PATH"
agentboard init
agentboard
```

---

## Platform binaries

| Binary name | OS | Arch |
|-------------|----|------|
| `agentboard-linux-x86_64` | Linux | x86_64 |
| `agentboard-linux-arm64` | Linux | ARM64 |
| `agentboard-darwin-x86_64` | macOS | Intel |
| `agentboard-darwin-arm64` | macOS | Apple Silicon |
| `agentboard-android-arm64` | Android (Termux) | ARM64 |

---

## Troubleshooting

**Release assets missing?**
Check the Actions tab for build failures. Common issues: Go version mismatch, out-of-memory on ARM builds.

**Install script returns "Not Found"?**
The release hasn't been published yet, or the binary names don't match exactly. Binary names must match exactly what the install script expects (see table above).

**tmux not installed?**
The install script tries to install it automatically. On failure, users must install tmux manually first:
- Ubuntu/Debian: `sudo apt-get install tmux`
- macOS: `brew install tmux`
- Termux: `pkg install tmux`