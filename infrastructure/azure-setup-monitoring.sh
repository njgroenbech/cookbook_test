#!/bin/bash

# Standalone Monitoring VM Deployment Script
# Creates a single monitoring VM (Prometheus + Grafana) with a public IP.
#   - monitoring VM : public-facing SSH (port 22) and Grafana (port 3001); Prometheus (9090) accessible within VNet only
# Deploys the monitoring stack (prom/prometheus + grafana/grafana) from ./monitoring/docker-compose.yaml.
#
# Independent of the other VMs in azure-setup.sh — creates resource group and VNet if missing,
# reuses them if they already exist. Prompts for the Prometheus scrape target (app VM IP).
#
# Usage: ./azure-setup-monitoring.sh [--no-colors]

set -e
export MSYS_NO_PATHCONV=1

NO_COLORS=false
while [[ $# -gt 0 ]]; do
    case "$1" in
        --help|-h)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --no-colors    Disable colored output"
            echo "  --help, -h     Show this help message"
            exit 0
            ;;
        --no-colors)
            NO_COLORS=true
            shift
            ;;
        *)
            shift
            ;;
    esac
done

# Configuration variables - CUSTOMIZE THESE
RESOURCE_GROUP="recipe-cookbook-backup"
LOCATION="norwayeast"
MONITORING_VM_NAME="recipe-cookbook-monitoring-vm"
VM_SIZE="Standard_D2s_v5"
VM_IMAGE="Canonical:0001-com-ubuntu-server-jammy:22_04-lts:latest"
ADMIN_USERNAME="azureuser"
SSH_KEY_PATH="$HOME/.ssh/azure_key.pub"
VNET_NAME="recipe-cookbook-vnet"
SUBNET_NAME="recipe-cookbook-subnet"
GITHUB_REPO="dendanskemetode/legacyproject"
REMOTE_APP_DIR="/home/azureuser/legacyProject"

# Runtime variables (populated during execution)
GIT_CLONE_URL=""
GRAFANA_PASSWORD=""
APP_PRIVATE_IP=""
MONITORING_PUBLIC_IP=""
MONITORING_PRIVATE_IP=""

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

if [ "$NO_COLORS" = true ]; then
    GREEN=''
    YELLOW=''
    RED=''
    NC=''
fi

echo "=========================================="
echo "Monitoring VM Deployment — Recipe Cookbook"
echo "Single-VM deployment: Prometheus + Grafana"
echo "=========================================="
echo ""

# ==========================================
# Pre-flight checks
# ==========================================

if ! command -v az &> /dev/null; then
    echo -e "${RED}❌ Azure CLI is not installed${NC}"
    echo "Install it from: https://docs.microsoft.com/en-us/cli/azure/install-azure-cli"
    exit 1
fi
echo -e "${GREEN}✅ Azure CLI is installed${NC}"

echo ""
echo "Checking Azure login status..."
if ! az account show &> /dev/null; then
    echo -e "${YELLOW}⚠️  Not logged in to Azure${NC}"
    echo "Logging in..."
    az login
else
    echo -e "${GREEN}✅ Already logged in to Azure${NC}"
    ACCOUNT=$(az account show --query name -o tsv)
    echo "Using subscription: $ACCOUNT"
fi

echo ""
if [ ! -f "$SSH_KEY_PATH" ]; then
    echo -e "${YELLOW}⚠️  SSH key not found at $SSH_KEY_PATH${NC}"
    echo "Generating new SSH key..."
    ssh-keygen -t rsa -b 4096 -f "${SSH_KEY_PATH%.pub}" -N "" -C "azure-vm-cicd"
    echo -e "${GREEN}✅ SSH key generated${NC}"
else
    echo -e "${GREEN}✅ SSH key found at $SSH_KEY_PATH${NC}"
fi

if ! command -v openssl &> /dev/null; then
    echo -e "${RED}❌ openssl is required for credential generation${NC}"
    exit 1
fi
echo -e "${GREEN}✅ openssl found${NC}"

# ==========================================
# Credential and token acquisition
# ==========================================

echo ""
echo "=========================================="
echo "Generating Grafana Admin Password"
echo "=========================================="
GRAFANA_PASSWORD=$(openssl rand -base64 32 | tr -dc 'a-zA-Z0-9' | head -c 32)
echo -e "${GREEN}✅ Grafana admin password generated (not shown)${NC}"

