// Intercept console errors for specific patterns
const originalConsoleError = console.error;
console.error = function() {
    // Check if this is a disconnect device error
    const errorString = Array.prototype.join.call(arguments, ' ');
    if (errorString.includes('disconnect') && errorString.includes('device')) {
        // Silently ignore disconnect device errors
        return;
    }
    // Pass through to original console.error for all other errors
    originalConsoleError.apply(console, arguments);
};

// Global variables
let ws;
let deviceData = [];
let messageData = [];
let securityGroupsData = [];
let tenants = [];
let currentTenantId = null;
let socket;
let isConnected = false;
let currentTenant = null;
let devicesList = [];

// Note: hiddenFieldDetails function is now defined in utils.js

let selectedDeviceId = null;

// Functions for hiding device and tenant details
function hideDeviceDetails() {
    const detailsCard = document.getElementById('device-details-card');
    if (detailsCard) {
        detailsCard.classList.add('d-none');
    }
    selectedDeviceId = null;
}

function hideTenantDetails() {
    const tenantDetailsCard = document.getElementById('tenant-details-card');
    if (tenantDetailsCard) {
        tenantDetailsCard.classList.add('d-none');
    }
}

// DOM Elements
const connectionStatus = document.getElementById('connection-status');
const tenantBadge = document.getElementById('tenant-badge');
const tenantInfo = document.getElementById('tenant-info');
const devicesTableBody = document.getElementById('devices-table-body');
const messageContainer = document.getElementById('message-container');
const linkTenantBtn = document.getElementById('link-tenant-btn');
const unlinkTenantBtn = document.getElementById('unlink-tenant-btn');
const otpInput = document.getElementById('otp-input');
const clearMessagesBtn = document.getElementById('clear-messages');

// Security Groups KPI DOM Elements
const refreshSecurityGroupsBtn = document.getElementById('refresh-security-groups');
const securityGroupsCount = document.getElementById('security-groups-count');
const deletedGroupsCount = document.getElementById('deleted-groups-count');
const tagValuesCount = document.getElementById('tag-values-count');
const securityGroupsLastUpdated = document.getElementById('security-groups-last-updated');

// Settings DOM Elements
const settingsForm = document.getElementById('settings-form');
const appIdInput = document.getElementById('app-id');
const apiKeyInput = document.getElementById('api-key');
const readStreamInput = document.getElementById('read-stream');
const writeStreamInput = document.getElementById('write-stream');
const groupIdInput = document.getElementById('group-id');
const restartBtn = document.getElementById('restart-btn');

// Initialize app when the page is fully loaded
document.addEventListener('DOMContentLoaded', function() {
    // Check if WebSocket is supported
    if (!window.WebSocket) {
        alert('WebSocket is not supported by your browser');
        return;
    }
    
    // Setup WebSocket connection
    setupWebSocket();
    
    // Setup event listeners
    setupEventListeners();
    
    // Get initial application status
    fetchAppStatus();
    
    // Load settings
    fetchSettings();
    
    // Load tenants data - this will initialize currentTenant
    fetchTenants();
    
    // Load security groups data after a short delay to ensure tenant data is loaded
    setTimeout(() => {
        fetchSecurityGroupsData();
    }, 1000);
});

function setupWebSocket() {
    // Get WebSocket URL based on the backend service
    const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws';
    const host = window.location.hostname;
    const wsUrl = `${protocol}://${host}:8080/ws`;
    
    console.log('Connecting to WebSocket:', wsUrl);
    
    // Create WebSocket connection
    socket = new WebSocket(wsUrl);
    
    // WebSocket event handlers
    socket.onopen = handleSocketOpen;
    socket.onmessage = handleSocketMessage;
    socket.onclose = handleSocketClose;
    socket.onerror = handleSocketError;
}

function handleSocketOpen() {
    isConnected = true;
    updateConnectionStatus();
    showToast('WebSocket connected', 'success');
}

function handleSocketMessage(event) {
    const data = JSON.parse(event.data);
    
    if (data.type === 'devices') {
        deviceData = data.devices;
        updateDevicesTable();
    } else if (data.type === 'message') {
        addMessage(data.message);
    } else if (data.type === 'tenants') {
        // Update tenants data
        tenants = data.tenants;
        updateTenantTabs(tenants);
    }

    // Update connection status
    updateConnectionStatus();
}

function handleSocketClose() {
    isConnected = false;
    updateConnectionStatus();
    
    // Try to reconnect after 5 seconds
    setTimeout(setupWebSocket, 5000);
    showToast('WebSocket disconnected. Attempting to reconnect...', 'warning');
}

function handleSocketError(error) {
    console.error('WebSocket error:', error);
    showToast('WebSocket error', 'danger');
}

function setupEventListeners() {
    // Security Groups refresh button click
    if (refreshSecurityGroupsBtn) {
        refreshSecurityGroupsBtn.addEventListener('click', function() {
            fetchSecurityGroupsData();
        });
    }
    
    // Link tenant button
    if (linkTenantBtn) {
        linkTenantBtn.addEventListener('click', linkTenant);
    }
    
    // Unlink tenant button
    if (unlinkTenantBtn) {
        unlinkTenantBtn.addEventListener('click', unlinkTenant);
    }
    
    // Clear messages button
    if (clearMessagesBtn) {
        clearMessagesBtn.addEventListener('click', clearMessages);
    }
    
    // View all SGTs button
    const viewAllSgtBtn = document.getElementById('view-all-sgt-btn');
    if (viewAllSgtBtn) {
        viewAllSgtBtn.addEventListener('click', showSecurityGroupsList);
    }
    
    // View all SGACLs button
    const viewAllSgaclBtn = document.getElementById('view-all-sgacl-btn');
    if (viewAllSgaclBtn) {
        viewAllSgaclBtn.addEventListener('click', showSecurityGroupAclsList);
    }
    
    // View egress policies button
    const viewEgressPoliciesBtn = document.getElementById('view-egress-policies-btn');
    if (viewEgressPoliciesBtn) {
        viewEgressPoliciesBtn.addEventListener('click', showEgressPolicies);
    }
    
    // Clear device and tenant selection when clicking outside relevant cards
    document.addEventListener('click', (event) => {
        const detailsCard = document.getElementById('device-details-card');
        const tenantDetailsCard = document.getElementById('tenant-details-card');
        const devicesTable = document.getElementById('devices-table');
        
        if (detailsCard && !detailsCard.contains(event.target) && 
            devicesTable && !devicesTable.contains(event.target)) {
            hideDeviceDetails();
        }
        
        if (tenantDetailsCard && !tenantDetailsCard.contains(event.target) &&
            !event.target.classList.contains('tenant-name')) {
            hideTenantDetails();
        }
    });
    
    // Settings form
    const settingsForm = document.getElementById('settings-form');
    if (settingsForm) {
        settingsForm.addEventListener('submit', function(e) {
            e.preventDefault();
            saveSettings(e);
        });
    }
    
    // Restart button
    const restartBtn = document.getElementById('restart-btn');
    if (restartBtn) {
        restartBtn.addEventListener('click', function() {
            restartApplication();
        });
    }
    
    // Set up tab navigation event listeners
    const tabs = document.querySelectorAll('[data-bs-toggle="tab"]');
    tabs.forEach(tab => {
        tab.addEventListener('shown.bs.tab', function (event) {
            // Store the currently active tab in session storage
            sessionStorage.setItem('activeTab', event.target.getAttribute('href'));
        });
    });
    
    // Restore active tab from session storage
    const activeTabId = sessionStorage.getItem('activeTab');
    if (activeTabId) {
        const activeTab = document.querySelector(`[href="${activeTabId}"]`);
        if (activeTab) {
            const tab = new bootstrap.Tab(activeTab);
            tab.show();
        }
    }
}

