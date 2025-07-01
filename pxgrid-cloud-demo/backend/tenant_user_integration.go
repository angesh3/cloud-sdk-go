package main

import (
	"fmt"
	"net/http"
	
	"github.com/gin-gonic/gin"
)

// User tenant integration
// This file contains functions that integrate the user system with tenant management

// linkTenantWithUserHandler links a tenant to the current authenticated user when OTP is used
func linkTenantWithUserHandler(c *gin.Context, tenantID string) {
	// Get the current user from context (set by authMiddleware)
	userInterface, exists := c.Get("user")
	if !exists {
		// If no user is authenticated, we just proceed with tenant linking
		// without assigning to a user. This maintains compatibility with
		// the original non-user-aware flow.
		return
	}
	
	user := userInterface.(*User)
	
	// Assign the tenant to the user
	err := userManager.AssignTenantToUser(user.ID, tenantID)
	if err != nil {
		// Just log the error but don't stop the process
		fmt.Printf("Warning: Failed to assign tenant %s to user %s: %v\n", 
			tenantID, user.ID, err)
	}
}

// getUserAccessibleTenantsHandler gets tenants that a user has access to
func getUserAccessibleTenantsHandler(c *gin.Context) {
	// Get the current user from context (set by authMiddleware)
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Not authenticated",
		})
		return
	}
	
	user := userInterface.(*User)
	
	// Check if user is an admin (admins can see all tenants)
	var tenantConfigs []map[string]interface{}
	
	if user.Role == "admin" {
		// For admin, return all tenants
		tenantManager.mu.RLock()
		for tenantID, tenantCtx := range tenantManager.tenants {
			if tenantCtx.Instance == nil {
				continue
			}
			
			tenantConfigs = append(tenantConfigs, map[string]interface{}{
				"id":   tenantID,
				"name": tenantCtx.Config.Name,
				"isDefault": tenantManager.defaultTenant == tenantID,
			})
		}
		tenantManager.mu.RUnlock()
	} else {
		// For regular users, return only assigned tenants
		tenantIDs, err := userManager.GetUserTenants(user.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to get user tenants",
			})
			return
		}
		
		tenantManager.mu.RLock()
		for _, tenantID := range tenantIDs {
			tenantCtx, exists := tenantManager.tenants[tenantID]
			if !exists || tenantCtx.Instance == nil {
				continue
			}
			
			tenantConfigs = append(tenantConfigs, map[string]interface{}{
				"id":   tenantID,
				"name": tenantCtx.Config.Name,
				"isDefault": tenantManager.defaultTenant == tenantID,
			})
		}
		tenantManager.mu.RUnlock()
	}
	
	c.JSON(http.StatusOK, gin.H{
		"tenants": tenantConfigs,
	})
}

// updateTenantLinkingAPI modifies the tenant linking API to integrate with user system
func updateTenantLinkingAPI() {
	// This function will be called to patch the original tenant linking functionality
	// to work with the user system. It will be implemented in the next phase when
	// we modify the existing tenant management system.
}

// checkTenantAccess checks if a user has access to a tenant
func checkTenantAccess(userID, tenantID string) bool {
	// Admins always have access to all tenants
	user, err := userManager.GetUserByID(userID)
	if err != nil {
		return false
	}
	
	if user.Role == "admin" {
		return true
	}
	
	// Check if tenant is in user's access list
	tenantIDs, err := userManager.GetUserTenants(userID)
	if err != nil {
		return false
	}
	
	for _, tid := range tenantIDs {
		if tid == tenantID {
			return true
		}
	}
	
	return false
}

// tenantAccessMiddleware checks if a user has access to requested tenant
func tenantAccessMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Run auth middleware first to get user
		authMiddleware()(c)
		
		// Check if request was aborted by auth middleware
		if len(c.Errors) > 0 {
			return
		}
		
		// Get user from context
		userInterface, exists := c.Get("user")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Not authenticated",
			})
			return
		}
		
		user := userInterface.(*User)
		
		// If user is admin, allow access to all tenants
		if user.Role == "admin" {
			c.Next()
			return
		}
		
		// Get requested tenant ID from query param or path param
		tenantID := c.Param("tenantId")
		if tenantID == "" {
			tenantID = c.Query("tenant_id")
		}
		
		// If no tenant ID specified, allow access (the API endpoint will handle appropriate defaults)
		if tenantID == "" {
			c.Next()
			return
		}
		
		// Check if user has access to the tenant
		if !checkTenantAccess(user.ID, tenantID) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "You don't have access to this tenant",
			})
			return
		}
		
		c.Next()
	}
}
