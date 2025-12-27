# Branch Protection Setup Guide

This document explains how to set up branch protection for the TrueNorth repository.

## Automatic Setup (Recommended)

Run the branch protection workflow:

1. Go to **Actions** → **Setup Branch Protection**
2. Click **Run workflow**
3. Enter the branch name (default: `main`)
4. Click **Run workflow**

> **Note**: This requires admin permissions on the repository.

## Manual Setup

If the automatic setup fails, configure branch protection manually:

### Step 1: Navigate to Settings

Go to: `https://github.com/[owner]/[repo]/settings/branches`

### Step 2: Add Branch Protection Rule

Click **Add rule** and configure:

#### Branch name pattern
```
main
```

#### Required Status Checks

Enable: **Require status checks to pass before merging**

- ✅ Require branches to be up to date before merging

Add these required checks:
- `Build & Lint`
- `Unit Tests`
- `Security Scan`
- `Integration Tests`

#### Pull Request Reviews

Enable: **Require a pull request before merging**

- Required approving reviews: `1`
- ✅ Dismiss stale pull request approvals when new commits are pushed
- ✅ Require review from Code Owners (if using CODEOWNERS)
- ✅ Require conversation resolution before merging

#### Additional Settings

- ❌ Require signed commits (optional, enable if needed)
- ❌ Require linear history (optional)
- ❌ Include administrators (allow admins to bypass)
- ❌ Allow force pushes
- ❌ Allow deletions

### Step 3: Save

Click **Create** or **Save changes**

## Required Secrets

For the full CI/CD pipeline, add these secrets in **Settings** → **Secrets and variables** → **Actions**:

| Secret | Description | Required |
|--------|-------------|----------|
| `ANTHROPIC_API_KEY` | Claude API key for code reviews | Optional |

## Verification

After setup, verify protection is working:

1. Create a test branch
2. Make a small change
3. Open a pull request to `main`
4. Verify CI runs and is required
5. Verify review is required

## CI Pipeline Jobs

The following jobs must pass before merging:

| Job | Description | Blocks Merge |
|-----|-------------|--------------|
| Build & Lint | Compiles code, runs `go vet` and format check | Yes |
| Unit Tests | Runs tests with coverage threshold (≥60%) | Yes |
| Security Scan | Runs `govulncheck` and secret detection | Yes |
| Integration Tests | Verifies server startup | Yes |
| Claude Code Review | AI-powered code review (PRs only) | No |

## Coverage Threshold

The CI pipeline enforces a minimum code coverage of **60%**.

To check current coverage locally:
```bash
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | grep total
```

## Troubleshooting

### "Required status check is missing"

The branch protection may reference checks that haven't run yet. Either:
1. Push a commit to trigger CI
2. Temporarily disable the missing check

### "Admin permissions required"

Branch protection requires repository admin access. Contact the repository owner or use a PAT with `admin:repo` scope.

### CI is slow

Go modules and build artifacts are cached. First run is slower; subsequent runs use cache.
