# Legacy Project

[![CI](https://github.com/DenDanskeMetode/legacyProject/actions/workflows/ci.yml/badge.svg)](https://github.com/DenDanskeMetode/legacyProject/actions/workflows/ci.yml)
[![CD](https://github.com/DenDanskeMetode/legacyProject/actions/workflows/cd.yml/badge.svg)](https://github.com/DenDanskeMetode/legacyProject/actions/workflows/cd.yml)
[![Go](https://img.shields.io/badge/Go-1.26.3-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![Coverage](https://img.shields.io/badge/coverage-%E2%89%A580%25_enforced-brightgreen)](https://github.com/DenDanskeMetode/legacyProject/actions/workflows/ci.yml)
[![Docker](https://img.shields.io/badge/image-ghcr.io-2496ED?logo=docker&logoColor=white)](https://github.com/DenDanskeMetode/legacyProject/pkgs/container/legacyproject)

A Go rewrite of a legacy Python/Flask recipe cookbook application. The project serves both an HTML frontend and a JSON REST API, backed by PostgreSQL and deployed across four Azure VMs.

---

## Table of Contents

- [Tech Stack & Framework Choice](#tech-stack--framework-choice)
- [Architecture](#architecture)
- [Branching Strategy](#branching-strategy)
- [Running Locally](#running-locally)
- [Environment Variables](#environment-variables)
- [Infrastructure](#infrastructure)
- [CI/CD](#cicd)
- [Branch Protection](#branch-protection-master)
- [Definition of Done](#definition-of-done)

---

## Tech Stack & Framework Choice

| Layer | Technology |
|---|---|
| Language | Go 1.26.3 |
| HTTP | `net/http` (standard library) |
| Database | PostgreSQL 16 |
| Reverse proxy | Nginx |
| Monitoring | Prometheus + Grafana |
| Container registry | GHCR |
| Cloud | Azure (4 × Standard_B1s VMs) |

Go with `net/http` was chosen over the legacy Python/Flask stack for its performance and simplicity — it is compiled, statically typed, and the standard library HTTP server requires no external framework dependencies.

---

## Architecture

The application is split across four Azure VMs in a shared private VNet. Only the nginx VM has a public IP.

```
Internet
    │
    ▼
┌─────────────┐
│  nginx VM   │  (public IP, ports 80/443)
│  Nginx      │  reverse proxy → app, /grafana/ → monitoring
└──────┬──────┘
       │ private VNet (10.0.1.0/24)
       ├─────────────────────┐────────────────────┐
       ▼                     ▼                    ▼
┌─────────────┐     ┌──────────────┐    ┌──────────────────┐
│   app VM    │     │ postgres VM  │    │  monitoring VM   │
│  Go app     │────▶│ PostgreSQL   │    │ Prometheus        │
│  :3000      │     │  :5432       │    │ Grafana  :3001   │
└─────────────┘     └──────────────┘    └──────────────────┘
```

- The app VM and postgres VM are reachable only within the VNet — no public IP.
- The monitoring VM is also private; Grafana is exposed through nginx at `/grafana/`.
- All VMs are provisioned and configured by the IaC scripts in [`infrastructure/`](infrastructure/README.md).

---

## Branching Strategy

This project uses **GitHub Flow**:

1. Create a short-lived feature branch from `master` (e.g. `fix-healthcheck`, `add-monitoring`)
2. Open a Pull Request targeting `master`
3. CI must pass (`Go quality checks` + `build-and-push`) before merge
4. Merge to `master` triggers automatic deployment to production
5. Delete the feature branch after merge

Branch protection rules on `master` enforce that no direct pushes are allowed and all status checks must pass. See [Branch Protection](#branch-protection-master) for details.

---

## Running Locally

**Prerequisites:** Docker, Docker Compose, and a running PostgreSQL instance (or use the database compose file).

**1. Start the database**
```bash
cd database
cp .env.example .env   # fill in credentials
docker compose up -d
```

**2. Start the application**
```bash
cd app
cp .env.example .env   # fill in DB credentials to match step 1
docker compose up -d
```

The app is now available at `http://localhost:3000`.
Swagger UI is at `http://localhost:3000/swagger`.
Prometheus metrics are at `http://localhost:3000/metrics`.

**3. (Optional) Start monitoring**
```bash
cd monitoring
cp .env.example .env
docker compose up -d
```

Grafana is available at `http://localhost:3001`.

---

## Environment Variables

The following variables must be set for the application to run. In production they are written to `.env` on each VM by the CD pipeline from GitHub Secrets.

| Variable | Description |
|---|---|
| `DB_HOST` | PostgreSQL hostname |
| `DB_PORT` | PostgreSQL port (typically `5432`) |
| `DB_USER` | Database user |
| `DB_PASSWORD` | Database password |
| `DB_NAME` | Database name |

See each service's `.env.example` for the full list including monitoring and nginx variables.

---

## Infrastructure

Infrastructure is managed as code using bash scripts in [`infrastructure/`](infrastructure/README.md).

- **`azure-setup.sh`** — provisions all four VMs, configures networking and firewall rules, installs Docker, and writes all GitHub Actions secrets automatically.
- **`azure-teardown.sh`** — deletes the entire resource group and all associated resources.

Every group member should be able to run the full setup → deploy → teardown cycle independently. See [`infrastructure/README.md`](infrastructure/README.md) for prerequisites and usage.

---

## CI/CD

Two GitHub Actions workflows run on every push:

| Workflow | Trigger | Purpose |
|---|---|---|
| [`ci.yml`](.github/workflows/ci.yml) | Every push + PR | `go vet`, unit tests, integration tests, ≥80% coverage, `govulncheck`, `hadolint` |
| [`cd.yml`](.github/workflows/cd.yml) | Push to `master` + PR to `master` | Build images, Trivy CVE scan, push to GHCR, deploy to all four VMs, rollback on failure, Discord notification |

On pull requests, `cd.yml` builds and scans the image but does not deploy. Full deployment only happens on merge to `master`.

See [`.github/workflows/README.md`](.github/workflows/README.md) for a detailed breakdown of every step.

---

## Branch Protection (master)

A GitHub Ruleset is configured on `master` with the following rules:

- **Restrict deletions** — the branch cannot be deleted
- **Require a pull request before merging** — no direct pushes; all changes must go through a PR
- **Require status checks to pass** — `Go quality checks` (ci.yml) and `build-and-push` (cd.yml) must pass before a PR can be merged; branches must also be up to date with master
- **Block force pushes** — history on master cannot be rewritten

No bypass list is configured, so these rules apply to everyone including admins.

---

## Definition of Done

### 1. Code Quality & Standards
- **Peer Reviewed:** At least one peer has reviewed and approved the Pull Request on important changes.
- **Linting & Style:** Code passes all static analysis and linting checks with zero critical warnings.
- **No Technical Debt:** No temporary workarounds or "TODO" comments are introduced unless tracked in the backlog.

### 2. Testing Automation
- **Unit Tests:** Minimum test coverage threshold is met (≥80%), and all tests pass.
- **Integration Tests:** API endpoints and component interactions are validated automatically in the pipeline.
- **Security Scanning (DevSecOps):** SAST and dependency vulnerability scans run with zero "High" or "Critical" vulnerabilities.

### 3. Continuous Integration & Deployment (CI/CD)
- **Green Build:** The CI pipeline builds the artifact/container successfully without manual intervention.
- **Automated Deployment:** The artifact is automatically deployed.
- **Environment Parity:** The deployment uses the exact same configuration templates and scripts that will be used for production.

### 4. Observability & Operations
- **Telemetry:** Logging and metrics are implemented. Prometheus metrics exposed at `/metrics`, visualised in Grafana.

### 5. Product & Compliance
- **Acceptance Criteria:** The feature fulfils all acceptance criteria defined in the user story / issue.
- **Documentation:** User-facing documentation, API specs (Swagger/OpenAPI), and internal architecture are updated.
- **Feature Flags:** Not applicable at this project scale.