function fetchAppStatus() {
    const apiUrl = `/api/status`;
    fetch(apiUrl)
        .then(response => response.json())
        .then(data => {
            if (data.tenant) {
                currentTenant = data.tenant;
                updateTenantUI();
            }
        })
        .catch(error => {
            console.error('Error fetching app status:', error);
            showToast('Failed to fetch application status', 'danger');
        });
}

// Fetch tenants from API
function fetchTenants() {
    console.log('Fetching tenants data...');
    const apiUrl = '/api/tenants';
    
    // Hide the "Link New Tenant" text that might appear in the message container
    updateMessageContainer();
    
    fetch(apiUrl)
        .then(response => response.json())
        .then(data => {
            console.log('Tenants data received:', data);
            tenants = data;
            
            // Find default tenant if any
            const defaultTenant = tenants.find(tenant => tenant.isDefault);
            if (defaultTenant) {
                console.log('Default tenant found:', defaultTenant);
                currentTenant = defaultTenant;
                currentTenantId = defaultTenant.id;
                updateTenantUI();
                
                // Update the message container to show the connected tenant
                const messageContainer = document.getElementById('message-container');
                if (messageContainer) {
                    const tenantInfo = document.createElement('div');
                    tenantInfo.className = 'alert alert-success';
                    tenantInfo.innerHTML = `
                        <i class="fa-solid fa-check-circle me-2"></i>
                        Connected to tenant: <strong>${defaultTenant.name}</strong> (ID: ${defaultTenant.id})
                    `;
                    
                    // Add the tenant info at the top of message container
                    if (messageContainer.firstChild) {
                        messageContainer.insertBefore(tenantInfo, messageContainer.firstChild);
                    } else {
                        messageContainer.appendChild(tenantInfo);
                    }
                }
            }
            
            // Fetch devices for the default tenant
            if (defaultTenant) {
                fetchTenantDevices(defaultTenant.id);
            }
        })
        .catch(error => {
            console.error('Error fetching tenants:', error);
        });
}

function linkTenant() {
    const otp = otpInput.value.trim();
    
    if (!otp) {
        showToast('Please enter a valid OTP', 'warning');
        return;
    }
    
    // Disable button and show loading state
    linkTenantBtn.disabled = true;
    linkTenantBtn.innerHTML = '<span class="spinner-border spinner-border-sm" role="status" aria-hidden="true"></span> Linking...';
    
    // Send API request to link tenant
    fetch(`/api/tenant/link`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({ otp }),
    })
    .then(async response => {
        if (!response.ok) {
            // Try to extract error message from response
            let errorMsg = 'Failed to link tenant';
            try {
                const errorData = await response.json();
                if (errorData && errorData.error) {
                    errorMsg = errorData.error;
                }
            } catch (e) { 
                // If response isn't valid JSON, use status text
                errorMsg = `Failed to link tenant: ${response.statusText || 'Unknown error'}`; 
            }
            throw new Error(errorMsg);
        }
        return response.json();
    })
    .then(data => {
        if (!data || !data.name) {
            throw new Error('Invalid response from server: Missing tenant name');
        }
        currentTenant = data.name;
        updateTenantUI();
        showToast(`Successfully linked tenant: ${data.name}`, 'success');
        otpInput.value = '';
    })
    .catch(error => {
        // Log the full error details including the message
        console.error('Error linking tenant:', {
            message: error.message,
            stack: error.stack,
            name: error.name
        });
        showToast(error.message || 'Failed to link tenant. Check the OTP and try again.', 'danger');
    })
    .finally(() => {
        // Reset button state
        linkTenantBtn.disabled = false;
        linkTenantBtn.innerHTML = '<i class="fa-solid fa-link me-2"></i>Link';
    });
}

function unlinkTenant() {
    if (!currentTenant) {
        showToast('No tenant currently linked', 'warning');
        return;
    }
    
    // Confirm before unlinking
    if (!confirm(`Are you sure you want to unlink tenant "${currentTenant}"?`)) {
        return;
    }
    
    // Disable button and show loading state
    unlinkTenantBtn.disabled = true;
    unlinkTenantBtn.innerHTML = '<span class="spinner-border spinner-border-sm" role="status" aria-hidden="true"></span> Unlinking...';
    
    // Send API request to unlink tenant
    fetch(`/api/tenant/unlink`, {
        method: 'POST',
    })
    .then(response => {
        if (!response.ok) {
            throw new Error('Failed to unlink tenant');
        }
        return response.json();
    })
    .then(data => {
        currentTenant = null;
        updateTenantUI();
        showToast(data.message, 'success');
        
        // Clear devices list
        devicesList = [];
        updateDevicesTable();
    })
    .catch(error => {
        console.error('Error unlinking tenant:', error);
        showToast('Failed to unlink tenant', 'danger');
    })
    .finally(() => {
        // Reset button state
        unlinkTenantBtn.disabled = true;
        unlinkTenantBtn.innerHTML = '<i class="fa-solid fa-unlink me-2"></i>Unlink Tenant';
    });
}

