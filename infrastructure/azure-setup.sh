#!/bin/bash

# Full-Stack Azure Deployment Script (SHTF Recovery)
# Creates four VMs in a shared VNet:
#   - nginx VM       : public-facing, ports 80/443 open
#   - app VM         : internal, port 3000 accessible within VNet only
#   - postgres VM    : internal, port 5432 accessible from app VM only
#   - monitoring VM  : internal, ports 9090/3001 accessible within VNet only
# Deploys the full application stack on each VM and sets all GitHub secrets.
#
# Usage: ./azure-setup.sh [--no-colors]
#
# NOTE: If the app VM's private IP ever changes (e.g. VM recreated), re-run
# this script so nginx.conf is regenerated with the correct upstream address.

set -e
export MSYS_NO_PATHCONV=1

# Parse command line arguments
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
NGINX_VM_NAME="recipe-cookbook-nginx-vm"
APP_VM_NAME="recipe-cookbook-app-vm"
POSTGRES_VM_NAME="recipe-cookbook-postgres-vm"
MONITORING_VM_NAME="recipe-cookbook-monitoring-vm"
VM_SIZE="Standard_B1s"  # Change to "Standard_B2s" for better performance
VM_IMAGE="Canonical:0001-com-ubuntu-server-jammy:22_04-lts:latest"
ADMIN_USERNAME="azureuser"
SSH_KEY_PATH="$HOME/.ssh/azure_key.pub"
VNET_NAME="recipe-cookbook-vnet"
SUBNET_NAME="recipe-cookbook-subnet"
GITHUB_REPO="dendanskemetode/legacyproject"
REMOTE_APP_DIR="/home/azureuser/legacyProject"

# Runtime variables (populated during execution)
DB_USER="cookbook_user"
DB_NAME="cookbook_db"
DB_PASSWORD=""
GHCR_TOKEN=""
GIT_CLONE_URL=""
GRAFANA_PASSWORD=""
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
echo "Full-Stack Azure Deployment — Recipe Cookbook"
echo "Four-VM deployment: nginx + app + postgres + monitoring"
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
echo "Generating Database Credentials"
echo "=========================================="
DB_PASSWORD=$(openssl rand -base64 32 | tr -dc 'a-zA-Z0-9' | head -c 32)
echo -e "${GREEN}✅ Database credentials generated (not shown)${NC}"

echo ""
echo "=========================================="
echo "Generating Grafana Admin Password"
echo "=========================================="
GRAFANA_PASSWORD=$(openssl rand -base64 32 | tr -dc 'a-zA-Z0-9' | head -c 32)
echo -e "${GREEN}✅ Grafana admin password generated (not shown)${NC}"

echo ""
echo "=========================================="
echo "Acquiring GHCR Token"
echo "=========================================="
if command -v gh &> /dev/null && gh auth status &> /dev/null 2>&1; then
    GHCR_TOKEN=$(gh auth token)
    echo -e "${GREEN}✅ GHCR token acquired from GitHub CLI${NC}"
else
    echo -e "${YELLOW}⚠️  GitHub CLI not available or not authenticated${NC}"
    read -s -p "Enter a GitHub PAT with read:packages scope: " GHCR_TOKEN
    echo
fi

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
echo "Creating Resource Group"
echo "=========================================="
echo "Name: $RESOURCE_GROUP"
echo "Location: $LOCATION"

if az group exists --name "$RESOURCE_GROUP" | grep -q "true"; then
    echo -e "${YELLOW}⚠️  Resource group already exists${NC}"
    read -p "Do you want to delete and recreate it? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo "Deleting existing resource group..."
        az group delete --name "$RESOURCE_GROUP" --yes
        echo "Resource group deleted."
    else
        echo "Using existing resource group"
    fi
fi

if ! az group exists --name "$RESOURCE_GROUP" | grep -q "true"; then
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
echo "Creating Shared Virtual Network"
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
    echo -e "${GREEN}✅ Virtual network already exists, skipping${NC}"
fi

# ==========================================
# Create VMs
# ==========================================

