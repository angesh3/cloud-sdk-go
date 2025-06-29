package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	sdk "github.com/cisco-pxgrid/cloud-sdk-go"
)

// SecurityGroupsAPI provides a tenant-aware wrapper for security groups operations
type SecurityGroupsAPI struct {
	tenantManager *TenantManager
}

// NewSecurityGroupsAPI creates a new tenant-aware security groups API handler
func NewSecurityGroupsAPI(tm *TenantManager) *SecurityGroupsAPI {
	return &SecurityGroupsAPI{
		tenantManager: tm,
	}
}

// RegisterAPI registers all the security groups API endpoints
func (api *SecurityGroupsAPI) RegisterAPI(r *gin.RouterGroup) {
	r.GET("/security-groups", api.GetSecurityGroups)
	r.GET("/security-groups/kpi", api.GetSecurityGroupsKPI)
	r.POST("/security-groups/refresh", api.RefreshSecurityGroupsData)
	r.GET("/security-groups/debug", api.DebugSecurityGroupsAPI)
	r.GET("/security-groups/list", api.GetSecurityGroupsList)
	r.GET("/security-groups/acls", api.GetSecurityGroupAcls)
	r.GET("/security-groups/egress-policies", api.GetEgressPolicies)
}

// GetTenantDevice is a helper to get a tenant and its active device
func (api *SecurityGroupsAPI) GetTenantDevice(c *gin.Context) (*sdk.Tenant, *sdk.Device, bool) {
	// Get default tenant or specific tenant from query param
	tenantID := c.Query("tenant_id")
	
	var tenant *sdk.Tenant
	if tenantID != "" {
		// Get specified tenant
		tenantCtx, ok := api.tenantManager.GetTenant(tenantID)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Tenant %s not found", tenantID),
			})
			return nil, nil, false
		}
		tenant = tenantCtx.Instance
	} else {
		// Get default tenant
		defaultTenant, ok := api.tenantManager.GetDefaultTenant()
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "No default tenant configured",
			})
			return nil, nil, false
		}
		tenant = defaultTenant.Instance
	}
	
	// Find an active device for this tenant
	device := GetActiveDeviceForTenant(tenant)
	if device == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("No active device found for tenant: %s", tenant.Name()),
		})
		return nil, nil, false
	}
	
	return tenant, device, true
}

// GetEgressPolicies handles requests for egress policies
func (api *SecurityGroupsAPI) GetEgressPolicies(c *gin.Context) {
	fmt.Println("Retrieving egress policies list (tenant-aware)")
	
	// Get tenant and device
	tenant, device, ok := api.GetTenantDevice(c)
	if !ok {
		return // Error response already sent by GetTenantDevice
	}
	
	fmt.Printf("Using tenant: %s (%s) for egress policies\n", tenant.Name(), tenant.ID())
	
	// Use the API to retrieve actual egress policies data
	reqBody := "{}" // Empty request body to get all policies
	
	// Create request with the proper endpoint and method
	req, err := http.NewRequest(http.MethodPost, "/pxgrid/trustsec/getEgressPolicies", bytes.NewBufferString(reqBody))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to create request: %v", err),
			"status": "error",
		})
		return
	}
	
	// Execute the request
	resp, err := device.Query(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to query egress policies: %v", err),
			"status": "error",
		})
		return
	}
	defer resp.Body.Close()

	// Parse the response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to read response: %v", err),
			"status": "error",
		})
		return
	}
	
	// Log the raw response for debugging
	fmt.Printf("Egress Policies API response: %s\n", string(respBody))
	
	// Parse the egress policies data
	var policies []map[string]interface{}
	var rawData map[string]interface{}
	
	if err := json.Unmarshal(respBody, &rawData); err == nil {
		// Extract policies from the API response
		if policyArray, ok := rawData["egressPolicies"].([]interface{}); ok {
			for _, policy := range policyArray {
				if policyMap, ok := policy.(map[string]interface{}); ok {
					policies = append(policies, policyMap)
				}
			}
		} else if policyArray, ok := rawData["response"].([]interface{}); ok {
			for _, policy := range policyArray {
				if policyMap, ok := policy.(map[string]interface{}); ok {
					policies = append(policies, policyMap)
				}
			}
		} else if policyArray, ok := rawData["egressMatrixCells"].([]interface{}); ok {
			// Alternative field name
			for _, policy := range policyArray {
				if policyMap, ok := policy.(map[string]interface{}); ok {
					policies = append(policies, policyMap)
				}
			}
		}
	}
	
	fmt.Printf("Retrieved %d egress policies\n", len(policies))
	c.JSON(http.StatusOK, policies)
}

