# Course Checklist: EK ITA Agil Cloud 2026 Spring

Derived from every week's README, learning goals, and exercises in the
[course repository](https://github.com/cookbookio/EK_ITA_Agil_Cloud_Ita_2026_Spring),
plus `exam_project_requirements.md` and `semester_overview.md`.

**Type**: MANDATORY = must be completed for exam. OPTIONAL = explicitly marked as optional in course materials.

- ✅ Done
- ⚠️ Partial / needs verification
- ❌ Missing

---

## 1. Tech Stack

| # | Item | Type | Status |
|---|------|------|--------|
| 1.1 | Server rewritten in a new framework (NOT Flask, Express, or Spring Boot) | MANDATORY | ✅ Go `net/http` |
| 1.2 | Framework choice argued and documented | MANDATORY | ✅ Documented in README (performance + simplicity, compiled, no external dependencies) |

---

## 2. Version Control & Organisation

| # | Item | Type | Status |
|---|------|------|--------|
| 2.1 | Git repository in use | MANDATORY | ✅ Yes |
| 2.2 | Branching strategy chosen (Git Flow / GitHub Flow / Trunk-Based) | MANDATORY | ✅ GitHub Flow |
| 2.3 | Branching strategy documented in README | MANDATORY | ✅ Documented in README |
| 2.4 | Issue management system (GitHub Issues) | MANDATORY | ✅ Issues managed in Kanban board |
| 2.5 | GitHub Issue template | MANDATORY | ✅ Bug report and feature request |
| 2.6 | GitHub PR template | MANDATORY | ✅ `.github/pull_request_template.md` exists |
| 2.7 | Repository URL submitted in `groups.py` (course repo) | MANDATORY | ⚠️ Cannot verify from code |
| 2.8 | Kanban board (GitHub Project) | MANDATORY | ✅ Done |
| 2.9 | Thorough documentation (README, architecture, how to run) | MANDATORY | ✅ README covers architecture, branching, how to run locally, env vars, IaC, CI/CD |

---

## 3. OpenAPI / Swagger

| # | Item | Type | Status |
|---|------|------|--------|
| 3.1 | OpenAPI specification file | MANDATORY | ✅ `app/api-schema.yaml` |
| 3.2 | Swagger UI served by the application | MANDATORY | ✅ `/swagger` endpoint in `main.go` |

---

## 4. GitHub Actions — CI (Continuous Integration)

| # | Item | Type | Status |
|---|------|------|--------|
| 4.1 | GitHub Actions workflow file | MANDATORY | ✅ `ci.yml` + `cd.yml` |
| 4.2 | Workflow triggers on push and pull request | MANDATORY | ✅ Both workflows cover push + PR |
| 4.3 | GitHub Secrets configured | MANDATORY | ✅ All secrets documented in `workflows/README.md` |
| 4.4 | Branch protection rules on `master` | MANDATORY | ✅ Documented in `readme.md` (ruleset requiring PR + status checks) |

---

## 5. Code Quality & Linting

| # | Item | Type | Status |
|---|------|------|--------|
| 5.1 | Linter integrated into GitHub Actions | MANDATORY | ✅ `go vet` runs in `ci.yml` |
| 5.2 | Dockerfile linting | MANDATORY | ✅ `hadolint` runs in `ci.yml` |
| 5.3 | Vulnerability scan for Go dependencies | MANDATORY | ✅ `govulncheck` runs in `ci.yml` |
| 5.4 | Container image vulnerability scan | MANDATORY | ✅ Trivy in `cd.yml` (CRITICAL/HIGH CVEs block deploy) |
| 5.5 | README badges (build status, coverage, etc.) | MANDATORY | ✅ CI, CD, Go version, coverage threshold, Docker image badges in `readme.md` |
| 5.6 | SonarQube / Code Climate | OPTIONAL | ❌ Not configured |

---

## 6. Testing

| # | Item | Type | Status |
|---|------|------|--------|
| 6.1 | Unit tests | MANDATORY | ✅ `handlers_test.go`, `templates_test.go`, `main_test.go` |
| 6.2 | Integration tests (real DB) | MANDATORY | ✅ `handlers_integration_test.go` with Postgres service in CI |
| 6.3 | Test coverage threshold enforced | MANDATORY | ✅ 80% minimum enforced in `ci.yml` |
| 6.4 | Race condition detection | MANDATORY | ✅ `-race` flag in `go test` |

---

## 7. Pre-commit Hooks (Git Hooks)

| # | Item | Type | Status |
|---|------|------|--------|
| 7.1 | Pre-commit hooks configured in project | MANDATORY | ✅ `lefthook.yml` + `lefthook install` documented in README |
| 7.2 | Linter or tests run automatically before commit | MANDATORY | ✅ `golangci-lint` + `go test -race` run in parallel on staged `app/**/*.go` files |

---

## 8. Azure Cloud & Deployment

| # | Item | Type | Status |
|---|------|------|--------|
| 8.1 | Application deployed to Azure VM(s) | MANDATORY | ✅ 4 VMs (nginx, app, postgres, monitoring) |
| 8.2 | SSH access with key pair | MANDATORY | ✅ Configured in IaC scripts |
| 8.3 | Static / public IP for nginx VM | MANDATORY | ✅ nginx VM has public IP |
| 8.4 | Necessary ports opened (80, 443, app port) | MANDATORY | ✅ Configured in `azure-setup.sh` |
| 8.5 | System deployed for entire exam period | MANDATORY | ✅ Yes |

---

## 9. Continuous Delivery / Continuous Deployment

| # | Item | Type | Status |
|---|------|------|--------|
| 9.1 | Automated Docker image build on push to master | MANDATORY | ✅ `cd.yml` → `build-and-push` |
| 9.2 | Image pushed to GHCR.io | MANDATORY | ✅ Two images: `legacyproject` + `legacyproject-nginx` |
| 9.3 | Automated deployment to VM(s) after successful build | MANDATORY | ✅ `cd.yml` → `deploy` job with matrix across 4 VMs |
| 9.4 | Healthcheck-gated deploy with rollback | MANDATORY | ✅ `--wait` + rollback to `latest-stable` |
| 9.5 | Concurrency lock (no racing deploys) | MANDATORY | ✅ `concurrency` group in `cd.yml` |
| 9.6 | PR builds image and scans without deploying | MANDATORY | ✅ PR-only path in `cd.yml` |

---

## 10. Nginx — Reverse Proxy

| # | Item | Type | Status |
|---|------|------|--------|
| 10.1 | Nginx reverse proxy configured | MANDATORY | ✅ `network/nginx.conf.template` |
| 10.2 | Nginx proxies traffic to backend app | MANDATORY | ✅ `upstream app` block |
| 10.3 | Nginx proxies Grafana under `/grafana/` | MANDATORY | ✅ `upstream grafana` block |
| 10.4 | Nginx deployed as its own Docker container | MANDATORY | ✅ `network/Dockerfile` + `docker-compose.yaml` |
| 10.5 | Load balancing strategy known/documented | MANDATORY | ⚠️ Single upstream only — no multi-instance load balancing; discuss in report |

---

## 11. Infrastructure as Code (IaC)

| # | Item | Type | Status |
|---|------|------|--------|
| 11.1 | `infrastructure/` folder at repo root | MANDATORY | ✅ Exists |
| 11.2 | Setup script (`azure-setup.sh`) | MANDATORY | ✅ Creates 4 VMs, VNet, installs Docker, sets GitHub secrets |
| 11.3 | Teardown script (`azure-teardown.sh`) | MANDATORY | ✅ Deletes entire resource group |
| 11.4 | Infrastructure `README.md` with prerequisites + usage | MANDATORY | ✅ `infrastructure/README.md` |
| 11.5 | Scripts committed to master branch | MANDATORY | ✅ Yes |
| 11.6 | nginx VM public-facing; backend VMs private-only | MANDATORY | ✅ nginx has public IP; app/postgres/monitoring have no public IP |
| 11.7 | NSG rules restrict backend access to nginx VM only | MANDATORY | ⚠️ Provisioned by `azure-setup.sh` — verify by running the full cycle |
| 11.8 | All group members can independently run full cycle | MANDATORY | ⚠️ Runtime concern — each member must test setup → deploy → teardown |
| 11.9 | Scripts work without modification for all team members | MANDATORY | ⚠️ Runtime concern |

---

## 12. Monitoring & Logging

| # | Item | Type | Status |
|---|------|------|--------|
| 12.1 | Prometheus running alongside application | MANDATORY | ✅ `monitoring/docker-compose.yaml` |
| 12.2 | Application exposes `/metrics` endpoint | MANDATORY | ✅ `promhttp.Handler()` at `/metrics` in `main.go` |
| 12.3 | Custom app metrics (request count, duration, DB query duration) | MANDATORY | ✅ Three metric vectors registered in `main.go` |
| 12.4 | Prometheus scrapes app metrics | MANDATORY | ✅ `prometheus.yml` uses `${APP_HOST}`/`${NGINX_PRIVATE_HOST}`/`${DB_HOST}` env vars; `cd.yml` renders the file via `envsubst` before copying it to the monitoring VM |
| 12.5 | Grafana dashboard configured | MANDATORY | ✅ Grafana service in `monitoring/docker-compose.yaml`; proxied via nginx |
| 12.6 | Monitoring deployed to its own VM | MANDATORY | ✅ Dedicated monitoring VM in IaC setup |
| 12.7 | node_exporter for system metrics | MANDATORY | ✅ Installed as a standalone Docker container on nginx, app, and postgres VMs by `azure-setup.sh` (port 9100); not in docker-compose by design — managed by IaC |

---

## 13. Deployment Strategy & SLA

| # | Item | Type | Status |
|---|------|------|--------|
| 13.1 | Deployment strategy chosen and documented (blue-green / canary / rolling) | MANDATORY | ⚠️ Documented in report. Needs to be explained in README as well. |
| 13.2 | Scaling and optimal deployment strategy discussed | MANDATORY | ❌ Not documented in repo |
| 13.3 | SLA (Service Level Agreement) written and published | MANDATORY | ⚠️ In report, not in README. |
| 13.4 | Definition of done documented | MANDATORY | ✅ Documented in README |
| 13.5 | Downtime / fault tolerance testing | OPTIONAL | ❌ Not done |

---

## 14. DevOps Culture & Process

| # | Item | Type | Status |
|---|------|------|--------|
| 14.1 | Commit activity reflects all group members participating | MANDATORY | ⚠️ Cannot verify from code — check GitHub contribution graphs |
| 14.2 | DevOps maturity assessment done (Week 19 exercise) | MANDATORY | ⚠️ Cannot verify from code |
| 14.3 | Psychological safety / group reflection prepared for exam | MANDATORY | ⚠️ Exam discussion requirement |

---

## Gap Summary — Prioritised Backlog

### Medium priority (needs verification or small fixes)
4. **⚠️ 13.1/13.2 — Deployment strategy**: document the chosen strategy (rolling update via Docker Compose `--wait`), its trade-offs, and scaling considerations
5. **⚠️ 13.3 — SLA**: write a simple SLA (uptime target, response time goal, RTO)
11. **⚠️ 2.7 — groups.py**: verify group entry is filled out in the course repository