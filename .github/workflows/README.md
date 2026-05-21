# CI/CD Pipeline

Two workflow files run automatically on every push.

---

## ci.yml — Quality gates (all branches + PRs)

Runs on every push to any branch and on every pull request.

| Step | Tool | Purpose |
|---|---|---|
| Static analysis | `go vet` | Catches common Go bugs (misused format strings, unreachable code, etc.) |
| Tests + race detector | `go test -race` | Runs all unit tests and detects data races |
| Coverage threshold | `go tool cover` | Fails if total test coverage drops below 30% |
| Vulnerability scan | `govulncheck` | Checks Go dependencies against the Go vulnerability database |
| Dockerfile lint | `hadolint` | Enforces Dockerfile best practices |

This workflow is the quality gate for pull requests. It must be configured as a **required status check** on `master` via branch protection rules so that cd.yml can rely on it having passed.

---

## cd.yml — Build, scan, and deploy

Runs on every push to `master`, on every pull request targeting `master`, and can be triggered manually via `workflow_dispatch`.

On **pull requests**, only `build-and-push` runs — the image is built and scanned for CVEs, but nothing is pushed to GHCR and no deploy happens. This catches Dockerfile errors and critical vulnerabilities before merge.

On **push to master**, the full pipeline runs. A concurrency lock prevents two deploys from racing.

```
build-and-push → deploy → promote-and-notify
```

### build-and-push
1. Builds the Docker image locally (not yet pushed to the registry).
2. Scans the local image with **Trivy** for `CRITICAL` and `HIGH` CVEs. The pipeline stops here if any unfixed vulnerabilities are found.
3. Pushes two tags to GHCR: `:latest` and `:<commit-sha>`.

### deploy
Deploys to the backend VM over SSH using a matrix (nginx VM will be added once configured). For each VM:

1. Copies `docker-compose.yaml` from the runner to `~/legacyProject/` on the VM via SCP.
2. Writes `~/legacyProject/.env` from the `ENV_FILE` secret — the value is passed as an environment variable and never echoed to logs.
3. Pulls the new image from GHCR.
4. Runs `docker compose up --wait`, which blocks until the container passes its **healthcheck**. If the healthcheck never passes within 60 seconds, the pipeline rolls back to `:latest-stable` (the last confirmed-healthy image) automatically.

### promote-and-notify
Runs once after all VMs in the deploy matrix are confirmed healthy.

1. Promotes the deployed commit SHA to the `:latest-stable` tag — a known-good reference that is never pushed speculatively.
2. Sends a Discord notification with the deploy status (requires the `DISCORD_WEBHOOK_URL` secret; silently skipped if not set).

---

## Secrets required

| Secret | Used by |
|---|---|
| `GITHUB_TOKEN` | GHCR login (provided automatically by GitHub Actions) |
| `SSH_HOST_BACKEND` | SSH target for the backend VM |
| `VM_USER` | SSH username for the VM |
| `AZURE_KEY` | SSH private key for the VM |
| `ENV_FILE` | Full contents of the `.env` file, written to the VM on each deploy |
| `DISCORD_WEBHOOK_URL` | Deploy notifications (optional) |

---

## Action version pinning

`actions/checkout` and `docker/login-action` are pinned to exact commit SHAs.
The remaining actions have `# TODO: pin to commit SHA` comments with instructions.
To get the SHA for any action tag, run:

```bash
gh api repos/<owner>/<repo>/git/ref/tags/<tag> --jq .object.sha
```
