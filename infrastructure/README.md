# Infrastructure

## Prerequisites for `azure-setup.sh`

### 1. Azure CLI
Install from PowerShell or CMD (not Git Bash):
```powershell
winget install Microsoft.AzureCLI
```
Then restart your terminal and log in:
```bash
az login
```

### 2. An active Azure subscription
The script uses `Azure for Students` or any subscription with sufficient quota.
Ensure your subscription has room for at least **2 Public IP addresses** (Standard SKU).

### 3. SSH key at `~/.ssh/azure_key`
The script expects a key pair at `~/.ssh/azure_key` (private) and `~/.ssh/azure_key.pub` (public).
If you don't have one, generate it:
```bash
ssh-keygen -t rsa -b 4096 -f ~/.ssh/azure_key -N "" -C "azure-vm-cicd"
```
The script will generate this automatically if it doesn't exist.

### 4. GitHub CLI (optional, for setting secrets automatically)
Install from PowerShell or CMD:
```powershell
winget install GitHub.cli
```
Then authenticate:
```bash
gh auth login
```
If not installed, the script will print the secrets you need to set manually in your repo's **Settings > Secrets and variables > Actions**.

### 5. Docker (on the Azure VMs)
Docker does not need to be installed locally. The script will check if Docker is already installed on each VM and install it if not.

---

## Running the script

```bash
bash azure-setup.sh
```

Run from Git Bash inside the `infrastructure/` directory. The script will:
1. Create a resource group, VNet, and two VMs (nginx + backend) in Azure
2. Configure network security rules
3. Install Docker on each VM if not already present
4. Deploy the Go app on the backend VM and nginx on the nginx VM
5. Set GitHub Actions secrets for CI/CD

To tear everything down afterwards:
```bash
./azure-teardown.sh
```
