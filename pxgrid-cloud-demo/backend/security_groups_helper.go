package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	sdk "github.com/cisco-pxgrid/cloud-sdk-go"
)

// Helper function to get the default tenant instance for security groups API functions
func getDefaultTenantForSecurityGroups(c *gin.Context) (*sdk.Tenant, bool) {
	fmt.Println("[DEBUG] Getting default tenant for security groups")
	defaultTenant, exists := tenantManager.GetDefaultTenant()
	if !exists {
		fmt.Println("[DEBUG] No default tenant found!")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No default tenant configured",
		})
		return nil, false
	}
	
	fmt.Printf("[DEBUG] Default tenant found: %s (%s)\n", defaultTenant.Instance.Name(), defaultTenant.Instance.ID())
	return defaultTenant.Instance, true
}

// Helper function to get a specific tenant instance by ID
func getTenantForSecurityGroups(c *gin.Context, tenantID string) (*sdk.Tenant, bool) {
	// If no tenant ID is specified, use default
	if tenantID == "" {
		return getDefaultTenantForSecurityGroups(c)
	}
	
	// Get the specified tenant
	tenantCtx, exists := tenantManager.GetTenant(tenantID)
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Tenant %s not found", tenantID),
		})
		return nil, false
	}
	
	return tenantCtx.Instance, true
}

// GetActiveDeviceForTenant gets an active device for a specific tenant
func GetActiveDeviceForTenant(tenant *sdk.Tenant) *sdk.Device {
	if tenant == nil {
		return nil
	}
	
	// This reuses the active device logic but ensures we use a device for the specific tenant
	for _, device := range activeDevices {
		if device.Tenant().ID() == tenant.ID() {
			return device
		}
	}
	
	return nil
}