echo ""
echo "=========================================="
echo "Git Repository Access"
echo "=========================================="

if command -v gh &> /dev/null && gh auth status &> /dev/null 2>&1; then
    GIT_TOKEN=$(gh auth token)
    GIT_CLONE_URL="https://oauth2:${GIT_TOKEN}@github.com/$GITHUB_REPO.git"
    echo -e "${GREEN}✅ Using GitHub CLI authentication${NC}"
else
    echo -e "${RED}❌ GitHub CLI not authenticated${NC}"
    echo "Run: gh auth login"
    exit 1
fi

echo ""
echo "=========================================="
echo "Prometheus Scrape Target"
echo "=========================================="
read -p "Enter the app VM IP for Prometheus to scrape (e.g. 10.0.1.5): " APP_PRIVATE_IP
if [ -z "$APP_PRIVATE_IP" ]; then
    echo -e "${RED}❌ app VM IP is required${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Prometheus will scrape $APP_PRIVATE_IP:3000${NC}"

echo ""
echo "=========================================="
echo "Git Repository Access"
echo "=========================================="
read -s -p "GitHub PAT with repo scope (press Enter if repo is public): " GIT_TOKEN
echo
if [ -n "$GIT_TOKEN" ]; then
    GIT_CLONE_URL="https://oauth2:${GIT_TOKEN}@github.com/$GITHUB_REPO.git"
    echo -e "${GREEN}✅ Git clone will use provided PAT${NC}"
else
    GIT_CLONE_URL="https://github.com/$GITHUB_REPO.git"
    echo -e "${GREEN}✅ Git clone will use public access${NC}"
fi

# ==========================================
# Resource group
# ==========================================

echo ""
echo "=========================================="
echo "Resource Group"
echo "=========================================="
echo "Name: $RESOURCE_GROUP"
echo "Location: $LOCATION"

if az group exists --name "$RESOURCE_GROUP" | grep -q "true"; then
    echo -e "${GREEN}✅ Resource group already exists, reusing${NC}"
else
    az group create \
        --name "$RESOURCE_GROUP" \
        --location "$LOCATION" \
        --output table
    echo -e "${GREEN}✅ Resource group created${NC}"
fi

# ==========================================
# Virtual Network
# ==========================================

echo ""
echo "=========================================="
echo "Virtual Network"
echo "=========================================="
echo "VNet: $VNET_NAME"
echo "Subnet: $SUBNET_NAME"

if ! az network vnet show \
        --resource-group "$RESOURCE_GROUP" \
        --name "$VNET_NAME" &>/dev/null; then
    az network vnet create \
        --resource-group "$RESOURCE_GROUP" \
        --name "$VNET_NAME" \
        --address-prefix 10.0.0.0/16 \
        --subnet-name "$SUBNET_NAME" \
        --subnet-prefix 10.0.1.0/24 \
        --output table
    echo -e "${GREEN}✅ Virtual network created${NC}"
else
    echo -e "${GREEN}✅ Virtual network already exists, reusing${NC}"
fi

# ==========================================
# Create monitoring VM
# ==========================================

echo ""
echo "=========================================="
echo "Creating monitoring VM (public-facing SSH)"
echo "=========================================="
echo "VM Name: $MONITORING_VM_NAME"
echo "Size: $VM_SIZE"

if az vm show --resource-group "$RESOURCE_GROUP" --name "$MONITORING_VM_NAME" &>/dev/null; then
    echo -e "${YELLOW}⚠️  monitoring VM already exists${NC}"
    read -p "Do you want to delete and recreate it? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo "Deleting existing monitoring VM..."
        az vm delete --resource-group "$RESOURCE_GROUP" --name "$MONITORING_VM_NAME" --yes
        echo "monitoring VM deleted."
    else
        echo -e "${RED}❌ Aborting — cannot proceed with existing VM${NC}"
        exit 1
    fi
fi

echo "This may take 2-5 minutes..."
echo ""

az vm create \
    --resource-group "$RESOURCE_GROUP" \
    --name "$MONITORING_VM_NAME" \
    --image "$VM_IMAGE" \
    --size "$VM_SIZE" \
    --admin-username "$ADMIN_USERNAME" \
    --ssh-key-values "$(cat "$SSH_KEY_PATH")" \
    --vnet-name "$VNET_NAME" \
    --subnet "$SUBNET_NAME" \
    --public-ip-sku Standard \
    --output table

