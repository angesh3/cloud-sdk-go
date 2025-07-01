/**
 * Admin Dashboard JavaScript
 * Handles user management and settings for pxGrid Cloud Demo
 */

// Global variables
let allUsers = [];
let allTenants = [];
let currentUser = null;
let editingUserId = null;

// DOM Ready
document.addEventListener('DOMContentLoaded', function() {
    // Check authentication
    checkAuthentication();
    
    // Initialize UI components
    initSidebar();
    initUserManagement();
    initSettingsManagement();
    
    // Set up logout button
    document.getElementById('logout-btn').addEventListener('click', logout);
});

/**
 * Authentication Functions
 */

// Check if user is authenticated and has admin privileges
function checkAuthentication() {
    const token = localStorage.getItem('auth_token');
    if (!token) {
        console.log('No authentication token found - redirecting to login');
        window.location.href = '../login.html';
        return;
    }
    
    // First check if we already have user data in local storage
    const userData = localStorage.getItem('user_data');
    if (userData) {
        try {
            const user = JSON.parse(userData);
            if (user.role === 'admin') {
                console.log('Found admin user in local storage');
                // Already verified as admin
                currentUser = user;
                
                // Update UI with user info
                document.getElementById('current-user').textContent = user.username;
                document.getElementById('username-display').textContent = user.username;
                
                // Load initial data
                loadUsers();
                loadTenants();
                return;
            } else {
                console.log('User is not an admin - redirecting to main app');
                // Not an admin - redirect
                window.location.href = '../index.html';
                return;
            }
        } catch (e) {
            console.error('Error parsing user data:', e);
            // Continue with server verification
        }
    }
    
    console.log('Verifying admin status with server');
    // Verify token with server
    fetch('/api/auth/me', {
        headers: {
            'Authorization': `Bearer ${token}`
        }
    })
    .then(response => {
        if (!response.ok) {
            throw new Error('Not authenticated');
        }
        return response.json();
    })
    .then(user => {
        if (user.role !== 'admin') {
            console.log('Server confirmed user is not an admin');
            // Redirect non-admin users to main app
            window.location.href = '../index.html';
            return;
        }
        
        console.log('Server confirmed admin status');
        // Store user data
        localStorage.setItem('user_data', JSON.stringify(user));
        
        // Store current user
        currentUser = user;
        
        // Update UI with user info
        document.getElementById('current-user').textContent = user.username;
        document.getElementById('username-display').textContent = user.username;
        
        // Load initial data
        loadUsers();
        loadTenants();
    })
    .catch(error => {
        console.error('Authentication error:', error);
        localStorage.removeItem('auth_token');
        localStorage.removeItem('user_data');
        window.location.href = '../login.html';
    });
}

// Logout function
function logout() {
    const token = localStorage.getItem('auth_token');
    if (token) {
        fetch('/api/auth/logout', {
            method: 'POST',
            headers: {
                'Authorization': `Bearer ${token}`
            }
        })
        .catch(error => {
            console.error('Logout error:', error);
        })
        .finally(() => {
            localStorage.removeItem('auth_token');
            localStorage.removeItem('user_data');
            window.location.href = '../login.html';
        });
    } else {
        window.location.href = '../login.html';
    }
}

/**
 * User Management Functions
 */

// Initialize user management UI
function initUserManagement() {
    // Add new user button
    document.getElementById('add-user-btn').addEventListener('click', () => {
        showUserModal();
    });
    
    // Save user button
    document.getElementById('save-user-btn').addEventListener('click', saveUser);
    
    // Confirm delete button
    document.getElementById('confirm-delete-btn').addEventListener('click', confirmDeleteUser);
    
    // Refresh tenants button
    document.getElementById('refresh-tenants-btn').addEventListener('click', loadTenants);
}

// Load all users
function loadUsers() {
    const token = localStorage.getItem('auth_token');
    
    fetch('/api/admin/users', {
        headers: {
            'Authorization': `Bearer ${token}`
        }
    })
    .then(response => {
        if (!response.ok) {
            throw new Error('Failed to load users');
        }
        return response.json();
    })
    .then(data => {
        allUsers = data.users || [];
        renderUsersTable();
    })
    .catch(error => {
        console.error('Error loading users:', error);
        showUsersError(error.message);
    });
}