function updateTenantUI() {
    // Update tenant indicator elements
    const activeTenantName = document.getElementById('active-tenant-name');
    const currentTenantSelector = document.getElementById('current-tenant-selector');
    const currentTenantNameSpan = document.getElementById('current-tenant-name');
    const navTenantName = document.getElementById('nav-tenant-name');
    const tenantNavItem = document.getElementById('tenant-nav-item');
    
    // Get the tenants loading elements
    const tenantsLoading = document.getElementById('tenantsLoading');
    const messageContainer = document.getElementById('message-container');
    
    if (currentTenant) {
        // Hide the loading spinner in the message stream area
        if (tenantsLoading) {
            tenantsLoading.style.display = 'none';
        }
        
        // Update badge
        tenantBadge.textContent = currentTenant.name || currentTenant;
        tenantBadge.classList.remove('bg-secondary');
        tenantBadge.classList.add('bg-info');
        
        // Update tenant info
        tenantInfo.innerHTML = `
            <div class="alert alert-success" role="alert">
                <i class="fa-solid fa-check-circle me-2"></i>
                Connected to tenant: <strong>${currentTenant.name || currentTenant}</strong>
            </div>
        `;
        
        // Update dropdown and navigation indicators
        if (activeTenantName) {
            activeTenantName.textContent = currentTenant.name || currentTenant;
        }
        
        if (currentTenantNameSpan) {
            currentTenantNameSpan.textContent = currentTenant.name || currentTenant;
        }
        
        // Update the nav tenant indicator
        if (navTenantName) {
            navTenantName.textContent = currentTenant.name || currentTenant;
        }
        
        if (tenantNavItem) {
            tenantNavItem.style.display = 'block';
        }
        
        unlinkTenantBtn.disabled = false;
        
        // Also update the Message Stream area to show the current tenant
        const tenantLoadingElement = document.getElementById('tenantsLoading');
        if (tenantLoadingElement) {
            tenantLoadingElement.innerHTML = `
                <p class="mt-2">
                    <i class="fa-solid fa-check-circle text-success me-2"></i>
                    Connected to tenant: <strong>${currentTenant.name || currentTenant}</strong>
                </p>
            `;
        }
    } else {
        // Update badge
        tenantBadge.textContent = 'No Tenant';
        tenantBadge.classList.remove('bg-info');
        tenantBadge.classList.add('bg-secondary');
        
        // Update tenant info
        tenantInfo.innerHTML = `
            <div class="alert alert-info" role="alert">
                <i class="fa-solid fa-info-circle me-2"></i>
                To connect to a tenant, either provide an OTP from the Cisco DNA-Cloud portal or use existing tenant credentials.
            </div>
        `;
        
        // Update dropdown and navigation indicators
        if (activeTenantName) {
            activeTenantName.textContent = 'Select Tenant';
        }
        
        if (currentTenantNameSpan) {
            currentTenantNameSpan.textContent = 'No Tenant';
        }
        
        // Hide the nav tenant indicator
        if (navTenantName) {
            navTenantName.textContent = 'None';
        }
        
        if (tenantNavItem) {
            tenantNavItem.style.display = 'none';
        }
        
        unlinkTenantBtn.disabled = true;
    }
}

function updateConnectionStatus() {
    if (isConnected) {
        connectionStatus.textContent = 'Connected';
        connectionStatus.classList.remove('bg-danger');
        connectionStatus.classList.add('bg-success');
    } else {
        connectionStatus.textContent = 'Disconnected';
        connectionStatus.classList.remove('bg-success');
        connectionStatus.classList.add('bg-danger');
    }
}

function updateDevicesList(devices) {
    devicesList = devices;
    updateDevicesTable();
}

// Fetch devices for a specific tenant
function fetchTenantDevices(tenantId) {
    console.log('Fetching devices for tenant:', tenantId);
    const apiUrl = `/api/tenants/${tenantId}/devices`;
    
    fetch(apiUrl)
        .then(response => response.json())
        .then(data => {
            console.log('Devices data received:', data);
            updateDevicesList(data);
        })
        .catch(error => {
            console.error('Error fetching devices:', error);
        });
}

function updateDevicesTable() {
    if (devicesList.length === 0) {
        devicesTableBody.innerHTML = `
            <tr>
                <td colspan="3" class="text-center">No devices connected</td>
            </tr>
        `;
        // Hide device details since there are no devices
        hideDeviceDetails();
        hideTenantDetails();
        return;
    }
    
    // Sort devices by status (active first) then by name
    const sortedDevices = [...devicesList].sort((a, b) => {
        if (a.status === b.status) {
            return a.name.localeCompare(b.name);
        }
        return a.status === 'active' ? -1 : 1;
    });
    
    devicesTableBody.innerHTML = sortedDevices.map(device => `
        <tr data-device-id="${device.id}" class="${selectedDeviceId === device.id ? 'table-active' : ''}" style="cursor:pointer;">
            <td>${device.name}</td>
            <td>
                <span class="status-${device.status.toLowerCase()}">
                    <i class="fa-solid fa-${device.status.toLowerCase() === 'active' ? 'circle-check' : 'circle-xmark'} me-1"></i>
                    ${device.status}
                </span>
            </td>
            <td>
                <span class="tenant-name" data-tenant-name="${device.tenantName}" style="cursor:pointer;">${device.tenantName}</span>
            </td>
        </tr>
    `).join('');
    
    // Add click event to table rows for device selection
    document.querySelectorAll('#devices-table-body tr[data-device-id]').forEach(row => {
        row.addEventListener('click', (event) => {
            // Only select device if the click wasn't on the tenant name
            if (!event.target.classList.contains('tenant-name')) {
                const deviceId = row.getAttribute('data-device-id');
                selectDevice(deviceId);
            }
        });
    });
    
    // Add click event for tenant names
    document.querySelectorAll('.tenant-name').forEach(tenantElement => {
        tenantElement.addEventListener('click', (event) => {
            event.stopPropagation(); // Prevent device selection
            const tenantName = tenantElement.getAttribute('data-tenant-name');
            showTenantDetails(tenantName);
        });
    });
    
    // If a device was selected, make sure it's still shown in details
    if (selectedDeviceId) {
        const device = devicesList.find(d => d.id === selectedDeviceId);
        if (device) {
            showDeviceDetails(device);
        } else {
            hideDeviceDetails();
        }
    }
}