echo -e "${GREEN}✅ monitoring VM created${NC}"

# ==========================================
# Network security
# ==========================================

echo ""
echo "=========================================="
echo "Configuring Network Security"
echo "=========================================="
echo "monitoring VM: adding NSG rules — allow TCP 9090 from VNet, TCP 3001 from Internet"

MONITORING_NIC_ID=$(az vm show \
    --resource-group "$RESOURCE_GROUP" \
    --name "$MONITORING_VM_NAME" \
    --query "networkProfile.networkInterfaces[0].id" \
    --output tsv)

MONITORING_NSG=$(az network nic show --ids "$MONITORING_NIC_ID" \
    --query "networkSecurityGroup.id" \
    --output tsv | xargs basename)

if ! az network nsg rule show \
        --resource-group "$RESOURCE_GROUP" \
        --nsg-name "$MONITORING_NSG" \
        --name "AllowVnetPrometheus" &>/dev/null; then
    az network nsg rule create \
        --resource-group "$RESOURCE_GROUP" \
        --nsg-name "$MONITORING_NSG" \
        --name "AllowVnetPrometheus" \
        --priority 200 \
        --source-address-prefixes "10.0.1.0/24" \
        --destination-port-ranges 9090 \
        --protocol Tcp \
        --access Allow \
        --direction Inbound \
        --output table
fi

if ! az network nsg rule show \
        --resource-group "$RESOURCE_GROUP" \
        --nsg-name "$MONITORING_NSG" \
        --name "AllowPublicGrafana" &>/dev/null; then
    az network nsg rule create \
        --resource-group "$RESOURCE_GROUP" \
        --nsg-name "$MONITORING_NSG" \
        --name "AllowPublicGrafana" \
        --priority 210 \
        --source-address-prefixes "*" \
        --destination-port-ranges 3001 \
        --protocol Tcp \
        --access Allow \
        --direction Inbound \
        --output table
fi

echo -e "${GREEN}✅ monitoring VM NSG rules created (9090 from VNet, 3001 public)${NC}"

# ==========================================
# Get VM IPs
# ==========================================

echo ""
echo "=========================================="
echo "Getting VM Information"
echo "=========================================="

MONITORING_PUBLIC_IP=$(az vm show \
    --resource-group "$RESOURCE_GROUP" \
    --name "$MONITORING_VM_NAME" \
    --show-details \
    --query publicIps \
    --output tsv)

MONITORING_PRIVATE_IP=$(az vm show \
    --resource-group "$RESOURCE_GROUP" \
    --name "$MONITORING_VM_NAME" \
    --show-details \
    --query privateIps \
    --output tsv)

echo ""
echo -e "monitoring VM — Public IP  : ${GREEN}$MONITORING_PUBLIC_IP${NC}"
echo -e "monitoring VM — Private IP : ${GREEN}$MONITORING_PRIVATE_IP${NC}"
echo ""
echo "Prometheus → app : $APP_PRIVATE_IP:3000"

# ==========================================
# Wait for VM
# ==========================================

echo ""
echo "Waiting for VM to reach provisioned state..."
az vm wait --resource-group "$RESOURCE_GROUP" --name "$MONITORING_VM_NAME" --created
echo "VM provisioned. Waiting for SSH to become available..."
sleep 30

# ==========================================
# Helper: update system on the VM
# ==========================================

setup_vm() {
    local VM_IP="$1"
    local VM_LABEL="$2"
    local SSH_KEY="${SSH_KEY_PATH%.pub}"
    # Host keys are unknown for freshly created VMs; strict checking is disabled intentionally.
    local -a SSH_OPTS=(-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=30 -i "$SSH_KEY")

    echo ""
    echo "=========================================="
    echo "Connecting to $VM_LABEL VM and updating system"
    echo "=========================================="

    local attempt=0
    while [ $attempt -lt 3 ]; do
        attempt=$((attempt + 1))
        if ssh "${SSH_OPTS[@]}" "$ADMIN_USERNAME@$VM_IP" "
            set -e
            echo 'Connected to VM, starting system update...'
            sudo apt update -y
            sudo apt upgrade -y
            sudo apt install -y curl wget git unzip
            sudo apt autoremove -y
            sudo apt autoclean
            echo 'System update complete.'
        "; then
            echo -e "${GREEN}✅ $VM_LABEL VM system update completed${NC}"
            return 0
        else
            echo -e "${YELLOW}⚠️  SSH connection to $VM_LABEL VM failed (attempt $attempt/3).${NC}"
            if [ $attempt -lt 3 ]; then
                echo "Retrying in 20s..."
                sleep 20
            fi
        fi
    done
    echo -e "${RED}❌ Failed to connect to $VM_LABEL VM after 3 attempts.${NC}"
    exit 1
}