echo ""
echo "=========================================="
echo "Creating nginx VM (public-facing)"
echo "=========================================="
echo "VM Name: $NGINX_VM_NAME"
echo "Size: $VM_SIZE"
echo "This may take 2-5 minutes..."
echo ""

az vm create \
    --resource-group "$RESOURCE_GROUP" \
    --name "$NGINX_VM_NAME" \
    --image "$VM_IMAGE" \
    --size "$VM_SIZE" \
    --admin-username "$ADMIN_USERNAME" \
    --ssh-key-values "$(cat "$SSH_KEY_PATH")" \
    --vnet-name "$VNET_NAME" \
    --subnet "$SUBNET_NAME" \
    --public-ip-sku Standard \
    --output table

echo -e "${GREEN}✅ nginx VM created${NC}"

echo ""
echo "=========================================="
echo "Creating app VM (internal)"
echo "=========================================="
echo "VM Name: $APP_VM_NAME"
echo "Size: $VM_SIZE"
echo "This may take 2-5 minutes..."
echo ""

az vm create \
    --resource-group "$RESOURCE_GROUP" \
    --name "$APP_VM_NAME" \
    --image "$VM_IMAGE" \
    --size "$VM_SIZE" \
    --admin-username "$ADMIN_USERNAME" \
    --ssh-key-values "$(cat "$SSH_KEY_PATH")" \
    --vnet-name "$VNET_NAME" \
    --subnet "$SUBNET_NAME" \
    --public-ip-address "" \
    --output table

echo -e "${GREEN}✅ app VM created${NC}"

echo ""
echo "=========================================="
echo "Creating postgres VM (internal)"
echo "=========================================="
echo "VM Name: $POSTGRES_VM_NAME"
echo "Size: $VM_SIZE"
echo "This may take 2-5 minutes..."
echo ""

az vm create \
    --resource-group "$RESOURCE_GROUP" \
    --name "$POSTGRES_VM_NAME" \
    --image "$VM_IMAGE" \
    --size "$VM_SIZE" \
    --admin-username "$ADMIN_USERNAME" \
    --ssh-key-values "$(cat "$SSH_KEY_PATH")" \
    --vnet-name "$VNET_NAME" \
    --subnet "$SUBNET_NAME" \
    --public-ip-address "" \
    --output table

echo -e "${GREEN}✅ postgres VM created${NC}"

echo ""
echo "=========================================="
echo "Creating monitoring VM (internal)"
echo "=========================================="
echo "VM Name: $MONITORING_VM_NAME"
echo "Size: $VM_SIZE"
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
    --public-ip-address "" \
    --output table

echo -e "${GREEN}✅ monitoring VM created${NC}"

# ==========================================
# Network security
# ==========================================

echo ""
echo "=========================================="
echo "Configuring Network Security"
echo "=========================================="
echo "nginx VM: opening ports 22 (SSH), 80 (HTTP), 443 (HTTPS)"

az vm open-port \
    --resource-group "$RESOURCE_GROUP" \
    --name "$NGINX_VM_NAME" \
    --port 80 \
    --priority 300 \
    --output table

az vm open-port \
    --resource-group "$RESOURCE_GROUP" \
    --name "$NGINX_VM_NAME" \
    --port 443 \
    --priority 310 \
    --output table

echo -e "${GREEN}✅ nginx VM ports configured${NC}"

# Fetch app VM private IP now so it can be used as the source filter for the postgres NSG rule
APP_PRIVATE_IP=$(az vm show \
    --resource-group "$RESOURCE_GROUP" \
    --name "$APP_VM_NAME" \
    --show-details \
    --query privateIps \
    --output tsv)

echo ""
echo "app VM: adding NSG rule — allow TCP 3000 inbound from VNet"

APP_NIC_ID=$(az vm show \
    --resource-group "$RESOURCE_GROUP" \
    --name "$APP_VM_NAME" \
    --query "networkProfile.networkInterfaces[0].id" \
    --output tsv)

APP_NSG=$(az network nic show --ids "$APP_NIC_ID" \
    --query "networkSecurityGroup.id" \
    --output tsv | xargs basename)

