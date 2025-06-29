// Tenant tabs management
let tenantTabsContainer = document.querySelector('.tenant-tabs');
let tenantTabsList = document.getElementById('tenantTabs');
let currentTab = 'all-tenants';

// Function to update tenant tabs
function updateTenantTabs(tenants) {
    // If there are fewer than 2 tenants, don't show tabs
    if (!tenants || tenants.length < 2) {
        tenantTabsContainer.style.display = 'none';
        return;
    }
    
    // Show the tenant tabs container
    tenantTabsContainer.style.display = 'block';
    
    // Keep the "All Tenants" tab
    const allTenantsTab = tenantTabsList.querySelector('#all-tenants-tab').closest('.nav-item');
    
    // Clear existing tenant tabs
    tenantTabsList.innerHTML = '';
    
    // Re-add All Tenants tab
    tenantTabsList.appendChild(allTenantsTab);
    
    // Add a tab for each tenant
    tenants.forEach(tenant => {
        const tabItem = document.createElement('li');
        tabItem.className = 'nav-item';
        
        const tabLink = document.createElement('a');
        tabLink.className = 'nav-link';
        tabLink.id = `tenant-${tenant.id}-tab`;
        tabLink.setAttribute('data-bs-toggle', 'tab');
        tabLink.setAttribute('href', `#tenant-${tenant.id}-content`);
        tabLink.setAttribute('role', 'tab');
        tabLink.setAttribute('data-tenant-id', tenant.id);
        
        // If this is the default tenant, show an indicator
        const badgeClass = tenant.isDefault ? 'bg-success' : 'bg-secondary';
        
        tabLink.innerHTML = `
            <i class="fas fa-building me-1"></i> ${tenant.name}
            <span class="badge ${badgeClass} ms-1" style="font-size: 0.7em;">${tenant.devices}</span>
        `;
        
        tabItem.appendChild(tabLink);
        tenantTabsList.appendChild(tabItem);
        
        // If this is the current tab, set it as active
        if (currentTab === tenant.id) {
            tabLink.classList.add('active');
        }
        
        // Add click event to tab
        tabLink.addEventListener('click', function() {
            // Update current tab
            currentTab = tenant.id;
            
            // Filter devices
            filterDevicesByTenant(tenant.id);
            
            // Filter messages
            filterMessagesByTenant(tenant.id);
        });
    });
    
    // Make sure the "All Tenants" tab has a click handler
    const allTenantsTabLink = allTenantsTab.querySelector('a');
    allTenantsTabLink.addEventListener('click', function() {
        currentTab = 'all-tenants';
        
        // Show all devices
        filterDevicesByTenant(null);
        
        // Show all messages
        filterMessagesByTenant(null);
    });
    
    // Set the active tab
    if (currentTab === 'all-tenants') {
        allTenantsTabLink.classList.add('active');
    } else {
        const activeTab = tenantTabsList.querySelector(`[data-tenant-id="${currentTab}"]`);
        if (activeTab) {
            activeTab.classList.add('active');
        } else {
            // If the current tab doesn't exist, default to "All Tenants"
            allTenantsTabLink.classList.add('active');
            currentTab = 'all-tenants';
        }
    }
}

// Filter devices by tenant
function filterDevicesByTenant(tenantId) {
    const deviceTableBody = document.getElementById('device-table-body');
    if (!deviceTableBody) return;
    
    const deviceRows = deviceTableBody.querySelectorAll('tr');
    deviceRows.forEach(row => {
        const rowTenantId = row.getAttribute('data-tenant-id');
        if (tenantId && rowTenantId !== tenantId) {
            row.style.display = 'none';
        } else {
            row.style.display = '';
        }
    });
    
    // Update counts and empty message
    updateDeviceCount();
}

// Filter messages by tenant
function filterMessagesByTenant(tenantId) {
    const messageContainer = document.getElementById('message-container');
    if (!messageContainer) return;
    
    const messages = messageContainer.querySelectorAll('.message');
    let visibleCount = 0;
    
    messages.forEach(msg => {
        const msgTenantId = msg.getAttribute('data-tenant-id');
        if (tenantId && msgTenantId !== tenantId) {
            msg.style.display = 'none';
        } else {
            msg.style.display = '';
            visibleCount++;
        }
    });
    
    // Check if empty
    const emptyMessage = document.getElementById('messages-empty');
    if (emptyMessage) {
        emptyMessage.style.display = visibleCount === 0 ? 'block' : 'none';
    }
}

// Update device count after filtering
function updateDeviceCount() {
    const deviceTableBody = document.getElementById('device-table-body');
    if (!deviceTableBody) return;
    
    const deviceRows = deviceTableBody.querySelectorAll('tr');
    let visibleCount = 0;
    let activeCount = 0;
    
    deviceRows.forEach(row => {
        if (row.style.display !== 'none') {
            visibleCount++;
            if (row.querySelector('.status-active')) {
                activeCount++;
            }
        }
    });
    
    // Update counts
    const deviceCountEl = document.getElementById('device-count');
    const activeDeviceCountEl = document.getElementById('active-device-count');
    
    if (deviceCountEl) deviceCountEl.textContent = visibleCount;
    if (activeDeviceCountEl) activeDeviceCountEl.textContent = activeCount;
    
    // Show/hide empty message
    const emptyDevices = document.getElementById('devices-empty');
    if (emptyDevices) {
        emptyDevices.style.display = visibleCount === 0 ? 'block' : 'none';
    }
}
