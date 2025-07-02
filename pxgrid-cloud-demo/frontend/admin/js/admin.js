/**
 * Simple Admin Dashboard JavaScript
 * Minimal functionality to ensure it works
 */

document.addEventListener('DOMContentLoaded', function() {
    console.log('Admin dashboard loaded');
    
    // Basic authentication check
    const token = localStorage.getItem('auth_token');
    const userData = localStorage.getItem('user_data');
    
    if (!token) {
        console.log('No authentication token found - redirecting to login');
        window.location.href = '../login.html';
        return;
    }
    
    try {
        // Update UI with user info if available
        if (userData) {
            const user = JSON.parse(userData);
            console.log('User data found:', user);
            
            // Display username if element exists
            const usernameDisplay = document.getElementById('username-display');
            if (usernameDisplay) {
                usernameDisplay.textContent = user.username || 'Admin User';
                console.log('Updated username display');
            } else {
                console.warn('Username display element not found');
            }
            
            // Setup logout button
            const logoutBtn = document.getElementById('logout-btn');
            if (logoutBtn) {
                logoutBtn.addEventListener('click', function() {
                    localStorage.removeItem('auth_token');
                    localStorage.removeItem('user_data');
                    window.location.href = '../login.html';
                });
                console.log('Logout button initialized');
            }
            
            // Add User button functionality
            const addUserBtn = document.getElementById('add-user-btn');
            if (addUserBtn) {
                addUserBtn.addEventListener('click', function() {
                    const userModal = new bootstrap.Modal(document.getElementById('user-modal'));
                    if (userModal) {
                        userModal.show();
                    } else {
                        console.warn('User modal not found');
                        alert('Add user functionality is currently unavailable');
                    }
                });
                console.log('Add user button initialized');
            }
        }
    } catch (error) {
        console.error('Error in admin dashboard initialization:', error);
    }
    
    // Display demo users in the table for testing
    const usersTableBody = document.getElementById('users-table-body');
    if (usersTableBody) {
        const demoUsers = [
            { username: 'admin', email: 'admin@example.com', role: 'admin', tenantAccess: ['Default'], lastLogin: new Date().toLocaleString() },
            { username: 'user1', email: 'user1@example.com', role: 'user', tenantAccess: [], lastLogin: '2025-06-28 14:32:15' }
        ];
        
        usersTableBody.innerHTML = demoUsers.map(user => {
            const roleBadge = user.role === 'admin' 
                ? '<span class="badge bg-primary">Admin</span>' 
                : '<span class="badge bg-secondary">User</span>';
            
            return `
                <tr>
                    <td>${user.username}</td>
                    <td>${user.email}</td>
                    <td>${roleBadge}</td>
                    <td>${user.tenantAccess.join(', ') || 'None'}</td>
                    <td>${user.lastLogin}</td>
                    <td>
                        <button class="btn btn-sm btn-outline-primary">Edit</button>
                        <button class="btn btn-sm btn-outline-danger">Delete</button>
                    </td>
                </tr>
            `;
        }).join('');
        
        console.log('Demo users displayed in table');
    } else {
        console.warn('Users table body element not found');
    }
});
