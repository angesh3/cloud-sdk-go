package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	sdk "github.com/cisco-pxgrid/cloud-sdk-go"
)

// Helper function to prepare a security groups operation with tenant check
// This centralizes the tenant and device checks used in multiple security group operations
func prepareSecurityGroupsOp(c *gin.Context) (*sdk.Tenant, *sdk.Device, bool) {
	// Get default tenant using helper
	tenantInstance, ok := getDefaultTenantForSecurityGroups(c)
	if !ok {
		return nil, nil, false
	}

	// Find an active device for this tenant
	device := GetActiveDeviceForTenant(tenantInstance)
	if device == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No active device found for tenant: " + tenantInstance.Name(),
		})
		return nil, nil, false
	}

	return tenantInstance, device, true
}
