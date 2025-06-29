// Tenant Manager JS
document.addEventListener('DOMContentLoaded', function() {
    // Initialize tenant tabs
    initializeTenantTabs();
    
    // Load initial tenants
    loadTenants();
    
    // Set up tenant-related event listeners
    setupTenantEventListeners();
});

// Global variables
let tenantData = [];
let currentTenantTab = 'all-tenants';

// Initialize tenant tabs
function initializeTenantTabs() {
    console.log('Initializing tenant tabs');
    const tenantTabsContainer = document.querySelector('.tenant-tabs');
    if (!tenantTabsContainer) {
        console.error('Tenant tabs container not found!');
        return;
    }
    
    // Always show tenant tabs on dashboard page
    tenantTabsContainer.style.display = 'block';
    console.log('Tenant tabs initialized and displayed');
    
    // Listen for main tab changes
    document.querySelectorAll('[data-bs-toggle="tab"]').forEach(tab => {
        tab.addEventListener('shown.bs.tab', function(e) {
            if (e.target.getAttribute('href') === '#dashboard-tab') {
                tenantTabsContainer.style.display = 'block';
            } else {
                tenantTabsContainer.style.display = 'none';
            }
        });
    });
}

// Load tenants from API
function loadTenants() {
    fetch('/api/tenants')
        .then(response => response.json())
        .then(data => {
            tenantData = data;
            updateTenantTabs(tenantData);
        })
        .catch(error => {
            console.error('Error loading tenants:', error);
        });
}

// Update tenant selector dropdown
function updateTenantTabs(tenants) {
    console.log('Updating tenant selector with:', tenants);
    const tenantDropdownMenu = document.getElementById('tenant-dropdown-menu');
    const currentTenantName = document.getElementById('current-tenant-name');
    const activeTenantName = document.getElementById('active-tenant-name');
    
    if (!tenantDropdownMenu || !currentTenantName || !activeTenantName) {
        console.error('Tenant dropdown elements not found');
        return;
    }
    
    // If we have at least one tenant, update the current tenant selection
    if (tenants && tenants.length > 0) {
        // Get the default tenant or the first tenant
        const defaultTenant = tenants.find(t => t.isDefault) || tenants[0];
        
        // Update dropdown labels
        currentTenantName.textContent = defaultTenant.name;
        activeTenantName.textContent = defaultTenant.name;
        
        // Set the data attribute for the current tenant selector
        const currentTenantSelector = document.getElementById('current-tenant-selector');
        if (currentTenantSelector) {
            currentTenantSelector.setAttribute('data-tenant-id', defaultTenant.id);
        }
    } else {
        // No tenants available
        activeTenantName.textContent = "Select Tenant";
        currentTenantName.textContent = "No Tenant";
    }
    
    // Setup dropdown item click handlers
    const allTenantsSelector = document.getElementById('all-tenants-selector');
    if (allTenantsSelector) {
        allTenantsSelector.addEventListener('click', function(e) {
            e.preventDefault();
            console.log('All tenants selected');
            selectTenant('all-tenants');
        });
    }
    
    const currentTenantSelector = document.getElementById('current-tenant-selector');
    if (currentTenantSelector && tenants && tenants.length > 0) {
        const defaultTenant = tenants.find(t => t.isDefault) || tenants[0];
        currentTenantSelector.setAttribute('data-tenant-id', defaultTenant.id);
        currentTenantSelector.addEventListener('click', function(e) {
            e.preventDefault();
            console.log('Current tenant selected');
            const tenantId = this.getAttribute('data-tenant-id');
            selectTenant(tenantId);
        });
    }
}

// Select a tenant
function selectTenant(tenantId) {
    console.log('Selecting tenant:', tenantId);
    
    // Update active dropdown item
    document.querySelectorAll('#tenant-dropdown-menu .dropdown-item').forEach(item => {
        item.classList.remove('active');
        if ((tenantId === 'all-tenants' && item.id === 'all-tenants-selector') || 
            item.getAttribute('data-tenant-id') === tenantId) {
            item.classList.add('active');
        }
    });
    
    // Update the active tenant name in the dropdown button
    const activeTenantName = document.getElementById('active-tenant-name');
    if (activeTenantName) {
        if (tenantId === 'all-tenants') {
            activeTenantName.textContent = 'All Tenants';
        } else {
            // Find the tenant name from the tenantData array
            const tenant = tenantData.find(t => t.id === tenantId);
            if (tenant) {
                activeTenantName.textContent = tenant.name;
            }
        }
    }
    
    // Filter devices by tenant
    filterDevicesByTenant(tenantId);
    
    // Filter messages by tenant
    filterMessagesByTenant(tenantId);
}

// Filter devices by tenant
function filterDevicesByTenant(tenantId) {
    const deviceRows = document.querySelectorAll('#device-table-body tr');
    let visibleCount = 0;
    let activeCount = 0;
    
    deviceRows.forEach(row => {
        const rowTenantId = row.getAttribute('data-tenant-id');
        
        if (tenantId === 'all-tenants' || rowTenantId === tenantId) {
            row.style.display = '';
            visibleCount++;
            if (row.querySelector('.status-active')) {
                activeCount++;
            }
        } else {
            row.style.display = 'none';
        }
    });
    
    // Update counts
    const deviceCount = document.getElementById('device-count');
    const activeDeviceCount = document.getElementById('active-device-count');
    if (deviceCount) deviceCount.textContent = visibleCount;
    if (activeDeviceCount) activeDeviceCount.textContent = activeCount;
    
    // Show/hide empty message
    const emptyDevices = document.getElementById('devices-empty');
    if (emptyDevices) {
        emptyDevices.style.display = visibleCount === 0 ? 'block' : 'none';
    }
}

