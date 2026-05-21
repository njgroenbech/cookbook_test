# Legacy Project

This is a Docker-based Go application with monitoring capabilities.

## Required Environment Variables

The following environment variables must be set for the application to function:

- **DB_HOST**: Database server hostname
- **DB_PORT**: Database server port
- **DB_USER**: Database user
- **DB_PASSWORD**: Database password
- **DB_NAME**: Database name

These variables should be configured in your GitHub Secrets and will be injected into the container via `docker-compose.yaml`.

---

## Branch Protection (master)

Branch protection is enforced via a GitHub Ruleset. To configure it:

1. Go to **Settings → Rules → Rulesets → New ruleset**
2. Enter the following:

| Field | Value |
|---|---|
| Ruleset Name | `Protect master` |
| Enforcement status | `Active` |
| Bypass list | *(empty — no bypasses)* |
| Target branches | `master` |

3. Under **Branch rules**, enable:

| Rule | Notes |
|---|---|
| Restrict deletions | Prevents the branch from being deleted |
| Require a pull request before merging | All changes must go through a PR — no direct pushes |
| Require status checks to pass | Add `Go quality checks` and `build-and-push` as required checks. Enable "Require branches to be up to date" |
| Block force pushes | Prevents rewriting history on master |

> **Note:** `Go quality checks` and `build-and-push` only appear in the status check search box after they have run at least once on a PR. Push a commit to any open PR to trigger the workflows before adding them.

For details on what the CI/CD workflows do, see [.github/workflows/README.md](.github/workflows/README.md).