setup_vm "$MONITORING_PUBLIC_IP" "monitoring"

# ==========================================
# Install Docker
# ==========================================

install_docker() {
    local VM_IP="$1"
    local VM_LABEL="$2"
    local SSH_KEY="${SSH_KEY_PATH%.pub}"
    local -a SSH_OPTS=(-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=30 -i "$SSH_KEY")

    local UFW_PORTS="9090 3001"

    echo ""
    echo "=========================================="
    echo "Installing Docker on $VM_LABEL VM"
    echo "=========================================="

    # Outer heredoc is unquoted (ENDSSH) so $REMOTE_APP_DIR and $UFW_PORTS expand locally.
    ssh "${SSH_OPTS[@]}" "$ADMIN_USERNAME@$VM_IP" << ENDSSH
set -e
echo "Updating package index..."
sudo apt update

echo "Installing prerequisites..."
sudo apt install -y apt-transport-https ca-certificates curl software-properties-common

echo "Adding Docker GPG key..."
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --batch --yes --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg

echo "Adding Docker repository..."
echo "deb [arch=amd64 signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu \$(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

echo "Installing Docker and Docker Compose..."
sudo apt update
sudo apt install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin

echo "Configuring firewall..."
sudo ufw --force enable
sudo ufw allow 22/tcp
for port in $UFW_PORTS; do
    sudo ufw allow \${port}/tcp
done

echo "Creating app directory..."
mkdir -p $REMOTE_APP_DIR

echo "Docker installation complete!"
docker --version
docker compose version
ENDSSH

    echo -e "${GREEN}✅ Docker installed on $VM_LABEL VM${NC}"
}

install_docker "$MONITORING_PUBLIC_IP" "monitoring"

echo -e "${YELLOW}⚠️  Note: docker group changes require re-login; all docker commands below use sudo${NC}"

# ==========================================
# Clone repo
# ==========================================

clone_repo() {
    local VM_IP="$1"
    local VM_LABEL="$2"
    local SSH_KEY="${SSH_KEY_PATH%.pub}"
    local -a SSH_OPTS=(-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=30 -i "$SSH_KEY")

    echo ""
    echo "Cloning repo on $VM_LABEL VM..."

    ssh "${SSH_OPTS[@]}" "$ADMIN_USERNAME@$VM_IP" "
        set -e
        if [ -d '$REMOTE_APP_DIR/.git' ]; then
            echo 'Repo already present, pulling latest...'
            cd '$REMOTE_APP_DIR' && git pull
        else
            git clone '$GIT_CLONE_URL' '$REMOTE_APP_DIR'
        fi
        echo 'Repo ready.'
    "
    echo -e "${GREEN}✅ Repo cloned on $VM_LABEL VM${NC}"
}

clone_repo "$MONITORING_PUBLIC_IP" "monitoring"

# ==========================================
# Deploy monitoring VM
# ==========================================

echo ""
echo "=========================================="
echo "Deploying Monitoring (Prometheus + Grafana)"
echo "=========================================="

SSH_KEY="${SSH_KEY_PATH%.pub}"

ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=30 -i "$SSH_KEY" \
    "$ADMIN_USERNAME@$MONITORING_PUBLIC_IP" << ENDSSH
set -e
cd $REMOTE_APP_DIR/monitoring

cat > .env << ENVEOF
GF_SECURITY_ADMIN_PASSWORD=$GRAFANA_PASSWORD
ENVEOF

cat > prometheus.yml << PROMEOF
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: "ultimate-bravery-cookbook"
    static_configs:
      - targets: ["$APP_PRIVATE_IP:3000"]
PROMEOF

echo "Removing any existing containers..."
sudo docker compose down --remove-orphans 2>/dev/null || true

echo "Starting monitoring containers..."
sudo docker compose up -d --force-recreate
echo "Monitoring containers started."
ENDSSH