function addMessage(message) {
    // Remove the "no messages" placeholder if it exists
    const placeholder = messageContainer.querySelector('.text-center');
    if (placeholder) {
        messageContainer.removeChild(placeholder);
    }
    
    // Format the message timestamp
    const timestamp = new Date(message.time);
    const formattedTime = timestamp.toLocaleTimeString();
    
    // Create the message element
    const messageElement = document.createElement('div');
    messageElement.className = 'message';
    messageElement.innerHTML = `
        <div class="d-flex justify-content-between">
            <div>
                <span class="device-name">${message.deviceName}</span>
                <span class="stream-name">${message.stream}</span>
            </div>
            <span class="time">${formattedTime}</span>
        </div>
        <div class="content">${formatMessageContent(message.content)}</div>
    `;
    
    // Add the new message to the container
    messageContainer.appendChild(messageElement);
    
    // Scroll to the bottom
    messageContainer.scrollTop = messageContainer.scrollHeight;
}

function loadMessages(messages) {
    // Clear existing messages
    messageContainer.innerHTML = '';
    
    if (messages.length === 0) {
        messageContainer.innerHTML = `
            <div class="text-center text-muted my-4">
                <i class="fa-solid fa-comment-slash fa-2x mb-3"></i>
                <p>No messages received yet</p>
            </div>
        `;
        return;
    }
    
    // Add all messages
    messages.forEach(message => addMessage(message));
}

function clearMessages() {
    messageContainer.innerHTML = `
        <div class="text-center text-muted my-4">
            <i class="fa-solid fa-comment-slash fa-2x mb-3"></i>
            <p>No messages received yet</p>
        </div>
    `;
}

// Function to update the message container, removing any unwanted content
function updateMessageContainer() {
    const msgContainer = document.getElementById('message-container');
    if (msgContainer) {
        // Clear any "Link New Tenant" text or buttons that might be showing
        const linkNewTenantElements = msgContainer.querySelectorAll('button, .link-tenant-btn');
        linkNewTenantElements.forEach(element => element.remove());
        
        // Make sure we have the standard "No messages" placeholder if empty
        if (!msgContainer.innerHTML.trim()) {
            msgContainer.innerHTML = `
                <div class="text-center text-muted my-4">
                    <i class="fa-solid fa-comment-slash fa-2x mb-3"></i>
                    <p>No messages received yet</p>
                </div>
            `;
        }
    }
}

function formatMessageContent(content) {
    // Try to parse the content as JSON for prettier display
    try {
        const json = JSON.parse(content);
        return `<pre class="mb-0">${JSON.stringify(json, null, 2)}</pre>`;
    } catch (e) {
        // If not JSON, return as is
        return `<pre class="mb-0">${content}</pre>`;
    }
}

// Settings Functions
function fetchSettings() {
    fetch(`/api/settings`)
        .then(response => {
            if (!response.ok) {
                throw new Error('Failed to fetch settings');
            }
            return response.json();
        })
        .then(data => {
            // Populate form fields
            appIdInput.value = data.app.id || '';
            // API key may be masked for security
            if (data.app.apiKey && !data.app.apiKey.includes('*')) {
                apiKeyInput.value = data.app.apiKey;
            }
            readStreamInput.value = data.app.readStream || '';
            writeStreamInput.value = data.app.writeStream || '';
            groupIdInput.value = data.app.groupId || '';
        })
        .catch(error => {
            console.error('Error fetching settings:', error);
            showToast('Failed to load settings. Please try again.', 'danger');
        });
}

function saveSettings(event) {
    if (event) event.preventDefault();
    
    // Get form elements regardless of which settings form was used
    const appIdElem = document.getElementById('app-id');
    const apiKeyElem = document.getElementById('api-key');
    const readStreamElem = document.getElementById('read-stream');
    const writeStreamElem = document.getElementById('write-stream');
    const groupIdElem = document.getElementById('group-id');
    
    if (!appIdElem) {
        console.error('Settings form elements not found');
        return;
    }
    
    const settings = {
        appId: appIdElem.value.trim(),
        apiKey: apiKeyElem.value.trim(),
        readStream: readStreamElem.value.trim(),
        writeStream: writeStreamElem.value.trim(),
        groupId: groupIdElem.value.trim()
    };
    
    // Don't send empty API key (to avoid overwriting existing key)
    if (!settings.apiKey) {
        delete settings.apiKey;
    }
    
    const settingsForm = document.getElementById('settings-form');
    const submitBtn = settingsForm.querySelector('button[type="submit"]');
    const originalBtnText = submitBtn.innerHTML;
    
    // Show loading state
    submitBtn.disabled = true;
    submitBtn.innerHTML = '<span class="spinner-border spinner-border-sm" role="status" aria-hidden="true"></span> Saving...';
    
    fetch(`/api/settings`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(settings),
    })
    .then(response => {
        if (!response.ok) {
            throw new Error('Failed to save settings');
        }
        return response.json();
    })
    .then(data => {
        showToast(data.message || 'Settings saved successfully', 'success');
        
        // Keep the settings tab active
        const settingsTab = document.querySelector('#settings-tab-link');
        if (settingsTab) {
            settingsTab.classList.add('active');
            document.querySelector('#settings-tab').classList.add('show', 'active');
            document.querySelector('#dashboard-tab').classList.remove('show', 'active');
            document.querySelector('#dashboard-tab-link').classList.remove('active');
        }
    })
    .catch(error => {
        console.error('Error saving settings:', error);
        showToast('Failed to save settings. Please try again.', 'danger');
    })
    .finally(() => {
        // Reset button state
        submitBtn.disabled = false;
        submitBtn.innerHTML = originalBtnText;
    });
}

