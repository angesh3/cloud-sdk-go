package main

import (
	"net/http"
	
	"github.com/gin-gonic/gin"
)

// CreateUserRequest represents a request to create a user
type CreateUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// UpdateUserRequest represents a request to update a user
type UpdateUserRequest struct {
	Username     string   `json:"username,omitempty"`
	Email        string   `json:"email,omitempty"`
	Password     string   `json:"password,omitempty"`
	Role         string   `json:"role,omitempty"`
	TenantAccess []string `json:"tenantAccess,omitempty"`
}

// UserTenantRequest represents a request to update a user's tenant access
type UserTenantRequest struct {
	TenantIDs []string `json:"tenantIds"`
}

// listUsersHandler lists all users (admin only)
func listUsersHandler(c *gin.Context) {
	users := userManager.ListUsers()
	c.JSON(http.StatusOK, gin.H{
		"users": users,
	})
}

// createUserHandler creates a new user (admin only)
func createUserHandler(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}
	
	// Validate input
	if req.Username == "" || req.Email == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Username, email, and password are required",
		})
		return
	}
	
	// Default to "user" role if not specified or invalid
	if req.Role != "admin" && req.Role != "user" {
		req.Role = "user"
	}
	
	// Create user
	user, err := userManager.CreateUser(req.Username, req.Email, req.Password, req.Role)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	
	// Create a copy without password hash
	userCopy := *user
	userCopy.PasswordHash = ""
	
	c.JSON(http.StatusCreated, gin.H{
		"message": "User created successfully",
		"user":    userCopy,
	})
}

// getUserHandler gets a user by ID (admin only)
func getUserHandler(c *gin.Context) {
	userID := c.Param("id")
	
	user, err := userManager.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User not found",
		})
		return
	}
	
	// Create a copy without password hash
	userCopy := *user
	userCopy.PasswordHash = ""
	
	c.JSON(http.StatusOK, userCopy)
}

// updateUserHandler updates a user (admin only)
func updateUserHandler(c *gin.Context) {
	userID := c.Param("id")
	
	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}
	
	// Check if user exists
	_, err := userManager.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User not found",
		})
		return
	}
	
	// Prepare updates
	updates := make(map[string]interface{})
	
	if req.Username != "" {
		updates["username"] = req.Username
	}
	
	if req.Email != "" {
		updates["email"] = req.Email
	}
	
	if req.Password != "" {
		updates["password"] = req.Password
	}
	
	if req.Role == "admin" || req.Role == "user" {
		updates["role"] = req.Role
	}
	
	if req.TenantAccess != nil {
		updates["tenantAccess"] = req.TenantAccess
	}
	
	// Update user
	user, err := userManager.UpdateUser(userID, updates)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	
	// Create a copy without password hash
	userCopy := *user
	userCopy.PasswordHash = ""
	
	c.JSON(http.StatusOK, gin.H{
		"message": "User updated successfully",
		"user":    userCopy,
	})
}

// deleteUserHandler deletes a user (admin only)
func deleteUserHandler(c *gin.Context) {
	userID := c.Param("id")
	
	// Check if user exists
	_, err := userManager.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User not found",
		})
		return
	}
	
	// Get the current admin user
	adminInterface, _ := c.Get("user")
	admin := adminInterface.(*User)
	
	// Don't allow an admin to delete themselves
	if userID == admin.ID {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Cannot delete your own admin account",
		})
		return
	}
	
	// Delete user
	err = userManager.DeleteUser(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete user",
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "User deleted successfully",
	})
}

// updateUserTenantsHandler updates a user's tenant access (admin only)
func updateUserTenantsHandler(c *gin.Context) {
	userID := c.Param("id")
	
	var req UserTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}
	
	// Check if user exists
	_, err := userManager.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User not found",
		})
		return
	}
	
	// Update user's tenant access
	updates := make(map[string]interface{})
	updates["tenantAccess"] = req.TenantIDs
	
	_, err = userManager.UpdateUser(userID, updates)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update user tenant access",
		})
		return
	}
	
	// Get updated user
	updatedUser, _ := userManager.GetUserByID(userID)
	
	// Create a copy without password hash
	userCopy := *updatedUser
	userCopy.PasswordHash = ""
	
	c.JSON(http.StatusOK, gin.H{
		"message":     "User tenant access updated successfully",
		"user":        userCopy,
		"tenantAccess": req.TenantIDs,
	})
}

// getUserTenantsHandler gets a user's tenant access
func getUserTenantsHandler(c *gin.Context) {
	userID := c.Param("id")
	
	// Check if user exists
	_, err := userManager.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User not found",
		})
		return
	}
	
	// Get user's tenant access
	tenants, err := userManager.GetUserTenants(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get user tenant access",
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"tenants": tenants,
	})
}

// addTenantToUserHandler adds a tenant to a user's access list
func addTenantToUserHandler(c *gin.Context) {
	userID := c.Param("id")
	tenantID := c.Param("tenantId")
	
	// Check if user exists
	_, err := userManager.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User not found",
		})
		return
	}
	
	// Add tenant to user
	err = userManager.AssignTenantToUser(userID, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to assign tenant to user",
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Tenant assigned to user successfully",
	})
}

// removeTenantFromUserHandler removes a tenant from a user's access list
func removeTenantFromUserHandler(c *gin.Context) {
	userID := c.Param("id")
	tenantID := c.Param("tenantId")
	
	// Check if user exists
	_, err := userManager.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User not found",
		})
		return
	}
	
	// Remove tenant from user
	err = userManager.RemoveTenantFromUser(userID, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to remove tenant from user",
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Tenant removed from user successfully",
	})
}