if ! az network nsg rule show \
        --resource-group "$RESOURCE_GROUP" \
        --nsg-name "$APP_NSG" \
        --name "AllowVnet3000" &>/dev/null; then
    az network nsg rule create \
        --resource-group "$RESOURCE_GROUP" \
        --nsg-name "$APP_NSG" \
        --name "AllowVnet3000" \
        --priority 200 \
        --source-address-prefixes "10.0.1.0/24" \
        --destination-port-ranges 3000 \
        --protocol Tcp \
        --access Allow \
        --direction Inbound \
        --output table
fi

echo -e "${GREEN}✅ app VM NSG rule created (port 3000 from VNet)${NC}"

echo ""
echo "postgres VM: adding NSG rule — allow TCP 5432 inbound from app VM only"

POSTGRES_NIC_ID=$(az vm show \
    --resource-group "$RESOURCE_GROUP" \
    --name "$POSTGRES_VM_NAME" \
    --query "networkProfile.networkInterfaces[0].id" \
    --output tsv)

POSTGRES_NSG=$(az network nic show --ids "$POSTGRES_NIC_ID" \
    --query "networkSecurityGroup.id" \
    --output tsv | xargs basename)

if ! az network nsg rule show \
        --resource-group "$RESOURCE_GROUP" \
        --nsg-name "$POSTGRES_NSG" \
        --name "AllowBackend5432" &>/dev/null; then
    az network nsg rule create \
        --resource-group "$RESOURCE_GROUP" \
        --nsg-name "$POSTGRES_NSG" \
        --name "AllowBackend5432" \
        --priority 200 \
        --source-address-prefixes "$APP_PRIVATE_IP/32" \
        --destination-port-ranges 5432 \
        --protocol Tcp \
        --access Allow \
        --direction Inbound \
        --output table
fi

echo -e "${GREEN}✅ postgres VM NSG rule created (port 5432 from app VM only)${NC}"

echo ""
echo "monitoring VM: adding NSG rules — allow TCP 9090/3001 inbound from VNet"

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
        --name "AllowVnetGrafana" &>/dev/null; then
    az network nsg rule create \
        --resource-group "$RESOURCE_GROUP" \
        --nsg-name "$MONITORING_NSG" \
        --name "AllowVnetGrafana" \
        --priority 210 \
        --source-address-prefixes "10.0.1.0/24" \
        --destination-port-ranges 3001 \
        --protocol Tcp \
        --access Allow \
        --direction Inbound \
        --output table
fi

echo -e "${GREEN}✅ monitoring VM NSG rules created (ports 9090/3001 from VNet)${NC}"

# ==========================================
# Get all VM IPs
# ==========================================

echo ""
echo "=========================================="
echo "Getting VM Information"
echo "=========================================="

NGINX_PUBLIC_IP=$(az vm show \
    --resource-group "$RESOURCE_GROUP" \
    --name "$NGINX_VM_NAME" \
    --show-details \
    --query publicIps \
    --output tsv)

POSTGRES_PRIVATE_IP=$(az vm show \
    --resource-group "$RESOURCE_GROUP" \
    --name "$POSTGRES_VM_NAME" \
    --show-details \
    --query privateIps \
    --output tsv)

MONITORING_PRIVATE_IP=$(az vm show \
    --resource-group "$RESOURCE_GROUP" \
    --name "$MONITORING_VM_NAME" \
    --show-details \
    --query privateIps \
    --output tsv)

echo ""
echo -e "nginx VM      — Public IP  : ${GREEN}$NGINX_PUBLIC_IP${NC}"
echo    "app VM        — Public IP  : (none — internal only)"
echo -e "app VM        — Private IP : ${GREEN}$APP_PRIVATE_IP${NC}"
echo    "postgres VM   — Public IP  : (none — internal only)"
echo -e "postgres VM   — Private IP : ${GREEN}$POSTGRES_PRIVATE_IP${NC}"
echo    "monitoring VM — Public IP  : (none — internal only)"
echo -e "monitoring VM — Private IP : ${GREEN}$MONITORING_PRIVATE_IP${NC}"
echo ""
echo "nginx   → app        : $APP_PRIVATE_IP:3000"
echo "app     → postgres   : $POSTGRES_PRIVATE_IP:5432"
echo "VNet    → monitoring : $MONITORING_PRIVATE_IP:9090 (Prometheus), $MONITORING_PRIVATE_IP:3001 (Grafana)"

