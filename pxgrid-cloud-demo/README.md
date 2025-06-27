# Cisco pxGrid Cloud SDK Demo

A containerized demo application showcasing the functionality of the Cisco pxGrid Cloud SDK with a modern web UI.

## Features

- Modern web-based UI for monitoring pxGrid Cloud messages
- Real-time device status monitoring
- WebSocket-based live updates
- Tenant linking/unlinking functionality
- Docker-based deployment for easy setup

## Prerequisites

- Docker and Docker Compose installed
- Cisco pxGrid Cloud app credentials:
  - App ID
  - API Key
  - Read/Write Stream IDs
- Access to Cisco DNA-Cloud portal for tenant OTPs

## Configuration

Before running the application, update the configuration file in `config/config.yaml`:

```yaml
app:
  id: your-app-id                    # Your pxGrid Cloud App ID
  apiKey: your-api-key               # Your pxGrid Cloud App API key
  globalFQDN: dnaservices.cisco.com  # Global FQDN (usually no need to change)
  regionalFQDN: neoffers.cisco.com   # Regional FQDN (usually no need to change)
  readStream: your-app-read-stream   # Your app's read stream ID
  writeStream: your-app-write-stream # Your app's write stream ID
  groupId: ""                        # Optional group ID

tenant:
  # Either provide an OTP to link a new tenant
  otp: ""
  # Or leave these blank and use the UI to link a tenant
  id: ""
  name: ""
  token: ""
```

## Running the Demo

1. Clone this repository

2. Configure the application
   - Edit `config/config.yaml` with your credentials

3. Build and start the container:
   ```bash
   docker-compose up -d
   ```

4. Access the web UI:
   Open your browser and navigate to http://localhost:8080

## Using the Demo

### Linking a Tenant

1. Obtain an OTP from the Cisco DNA-Cloud portal
2. Enter the OTP in the "Link tenant with OTP" field
3. Click the "Link" button

### Monitoring Devices and Messages

- Connected devices will be displayed in the "Connected Devices" table
- Messages from devices will appear in real-time in the "Message Stream" section

### Unlinking a Tenant

1. Click the "Unlink Tenant" button to disconnect from the current tenant

## Development

The demo application consists of:

- **Backend**: Go application using the Cisco pxGrid Cloud SDK
- **Frontend**: HTML/CSS/JavaScript web application
- **Docker**: Containerized deployment

## License

This demo application is covered by the same license as the Cisco pxGrid Cloud SDK.