function restartApplication() {
    if (!confirm('Are you sure you want to restart the application? This will disconnect all active sessions.')) {
        return;
    }
    
    // Get the restart button specifically to avoid DOM element issues
    const restartBtn = document.getElementById('restart-btn');
    if (!restartBtn) {
        console.error('Restart button not found');
        return;
    }
    
    // Show loading state
    restartBtn.disabled = true;
    restartBtn.innerHTML = '<span class="spinner-border spinner-border-sm" role="status" aria-hidden="true"></span> Restarting...';
    
    showToast('Initiating application restart...', 'info');
    
    fetch(`/api/restart`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({})
    })
    .then(response => {
        if (!response.ok) {
            throw new Error('Failed to restart application');
        }
        return response.json();
    })
    .then(data => {
        showToast(data.message || 'Application is restarting...', 'success');
        
        // After successful restart request, wait for 5 seconds then try to reconnect
        showToast('Application restarting. This page will reload in 5 seconds...', 'info');
        setTimeout(() => {
            window.location.reload();
        }, 5000);
    })
    .catch(error => {
        console.error('Error restarting application:', error);
        showToast('Failed to restart application: ' + error.message, 'danger');
        restartBtn.disabled = false;
        restartBtn.innerHTML = '<i class="fas fa-sync-alt me-1"></i> Restart Application';
    });
}

function selectDevice(deviceId) {
    selectedDeviceId = deviceId;
    
    // Highlight the selected row
    document.querySelectorAll('#devices-table-body tr').forEach(row => {
        row.classList.remove('table-active');
    });
    
    const selectedRow = document.querySelector(`#devices-table-body tr[data-device-id="${deviceId}"]`);
    if (selectedRow) {
        selectedRow.classList.add('table-active');
    }
    
    // Show device details
    const device = devicesList.find(d => d.id === deviceId);
    if (device) {
        showDeviceDetails(device);
    }
}

function showDeviceDetails(device) {
    const detailsCard = document.getElementById('device-details-card');
    if (!detailsCard) {
        return;
    }
    
    detailsCard.style.display = 'block';
    
    // Update device details content
    const detailsContent = document.getElementById('device-details-content');
    
    const lastSeen = new Date().toLocaleString();
    
    detailsContent.innerHTML = `
        <div class="device-detail-header mb-3 d-flex justify-content-between align-items-center">
            <h4>${device.name}</h4>
            <span class="badge ${device.status === 'active' ? 'bg-success' : 'bg-danger'}">${device.status}</span>
        </div>
        
        <table class="table table-sm">
            <tbody>
                <tr>
                    <th>Device ID</th>
                    <td><code>${device.id}</code></td>
                </tr>
                <tr>
                    <th>Tenant</th>
                    <td>${device.tenantName}</td>
                </tr>
                <tr>
                    <th>Status</th>
                    <td>
                        <span class="status-${device.status.toLowerCase()}">
                            <i class="fa-solid fa-${device.status.toLowerCase() === 'active' ? 'circle-check' : 'circle-xmark'} me-1"></i>
                            ${device.status}
                        </span>
                    </td>
                </tr>
                <tr>
                    <th>Last Seen</th>
                    <td>${lastSeen}</td>
                </tr>
            </tbody>
        </table>
        
        <div class="mt-3 d-flex justify-content-between">
            <button id="view-messages-btn" class="btn btn-primary btn-sm">
                <i class="fa-solid fa-comment-dots me-1"></i> View Messages
            </button>
            <div>
                <button id="copy-device-id-btn" class="btn btn-secondary btn-sm" data-device-id="${device.id}">
                    <i class="fa-solid fa-copy me-1"></i> Copy ID
                </button>
            </div>
        </div>
    `;
    
    // Add event listener for copy button
    const copyBtn = document.getElementById('copy-device-id-btn');
    if (copyBtn) {
        // Directly use the device ID from the device object instead of relying on the attribute
        copyBtn.addEventListener('click', () => {
            console.log('Copying device ID:', device.id);
            navigator.clipboard.writeText(device.id)
                .then(() => showToast('Device ID copied to clipboard', 'success'))
                .catch(err => console.error('Failed to copy: ', err));
        });
    }
    
    // Add event listener for view messages button
    const viewMessagesBtn = document.getElementById('view-messages-btn');
    if (viewMessagesBtn) {
        viewMessagesBtn.addEventListener('click', () => {
            filterMessagesByDevice(device.name);
        });
    }
}

// This function is a placeholder since the backend API endpoint is not implemented
function disconnectDevice(deviceId) {
    try {
        console.log('Disconnect device requested for ID:', deviceId);
        
        if (!deviceId) {
            showToast('Cannot disconnect device: Invalid device ID', 'danger');
            return;
        }
        
        if (!currentTenant) {
            showToast('Cannot disconnect device: No tenant connected', 'danger');
            return;
        }
        
        if (!confirm('Are you sure you want to disconnect this device?')) {
            return;
        }
        
        // Show message that this feature is not available in this version
        showToast('Disconnect device feature is not available in this version of the SDK demo.', 'warning');
        
        // Hide device details panel
        const detailsCard = document.getElementById('device-details-card');
        if (detailsCard) {
            detailsCard.style.display = 'none';
        }
        
        // Clear any selected device to prevent further actions
        selectedDeviceId = null;
    } catch (error) {
        // Prevent any errors from bubbling up to the console
        showToast('An error occurred while processing the disconnect request', 'danger');
    }
}

function highlightDeviceMessages(deviceName) {
    const messageContainer = document.getElementById('messages-container');
    if (!messageContainer) return;
    
    const messages = messageContainer.querySelectorAll('.message');
    messages.forEach(msg => {
        const msgDeviceName = msg.querySelector('.device-name')?.textContent;
        
        if (msgDeviceName === deviceName) {
            msg.classList.add('message-highlight');
            // Scroll to the first message from this device
            msg.scrollIntoView({ behavior: 'smooth', block: 'center' });
        } else {
            msg.classList.remove('message-highlight');
        }
    });
    
    // Show toast
    showToast(`Showing messages from device: ${deviceName}`, 'info');
}