# ==========================================
# Wait for VMs
# ==========================================

echo ""
echo "Waiting for VMs to reach provisioned state..."
az vm wait --resource-group "$RESOURCE_GROUP" --name "$NGINX_VM_NAME" --created
az vm wait --resource-group "$RESOURCE_GROUP" --name "$APP_VM_NAME" --created
az vm wait --resource-group "$RESOURCE_GROUP" --name "$POSTGRES_VM_NAME" --created
az vm wait --resource-group "$RESOURCE_GROUP" --name "$MONITORING_VM_NAME" --created
echo "VMs provisioned. Waiting for SSH to become available..."
sleep 30

# ==========================================
# Helper: update system on a VM
# ==========================================

setup_vm() {
    local VM_IP="$1"
    local VM_LABEL="$2"
    local JUMP="${3:-}"
    local SSH_KEY="${SSH_KEY_PATH%.pub}"
    # Host keys are unknown for freshly created VMs; strict checking is disabled intentionally.
    local -a SSH_OPTS=(-o StrictHostKeyChecking=no -o ConnectTimeout=30 -i "$SSH_KEY")
    if [ -n "$JUMP" ]; then
        SSH_OPTS+=(-o "ProxyCommand=ssh -o StrictHostKeyChecking=no -o ConnectTimeout=30 -i '$SSH_KEY' -W %h:%p $JUMP")
    fi

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

setup_vm "$NGINX_PUBLIC_IP" "nginx"
setup_vm "$APP_PRIVATE_IP" "app" "$ADMIN_USERNAME@$NGINX_PUBLIC_IP"
setup_vm "$POSTGRES_PRIVATE_IP" "postgres" "$ADMIN_USERNAME@$NGINX_PUBLIC_IP"
setup_vm "$MONITORING_PRIVATE_IP" "monitoring" "$ADMIN_USERNAME@$NGINX_PUBLIC_IP"

# ==========================================
# Install Docker on all VMs
# ==========================================

install_docker() {
    local VM_IP="$1"
    local VM_LABEL="$2"
    local ROLE="$3"       # nginx | app | postgres | monitoring
    local JUMP="${4:-}"
    local SSH_KEY="${SSH_KEY_PATH%.pub}"
    # Host keys are unknown for freshly created VMs; strict checking is disabled intentionally.
    local -a SSH_OPTS=(-o StrictHostKeyChecking=no -o ConnectTimeout=30 -i "$SSH_KEY")
    if [ -n "$JUMP" ]; then
        SSH_OPTS+=(-o "ProxyCommand=ssh -o StrictHostKeyChecking=no -o ConnectTimeout=30 -i '$SSH_KEY' -W %h:%p $JUMP")
    fi

    # Compute the ports this VM role needs open (expanded locally into the heredoc).
    local UFW_PORTS
    case "$ROLE" in
        nginx)      UFW_PORTS="80 443" ;;
        app)        UFW_PORTS="3000" ;;
        postgres)   UFW_PORTS="5432" ;;
        monitoring) UFW_PORTS="9090 3001" ;;
        *)          UFW_PORTS="" ;;
    esac

    echo ""
    echo "=========================================="
    echo "Installing Docker on $VM_LABEL VM"
    echo "=========================================="

    # Outer heredoc is unquoted (ENDSSH) so $REMOTE_APP_DIR and $UFW_PORTS expand locally.
    # Remote-side variables ($USER, $(lsb_release)) are escaped with \ to defer to the remote shell.
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

install_docker "$NGINX_PUBLIC_IP"     "nginx"      "nginx"
install_docker "$APP_PRIVATE_IP"      "app"        "app"        "$ADMIN_USERNAME@$NGINX_PUBLIC_IP"
install_docker "$POSTGRES_PRIVATE_IP" "postgres"   "postgres"   "$ADMIN_USERNAME@$NGINX_PUBLIC_IP"
install_docker "$MONITORING_PRIVATE_IP" "monitoring" "monitoring" "$ADMIN_USERNAME@$NGINX_PUBLIC_IP"