// Load all tenants
function loadTenants() {
    const token = localStorage.getItem('auth_token');
    
    fetch('/api/tenants', {
        headers: {
            'Authorization': `Bearer ${token}`
        }
    })
    .then(response => {
        if (!response.ok) {
            throw new Error('Failed to load tenants');
        }
        return response.json();
    })
    .then(data => {
        allTenants = Array.isArray(data) ? data : (data.tenants || []);
        
        // If modal is open, update the tenant checkboxes
        if (document.getElementById('user-modal').classList.contains('show')) {
            renderTenantsList();
        }
    })
    .catch(error => {
        console.error('Error loading tenants:', error);
    });
}

// Render users table
function renderUsersTable() {
    const tableBody = document.getElementById('users-table-body');
    
    if (!allUsers.length) {
        tableBody.innerHTML = `
            <tr>
                <td colspan="6" class="text-center">
                    <p class="my-3">No users found</p>
                </td>
            </tr>
        `;
        return;
    }
    
    tableBody.innerHTML = '';
    
    allUsers.forEach(user => {
        // Format last login date
        const lastLogin = user.lastLogin 
            ? new Date(user.lastLogin).toLocaleString() 
            : 'Never';
            
        // Create tenant count badge
        const tenantCount = user.tenantAccess ? user.tenantAccess.length : 0;
        
        const row = document.createElement('tr');
        row.innerHTML = `
            <td>${user.username}</td>
            <td>${user.email}</td>
            <td>
                ${user.role === 'admin' 
                    ? '<span class="badge bg-danger user-role-badge">Admin</span>' 
                    : '<span class="badge bg-info user-role-badge">User</span>'}
                ${user.role}
            </td>
            <td>
                <span class="badge bg-secondary">${tenantCount}</span>
                <button class="btn btn-sm btn-outline-secondary view-tenants-btn" data-user-id="${user.id}">
                    View
                </button>
            </td>
            <td>${lastLogin}</td>
            <td class="table-actions">
                <button class="btn btn-sm btn-primary edit-user-btn" data-user-id="${user.id}">
                    <i class="fas fa-edit"></i> Edit
                </button>
                <button class="btn btn-sm btn-danger delete-user-btn" data-user-id="${user.id}" ${user.id === currentUser.id ? 'disabled' : ''}>
                    <i class="fas fa-trash-alt"></i> Delete
                </button>
            </td>
        `;
        
        tableBody.appendChild(row);
    });
    
    // Add event listeners to buttons
    document.querySelectorAll('.edit-user-btn').forEach(button => {
        button.addEventListener('click', () => {
            const userId = button.getAttribute('data-user-id');
            const user = allUsers.find(u => u.id === userId);
            if (user) {
                showUserModal(user);
            }
        });
    });
    
    document.querySelectorAll('.delete-user-btn').forEach(button => {
        button.addEventListener('click', () => {
            const userId = button.getAttribute('data-user-id');
            const user = allUsers.find(u => u.id === userId);
            if (user) {
                showDeleteUserModal(user);
            }
        });
    });
    
    document.querySelectorAll('.view-tenants-btn').forEach(button => {
        button.addEventListener('click', () => {
            const userId = button.getAttribute('data-user-id');
            const user = allUsers.find(u => u.id === userId);
            if (user) {
                showUserTenants(user);
            }
        });
    });
}

// Render tenants list in user modal
function renderTenantsList() {
    const tenantsList = document.getElementById('user-tenants-list');
    
    if (!allTenants.length) {
        tenantsList.innerHTML = `
            <div class="alert alert-info mb-0">
                No tenants available. Link tenants in the main app first.
            </div>
        `;
        return;
    }
    
    tenantsList.innerHTML = '';
    
    // Get selected tenants for the user being edited
    const selectedTenants = [];
    if (editingUserId) {
        const user = allUsers.find(u => u.id === editingUserId);
        if (user && user.tenantAccess) {
            selectedTenants.push(...user.tenantAccess);
        }
    }
    
    allTenants.forEach(tenant => {
        const tenantId = tenant.id;
        const isChecked = selectedTenants.includes(tenantId);
        
        const item = document.createElement('div');
        item.className = 'form-check';
        item.innerHTML = `
            <input class="form-check-input tenant-checkbox" type="checkbox" 
                id="tenant-${tenantId}" value="${tenantId}" ${isChecked ? 'checked' : ''}>
            <label class="form-check-label" for="tenant-${tenantId}">
                ${tenant.name} (ID: ${tenantId})
            </label>
        `;
        
        tenantsList.appendChild(item);
    });
}