echo -e "${GREEN}✅ Monitoring deployed${NC}"

# ==========================================
# Verify deployment
# ==========================================

echo ""
echo "=========================================="
echo "Verifying Deployment"
echo "=========================================="

echo -n "Grafana responding (2xx/3xx)... "
GRAFANA_STATUS=$(ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=10 -i "$SSH_KEY" \
    "$ADMIN_USERNAME@$MONITORING_PUBLIC_IP" \
    "curl -s -o /dev/null -w '%{http_code}' http://localhost:3001/login"; true)
GRAFANA_STATUS="${GRAFANA_STATUS:-000}"
if [[ "$GRAFANA_STATUS" =~ ^[23] ]]; then
    echo -e "${GREEN}✅ OK (HTTP $GRAFANA_STATUS)${NC}"
else
    echo -e "${RED}❌ FAIL (HTTP $GRAFANA_STATUS — Grafana may still be starting, try again in a moment)${NC}"
fi

echo -n "Prometheus responding (2xx/3xx)... "
PROM_STATUS=$(ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=10 -i "$SSH_KEY" \
    "$ADMIN_USERNAME@$MONITORING_PUBLIC_IP" \
    "curl -s -o /dev/null -w '%{http_code}' http://localhost:9090/-/ready"; true)
PROM_STATUS="${PROM_STATUS:-000}"
if [[ "$PROM_STATUS" =~ ^[23] ]]; then
    echo -e "${GREEN}✅ OK (HTTP $PROM_STATUS)${NC}"
else
    echo -e "${RED}❌ FAIL (HTTP $PROM_STATUS — Prometheus may still be starting, try again in a moment)${NC}"
fi

# ==========================================
# Set GitHub Secrets
# ==========================================

echo ""
echo "=========================================="
echo "Setting GitHub Secrets"
echo "=========================================="

if ! command -v gh &> /dev/null; then
    echo -e "${YELLOW}⚠️  GitHub CLI is not installed${NC}"
    echo "Install it from: https://cli.github.com/"
    echo ""
    echo "Set these secrets manually in repository Settings > Secrets and variables > Actions:"
    echo ""
    echo "  SSH_HOST_MONITORING = $MONITORING_PUBLIC_IP"
    echo "  GRAFANA_PASSWORD    = (generated randomly — re-run the script to provision new credentials)"
    echo ""
else
    if ! gh auth status &> /dev/null; then
        echo -e "${YELLOW}⚠️  Not authenticated with GitHub CLI${NC}"
        gh auth login
    fi

    echo "Setting GitHub secrets..."

    echo "$MONITORING_PUBLIC_IP" | gh secret set SSH_HOST_MONITORING
    echo "$GRAFANA_PASSWORD"     | gh secret set GRAFANA_PASSWORD

    echo -e "${GREEN}✅ GitHub secrets set successfully${NC}"
fi

# ==========================================
# Summary
# ==========================================

echo ""
echo "=========================================="
echo "Deployment Complete!"
echo "=========================================="
echo ""
echo -e "Resource Group : ${GREEN}$RESOURCE_GROUP${NC}"
echo -e "VNet           : ${GREEN}$VNET_NAME${NC}"
echo ""
echo "monitoring VM"
echo -e "  Name        : ${GREEN}$MONITORING_VM_NAME${NC}"
echo -e "  Public IP   : ${GREEN}$MONITORING_PUBLIC_IP${NC}"
echo -e "  Private IP  : ${GREEN}$MONITORING_PRIVATE_IP${NC}"
echo    "  Ports open  : 22 (public), 3001 (public), 9090 (VNet only)"
echo -e "  Scraping    : ${GREEN}$APP_PRIVATE_IP:3000${NC}"
echo ""
echo "SSH access:"
echo -e "  ${YELLOW}ssh $ADMIN_USERNAME@$MONITORING_PUBLIC_IP${NC}"
echo ""
echo "Grafana (publicly accessible):"
echo -e "  ${GREEN}http://$MONITORING_PUBLIC_IP:3001/${NC}"
echo ""
echo "Prometheus (SSH tunnel to access from your machine):"
echo -e "  ${YELLOW}ssh -L 9090:localhost:9090 $ADMIN_USERNAME@$MONITORING_PUBLIC_IP${NC}"
echo    "  Then open: http://localhost:9090/"
echo ""
echo "=========================================="
