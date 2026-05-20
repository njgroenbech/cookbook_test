# CI/CD Pipeline

Two workflow files run automatically on every push.

---

## linting.yml — Quality gates (all branches + PRs)

Runs on every push to any branch and on every pull request.

| Step | Tool | Purpose |
|---|---|---|
| Static analysis | `go vet` | Catches common Go bugs (misused format strings, unreachable code, etc.) |
| Tests + race detector | `go test -race` | Runs all unit tests and detects data races |
| Coverage threshold | `go tool cover` | Fails if total test coverage drops below 30% |
| Vulnerability scan | `govulncheck` | Checks Go dependencies against the Go vulnerability database |
| Dockerfile lint | `hadolint` | Enforces Dockerfile best practices |

This workflow is the quality gate for pull requests. It must pass before merging to master.

---

## continuous_delivery.yml — Build, scan, and deploy (master only)

Runs on every push to `master`. A concurrency lock prevents two deploys from racing.

```
test → build-and-push → deploy
```

### test
Runs `go test -race` as a final deployment gate, independent of the linting workflow.

### build-and-push
1. Builds the Docker image locally (not yet pushed to the registry).
2. Scans the local image with **Trivy** for `CRITICAL` and `HIGH` CVEs. The pipeline stops here if any unfixed vulnerabilities are found.
3. Pushes two tags to GHCR: `:latest` and `:<commit-sha>`.

### deploy
Deploys to two Azure VMs (backend and nginx) over SSH. On each VM:

1. Pulls the new image from GHCR.
2. Records the current image ID for rollback.
3. Runs `docker compose up --wait`, which blocks until the container passes its **healthcheck** (`GET /health`). If the healthcheck never passes within 60 seconds, the pipeline rolls back to the previously recorded image automatically.
4. After both VMs are confirmed healthy, the commit is promoted to the `:latest-stable` tag — a known-good reference that is never pushed speculatively.
5. Sends a Discord notification with the deploy status (requires the `DISCORD_WEBHOOK_URL` secret; silently skipped if not set).

---

## Secrets required

| Secret | Used by |
|---|---|
| `GITHUB_TOKEN` | GHCR login (provided automatically by GitHub Actions) |
| `SSH_HOST_BACKEND` | SSH target for the backend VM |
| `SSH_HOST_NGINX` | SSH target for the nginx VM |
| `VM_USER` | SSH username for both VMs |
| `AZURE_KEY` | SSH private key for both VMs |
| `DISCORD_WEBHOOK_URL` | Deploy notifications (optional) |

---

## Action version pinning

`actions/checkout` and `docker/login-action` are pinned to exact commit SHAs.
The remaining actions have `# TODO: pin to commit SHA` comments with instructions.
To get the SHA for any action tag, run:

```bash
gh api repos/<owner>/<repo>/git/ref/tags/<tag> --jq .object.sha
```
