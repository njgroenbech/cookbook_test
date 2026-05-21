#!/bin/bash

# Azure Monitoring VM Teardown Script
# Deletes only the monitoring VM and its associated resources (NIC, NSG, public IP, OS disk).
# The shared resource group and VNet are left intact so other VMs in the deployment are unaffected.

set -e

# Configuration - MUST MATCH azure-setup-monitoring.sh
RESOURCE_GROUP="recipe-cookbook-backup"
MONITORING_VM_NAME="recipe-cookbook-monitoring-vm"

if [ -t 1 ]; then
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    RED='\033[0;31m'
    NC='\033[0m'
else
    GREEN=''
    YELLOW=''
    RED=''
    NC=''
fi

echo "=========================================="
echo "Monitoring VM Teardown"
echo "=========================================="
echo ""

if ! command -v az &> /dev/null; then
    echo -e "${RED}❌ Azure CLI is not installed${NC}"
    exit 1
fi

if ! az account show &> /dev/null; then
    echo -e "${RED}❌ Not logged in to Azure${NC}"
    echo "Please login first: az login"
    exit 1
fi

echo -e "${GREEN}✅ Logged in to Azure${NC}"
ACCOUNT=$(az account show --query name -o tsv)
echo "Using subscription: $ACCOUNT"
echo ""

if ! az group exists --name "$RESOURCE_GROUP" | grep -q "true"; then
    echo -e "${YELLOW}⚠️  Resource group '$RESOURCE_GROUP' does not exist${NC}"
    echo "Nothing to delete."
    exit 0
fi

if ! az vm show --resource-group "$RESOURCE_GROUP" --name "$MONITORING_VM_NAME" &>/dev/null; then
    echo -e "${YELLOW}⚠️  Monitoring VM '$MONITORING_VM_NAME' not found in '$RESOURCE_GROUP'${NC}"
    echo "Nothing to delete."
    exit 0
fi

# ==========================================
# Collect associated resource IDs
# ==========================================

echo "Resolving monitoring VM resources..."

NIC_ID=$(az vm show \
    --resource-group "$RESOURCE_GROUP" \
    --name "$MONITORING_VM_NAME" \
    --query "networkProfile.networkInterfaces[0].id" \
    --output tsv)

NIC_NAME=$(basename "$NIC_ID")

NSG_ID=$(az network nic show --ids "$NIC_ID" \
    --query "networkSecurityGroup.id" \
    --output tsv 2>/dev/null || echo "")
NSG_NAME=""
if [ -n "$NSG_ID" ]; then
    NSG_NAME=$(basename "$NSG_ID")
fi

PUBLIC_IP_ID=$(az network nic show --ids "$NIC_ID" \
    --query "ipConfigurations[0].publicIPAddress.id" \
    --output tsv 2>/dev/null || echo "")
PUBLIC_IP_NAME=""
if [ -n "$PUBLIC_IP_ID" ]; then
    PUBLIC_IP_NAME=$(basename "$PUBLIC_IP_ID")
fi

OS_DISK_NAME=$(az vm show \
    --resource-group "$RESOURCE_GROUP" \
    --name "$MONITORING_VM_NAME" \
    --query "storageProfile.osDisk.name" \
    --output tsv)

echo ""
echo "=========================================="
echo "Resources to be deleted:"
echo "=========================================="
echo "  VM        : $MONITORING_VM_NAME"
echo "  NIC       : $NIC_NAME"
[ -n "$NSG_NAME" ]       && echo "  NSG       : $NSG_NAME"
[ -n "$PUBLIC_IP_NAME" ] && echo "  Public IP : $PUBLIC_IP_NAME"
echo "  OS Disk   : $OS_DISK_NAME"
echo ""
echo -e "${YELLOW}Resource group '$RESOURCE_GROUP' and the shared VNet will NOT be deleted.${NC}"
echo ""

echo -e "${RED}⚠️  WARNING: This will permanently delete the monitoring VM and its associated resources${NC}"
echo ""
read -p "Are you sure you want to continue? (yes/NO): " -r
echo

if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
    echo "Teardown cancelled."
    exit 0
fi

echo ""
echo -e "${RED}⚠️  FINAL CONFIRMATION${NC}"
read -p "Type the VM name '$MONITORING_VM_NAME' to confirm deletion: " -r
echo

if [[ $REPLY != "$MONITORING_VM_NAME" ]]; then
    echo -e "${RED}❌ VM name doesn't match. Teardown cancelled.${NC}"
    exit 1
fi

# ==========================================
# Delete resources in dependency order
# ==========================================

echo ""
echo "=========================================="
echo "Deleting monitoring VM"
echo "=========================================="
az vm delete \
    --resource-group "$RESOURCE_GROUP" \
    --name "$MONITORING_VM_NAME" \
    --yes
echo -e "${GREEN}✅ VM deleted${NC}"

echo ""
echo "=========================================="
echo "Deleting NIC"
echo "=========================================="
az network nic delete \
    --resource-group "$RESOURCE_GROUP" \
    --name "$NIC_NAME"
echo -e "${GREEN}✅ NIC deleted${NC}"

if [ -n "$NSG_NAME" ]; then
    echo ""
    echo "=========================================="
    echo "Deleting NSG"
    echo "=========================================="
    if az network nsg delete \
        --resource-group "$RESOURCE_GROUP" \
        --name "$NSG_NAME" 2>/dev/null; then
        echo -e "${GREEN}✅ NSG deleted${NC}"
    else
        echo -e "${YELLOW}⚠️  NSG '$NSG_NAME' could not be deleted (may be in use elsewhere)${NC}"
    fi
fi

if [ -n "$PUBLIC_IP_NAME" ]; then
    echo ""
    echo "=========================================="
    echo "Deleting public IP"
    echo "=========================================="
    az network public-ip delete \
        --resource-group "$RESOURCE_GROUP" \
        --name "$PUBLIC_IP_NAME"
    echo -e "${GREEN}✅ Public IP deleted${NC}"
fi

echo ""
echo "=========================================="
echo "Deleting OS disk"
echo "=========================================="
if az disk delete \
    --resource-group "$RESOURCE_GROUP" \
    --name "$OS_DISK_NAME" \
    --yes 2>/dev/null; then
    echo -e "${GREEN}✅ OS disk deleted${NC}"
else
    echo -e "${YELLOW}⚠️  OS disk '$OS_DISK_NAME' could not be deleted (may have already been removed)${NC}"
fi

echo ""
echo "=========================================="
echo "Teardown Complete!"
echo "=========================================="
echo ""
echo "Monitoring VM and its associated resources have been deleted."
echo "Resource group '$RESOURCE_GROUP' is preserved."
echo ""
