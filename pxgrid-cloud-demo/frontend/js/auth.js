/**
 * Authentication utilities for pxGrid Cloud Demo
 */

// Check if user is authenticated
function checkAuthentication(callback) {
    const token = localStorage.getItem('auth_token');
    if (!token) {
        // Not authenticated, redirect to login
        window.location.href = 'login.html';
        return;
    }
    
    // Special handling for our direct admin login override
    if (token.startsWith('admin-token-')) {
        console.log('Using direct admin login token');
        const userData = localStorage.getItem('user_data');
        if (userData) {
            const user = JSON.parse(userData);
            // Update UI with user info
            updateUserUI(user);
            
            // Run callback if provided
            if (typeof callback === 'function') {
                callback(user);
            }
            return;
        }
    }
    
    // Verify token with server
    fetch('/api/auth/me', {
        headers: {
            'Authorization': `Bearer ${token}`
        }
    })
    .then(response => {
        if (!response.ok) {
            throw new Error('Invalid authentication');
        }
        return response.json();
    })
    .then(user => {
        // Store user data
        localStorage.setItem('user_data', JSON.stringify(user));
        
        // Update UI with user info
        updateUserUI(user);
        
        // Run callback if provided
        if (typeof callback === 'function') {
            callback(user);
        }
    })
    .catch(error => {
        console.error('Authentication error:', error);
        // Clear invalid token and redirect to login
        localStorage.removeItem('auth_token');
        localStorage.removeItem('user_data');
        window.location.href = 'login.html';
    });
}

// Update UI based on user data
function updateUserUI(user) {
    // Update user dropdown
    document.getElementById('current-username').textContent = user.username;
    document.getElementById('dropdown-user-email').textContent = user.email;
    
    // Show user dropdown, hide login button
    document.getElementById('user-dropdown').style.display = 'block';
    document.getElementById('login-btn').style.display = 'none';
    
    // Show or hide admin link based on role
    const adminNavItem = document.getElementById('admin-nav-item');
    if (user.role === 'admin') {
        adminNavItem.style.display = 'block';
    } else {
        adminNavItem.style.display = 'none';
    }
}

// Handle user logout
function logout() {
    const token = localStorage.getItem('auth_token');
    
    if (token) {
        // Send logout request to server
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
            // Clear local storage and redirect regardless of server response
            localStorage.removeItem('auth_token');
            localStorage.removeItem('user_data');
            window.location.href = 'login.html';
        });
    } else {
        // No token, just redirect to login
        window.location.href = 'login.html';
    }
}

// Get authenticated API request headers
function getAuthHeaders() {
    const token = localStorage.getItem('auth_token');
    return {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json'
    };
}

// Get current user data from localStorage
function getCurrentUser() {
    const userData = localStorage.getItem('user_data');
    if (userData) {
        try {
            return JSON.parse(userData);
        } catch (e) {
            console.error('Error parsing user data:', e);
            return null;
        }
    }
    return null;
}