// Filter messages by tenant
function filterMessagesByTenant(tenantId) {
    const messages = document.querySelectorAll('.message');
    let visibleCount = 0;
    
    messages.forEach(msg => {
        const msgTenantId = msg.getAttribute('data-tenant-id');
        
        if (tenantId === 'all-tenants' || msgTenantId === tenantId) {
            msg.style.display = '';
            visibleCount++;
        } else {
            msg.style.display = 'none';
        }
    });
    
    // Show/hide empty message
    const emptyMessages = document.getElementById('messages-empty');
    if (emptyMessages) {
        emptyMessages.style.display = visibleCount === 0 ? 'block' : 'none';
    }
}

// Set up tenant event listeners
function setupTenantEventListeners() {
    // Link tenant button
    const linkTenantBtn = document.getElementById('link-tenant-btn');
    if (linkTenantBtn) {
        linkTenantBtn.addEventListener('click', linkTenant);
    }
    
    // Confirm link tenant button
    const confirmLinkBtn = document.getElementById('submitLinkTenant');
    if (confirmLinkBtn) {
        confirmLinkBtn.addEventListener('click', confirmLinkTenant);
    }
    
    // Confirm unlink tenant button
    const confirmUnlinkBtn = document.getElementById('confirmUnlinkTenant');
    if (confirmUnlinkBtn) {
        confirmUnlinkBtn.addEventListener('click', confirmUnlinkTenant);
    }
}

// Link tenant with OTP
function confirmLinkTenant() {
    const otp = document.getElementById('tenantOtp').value.trim();
    const setAsDefault = document.getElementById('setAsDefault').checked;
    const statusDiv = document.getElementById('linkStatus');
    const submitBtn = document.getElementById('submitLinkTenant');
    
    if (!otp) {
        statusDiv.className = 'alert alert-danger';
        statusDiv.textContent = 'Please enter a tenant OTP';
        statusDiv.style.display = 'block';
        return;
    }
    
    submitBtn.disabled = true;
    statusDiv.className = 'alert alert-info';
    statusDiv.textContent = 'Linking tenant...';
    statusDiv.style.display = 'block';
    
    fetch('/api/tenant/link', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({
            otp,
            setAsDefault
        })
    })
    .then(response => {
        if (!response.ok) {
            return response.json().then(data => { throw new Error(data.error || response.statusText); });
        }
        return response.json();
    })
    .then(data => {
        statusDiv.className = 'alert alert-success';
        statusDiv.textContent = `Tenant "${data.name}" linked successfully!`;
        
        // Close modal after short delay
        setTimeout(() => {
            const modal = bootstrap.Modal.getInstance(document.getElementById('linkTenantModal'));
            if (modal) modal.hide();
            
            // Reload tenants
            loadTenants();
        }, 1500);
    })
    .catch(error => {
        console.error('Error linking tenant:', error);
        statusDiv.className = 'alert alert-danger';
        statusDiv.textContent = 'Failed to link tenant: ' + error.message;
        submitBtn.disabled = false;
    });
}

// Confirm unlink tenant action
function confirmUnlinkTenant() {
    const tenantId = document.getElementById('unlinkTenantId').value;
    const statusDiv = document.getElementById('unlinkStatus');
    const confirmBtn = document.getElementById('confirmUnlinkTenant');
    
    confirmBtn.disabled = true;
    statusDiv.className = 'alert alert-info';
    statusDiv.textContent = 'Unlinking tenant...';
    statusDiv.style.display = 'block';
    
    fetch(`/api/tenants/${tenantId}/unlink`, {
        method: 'POST'
    })
    .then(response => {
        if (!response.ok) {
            return response.json().then(data => { throw new Error(data.error || response.statusText); });
        }
        return response.json();
    })
    .then(data => {
        statusDiv.className = 'alert alert-success';
        statusDiv.textContent = 'Tenant unlinked successfully!';
        
        // Close modal after short delay
        setTimeout(() => {
            const modal = bootstrap.Modal.getInstance(document.getElementById('unlinkTenantModal'));
            if (modal) modal.hide();
            
            // Reload tenants
            loadTenants();
            
            // Reset to all-tenants view
            selectTenant('all-tenants');
        }, 1500);
    })
    .catch(error => {
        console.error('Error unlinking tenant:', error);
        statusDiv.className = 'alert alert-danger';
        statusDiv.textContent = 'Failed to unlink tenant: ' + error.message;
        confirmBtn.disabled = false;
    });
}

// Show unlink tenant confirmation modal
function showUnlinkModal(tenantId, tenantName) {
    document.getElementById('unlinkTenantName').textContent = tenantName;
    document.getElementById('unlinkTenantId').value = tenantId;
    document.getElementById('unlinkStatus').style.display = 'none';
    
    const modal = new bootstrap.Modal(document.getElementById('unlinkTenantModal'));
    modal.show();
}

// Process WebSocket tenant data updates
function processTenantUpdate(tenantData) {
    updateTenantTabs(tenantData);
}

// WebSocket integration
if (typeof ws !== 'undefined') {
    const originalHandler = ws.onmessage;
    
    ws.onmessage = function(event) {
        if (originalHandler) {
            originalHandler(event);
        }
        
        const data = JSON.parse(event.data);
        if (data.type === 'tenants') {
            processTenantUpdate(data.tenants);
        }
    };
}