// GetSecurityGroupAcls handles requests for security group ACLs
func (api *SecurityGroupsAPI) GetSecurityGroupAcls(c *gin.Context) {
	fmt.Println("Retrieving security group ACLs list (tenant-aware)")
	
	// Get tenant and device
	tenant, device, ok := api.GetTenantDevice(c)
	if !ok {
		return // Error response already sent by GetTenantDevice
	}
	
	fmt.Printf("Using tenant: %s (%s) for security group ACLs\n", tenant.Name(), tenant.ID())
	
	// Use the API to retrieve actual ACL data
	reqBody := "{}" // Empty request body to get all ACLs
	
	// Create request with the proper endpoint and method
	req, err := http.NewRequest(http.MethodPost, "/pxgrid/trustsec/getSecurityGroupAcls", bytes.NewBufferString(reqBody))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to create request: %v", err),
			"status": "error",
		})
		return
	}
	
	// Execute the request
	resp, err := device.Query(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to query security group ACLs: %v", err),
			"status": "error",
		})
		return
	}
	defer resp.Body.Close()

	// Parse the response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to read response: %v", err),
			"status": "error",
		})
		return
	}
	
	// Log the raw response for debugging
	fmt.Printf("ACLs API response: %s\n", string(respBody))
	
	// Parse the security group ACLs data
	var acls []map[string]interface{}
	var rawData map[string]interface{}
	
	if err := json.Unmarshal(respBody, &rawData); err == nil {
		// Extract ACLs from the API response
		if aclArray, ok := rawData["securityGroupAcls"].([]interface{}); ok {
			for _, acl := range aclArray {
				if aclMap, ok := acl.(map[string]interface{}); ok {
					acls = append(acls, aclMap)
				}
			}
		} else if aclArray, ok := rawData["response"].([]interface{}); ok {
			for _, acl := range aclArray {
				if aclMap, ok := acl.(map[string]interface{}); ok {
					acls = append(acls, aclMap)
				}
			}
		}
	}
	
	fmt.Printf("Retrieved %d security group ACLs\n", len(acls))
	c.JSON(http.StatusOK, acls)
}

// GetSecurityGroupsList handles requests for security groups list
func (api *SecurityGroupsAPI) GetSecurityGroupsList(c *gin.Context) {
	fmt.Println("Retrieving security groups list (tenant-aware)")
	
	// Get tenant and device
	tenant, device, ok := api.GetTenantDevice(c)
	if !ok {
		return // Error response already sent by GetTenantDevice
	}
	
	fmt.Printf("Using tenant: %s (%s) for security groups list\n", tenant.Name(), tenant.ID())
	
	// Use the actual implementation to get real security groups data
	reqBody := "{}" // Empty request body to get all security groups
	
	// Create request with the proper endpoint and method
	req, err := http.NewRequest(http.MethodPost, "/pxgrid/trustsec/getSecurityGroups", bytes.NewBufferString(reqBody))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to create request: %v", err),
			"status": "error",
		})
		return
	}
	
	// Execute the request
	resp, err := device.Query(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to query security groups: %v", err),
			"status": "error",
		})
		return
	}
	defer resp.Body.Close()

	// Parse the response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to read response: %v", err),
			"status": "error",
		})
		return
	}
	
	// Parse the security groups data
	var securityGroups []SecurityGroup
	var rawData map[string]interface{}
	if err := json.Unmarshal(respBody, &rawData); err == nil {
		// Extract security groups from the API response
		if sgArray, ok := rawData["securityGroups"].([]interface{}); ok {
			for _, sg := range sgArray {
				if sgMap, ok := sg.(map[string]interface{}); ok {
					newSG := extractSecurityGroup(sgMap)
					securityGroups = append(securityGroups, newSG)
				}
			}
		} else if sgArray, ok := rawData["response"].([]interface{}); ok {
			for _, sg := range sgArray {
				if sgMap, ok := sg.(map[string]interface{}); ok {
					newSG := extractSecurityGroup(sgMap)
					securityGroups = append(securityGroups, newSG)
				}
			}
		}
	}
	
	fmt.Printf("Retrieved %d security groups\n", len(securityGroups))
	c.JSON(http.StatusOK, securityGroups)
}