// Show user modal for add/edit
function showUserModal(user = null) {
    const modal = new bootstrap.Modal(document.getElementById('user-modal'));
    const modalTitle = document.getElementById('user-modal-label');
    const form = document.getElementById('user-form');
    const passwordHelp = document.getElementById('password-help');
    
    // Reset form
    form.reset();
    
    // Set mode (add or edit)
    if (user) {
        modalTitle.textContent = 'Edit User';
        passwordHelp.textContent = 'Leave blank to keep existing password.';
        
        // Set form values
        document.getElementById('user-id').value = user.id;
        document.getElementById('user-username').value = user.username;
        document.getElementById('user-email').value = user.email;
        document.getElementById('user-role').value = user.role;
        
        editingUserId = user.id;
    } else {
        modalTitle.textContent = 'Add New User';
        passwordHelp.textContent = 'Password must be at least 8 characters.';
        document.getElementById('user-password').setAttribute('required', 'required');
        
        editingUserId = null;
    }
    
    // Render tenants list
    renderTenantsList();
    
    // Show modal
    modal.show();
}

// Save user (add or edit)
function saveUser() {
    const userId = document.getElementById('user-id').value;
    const username = document.getElementById('user-username').value;
    const email = document.getElementById('user-email').value;
    const password = document.getElementById('user-password').value;
    const role = document.getElementById('user-role').value;
    
    // Get selected tenants
    const tenantAccess = [];
    document.querySelectorAll('.tenant-checkbox:checked').forEach(checkbox => {
        tenantAccess.push(checkbox.value);
    });
    
    // Validate form
    if (!username || !email) {
        alert('Username and email are required');
        return;
    }
    
    if (!userId && !password) {
        alert('Password is required for new users');
        return;
    }
    
    const token = localStorage.getItem('auth_token');
    const isNewUser = !userId;
    
    // Prepare user data
    const userData = {
        username,
        email,
        role
    };
    
    if (password) {
        userData.password = password;
    }
    
    if (tenantAccess.length) {
        userData.tenantAccess = tenantAccess;
    }
    
    // Determine URL and method based on whether we're adding or editing
    const url = isNewUser 
        ? '/api/admin/users' 
        : `/api/admin/users/${userId}`;
    
    const method = isNewUser ? 'POST' : 'PUT';
    
    // Send request
    fetch(url, {
        method,
        headers: {
            'Authorization': `Bearer ${token}`,
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(userData)
    })
    .then(response => {
        if (!response.ok) {
            return response.json().then(err => {
                throw new Error(err.error || 'Failed to save user');
            });
        }
        return response.json();
    })
    .then(data => {
        // Close modal
        const modal = bootstrap.Modal.getInstance(document.getElementById('user-modal'));
        modal.hide();
        
        // Update users list
        loadUsers();
    })
    .catch(error => {
        console.error('Error saving user:', error);
        alert(error.message);
    });
}

// Show delete user confirmation modal
function showDeleteUserModal(user) {
    document.getElementById('delete-username').textContent = user.username;
    editingUserId = user.id;
    
    const modal = new bootstrap.Modal(document.getElementById('delete-user-modal'));
    modal.show();
}

// Confirm delete user
function confirmDeleteUser() {
    if (!editingUserId) {
        return;
    }
    
    const token = localStorage.getItem('auth_token');
    
    fetch(`/api/admin/users/${editingUserId}`, {
        method: 'DELETE',
        headers: {
            'Authorization': `Bearer ${token}`
        }
    })
    .then(response => {
        if (!response.ok) {
            return response.json().then(err => {
                throw new Error(err.error || 'Failed to delete user');
            });
        }
        return response.json();
    })
    .then(data => {
        // Close modal
        const modal = bootstrap.Modal.getInstance(document.getElementById('delete-user-modal'));
        modal.hide();
        
        // Update users list
        loadUsers();
    })
    .catch(error => {
        console.error('Error deleting user:', error);
        alert(error.message);
    });
}

// Show user tenants in a modal
function showUserTenants(user) {
    // This would typically open a modal with a list of the user's tenants
    // For simplicity, we'll just show an alert
    
    if (!user.tenantAccess || user.tenantAccess.length === 0) {
        alert(`User ${user.username} has no tenant access`);
        return;
    }
    
    const tenantNames = user.tenantAccess.map(tenantId => {
        const tenant = allTenants.find(t => t.id === tenantId);
        return tenant ? tenant.name : tenantId;
    });
    
    alert(`User ${user.username} has access to tenants:\n\n${tenantNames.join('\n')}`);
}

// Show error message in users section
function showUsersError(message) {
    const errorAlert = document.getElementById('users-error-alert');
    const errorMessage = document.getElementById('users-error-message');
    
    errorMessage.textContent = message;
    errorAlert.classList.remove('d-none');
    
    // Hide after 5 seconds
    setTimeout(() => {
        errorAlert.classList.add('d-none');
    }, 5000);
}

/**
 * Settings Management Functions
 */

function initSettingsManagement() {
    // Load settings when settings section is shown
    document.querySelector('[data-section="settings-section"]').addEventListener('click', function() {
        loadSettings();
    });
    
    // Save settings button
    document.getElementById('save-settings-btn').addEventListener('click', saveSettings);
}

// Load application settings
function loadSettings() {
    // This would typically load the existing settings form from the original settings page
    const settingsContainer = document.getElementById('settings-container');
    
    // We'll load the settings HTML via AJAX
    fetch('../settings.html')
        .then(response => response.text())
        .then(html => {
            // Extract just the settings form portion
            const tempDiv = document.createElement('div');
            tempDiv.innerHTML = html;
            
            // Find the settings form content
            const settingsForm = tempDiv.querySelector('.settings-form') || 
                                tempDiv.querySelector('form') ||
                                tempDiv.querySelector('.card-body');
            
            if (settingsForm) {
                settingsContainer.innerHTML = settingsForm.innerHTML;
                
                // Load actual settings values
                loadSettingsValues();
            } else {
                settingsContainer.innerHTML = `
                    <div class="alert alert-warning">
                        <i class="fas fa-exclamation-triangle me-2"></i>
                        Could not load settings form. Please use the main settings page.
                    </div>
                    <div class="d-grid gap-2">
                        <a href="../settings.html" class="btn btn-primary">
                            <i class="fas fa-cogs me-2"></i>
                            Go to Settings Page
                        </a>
                    </div>
                `;
            }
        })
        .catch(error => {
            console.error('Error loading settings:', error);
            settingsContainer.innerHTML = `
                <div class="alert alert-danger">
                    <i class="fas fa-exclamation-circle me-2"></i>
                    Failed to load settings.
                </div>
            `;
        });
}

// Load settings values from API
function loadSettingsValues() {
    const token = localStorage.getItem('auth_token');
    
    fetch('/api/settings', {
        headers: {
            'Authorization': `Bearer ${token}`
        }
    })
    .then(response => {
        if (!response.ok) {
            throw new Error('Failed to load settings');
        }
        return response.json();
    })
    .then(settings => {
        // Populate form with settings values
        // This will depend on the structure of your settings
        console.log('Settings loaded:', settings);
    })
    .catch(error => {
        console.error('Error loading settings:', error);
    });
}

// Save settings
function saveSettings() {
    // This would collect form data and send to API
    alert('Settings saving is not implemented in this demo');
}

/**
 * Sidebar Navigation
 */

function initSidebar() {
    // Handle section navigation
    const sectionLinks = document.querySelectorAll('[data-section]');
    sectionLinks.forEach(link => {
        link.addEventListener('click', function(e) {
            e.preventDefault();
            
            // Get target section
            const targetId = this.getAttribute('data-section');
            const targetSection = document.getElementById(targetId);
            
            // Hide all sections
            document.querySelectorAll('main > section').forEach(section => {
                section.classList.add('d-none');
            });
            
            // Show target section
            targetSection.classList.remove('d-none');
            
            // Update active link
            sectionLinks.forEach(link => {
                link.classList.remove('active');
            });
            this.classList.add('active');
        });
    });
}
