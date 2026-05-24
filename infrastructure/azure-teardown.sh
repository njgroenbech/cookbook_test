#!/bin/bash

# Azure Resource Teardown Script
# This script deletes the entire resource group and all its resources

set -e  # Exit on any error

# Configuration - MUST MATCH azure-setup.sh
RESOURCE_GROUP="recipe-cookbook-backup"
NGINX_PUBLIC_IP_NAME="recipe-cookbook-nginx-public-ip"

# Check if terminal supports ANSI colors
if [ -t 1 ]; then
    # Terminal supports ANSI colors
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    RED='\033[0;31m'
    NC='\033[0m' # No Color
else
    # Terminal does not support ANSI colors
    GREEN=''
    YELLOW=''
    RED=''
    NC=''
fi

echo "=========================================="
echo "Azure Resource Teardown"
echo "=========================================="
echo ""

# Check if Azure CLI is installed
if ! command -v az &> /dev/null; then
    echo -e "${RED}❌ Azure CLI is not installed${NC}"
    exit 1
fi

# Check if logged in to Azure
if ! az account show &> /dev/null; then
    echo -e "${RED}❌ Not logged in to Azure${NC}"
    echo "Please login first: az login"
    exit 1
fi

echo -e "${GREEN}✅ Logged in to Azure${NC}"
ACCOUNT=$(az account show --query name -o tsv)
echo "Using subscription: $ACCOUNT"
echo ""

# Check if resource group exists
if ! az group exists --name "$RESOURCE_GROUP" | grep -q "true"; then
    echo -e "${YELLOW}⚠️  Resource group '$RESOURCE_GROUP' does not exist${NC}"
    echo "Nothing to delete."
    exit 0

fi 

# Confirmation prompt
echo -e "${RED}⚠️  WARNING: This will permanently delete all resources in '$RESOURCE_GROUP'.${NC}"
echo "This includes all VMs, disks, network interfaces, NSGs, and the VNet."
echo "Note: the nginx public IP ('$NGINX_PUBLIC_IP_NAME') is NOT deleted."
echo ""
read -p "Type the resource group name to confirm: " CONFIRM_GROUP
if [ "$CONFIRM_GROUP" != "$RESOURCE_GROUP" ]; then
    echo -e "${RED}❌ Input does not match. Teardown cancelled.${NC}"
    exit 1
fi
echo ""

echo "Deleting VMs..."
az vm delete --resource-group "$RESOURCE_GROUP" --name recipe-cookbook-nginx-vm --yes
az vm delete --resource-group "$RESOURCE_GROUP" --name recipe-cookbook-app-vm --yes
az vm delete --resource-group "$RESOURCE_GROUP" --name recipe-cookbook-postgres-vm --yes
az vm delete --resource-group "$RESOURCE_GROUP" --name recipe-cookbook-monitoring-vm --yes
echo -e "${GREEN}✅ VMs deleted${NC}"

echo "Deleting network interfaces..."
az network nic delete --resource-group "$RESOURCE_GROUP" --name recipe-cookbook-monitoring-vmVMNic
az network nic delete --resource-group "$RESOURCE_GROUP" --name recipe-cookbook-postgres-vmVMNic
az network nic delete --resource-group "$RESOURCE_GROUP" --name recipe-cookbook-nginx-vmVMNic
az network nic delete --resource-group "$RESOURCE_GROUP" --name recipe-cookbook-app-vmVMNic
echo -e "${GREEN}✅ Network interfaces deleted${NC}"

echo "Deleting disks..."
for disk_id in $(az disk list --resource-group "$RESOURCE_GROUP" --query "[].id" --output tsv); do
    echo "  $disk_id"
    MSYS_NO_PATHCONV=1 az disk delete --ids "$disk_id" --yes --no-wait
done
echo -e "${GREEN}✅ Disks deleted${NC}"

echo "Deleting network security groups..."
az network nsg delete --resource-group "$RESOURCE_GROUP" --name "recipe-cookbook-nginx-vmNSG"
az network nsg delete --resource-group "$RESOURCE_GROUP" --name "recipe-cookbook-postgres-vmNSG"
az network nsg delete --resource-group "$RESOURCE_GROUP" --name "recipe-cookbook-monitoring-vmNSG"
az network nsg delete --resource-group "$RESOURCE_GROUP" --name "recipe-cookbook-app-vmNSG"
echo -e "${GREEN}✅ NSGs deleted${NC}"

echo "Deleting virtual network..."
az network vnet delete --resource-group "$RESOURCE_GROUP" --name "recipe-cookbook-vnet"
echo -e "${GREEN}✅ VNet deleted${NC}"

echo ""
echo "=========================================="
echo "Teardown Complete! 🗑️"
echo "=========================================="
echo ""
echo "All resources in '$RESOURCE_GROUP' have been deleted (or deletion is in progress)."
echo ""