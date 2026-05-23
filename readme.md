# Legacy Project (needs to be updated)

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

A GitHub Ruleset is configured on `master` with the following rules:

- **Restrict deletions** — the branch cannot be deleted
- **Require a pull request before merging** — no direct pushes; all changes must go through a PR
- **Require status checks to pass** — `Go quality checks` (ci.yml) and `build-and-push` (cd.yml) must pass before a PR can be merged; branches must also be up to date with master
- **Block force pushes** — history on master cannot be rewritten

No bypass list is configured, so these rules apply to everyone including admins.

For details on what the CI/CD workflows do, see [.github/workflows/README.md](.github/workflows/README.md).