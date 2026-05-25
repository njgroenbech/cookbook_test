# Infrastructure

![Azure Architecture](../docs/azure-architecture.png)

## Prerequisites

> **Windows:** These are bash scripts. Run them in **WSL** or **Git Bash** — not PowerShell or Command Prompt.

| Tool | Purpose | Install |
|---|---|---|
| [Azure CLI](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli) | Create VMs | Windows: `winget install Microsoft.AzureCLI`<br>macOS: `brew install azure-cli`<br>Linux: `curl -sL https://aka.ms/InstallAzureCLIDeb &#124; sudo bash` |
| Azure subscription | Quota for 1 Standard public IP | Azure for Students works |
| `openssl` | Generate DB/Grafana passwords | Pre-installed on macOS/Linux; included in Git Bash / WSL on Windows |
| SSH key at `~/.ssh/azure_key` | VM auth | Auto-generated if missing |
| [GitHub CLI](https://cli.github.com/) (optional) | Set secrets automatically | Windows: `winget install GitHub.cli`<br>macOS: `brew install gh`<br>Linux: `sudo apt install gh`<br>Then run `gh auth login` |

## What `azure-setup.sh` does

Run once to provision infrastructure and perform the initial container deployment. Subsequent deployments are handled by GitHub Actions on every push.

1. Creates resource group `recipe-cookbook-backup` and a shared VNet (`10.0.0.0/16`) in `norwayeast`
2. Provisions four VMs (all `Standard_B1s`, Ubuntu 22.04):
   - **nginx VM** — public IP, ports 22/80/443 open
   - **app VM** — no public IP, port 3000 reachable within VNet only
   - **postgres VM** — no public IP, port 5432 reachable from app VM only
   - **monitoring VM** — no public IP, ports 9090/3001 reachable within VNet only
3. Generates random DB credentials and Grafana admin password
4. Prompts for a GHCR token (or reads from `gh auth token`) and an optional GitHub PAT for cloning
5. Installs Docker on all four VMs; installs node_exporter (port 9100) on nginx, app, and postgres VMs; clones the repo on all VMs
6. Deploys PostgreSQL → waits for it to be ready → deploys the Go app → deploys nginx → deploys Prometheus + Grafana (with generated `prometheus.yml` scraping app and all node_exporter targets)
7. Runs a quick end-to-end check (postgres, backend HTTP, nginx public IP)
8. Sets GitHub Actions secrets via `gh secret set` (or prints them for manual entry)

GitHub secrets written: `VM_USER`, `SSH_HOST_NGINX`, `SSH_HOST_NGINX_PRIVATE`, `SSH_HOST_APP`, `SSH_HOST_POSTGRES`, `SSH_HOST_MONITORING`, `SSH_PROXY_HOST`, `AZURE_KEY`, `DB_HOST`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `GRAFANA_PASSWORD`

## What `azure-teardown.sh` does

Deletes all resources in `recipe-cookbook-backup` in order: VMs → NICs → disks (background, `--no-wait`) → NSGs → VNet. Non-interactive — no confirmation prompt.

## Azure Region

The setup script auto-detects which region to use — you do not need to configure anything manually beforehand. When it runs, it checks your account against a list of preferred European regions (`norwayeast`, `swedencentral`, `northeurope`, `westeurope`) and handles it as follows:

| Scenario | What happens |
|---|---|
| Exactly one preferred region is available | Auto-selected — no prompt |
| Multiple preferred regions are available | Script shows a numbered list, you pick one |
| None of the preferred regions are available | Script prints **all** regions available on your account and prompts you to type one |

The third case is the most common when using accounts with restricted quotas (e.g. enterprise accounts with policy restrictions, or Azure for Students accounts where some regions are blocked). To check your available regions ahead of time, run this after `az login`:

```bash
az account list-locations --query "[].{Name:name, DisplayName:displayName}" --output table
```

Pick any region from the `Name` column where you have quota for 4 × `Standard_B1s` VMs and 1 Standard public IP. `Standard_B1s` is a small general-purpose VM available in virtually all regions — if a region appears in the list, it will almost certainly work.

## Usage

The full setup takes **20–40 minutes** on a standard connection (VM provisioning ~5 min per VM, Docker installation and container startup account for the rest).

```bash
# Provision everything
bash azure-setup.sh

# Disable color output (useful in CI / non-TTY terminals)
bash azure-setup.sh --no-colors

# SSH access
ssh azureuser@<nginx-public-ip>
ssh -J azureuser@<nginx-public-ip> azureuser@<app-private-ip>
ssh -J azureuser@<nginx-public-ip> azureuser@<postgres-private-ip>
ssh -J azureuser@<nginx-public-ip> azureuser@<monitoring-private-ip>

# Tear everything down
bash azure-teardown.sh
```

> If the app VM's private IP ever changes (e.g. VM recreated), re-run `azure-setup.sh` so nginx.conf is regenerated with the correct upstream address.