echo -e "${YELLOW}⚠️  Note: docker group changes require re-login; all docker commands below use sudo${NC}"

# ==========================================
# Clone repo on all VMs
# ==========================================

clone_repo() {
    local VM_IP="$1"
    local VM_LABEL="$2"
    local JUMP="${3:-}"
    local SSH_KEY="${SSH_KEY_PATH%.pub}"
    local -a SSH_OPTS=(-o StrictHostKeyChecking=no -o ConnectTimeout=30 -i "$SSH_KEY")
    if [ -n "$JUMP" ]; then
        SSH_OPTS+=(-o "ProxyCommand=ssh -o StrictHostKeyChecking=no -o ConnectTimeout=30 -i '$SSH_KEY' -W %h:%p $JUMP")
    fi

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

clone_repo "$NGINX_PUBLIC_IP" "nginx"
clone_repo "$APP_PRIVATE_IP" "app" "$ADMIN_USERNAME@$NGINX_PUBLIC_IP"
clone_repo "$POSTGRES_PRIVATE_IP" "postgres" "$ADMIN_USERNAME@$NGINX_PUBLIC_IP"
clone_repo "$MONITORING_PRIVATE_IP" "monitoring" "$ADMIN_USERNAME@$NGINX_PUBLIC_IP"

# ==========================================
# Deploy postgres VM
# ==========================================

echo ""
echo "=========================================="
echo "Deploying PostgreSQL"
echo "=========================================="

SSH_KEY="${SSH_KEY_PATH%.pub}"

ssh -o StrictHostKeyChecking=no -o ConnectTimeout=30 -i "$SSH_KEY" \
    -o "ProxyCommand=ssh -o StrictHostKeyChecking=no -o ConnectTimeout=30 -i '$SSH_KEY' -W %h:%p $ADMIN_USERNAME@$NGINX_PUBLIC_IP" \
    "$ADMIN_USERNAME@$POSTGRES_PRIVATE_IP" << ENDSSH
set -e
cd $REMOTE_APP_DIR/database

cat > .env << ENVEOF
POSTGRES_USER=$DB_USER
POSTGRES_PASSWORD=$DB_PASSWORD
POSTGRES_DB=$DB_NAME
ENVEOF

echo "Starting PostgreSQL container..."
sudo docker compose up -d
echo "PostgreSQL container started."
ENDSSH

echo -e "${GREEN}✅ PostgreSQL deployed${NC}"

# ==========================================
# Wait for postgres to be ready
# ==========================================

echo ""
echo "Waiting for PostgreSQL to accept connections..."
MAX_ATTEMPTS=30
ATTEMPT=0
while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
    ATTEMPT=$((ATTEMPT + 1))
    if ssh -o StrictHostKeyChecking=no -o ConnectTimeout=10 -i "$SSH_KEY" \
        -o "ProxyCommand=ssh -o StrictHostKeyChecking=no -o ConnectTimeout=10 -i '$SSH_KEY' -W %h:%p $ADMIN_USERNAME@$NGINX_PUBLIC_IP" \
        "$ADMIN_USERNAME@$POSTGRES_PRIVATE_IP" \
        "sudo docker exec recipe_app_postgres pg_isready -U $DB_USER -d $DB_NAME" 2>/dev/null; then
        echo -e "${GREEN}✅ PostgreSQL is ready${NC}"
        break
    fi
    echo "  attempt $ATTEMPT/$MAX_ATTEMPTS — not ready yet, waiting 10s..."
    sleep 10
done
if [ $ATTEMPT -eq $MAX_ATTEMPTS ]; then
    echo -e "${RED}❌ PostgreSQL did not become ready after $((MAX_ATTEMPTS * 10))s${NC}"
    exit 1
fi

# ==========================================
# Deploy app VM
# ==========================================

echo ""
echo "=========================================="
echo "Deploying Backend"
echo "=========================================="

ssh -o StrictHostKeyChecking=no -o ConnectTimeout=30 -i "$SSH_KEY" \
    -o "ProxyCommand=ssh -o StrictHostKeyChecking=no -o ConnectTimeout=30 -i '$SSH_KEY' -W %h:%p $ADMIN_USERNAME@$NGINX_PUBLIC_IP" \
    "$ADMIN_USERNAME@$APP_PRIVATE_IP" << ENDSSH
set -e
cd $REMOTE_APP_DIR/app

cat > .env << ENVEOF
DB_HOST=$POSTGRES_PRIVATE_IP
DB_PORT=5432
DB_USER=$DB_USER
DB_PASSWORD=$DB_PASSWORD
DB_NAME=$DB_NAME
PORT=3000
ENVEOF

echo "Authenticating to GHCR..."
echo "$GHCR_TOKEN" | sudo docker login ghcr.io -u github-token --password-stdin

echo "Pulling app image..."
sudo docker compose pull

echo "Starting app container..."
sudo docker compose up -d
echo "Backend container started."
ENDSSH

echo -e "${GREEN}✅ Backend deployed${NC}"

# ==========================================
# Wait for app to be ready
# ==========================================

echo ""
echo "Waiting for Go app to accept connections..."
MAX_ATTEMPTS=18
ATTEMPT=0
while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
    ATTEMPT=$((ATTEMPT + 1))
    APP_READY_STATUS=$(ssh -o StrictHostKeyChecking=no -o ConnectTimeout=10 -i "$SSH_KEY" \
        -o "ProxyCommand=ssh -o StrictHostKeyChecking=no -o ConnectTimeout=10 -i '$SSH_KEY' -W %h:%p $ADMIN_USERNAME@$NGINX_PUBLIC_IP" \
        "$ADMIN_USERNAME@$APP_PRIVATE_IP" \
        "curl -s -o /dev/null -w '%{http_code}' http://localhost:3000/" 2>/dev/null || echo "000")
    APP_READY_STATUS="${APP_READY_STATUS:-000}"
    if [[ "$APP_READY_STATUS" =~ ^[23] ]]; then
        echo -e "${GREEN}✅ Go app is ready (HTTP $APP_READY_STATUS)${NC}"
        break
    fi
    echo "  attempt $ATTEMPT/$MAX_ATTEMPTS — not ready yet (HTTP $APP_READY_STATUS), waiting 10s..."
    sleep 10
done
if [ $ATTEMPT -eq $MAX_ATTEMPTS ]; then
    echo -e "${RED}❌ Go app did not become ready after $((MAX_ATTEMPTS * 10))s${NC}"
    exit 1
fi

# ==========================================
# Deploy nginx VM
# ==========================================

echo ""
echo "=========================================="
echo "Deploying nginx"
echo "=========================================="

# Outer heredoc is unquoted (ENDSSH) so $APP_PRIVATE_IP and $REMOTE_APP_DIR expand here.
ssh -o StrictHostKeyChecking=no -o ConnectTimeout=30 -i "$SSH_KEY" \
    "$ADMIN_USERNAME@$NGINX_PUBLIC_IP" << ENDSSH
set -e
cd $REMOTE_APP_DIR/network

cat > .env << ENVEOF
APP_HOST=$APP_PRIVATE_IP
ENVEOF

echo "Authenticating to GHCR..."
echo "$GHCR_TOKEN" | sudo docker login ghcr.io -u github-token --password-stdin

echo "Pulling and starting nginx container..."
sudo docker compose pull
sudo docker compose up -d
echo "nginx container started."
ENDSSH

echo -e "${GREEN}✅ nginx deployed${NC}"

# ==========================================
# Deploy monitoring VM
# ==========================================

echo ""
echo "=========================================="
echo "Deploying Monitoring (Prometheus + Grafana)"
echo "=========================================="

ssh -o StrictHostKeyChecking=no -o ConnectTimeout=30 -i "$SSH_KEY" \
    -o "ProxyCommand=ssh -o StrictHostKeyChecking=no -o ConnectTimeout=30 -i '$SSH_KEY' -W %h:%p $ADMIN_USERNAME@$NGINX_PUBLIC_IP" \
    "$ADMIN_USERNAME@$MONITORING_PRIVATE_IP" << ENDSSH
set -e
cd $REMOTE_APP_DIR/monitoring

cat > .env << ENVEOF
GF_SECURITY_ADMIN_PASSWORD=$GRAFANA_PASSWORD
ENVEOF

echo "Authenticating to GHCR..."
echo "$GHCR_TOKEN" | sudo docker login ghcr.io -u github-token --password-stdin

echo "Starting monitoring containers..."
sudo docker compose up -d
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

# 1. Postgres
echo -n "PostgreSQL reachable... "
if ssh -o StrictHostKeyChecking=no -o ConnectTimeout=10 -i "$SSH_KEY" \
    -o "ProxyCommand=ssh -o StrictHostKeyChecking=no -o ConnectTimeout=10 -i '$SSH_KEY' -W %h:%p $ADMIN_USERNAME@$NGINX_PUBLIC_IP" \
    "$ADMIN_USERNAME@$POSTGRES_PRIVATE_IP" \
    "sudo docker exec recipe_app_postgres pg_isready -U $DB_USER -d $DB_NAME" 2>/dev/null; then
    echo -e "${GREEN}✅ OK${NC}"
else
    echo -e "${RED}❌ FAIL${NC}"
fi

# 2. Backend (curl from within the app VM so we don't need cross-VM routing for the check)
echo -n "Backend responding (2xx/3xx)... "
APP_STATUS=$(ssh -o StrictHostKeyChecking=no -o ConnectTimeout=10 -i "$SSH_KEY" \
    -o "ProxyCommand=ssh -o StrictHostKeyChecking=no -o ConnectTimeout=10 -i '$SSH_KEY' -W %h:%p $ADMIN_USERNAME@$NGINX_PUBLIC_IP" \
    "$ADMIN_USERNAME@$APP_PRIVATE_IP" \
    "curl -s -o /dev/null -w '%{http_code}' http://localhost:3000/"; true)
APP_STATUS="${APP_STATUS:-000}"
if [[ "$APP_STATUS" =~ ^[23] ]]; then
    echo -e "${GREEN}✅ OK (HTTP $APP_STATUS)${NC}"
else
    echo -e "${RED}❌ FAIL (HTTP $APP_STATUS)${NC}"
fi

# 3. End-to-end via nginx public IP
echo -n "App accessible via nginx (2xx/3xx)... "
sleep 5
E2E_STATUS=$(curl -s -o /dev/null -w "%{http_code}" --max-time 10 "http://$NGINX_PUBLIC_IP/"; true)
E2E_STATUS="${E2E_STATUS:-000}"
if [[ "$E2E_STATUS" =~ ^[23] ]]; then
    echo -e "${GREEN}✅ OK (HTTP $E2E_STATUS)${NC}"
else
    echo -e "${RED}❌ FAIL (HTTP $E2E_STATUS — nginx may still be starting, try again in a moment)${NC}"
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
    echo "  VM_USER            = $ADMIN_USERNAME"
    echo "  SSH_HOST_NGINX     = $NGINX_PUBLIC_IP"
    echo "  SSH_HOST_APP       = $APP_PRIVATE_IP"
    echo "  SSH_PROXY_HOST     = $NGINX_PUBLIC_IP"
    echo "  AZURE_KEY          = (contents of ${SSH_KEY_PATH%.pub})"
    echo "  SSH_HOST_POSTGRES  = $POSTGRES_PRIVATE_IP"
    echo "  DB_HOST            = $POSTGRES_PRIVATE_IP"
    echo "  DB_USER            = $DB_USER"
    echo "  DB_PASSWORD        = (generated randomly — re-run the script to provision new credentials)"
    echo "  DB_NAME            = $DB_NAME"
    echo "  GRAFANA_PASSWORD   = (generated randomly — re-run the script to provision new credentials)"
    echo ""
else
    if ! gh auth status &> /dev/null; then
        echo -e "${YELLOW}⚠️  Not authenticated with GitHub CLI${NC}"
        gh auth login
    fi

    echo "Setting GitHub secrets..."

    echo "$ADMIN_USERNAME"      | gh secret set VM_USER
    echo "$NGINX_PUBLIC_IP"     | gh secret set SSH_HOST_NGINX
    echo "$APP_PRIVATE_IP"      | gh secret set SSH_HOST_APP
    echo "$NGINX_PUBLIC_IP"     | gh secret set SSH_PROXY_HOST
    gh secret set AZURE_KEY < "${SSH_KEY_PATH%.pub}"
    # Stored for recovery only — used when re-provisioning a single VM without a full teardown.
    echo "$POSTGRES_PRIVATE_IP" | gh secret set SSH_HOST_POSTGRES
    echo "$POSTGRES_PRIVATE_IP" | gh secret set DB_HOST
    echo "$DB_USER"             | gh secret set DB_USER
    echo "$DB_PASSWORD"         | gh secret set DB_PASSWORD
    echo "$DB_NAME"             | gh secret set DB_NAME
    echo "$GRAFANA_PASSWORD"    | gh secret set GRAFANA_PASSWORD

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
echo "nginx VM"
echo -e "  Name       : ${GREEN}$NGINX_VM_NAME${NC}"
echo -e "  Public IP  : ${GREEN}$NGINX_PUBLIC_IP${NC}"
echo    "  Ports open : 22, 80, 443"
echo ""
echo "app VM"
echo -e "  Name        : ${GREEN}$APP_VM_NAME${NC}"
echo    "  Public IP   : (none — internal only)"
echo -e "  Private IP  : ${GREEN}$APP_PRIVATE_IP${NC}"
echo    "  Ports open  : 22 (public), 3000 (VNet only)"
echo ""
echo "postgres VM"
echo -e "  Name        : ${GREEN}$POSTGRES_VM_NAME${NC}"
echo    "  Public IP   : (none — internal only)"
echo -e "  Private IP  : ${GREEN}$POSTGRES_PRIVATE_IP${NC}"
echo    "  Ports open  : 22 (public), 5432 (app VM only)"
echo ""
echo "monitoring VM"
echo -e "  Name        : ${GREEN}$MONITORING_VM_NAME${NC}"
echo    "  Public IP   : (none — internal only)"
echo -e "  Private IP  : ${GREEN}$MONITORING_PRIVATE_IP${NC}"
echo    "  Ports open  : 22 (public), 9090/3001 (VNet only)"
echo ""
echo "=========================================="
echo "GitHub Secrets:"
echo "=========================================="
echo "  VM_USER            = $ADMIN_USERNAME"
echo "  SSH_HOST_NGINX     = $NGINX_PUBLIC_IP"
echo "  SSH_HOST_APP       = $APP_PRIVATE_IP"
echo "  SSH_PROXY_HOST     = $NGINX_PUBLIC_IP"
echo "  AZURE_KEY          = (from ${SSH_KEY_PATH%.pub})"
echo ""
echo "=========================================="
echo "App is live at:"
echo "=========================================="
echo ""
echo -e "  ${GREEN}http://$NGINX_PUBLIC_IP/${NC}"
echo ""
echo "SSH access:"
echo -e "  ${YELLOW}ssh $ADMIN_USERNAME@$NGINX_PUBLIC_IP${NC}   (nginx)"
echo -e "  ${YELLOW}ssh -J $ADMIN_USERNAME@$NGINX_PUBLIC_IP $ADMIN_USERNAME@$APP_PRIVATE_IP${NC}   (app via jump)"
echo -e "  ${YELLOW}ssh -J $ADMIN_USERNAME@$NGINX_PUBLIC_IP $ADMIN_USERNAME@$POSTGRES_PRIVATE_IP${NC}  (postgres via jump)"
echo -e "  ${YELLOW}ssh -J $ADMIN_USERNAME@$NGINX_PUBLIC_IP $ADMIN_USERNAME@$MONITORING_PRIVATE_IP${NC} (monitoring via jump)"
echo ""
echo "Grafana:"
echo -e "  ${YELLOW}http://$MONITORING_PRIVATE_IP:3001/${NC}  (accessible from within VNet; SSH tunnel to reach it)"
echo ""
echo "=========================================="
echo ""
echo "To tear down all resources:"
echo -e "   ${RED}./azure-teardown.sh${NC}"
echo ""
