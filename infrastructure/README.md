# Infrastructure

## Prerequisites

| Tool | Purpose | Install |
|---|---|---|
| [Azure CLI](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli) | Create VMs | `winget install Microsoft.AzureCLI` |
| Azure subscription | Quota for 1 Standard public IP | Azure for Students works |
| SSH key at `~/.ssh/azure_key` | VM auth | `ssh-keygen -t rsa -b 4096 -f ~/.ssh/azure_key -N ""` (auto-generated if missing) |
| [GitHub CLI](https://cli.github.com/) (optional) | Set secrets automatically | `winget install GitHub.cli` then `gh auth login` |

## What the script does

Run once to provision infrastructure. **Not a deployment script** — containers are deployed by GitHub Actions on every push.

1. Creates a resource group, VNet, and two VMs in `norwayeast`
2. **nginx VM** — public IP, ports 22/80/443 open
3. **backend VM** — no public IP, port 8080 reachable within VNet only
4. Installs Docker on both VMs
5. Writes GitHub Actions secrets (`SSH_HOST_NGINX`, `SSH_HOST_BACKEND`, `BACKEND_PRIVATE_IP`, `SSH_USER`, `SSH_PRIVATE_KEY`)

If GitHub CLI is not installed, secrets are printed for manual entry in **Settings > Secrets and variables > Actions**.

## Usage

```bash
# From the infrastructure/ directory
bash azure-setup.sh

# SSH to nginx
ssh azureuser@<nginx-public-ip>

# SSH to backend (jump through nginx)
ssh -J azureuser@<nginx-public-ip> azureuser@<backend-private-ip>

# Tear everything down
bash azure-teardown.sh
```