function showToast(message, type = 'info') {
    const toastId = `toast-${Date.now()}`;
    const toastContainer = document.getElementById('toast-container');
    
    const toast = document.createElement('div');
    toast.className = `toast show bg-${type} text-light`;
    toast.setAttribute('role', 'alert');
    toast.setAttribute('aria-live', 'assertive');
    toast.setAttribute('aria-atomic', 'true');
    toast.setAttribute('id', toastId);
    
    toast.innerHTML = `
        <div class="toast-header bg-${type} text-light">
            <strong class="me-auto">Notification</strong>
            <button type="button" class="btn-close btn-close-white" data-bs-dismiss="toast" aria-label="Close"></button>
        </div>
        <div class="toast-body">
            ${message}
        </div>
    `;
    
    toastContainer.appendChild(toast);
    
    // Auto-hide the toast after 5 seconds
    setTimeout(() => {
        toast.classList.remove('show');
        
        // Remove from DOM after hide animation
        setTimeout(() => {
            if (document.getElementById(toastId)) {
                toastContainer.removeChild(toast);
            }
        }, 500);
    }, 5000);
    
    // Add click event to close button
    toast.querySelector('.btn-close').addEventListener('click', () => {
        toast.classList.remove('show');
        setTimeout(() => {
            if (document.getElementById(toastId)) {
                toastContainer.removeChild(toast);
            }
        }, 500);
    });
}
// Fetch security groups data from API
function fetchSecurityGroupsData() {
    console.log('Fetching security groups data, tenant:', currentTenant, 'devices:', devicesList.length);
    
    // If currentTenant is not set, try to get it from the tenants list first
    if (!currentTenant && tenants && tenants.length > 0) {
        // Find default tenant
        const defaultTenant = tenants.find(tenant => tenant.isDefault);
        if (defaultTenant) {
            console.log('Using default tenant from list:', defaultTenant);
            currentTenant = defaultTenant;
        }
    }
    
    // Only fetch if there's a linked tenant
    if (!currentTenant) {
        resetSecurityGroupsUI();
        securityGroupsLastUpdated.textContent = 'Not available - connect a tenant first';
        showToast('Please link a tenant before accessing security groups data', 'warning');
        return;
    }
    
    // Check if we have any active devices
    const activeDevices = devicesList.filter(device => device.status.toLowerCase() === 'active');
    if (activeDevices.length === 0) {
        resetSecurityGroupsUI();
        securityGroupsLastUpdated.textContent = 'Not available - no active devices';
        showToast('Active devices are needed to query security groups data', 'warning');
        return;
    }
    
    // Add loading indicator
    securityGroupsLastUpdated.textContent = 'Loading data...';
    showToast('Fetching security groups data...', 'info');
    
    fetch('/api/security-groups/kpi')
        .then(async response => {
            // Get the raw text regardless of status code
            const responseText = await response.text();
            console.log('Security Groups API Response:', response.status, responseText);
            
            if (!response.ok) {
                // Try to parse as JSON if possible
                let errorMsg = 'Failed to fetch security groups data';
                
                try {
                    const errorData = JSON.parse(responseText);
                    if (errorData.error) {
                        errorMsg = errorData.error;
                    }
                } catch (e) {
                    // If it's not valid JSON, use the response text as the error message
                    if (responseText && responseText.length < 100) {
                        errorMsg = responseText;
                    }
                }
                
                // Handle specific error cases
                if (errorMsg.includes('No active device found')) {
                    errorMsg = 'No active devices connected. Devices are needed to query security groups.';
                    securityGroupsLastUpdated.textContent = 'Waiting for device connection...';
                } else if (errorMsg.includes('No tenant linked')) {
                    errorMsg = 'Please link a tenant before accessing security groups data';
                    securityGroupsLastUpdated.textContent = 'Link a tenant to view data';
                } else {
                    // This is a server error - show details to help debugging
                    securityGroupsLastUpdated.textContent = 'API Error - See console for details';
                }
                
                throw new Error(errorMsg);
            }
            
            // Try to parse response as JSON
            try {
                return JSON.parse(responseText);
            } catch (e) {
                throw new Error('Invalid JSON response from server');
            }
        })
        .then(data => {
            securityGroupsData = data;
            updateSecurityGroupsUI(data);
            showToast('Security Groups data updated successfully', 'info');
        })
        .catch(error => {
            console.error('Error fetching security groups data:', error);
            // Always show the error to help with debugging
            showToast(`Security Groups API Error: ${error.message}`, 'danger');
            
            // Still update UI with empty data
            securityGroupsCount.textContent = '-';
            deletedGroupsCount.textContent = '-';
            tagValuesCount.textContent = '-';
        });
}

// Update security groups UI with data
function updateSecurityGroupsUI(data) {
    if (!data) return;
    
    // Update counts
    securityGroupsCount.textContent = data.totalCount || '0';
    deletedGroupsCount.textContent = data.deletedCount || '0';
    
    // Calculate tag values count
    const tagCount = Object.keys(data.tagDistribution || {}).length;
    tagValuesCount.textContent = tagCount;
    
    // Update last updated timestamp
    securityGroupsLastUpdated.textContent = 'Last updated: ' + new Date().toLocaleTimeString();
    
    // Add "View All" button if it doesn't exist
    // The button should be added at the bottom of the Security Groups KPI card body
    if (!document.getElementById('view-all-sgt-btn')) {
        const securityGroupsCard = document.querySelector('.card-header')
            ? Array.from(document.querySelectorAll('.card-header'))
                .find(header => header.textContent.includes('Security Groups KPI'))
                ?.closest('.card')
            : null;
            
        if (securityGroupsCard) {
            const cardBody = securityGroupsCard.querySelector('.card-body');
            if (cardBody) {
                // Create a container for the button
                const btnContainer = document.createElement('div');
                btnContainer.className = 'mt-3 text-center';
                
                // Create the button
                const viewAllBtn = document.createElement('button');
                viewAllBtn.id = 'view-all-sgt-btn';
                viewAllBtn.className = 'btn btn-primary';
                viewAllBtn.innerHTML = '<i class="fa-solid fa-list me-1"></i> View All Security Group Tags';
                viewAllBtn.addEventListener('click', showSecurityGroupsList);
                
                // Add the button to the container
                btnContainer.appendChild(viewAllBtn);
                
                // Add the container to the card body
                cardBody.appendChild(btnContainer);
                console.log('"View All SGTs" button added to Security Groups KPI card');
            }
        }
    }
}