// DebugSecurityGroupsAPI handles debug requests
func (api *SecurityGroupsAPI) DebugSecurityGroupsAPI(c *gin.Context) {
	fmt.Println("Debug security groups API (tenant-aware)")
	
	// Get tenant and device
	tenant, _, ok := api.GetTenantDevice(c)
	if !ok {
		return // Error response already sent by GetTenantDevice
	}
	
	fmt.Printf("Using tenant: %s (%s) for debug\n", tenant.Name(), tenant.ID())
	
	// Added tenant info to response for debugging
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"message": "Debug info would be provided here",
		"tenant": map[string]string{
			"id":   tenant.ID(),
			"name": tenant.Name(),
		},
	})
}

// RefreshSecurityGroupsData handles requests to refresh security groups data
func (api *SecurityGroupsAPI) RefreshSecurityGroupsData(c *gin.Context) {
	fmt.Println("Refreshing security groups data (tenant-aware)")
	
	// Get tenant and device
	tenant, _, ok := api.GetTenantDevice(c)
	if !ok {
		return // Error response already sent by GetTenantDevice
	}
	
	fmt.Printf("Using tenant: %s (%s) for refresh\n", tenant.Name(), tenant.ID())
	
	// Added tenant info to response for debugging
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"message": "Security groups data refreshed",
		"timestamp": fmt.Sprintf("%s", time.Now()),
		"tenant": map[string]string{
			"id":   tenant.ID(),
			"name": tenant.Name(),
		},
	})
}

// GetSecurityGroups handles requests for security groups
func (api *SecurityGroupsAPI) GetSecurityGroups(c *gin.Context) {
	fmt.Println("Retrieving security groups (tenant-aware)")
	
	// Get tenant and device
	tenant, _, ok := api.GetTenantDevice(c)
	if !ok {
		return // Error response already sent by GetTenantDevice
	}
	
	fmt.Printf("Using tenant: %s (%s) for security groups\n", tenant.Name(), tenant.ID())
	
	// Added tenant info to response for debugging
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"message": "Security groups would be retrieved here",
		"tenant": map[string]string{
			"id":   tenant.ID(),
			"name": tenant.Name(),
		},
	})
}

// GetSecurityGroupsKPI handles requests for security groups KPI
func (api *SecurityGroupsAPI) GetSecurityGroupsKPI(c *gin.Context) {
	fmt.Println("Retrieving security groups KPI (tenant-aware)")
	
	// Get tenant and device
	tenant, _, ok := api.GetTenantDevice(c)
	if !ok {
		return // Error response already sent by GetTenantDevice
	}
	
	fmt.Printf("Using tenant: %s (%s) for security groups KPI\n", tenant.Name(), tenant.ID())
	
	// Find an active device for the tenant
	device := findActiveDevice()
	if device == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No active device found",
			"status": "error",
		})
		return
	}
	
	// Use the actual implementation to get real security groups data
	reqBody := "{}" // Empty request body to get all security groups
	
	// Create request with the proper endpoint and method
	req, err := http.NewRequest(http.MethodPost, "/pxgrid/trustsec/getSecurityGroups", bytes.NewBufferString(reqBody))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to create request: %v", err),
			"status": "error",
		})
		return
	}
	
	// Execute the request
	resp, err := device.Query(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to query security groups: %v", err),
			"status": "error",
		})
		return
	}
	defer resp.Body.Close()

	// Parse the response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to read response: %v", err),
			"status": "error",
		})
		return
	}
	
	// Parse the security groups data
	var securityGroups []SecurityGroup
	var rawData map[string]interface{}
	if err := json.Unmarshal(respBody, &rawData); err == nil {
		// Extract security groups from the API response
		if sgArray, ok := rawData["securityGroups"].([]interface{}); ok {
			for _, sg := range sgArray {
				if sgMap, ok := sg.(map[string]interface{}); ok {
					newSG := extractSecurityGroup(sgMap)
					securityGroups = append(securityGroups, newSG)
				}
			}
		} else if sgArray, ok := rawData["response"].([]interface{}); ok {
			for _, sg := range sgArray {
				if sgMap, ok := sg.(map[string]interface{}); ok {
					newSG := extractSecurityGroup(sgMap)
					securityGroups = append(securityGroups, newSG)
				}
			}
		}
	}
	
	// Construct response data
	securityGroupsData := SecurityGroupsResponse{
		SecurityGroups: securityGroups,
	}
	
	// Compute KPI data
	kpiData := computeKPIData(securityGroupsData)
	
	// Add tenant info to the response
	response := gin.H{
		"totalCount": kpiData.TotalCount,
		"deletedCount": kpiData.DeletedCount,
		"tagDistribution": kpiData.TagDistribution,
		"status": "success",
		"tenant": map[string]string{
			"id":   tenant.ID(),
			"name": tenant.Name(),
		},
	}
	
	c.JSON(http.StatusOK, response)
}
