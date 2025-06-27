#!/bin/bash

CONFIG_FILE="./config/config-real.yaml"
echo "PxGrid Cloud SDK Demo - Credentials Setup"
echo "========================================"
echo ""

# Check if the config file exists
if [ ! -f "$CONFIG_FILE" ]; then
  echo "Error: Config file not found at $CONFIG_FILE"
  exit 1
fi

# Get app credentials
read -p "Enter your App ID: " APP_ID
read -p "Enter your API Key: " API_KEY
read -p "Enter your Read Stream ID: " READ_STREAM
read -p "Enter your Write Stream ID: " WRITE_STREAM
read -p "Enter your Group ID: " GROUP_ID

# Ask about tenant
echo ""
echo "Tenant configuration:"
echo "1. I want to link a new tenant through the UI"
echo "2. I want to use existing tenant credentials"
read -p "Choose an option (1/2): " TENANT_OPTION

if [ "$TENANT_OPTION" = "2" ]; then
  read -p "Enter Tenant ID: " TENANT_ID
  read -p "Enter Tenant Name: " TENANT_NAME
  read -p "Enter Tenant Token: " TENANT_TOKEN
fi

# Update the config file
sed -i '' "s|id: \"your-app-id\"|id: \"$APP_ID\"|g" "$CONFIG_FILE"
sed -i '' "s|apiKey: \"your-api-key\"|apiKey: \"$API_KEY\"|g" "$CONFIG_FILE"
sed -i '' "s|readStream: \"your-read-stream-id\"|readStream: \"$READ_STREAM\"|g" "$CONFIG_FILE"
sed -i '' "s|writeStream: \"your-write-stream-id\"|writeStream: \"$WRITE_STREAM\"|g" "$CONFIG_FILE"
sed -i '' "s|groupId: \"your-group-id\"|groupId: \"$GROUP_ID\"|g" "$CONFIG_FILE"

if [ "$TENANT_OPTION" = "2" ]; then
  sed -i '' "s|id: \"\"|id: \"$TENANT_ID\"|g" "$CONFIG_FILE"
  sed -i '' "s|name: \"\"|name: \"$TENANT_NAME\"|g" "$CONFIG_FILE"
  sed -i '' "s|token: \"\"|token: \"$TENANT_TOKEN\"|g" "$CONFIG_FILE"
fi

echo ""
echo "Configuration updated successfully!"
echo "You can now run the application with: docker-compose -f docker-compose.real.yml up --build"