// Show Security Group ACLs list in a modal
function showSecurityGroupAclsList() {
    // Show loading indicator
    const modalContent = `
        <div class="modal fade" id="sgacl-list-modal" tabindex="-1" aria-labelledby="sgacl-list-modal-label" aria-hidden="true">
            <div class="modal-dialog modal-lg">
                <div class="modal-content">
                    <div class="modal-header">
                        <h5 class="modal-title" id="sgacl-list-modal-label">Security Group ACLs</h5>
                        <button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
                    </div>
                    <div class="modal-body">
                        <div id="sgacl-loading" class="text-center">
                            <div class="spinner-border" role="status">
                                <span class="visually-hidden">Loading...</span>
                            </div>
                            <p>Loading security group ACLs...</p>
                        </div>
                        <div id="sgacl-error" class="alert alert-danger d-none"></div>
                        <div id="sgacl-table-container" class="d-none">
                            <table class="table table-striped table-hover">
                                <thead>
                                    <tr>
                                        <th>Name</th>
                                        <th>Description</th>
                                        <th>IP Version</th>
                                        <th>ACL Content</th>
                                    </tr>
                                </thead>
                                <tbody id="sgacl-table-body">
                                </tbody>
                            </table>
                        </div>
                    </div>
                    <div class="modal-footer">
                        <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">Close</button>
                    </div>
                </div>
            </div>
        </div>
    `;
    
    // Add modal to body if it doesn't exist
    if (!document.getElementById('sgacl-list-modal')) {
        const modalDiv = document.createElement('div');
        modalDiv.innerHTML = modalContent;
        document.body.appendChild(modalDiv);
    }
    
    // Show the modal
    const modal = new bootstrap.Modal(document.getElementById('sgacl-list-modal'));
    modal.show();
    
    // Fetch security group ACLs list
    fetch('/api/security-groups/acls')
        .then(async response => {
            const responseText = await response.text();
            console.log('Security Group ACLs Response:', response.status, responseText);
            
            if (!response.ok) {
                throw new Error('Failed to fetch security group ACLs');
            }
            
            try {
                return JSON.parse(responseText);
            } catch (e) {
                throw new Error('Invalid JSON response from server');
            }
        })
        .then(sgacls => {
            // Hide loading, show table
            document.getElementById('sgacl-loading').classList.add('d-none');
            document.getElementById('sgacl-table-container').classList.remove('d-none');
            
            // Populate table
            const tableBody = document.getElementById('sgacl-table-body');
            tableBody.innerHTML = '';
            
            if (sgacls && sgacls.length > 0) {
                // Sort by name
                sgacls.sort((a, b) => {
                    return (a.name || '').localeCompare(b.name || '');
                });
                
                sgacls.forEach(acl => {
                    const row = document.createElement('tr');
                    
                    // Format ACL content
                    let aclContent = '';
                    if (acl.aclcontent) {
                        aclContent = acl.aclcontent;
                    } else if (acl.aces && acl.aces.length > 0) {
                        aclContent = acl.aces.join('<br>');
                    }
                    
                    row.innerHTML = `
                        <td>${acl.name || '-'}</td>
                        <td>${acl.description || '-'}</td>
                        <td>${acl.ipVersion || '-'}</td>
                        <td><pre class="mb-0 small">${aclContent || '-'}</pre></td>
                    `;
                    tableBody.appendChild(row);
                });
            } else {
                const row = document.createElement('tr');
                row.innerHTML = '<td colspan="4" class="text-center">No security group ACLs found</td>';
                tableBody.appendChild(row);
            }
        })
        .catch(error => {
            console.error('Error fetching security group ACLs:', error);
            document.getElementById('sgacl-loading').classList.add('d-none');
            const errorElement = document.getElementById('sgacl-error');
            errorElement.classList.remove('d-none');
            errorElement.textContent = `Error loading security group ACLs: ${error.message}`;
        });
}

// Show Egress Policies in a modal
function showEgressPolicies() {
    // Show loading indicator
    const modalContent = `
        <div class="modal fade" id="egress-policies-modal" tabindex="-1" aria-labelledby="egress-policies-modal-label" aria-hidden="true">
            <div class="modal-dialog modal-xl">
                <div class="modal-content">
                    <div class="modal-header">
                        <h5 class="modal-title" id="egress-policies-modal-label">Egress Policies</h5>
                        <button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
                    </div>
                    <div class="modal-body">
                        <div id="egress-policies-loading" class="text-center">
                            <div class="spinner-border" role="status">
                                <span class="visually-hidden">Loading...</span>
                            </div>
                            <p>Loading egress policies...</p>
                        </div>
                        <div id="egress-policies-error" class="alert alert-danger d-none"></div>
                        <div id="egress-policies-table-container" class="d-none">
                            <table class="table table-striped table-hover">
                                <thead>
                                    <tr>
                                        <th>Source SGT</th>
                                        <th>Destination SGT</th>
                                        <th>ACL Name</th>
                                        <th>Default Rule</th>
                                        <th>Status</th>
                                    </tr>
                                </thead>
                                <tbody id="egress-policies-table-body">
                                </tbody>
                            </table>
                        </div>
                    </div>
                    <div class="modal-footer">
                        <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">Close</button>
                    </div>
                </div>
            </div>
        </div>
    `;
    
    // Add modal to body if it doesn't exist
    if (!document.getElementById('egress-policies-modal')) {
        const modalDiv = document.createElement('div');
        modalDiv.innerHTML = modalContent;
        document.body.appendChild(modalDiv);
    }
    
    // Show the modal
    const modal = new bootstrap.Modal(document.getElementById('egress-policies-modal'));
    modal.show();
    
    // Fetch egress policies list
    fetch('/api/security-groups/egress-policies')
        .then(async response => {
            const responseText = await response.text();
            console.log('Egress Policies Response:', response.status, responseText);
            
            if (!response.ok) {
                throw new Error('Failed to fetch egress policies');
            }
            
            try {
                return JSON.parse(responseText);
            } catch (e) {
                throw new Error('Invalid JSON response from server');
            }
        })
        .then(policies => {
            // Hide loading, show table
            document.getElementById('egress-policies-loading').classList.add('d-none');
            document.getElementById('egress-policies-table-container').classList.remove('d-none');
            
            // Populate table
            const tableBody = document.getElementById('egress-policies-table-body');
            tableBody.innerHTML = '';
            
            if (policies && policies.length > 0) {
                // Sort policies by source and destination SGT
                policies.sort((a, b) => {
                    const sourceA = parseInt(a.sourceSgt?.tag || '0', 10);
                    const sourceB = parseInt(b.sourceSgt?.tag || '0', 10);
                    if (sourceA !== sourceB) return sourceA - sourceB;
                    
                    const destA = parseInt(a.destinationSgt?.tag || '0', 10);
                    const destB = parseInt(b.destinationSgt?.tag || '0', 10);
                    return destA - destB;
                });
                
                policies.forEach(policy => {
                    const row = document.createElement('tr');
                    
                    // Format source SGT
                    const sourceSGT = policy.sourceSgt ? 
                        `${policy.sourceSgt.name || 'Unknown'} (${policy.sourceSgt.tag || 'No Tag'})` : 'N/A';
                    
                    // Format destination SGT
                    const destSGT = policy.destinationSgt ? 
                        `${policy.destinationSgt.name || 'Unknown'} (${policy.destinationSgt.tag || 'No Tag'})` : 'N/A';
                    
                    // Format ACL name
                    const aclName = policy.acl ? policy.acl.name : 'N/A';
                    
                    // Add status class based on matrixCellStatus
                    let statusClass = '';
                    if (policy.matrixCellStatus === 'ENABLED') {
                        statusClass = 'text-success';
                    } else if (policy.matrixCellStatus === 'DISABLED') {
                        statusClass = 'text-danger';
                    } else if (policy.matrixCellStatus === 'MONITOR') {
                        statusClass = 'text-warning';
                    }
                    
                    row.innerHTML = `
                        <td>${sourceSGT}</td>
                        <td>${destSGT}</td>
                        <td>${aclName}</td>
                        <td>${policy.defaultRule || 'N/A'}</td>
                        <td class="${statusClass}">${policy.matrixCellStatus || 'N/A'}</td>
                    `;
                    tableBody.appendChild(row);
                });
            } else {
                const row = document.createElement('tr');
                row.innerHTML = '<td colspan="5" class="text-center">No egress policies found</td>';
                tableBody.appendChild(row);
            }
        })
        .catch(error => {
            console.error('Error fetching egress policies:', error);
            document.getElementById('egress-policies-loading').classList.add('d-none');
            const errorElement = document.getElementById('egress-policies-error');
            errorElement.classList.remove('d-none');
            errorElement.textContent = `Error loading egress policies: ${error.message}`;
        });
}

