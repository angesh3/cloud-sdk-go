/**
 * Admin Dashboard JavaScript
 * Handles user management and settings for pxGrid Cloud Demo
 */

// Global variables
let allUsers = [];
let allTenants = [];
let currentUser = null;
let editingUserId = null;

// Helper function to format the authorization header
function getAuthHeader() {
    const token = localStorage.getItem('auth_token');
    return token.startsWith('admin-token-') ? token : `Bearer ${token}`;
}

// DOM Ready
document.addEventListener('DOMContentLoaded', function() {
    // Check authentication
    checkAuthentication();
    
    // Initialize UI components
    initUI();
    
    // Set up logout button
    safeAddEventListener('logout-btn', 'click', logout);
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
                updateUserUI(user);
                
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
    // Regular token verification with server
    fetch('/api/auth/me', {
        headers: {
            'Authorization': getAuthHeader()
        },
        credentials: 'include'
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
        updateUserUI(user);
        
        // Load initial data
        loadUsers();
        loadTenants();
    })
    .catch(error => {
        console.error('Authentication error:', error);
        // Clear invalid token and redirect to login
        localStorage.removeItem('auth_token');
        localStorage.removeItem('user_data');
        window.location.href = '../login.html';
    });
}

// Logout function
function logout() {
    const token = localStorage.getItem('auth_token');
    
    if (token) {
        if (token.startsWith('admin-token-')) {
            // For direct admin tokens, just clear local storage
            localStorage.removeItem('auth_token');
            localStorage.removeItem('user_data');
            window.location.href = '../login.html';
            return;
        }
        
        // Send logout request to server for regular tokens
        fetch('/api/auth/logout', {
            method: 'POST',
            headers: {
                'Authorization': getAuthHeader()
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
 * UI Initialization
 */

// Initialize all UI components
function initUI() {
    initSidebar();
    initUserManagement();
    initSettingsManagement();
}

// Initialize sidebar navigation
function initSidebar() {
    // Find all sidebar navigation links
    const navLinks = document.querySelectorAll('.sidebar .nav-link');
    if (navLinks.length === 0) {
        console.warn('No sidebar navigation links found');
        return;
    }
    
    // Add click event to each link
    navLinks.forEach(link => {
        link.addEventListener('click', function(e) {
            e.preventDefault();
            const sectionId = this.getAttribute('data-section');
            if (!sectionId) return;
            
            // Hide all sections
            document.querySelectorAll('main .section').forEach(section => {
                section.style.display = 'none';
            });
            
            // Show the selected section
            const targetSection = document.getElementById(sectionId);
            if (targetSection) {
                targetSection.style.display = 'block';
                
                // Update active state
                navLinks.forEach(l => l.classList.remove('active'));
                this.classList.add('active');
            }
        });
    });
}

/**
 * User Management Functions
 */

// Initialize user management UI
function initUserManagement() {
    // Add new user button
    safeAddEventListener('add-user-btn', 'click', () => {
        showUserModal();
    });
    
    // Save user button
    safeAddEventListener('save-user-btn', 'click', saveUser);
    
    // Confirm delete button
    safeAddEventListener('confirm-delete-btn', 'click', confirmDeleteUser);
    
    // Refresh users button
    safeAddEventListener('refresh-users-btn', 'click', loadUsers);
    
    // Refresh tenants button
    safeAddEventListener('refresh-tenants-btn', 'click', loadTenants);
}

// Load all users
function loadUsers() {
    const token = localStorage.getItem('auth_token');
    
    fetch('/api/admin/users', {
        headers: {
            'Authorization': getAuthHeader()
        }
    })
    .then(response => {
        if (!response.ok) {
            throw new Error('Failed to load users');
        }
        return response.json();
    })
    .then(data => {
        allUsers = Array.isArray(data) ? data : (data.users || []);
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
            'Authorization': getAuthHeader()
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
        if (document.getElementById('user-modal') && 
            document.getElementById('user-modal').classList.contains('show')) {
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
    if (!tableBody) {
        console.error('Users table body element not found');
        return;
    }
    
    if (!allUsers || !allUsers.length) {
        tableBody.innerHTML = `
            <tr>
                <td colspan="5" class="text-center">
                    <div class="alert alert-info mb-0">
                        No users found. Click "Add User" to create one.
                    </div>
                </td>
            </tr>
        `;
        return;
    }
    
    // Clear any errors
    safeHideElement('users-error');
    
    // Sort users by username
    const sortedUsers = [...allUsers].sort((a, b) => a.username.localeCompare(b.username));
    
    // Generate table rows
    tableBody.innerHTML = sortedUsers.map(user => {
        const roleBadge = user.role === 'admin' 
            ? '<span class="badge bg-admin">Admin</span>' 
            : '<span class="badge bg-user">User</span>';
        
        const tenantAccess = user.tenantAccess && user.tenantAccess.length 
            ? user.tenantAccess.length + ' tenants' 
            : 'None';
        
        const lastLogin = user.lastLogin 
            ? new Date(user.lastLogin).toLocaleString() 
            : 'Never';
        
        return `
            <tr>
                <td>${user.username}</td>
                <td>${user.email}</td>
                <td>${roleBadge}</td>
                <td>${tenantAccess}</td>
                <td>${lastLogin}</td>
                <td class="table-actions">
                    <button class="btn btn-sm btn-outline-primary edit-user-btn" data-user-id="${user.id}">
                        <i class="fas fa-edit"></i>
                    </button>
                    <button class="btn btn-sm btn-outline-danger delete-user-btn" data-user-id="${user.id}">
                        <i class="fas fa-trash-alt"></i>
                    </button>
                </td>
            </tr>
        `;
    }).join('');
    
    // Add event listeners for edit and delete buttons
    document.querySelectorAll('.edit-user-btn').forEach(button => {
        button.addEventListener('click', function() {
            const userId = this.getAttribute('data-user-id');
            editUser(userId);
        });
    });
    
    document.querySelectorAll('.delete-user-btn').forEach(button => {
        button.addEventListener('click', function() {
            const userId = this.getAttribute('data-user-id');
            showDeleteConfirmation(userId);
        });
    });
}

// Show user management error
function showUsersError(message) {
    const errorElement = document.getElementById('users-error');
    if (errorElement) {
        errorElement.textContent = message || 'An error occurred';
        errorElement.style.display = 'block';
    }
}

/**
 * Settings Management Functions
 */

// Initialize settings management UI
function initSettingsManagement() {
    // Save settings button
    safeAddEventListener('save-settings-btn', 'click', saveSettings);
}

// Handle any UI errors gracefully
window.addEventListener('error', function(e) {
    console.error('JavaScript error:', e.error);
    // Log the error but don't crash the page
    return true;
});
