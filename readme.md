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

## Definition of Done
### 1. Code Quality & Standards
Peer Reviewed: At least one peer has reviewed and approved the Pull Request (PR) on important changes.

Linting & Style: Code passes all static analysis and linting checks with zero critical warnings.

No Technical Debt: No temporary workarounds or "TODO" comments are introduced unless tracked in the backlog.

### 2. Testing Automation
Unit Tests: Minimum test coverage threshold is met 80%+, and all tests pass.

Integration Tests: API endpoints and component interactions are validated automatically in the pipeline.

Security Scanning (DevSecOps): Static Application Security Testing (SAST) and dependency vulnerability scans run with zero "High" or "Critical" vulnerabilities.

### 3. Continuous Integration & Deployment (CI/CD)
Green Build: The CI pipeline builds the artifact/container successfully without manual intervention.

Automated Deployment: The artifact is automatically deployed.

Environment Parity: The deployment uses the exact same configuration templates and scripts that will be used for production.

### 4. Observability & Operations
Telemetry: Logging, metrics, and distributed tracing are implemented following architectural standards.

### 5. Product & Compliance
Documentation: User-facing documentation, API specs (e.g., Swagger/OpenAPI), and internal architecture diagrams are updated. 

## Branch Protection (master)

A GitHub Ruleset is configured on `master` with the following rules:

- **Restrict deletions** — the branch cannot be deleted
- **Require a pull request before merging** — no direct pushes; all changes must go through a PR
- **Require status checks to pass** — `Go quality checks` (ci.yml) and `build-and-push` (cd.yml) must pass before a PR can be merged; branches must also be up to date with master
- **Block force pushes** — history on master cannot be rewritten

No bypass list is configured, so these rules apply to everyone including admins.

For details on what the CI/CD workflows do, see [.github/workflows/README.md](.github/workflows/README.md).