// Reset security groups UI when tenant disconnects
function resetSecurityGroupsUI() {
    securityGroupsData = null;
    securityGroupsCount.textContent = '--';
    deletedGroupsCount.textContent = '--';
    tagValuesCount.textContent = '--';
    securityGroupsLastUpdated.textContent = 'Not available - connect a tenant first';
}

// Show security groups list in a modal
function showSecurityGroupsList() {
    // Show loading indicator
    const modalContent = `
        <div class="modal fade" id="sgt-list-modal" tabindex="-1" aria-labelledby="sgt-list-modal-label" aria-hidden="true">
            <div class="modal-dialog modal-lg">
                <div class="modal-content">
                    <div class="modal-header">
                        <h5 class="modal-title" id="sgt-list-modal-label">Security Group Tags (SGT) List</h5>
                        <button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
                    </div>
                    <div class="modal-body">
                        <div id="sgt-loading" class="text-center">
                            <div class="spinner-border" role="status">
                                <span class="visually-hidden">Loading...</span>
                            </div>
                            <p>Loading security groups...</p>
                        </div>
                        <div id="sgt-error" class="alert alert-danger d-none"></div>
                        <div id="sgt-table-container" class="d-none">
                            <table class="table table-striped table-hover">
                                <thead>
                                    <tr>
                                        <th>Name</th>
                                        <th>Tag</th>
                                        <th>ID</th>
                                        <th>Description</th>
                                    </tr>
                                </thead>
                                <tbody id="sgt-table-body">
                                </tbody>
                            </table>
                        </div>
                    </div>
                    <div class="modal-footer">
                        <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">Close</button>
                    </div>
                </div>
            </div>
        </div>
    `;
    
    // Add modal to body if it doesn't exist
    if (!document.getElementById('sgt-list-modal')) {
        const modalDiv = document.createElement('div');
        modalDiv.innerHTML = modalContent;
        document.body.appendChild(modalDiv);
    }
    
    // Show the modal
    const modal = new bootstrap.Modal(document.getElementById('sgt-list-modal'));
    modal.show();
    
    // Fetch security groups list
    fetch('/api/security-groups/list')
        .then(async response => {
            const responseText = await response.text();
            console.log('Security Groups List Response:', response.status, responseText);
            
            if (!response.ok) {
                throw new Error('Failed to fetch security groups list');
            }
            
            try {
                return JSON.parse(responseText);
            } catch (e) {
                throw new Error('Invalid JSON response from server');
            }
        })
        .then(securityGroups => {
            // Hide loading, show table
            document.getElementById('sgt-loading').classList.add('d-none');
            document.getElementById('sgt-table-container').classList.remove('d-none');
            
            // Populate table
            const tableBody = document.getElementById('sgt-table-body');
            tableBody.innerHTML = '';
            
            if (securityGroups && securityGroups.length > 0) {
                // Sort by tag value (numeric)
                securityGroups.sort((a, b) => {
                    const tagA = parseInt(a.tag || '0', 10);
                    const tagB = parseInt(b.tag || '0', 10);
                    return tagA - tagB;
                });
                
                securityGroups.forEach(sg => {
                    const row = document.createElement('tr');
                    row.innerHTML = `
                        <td>${sg.name || '-'}</td>
                        <td>${sg.tag || '-'}</td>
                        <td>${sg.id || '-'}</td>
                        <td>${sg.description || '-'}</td>
                    `;
                    tableBody.appendChild(row);
                });
            } else {
                const row = document.createElement('tr');
                row.innerHTML = '<td colspan="4" class="text-center">No security groups found</td>';
                tableBody.appendChild(row);
            }
        })
        .catch(error => {
            console.error('Error fetching security groups list:', error);
            document.getElementById('sgt-loading').classList.add('d-none');
            const errorElement = document.getElementById('sgt-error');
            errorElement.classList.remove('d-none');
            errorElement.textContent = `Error loading security groups: ${error.message}`;
        });
}
