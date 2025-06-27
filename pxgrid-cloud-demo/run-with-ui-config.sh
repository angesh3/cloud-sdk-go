#!/bin/bash

# Make sure config directory exists
mkdir -p config

# Create a basic config file if it doesn't exist
if [ ! -f ./config/config.yaml ]; then
    echo "Creating default config file..."
    cat > ./config/config.yaml <<EOF
app:
  id: "demo-app"
  apiKey: ""
  globalFQDN: "api.pxgrid.cisco.com"
  regionalFQDN: "api.pxgrid.cisco.com"
  regionalFQDNs:
    - "api.pxgrid.cisco.com"
  readStream: ""
  writeStream: ""
  groupId: ""
tenant:
  otp: ""
  id: ""
  name: ""
  token: ""
EOF
fi

# Stop any existing containers
docker-compose -f docker-compose.ui-config.yml down

# Build and start the container
docker-compose -f docker-compose.ui-config.yml up --build -d

echo ""
echo "pxGrid Cloud Demo is starting..."
echo "Access the UI at: http://localhost:8084"
echo "Go to http://localhost:8084/settings.html to configure your credentials"
echo ""
echo "Press Ctrl+C to stop viewing logs"
echo ""

# Show logs
docker-compose -f docker-compose.ui-config.yml logs -f
