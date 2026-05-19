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