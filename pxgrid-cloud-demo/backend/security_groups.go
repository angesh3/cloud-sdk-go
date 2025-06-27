package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	sdk "github.com/cisco-pxgrid/cloud-sdk-go"
)

// SecurityGroup represents a security group from the API response
// As per https://github.com/cisco-pxgrid/pxgrid-rest-ws/wiki/TrustSec-configuration#post-restbaseurlgetsecuritygroups
type SecurityGroup struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Tag         string `json:"tag,omitempty"`
	Propagate   bool   `json:"propagateToApic,omitempty"`
	Default     bool   `json:"defaultSGT,omitempty"`
}

// SecurityGroupsResponse represents the response from the getSecurityGroups API
type SecurityGroupsResponse struct {
	SecurityGroups []SecurityGroup `json:"securityGroups"`
}

// KPIData represents computed KPI metrics for security groups
type KPIData struct {
	TotalCount        int                   `json:"totalCount"`
	DeletedCount      int                   `json:"deletedCount"`
	StatusDistribution map[string]int       `json:"statusDistribution"`
	TagDistribution   map[string]int        `json:"tagDistribution"`
	CreationTrend     map[string]int        `json:"creationTrend"` // Date -> count
	DeletionTrend     map[string]int        `json:"deletionTrend"` // Date -> count
	HistoricalData    []HistoricalDataPoint `json:"historicalData"`
}

// HistoricalDataPoint represents historical data for KPI trends
type HistoricalDataPoint struct {
	Timestamp    time.Time `json:"timestamp"`
	TotalCount   int       `json:"totalCount"`
	NewCount     int       `json:"newCount"`
	DeletedCount int       `json:"deletedCount"`
}

// Store for historical data
var historicalKPIData []HistoricalDataPoint
var lastPolledTime time.Time
var activeDevices = make(map[string]*sdk.Device)

// SGACL represents a security group ACL from the API response
type SGACL struct {
	ID          string   `json:"id,omitempty"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	ACLContent  string   `json:"aclcontent,omitempty"`
	IPVersion   string   `json:"ipVersion,omitempty"`
	GenerationID string  `json:"generationId,omitempty"`
	ACEs        []string `json:"aces,omitempty"` // Access Control Entries
}

// EgressPolicy represents an egress policy from the API response
type EgressPolicy struct {
	ID              string `json:"id,omitempty"`
	Name            string `json:"name"`
	Description     string `json:"description,omitempty"`
	SourceSGT       SGT    `json:"sourceSgt,omitempty"`
	DestinationSGT  SGT    `json:"destinationSgt,omitempty"`
	ACL             SGACL  `json:"acl,omitempty"`
	MatrixCellStatus string `json:"matrixCellStatus,omitempty"`
	DefaultRule     string `json:"defaultRule,omitempty"`
}

// SGT represents a simplified Security Group Tag for references in policies
type SGT struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	Tag  string `json:"tag,omitempty"`
}

// Register security groups API
func registerSecurityGroupsAPI(r *gin.RouterGroup) {
	r.GET("/security-groups", getSecurityGroups)
	r.GET("/security-groups/kpi", getSecurityGroupsKPI)
	r.POST("/security-groups/refresh", refreshSecurityGroupsData)
	r.GET("/security-groups/debug", debugSecurityGroupsAPI)
	r.GET("/security-groups/list", getSecurityGroupsList)
	r.GET("/security-groups/acls", getSecurityGroupAcls)
	r.GET("/security-groups/egress-policies", getEgressPolicies)
}

// Get complete list of egress policies
func getEgressPolicies(c *gin.Context) {
	fmt.Println("Retrieving egress policies list")
	
	if tenant == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No tenant linked",
		})
		return
	}

	// Find an active device
	device := findActiveDevice()
	if device == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No active device found",
		})
		return
	}
	
	// Send request to get egress policies
	// Using the endpoint from documentation: POST [restBaseUrl]/getEgressPolicies
	reqBody := "{}" // Empty request body to get all policies
	req, err := http.NewRequest(http.MethodPost, "/pxgrid/trustsec/getEgressPolicies", bytes.NewBufferString(reqBody))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to create request: %v", err),
		})
		return
	}

	// Execute the request
	fmt.Println("Executing egress policies API request...")
	resp, err := device.Query(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to query egress policies: %v", err),
		})
		return
	}
	defer resp.Body.Close()

	// Log response status
	fmt.Printf("API Response Status: %d %s\n", resp.StatusCode, resp.Status)
	
	// Parse the response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to read response: %v", err),
		})
		return
	}
	
	// For debugging
	fmt.Printf("Egress Policies API Response: %s\n", string(respBody))
	
	// Parse the response into structured data
	var rawData map[string]interface{}
	if err := json.Unmarshal(respBody, &rawData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to parse response: %v", err),
		})
		return
	}
	
	// Extract egress policies from the response
	var policies []EgressPolicy
	
	// Try to extract from different possible formats
	if policiesArray, ok := rawData["egressPolicies"].([]interface{}); ok {
		for _, policy := range policiesArray {
			if policyMap, ok := policy.(map[string]interface{}); ok {
				policies = append(policies, extractEgressPolicy(policyMap))
			}
		}
	} else if policiesArray, ok := rawData["policies"].([]interface{}); ok {
		for _, policy := range policiesArray {
			if policyMap, ok := policy.(map[string]interface{}); ok {
				policies = append(policies, extractEgressPolicy(policyMap))
			}
		}
	} else if policiesArray, ok := rawData["response"].([]interface{}); ok {
		for _, policy := range policiesArray {
			if policyMap, ok := policy.(map[string]interface{}); ok {
				policies = append(policies, extractEgressPolicy(policyMap))
			}
		}
	}
	
	// Return the policies list
	c.JSON(http.StatusOK, policies)
}

// Helper function to extract egress policy data from map
func extractEgressPolicy(data map[string]interface{}) EgressPolicy {
	policy := EgressPolicy{}
	
	// Extract basic fields with proper type conversion
	if id, ok := data["id"].(string); ok {
		policy.ID = id
	}
	if name, ok := data["name"].(string); ok {
		policy.Name = name
	}
	if desc, ok := data["description"].(string); ok {
		policy.Description = desc
	}
	if status, ok := data["matrixCellStatus"].(string); ok {
		policy.MatrixCellStatus = status
	}
	if rule, ok := data["defaultRule"].(string); ok {
		policy.DefaultRule = rule
	}
	
	// Extract source SGT
	if srcSGTData, ok := data["sourceSgt"].(map[string]interface{}); ok {
		policy.SourceSGT = extractSGT(srcSGTData)
	}
	
	// Extract destination SGT
	if dstSGTData, ok := data["destinationSgt"].(map[string]interface{}); ok {
		policy.DestinationSGT = extractSGT(dstSGTData)
	}
	
	// Extract ACL
	if aclData, ok := data["acl"].(map[string]interface{}); ok {
		policy.ACL = extractSGACL(aclData)
	}
	
	return policy
}

// Helper function to extract SGT data from map
func extractSGT(data map[string]interface{}) SGT {
	sgt := SGT{}
	
	// Extract fields with proper type conversion
	if id, ok := data["id"].(string); ok {
		sgt.ID = id
	}
	if name, ok := data["name"].(string); ok {
		sgt.Name = name
	}
	if tag, ok := data["tag"].(string); ok {
		sgt.Tag = tag
	} else if value, ok := data["value"].(string); ok {
		// Some APIs use "value" instead of "tag"
		sgt.Tag = value
	}
	
	return sgt
}

// Get complete list of security group ACLs
func getSecurityGroupAcls(c *gin.Context) {
	fmt.Println("Retrieving security group ACLs list")
	
	if tenant == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No tenant linked",
		})
		return
	}

	// Find an active device
	device := findActiveDevice()
	if device == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No active device found",
		})
		return
	}
	
	// Send request to get security group ACLs
	// Using the endpoint from documentation: POST [restBaseUrl]/getSecurityGroupAcls
	reqBody := "{}" // Empty request body to get all ACLs
	req, err := http.NewRequest(http.MethodPost, "/pxgrid/trustsec/getSecurityGroupAcls", bytes.NewBufferString(reqBody))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to create request: %v", err),
		})
		return
	}

	// Execute the request
	resp, err := device.Query(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to query security group ACLs: %v", err),
		})
		return
	}
	defer resp.Body.Close()
	
	// Parse the response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to read response: %v", err),
		})
		return
	}
	
	// For debugging
	fmt.Printf("SGACL API Response: %s\n", string(respBody))
	
	// Parse the response into structured data
	var rawData map[string]interface{}
	if err := json.Unmarshal(respBody, &rawData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to parse response: %v", err),
		})
		return
	}
	
	// Extract ACLs from the response
	var acls []SGACL
	
	// Try to extract from standard format (securityGroupAcls field)
	if sgaclsArray, ok := rawData["securityGroupAcls"].([]interface{}); ok {
		for _, sgacl := range sgaclsArray {
			if sgaclMap, ok := sgacl.(map[string]interface{}); ok {
				acls = append(acls, extractSGACL(sgaclMap))
			}
		}
	} else if sgaclsArray, ok := rawData["acls"].([]interface{}); ok { // Alternative format
		for _, sgacl := range sgaclsArray {
			if sgaclMap, ok := sgacl.(map[string]interface{}); ok {
				acls = append(acls, extractSGACL(sgaclMap))
			}
		}
	} else if sgaclsArray, ok := rawData["response"].([]interface{}); ok { // Another alternative format
		for _, sgacl := range sgaclsArray {
			if sgaclMap, ok := sgacl.(map[string]interface{}); ok {
				acls = append(acls, extractSGACL(sgaclMap))
			}
		}
	}
	
	// Return the ACLs list
	c.JSON(http.StatusOK, acls)
}

// Helper function to extract SGACL data from map
func extractSGACL(data map[string]interface{}) SGACL {
	acl := SGACL{}
	
	// Extract basic fields with proper type conversion
	if id, ok := data["id"].(string); ok {
		acl.ID = id
	}
	if name, ok := data["name"].(string); ok {
		acl.Name = name
	}
	if desc, ok := data["description"].(string); ok {
		acl.Description = desc
	}
	if content, ok := data["aclcontent"].(string); ok {
		acl.ACLContent = content
	}
	if ipver, ok := data["ipVersion"].(string); ok {
		acl.IPVersion = ipver
	}
	if genID, ok := data["generationId"].(string); ok {
		acl.GenerationID = genID
	}
	
	// Extract ACEs (Access Control Entries) which are typically an array of strings
	if acesArray, ok := data["aces"].([]interface{}); ok {
		for _, ace := range acesArray {
			if aceStr, ok := ace.(string); ok {
				acl.ACEs = append(acl.ACEs, aceStr)
			}
		}
	}
	
	// If we have aclcontent but no ACEs, try to parse the content
	if len(acl.ACEs) == 0 && acl.ACLContent != "" {
		// Simple parsing of ACL content, assuming entries are separated by newlines
		lines := strings.Split(acl.ACLContent, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				acl.ACEs = append(acl.ACEs, line)
			}
		}
	}
	
	return acl
}

// Get complete list of security groups
func getSecurityGroupsList(c *gin.Context) {
	fmt.Println("Retrieving security groups list")
	
	if tenant == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No tenant linked",
		})
		return
	}

	// Find an active device
	device := findActiveDevice()
	if device == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No active device found",
		})
		return
	}
	
	// Send request to get security groups
	reqBody := "{}" // Empty request body to get all security groups
	req, err := http.NewRequest(http.MethodPost, "/pxgrid/trustsec/getSecurityGroups", bytes.NewBufferString(reqBody))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to create request: %v", err),
		})
		return
	}

	// Execute the request
	resp, err := device.Query(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to query security groups: %v", err),
		})
		return
	}
	defer resp.Body.Close()
	
	// Parse the response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to read response: %v", err),
		})
		return
	}
	
	// Parse the response into structured data
	var rawData map[string]interface{}
	if err := json.Unmarshal(respBody, &rawData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to parse response: %v", err),
		})
		return
	}
	
	// Extract security groups
	var securityGroups []SecurityGroup
	
	// Try to extract from standard format
	if sgArray, ok := rawData["securityGroups"].([]interface{}); ok {
		for _, sg := range sgArray {
			if sgMap, ok := sg.(map[string]interface{}); ok {
				newSG := extractSecurityGroup(sgMap)
				securityGroups = append(securityGroups, newSG)
			}
		}
	} else {
		// Try alternative formats
		if sgArray, ok := rawData["response"].([]interface{}); ok {
			for _, sg := range sgArray {
				if sgMap, ok := sg.(map[string]interface{}); ok {
					newSG := extractSecurityGroup(sgMap)
					securityGroups = append(securityGroups, newSG)
				}
			}
		}
	}
	
	// Return the security groups list
	c.JSON(http.StatusOK, securityGroups)
}

// Debug endpoint to test API paths and parameters
func debugSecurityGroupsAPI(c *gin.Context) {
	if tenant == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No tenant linked",
		})
		return
	}
	
	// Find an active device
	device := findActiveDevice()
	if device == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No active device found",
		})
		return
	}
	
	// Get debug info
	debugInfo := map[string]interface{}{
		"device": map[string]string{
			"id":   device.ID(),
			"name": device.Name(),
		},
		"tenant": map[string]string{
			"id":   tenant.ID(),
			"name": tenant.Name(),
		},
		"availablePaths": []string{
			"/pxgrid/trustsec/getSecurityGroups",
			"/pxgrid/ise/config/trustsec/getSecurityGroups",
			"/pxgrid/ise/config/trustsec/securitygroups",
			"/pxgrid/ise/config/trustsec/security-groups",
			"/pxgrid/trustsec/getBulkSecurityGroups",
		},
	}
	
	// Try a test query with minimal path
	testPath := "/pxgrid/ise/config/trustsec"
	req, _ := http.NewRequest(http.MethodGet, testPath, nil)
	resp, err := device.Query(req)
	if err != nil {
		debugInfo["testQuery"] = map[string]interface{}{
			"path":  testPath,
			"error": err.Error(),
		}
	} else {
		respBody, _ := io.ReadAll(resp.Body)
		debugInfo["testQuery"] = map[string]interface{}{
			"path":     testPath,
			"status":   resp.Status,
			"response": string(respBody),
		}
		resp.Body.Close()
	}
	
	c.JSON(http.StatusOK, debugInfo)
}

// findActiveDevice looks for an active device to use for API calls
func findActiveDevice() *sdk.Device {
	deviceMutex.RLock()
	defer deviceMutex.RUnlock()
	
	for id, deviceInfo := range deviceStore {
		if deviceInfo.Status == "active" {
			device, exists := activeDevices[id]
			if exists {
				return device
			}
		}
	}
	return nil
}

// Store device in activation handler
func storeActiveDevice(d *sdk.Device) {
	activeDevices[d.ID()] = d
}

// Remove inactive device
func removeInactiveDevice(d *sdk.Device) {
	delete(activeDevices, d.ID())
}

// Initialize the security groups module
func init() {
	// Subscribe to device activation/deactivation to maintain our device cache
	// This is already handled in main.go via the handler functions, just accessing them here
}

// Retrieve security groups from the API
func getSecurityGroups(c *gin.Context) {
	if tenant == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No tenant linked",
		})
		return
	}

	// Find an active device
	device := findActiveDevice()
	if device == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No active device found",
		})
		return
	}

	// Create request to getSecurityGroups API
	reqBody := "{}" // Empty request for all security groups
	req, err := http.NewRequest(http.MethodPost, "/pxgrid/trustsec/getSecurityGroups", bytes.NewBufferString(reqBody))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to create request: %v", err),
		})
		return
	}

	// Execute the request
	resp, err := device.Query(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to query security groups: %v", err),
		})
		return
	}
	defer resp.Body.Close()

	// Parse the response
	var result SecurityGroupsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to parse response: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// Get computed KPI data for security groups
func getSecurityGroupsKPI(c *gin.Context) {
	fmt.Println("Received security groups KPI request")
	
	if tenant == nil {
		fmt.Println("Error: No tenant linked")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No tenant linked",
		})
		return
	}
	fmt.Printf("Using tenant: %s (%s)\n", tenant.Name(), tenant.ID())

	// Find an active device
	device := findActiveDevice()
	if device == nil {
		fmt.Println("Error: No active device found")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No active device found",
		})
		return
	}
	fmt.Printf("Using device: %s (%s)\n", device.Name(), device.ID())

	// Use the correct API endpoint as specified in the Cloud SDK README
	// See: /Users/anvikram/code/cisco/spike/cloud-sdk-go/README.md
	reqBody := "{}" // Empty request body to get all security groups
	fmt.Printf("Sending security groups request: %s\n", reqBody)
	
	// Create request with the proper endpoint and method
	req, err := http.NewRequest(http.MethodPost, "/pxgrid/trustsec/getSecurityGroups", bytes.NewBufferString(reqBody))
	if err != nil {
		fmt.Printf("Failed to create request: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to create request: %v", err),
		})
		return
	}

	// Log device details
	fmt.Printf("Device details:\n")
	fmt.Printf("  - ID: %s\n", device.ID())
	fmt.Printf("  - Name: %s\n", device.Name())
	fmt.Printf("  - Tenant ID: %s\n", device.Tenant().ID())
	fmt.Printf("  - Tenant Name: %s\n", device.Tenant().Name())
	
	// Log request details
	fmt.Printf("Request URL: %s\n", req.URL.String())
	fmt.Printf("Request Method: %s\n", req.Method)
	fmt.Printf("Request Headers:\n")
	for k, v := range req.Header {
		fmt.Printf("  %s: %v\n", k, v)
	}
	fmt.Printf("Request Body: %s\n", reqBody)
	
	// Execute the request
	fmt.Println("Executing security groups API request...")
	resp, err := device.Query(req)
	if err != nil {
		fmt.Printf("Failed to query security groups: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to query security groups: %v", err),
		})
		return
	}
	defer resp.Body.Close()

	// Log response status and headers
	fmt.Printf("API Response Status: %d %s\n", resp.StatusCode, resp.Status)
	fmt.Printf("API Response Headers:\n")
	for k, v := range resp.Header {
		fmt.Printf("  %s: %v\n", k, v)
	}
	
	// Parse the response - use a map for flexibility with API format
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Failed to read response body: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to read response: %v", err),
		})
		return
	}
	
	// Log the raw response for debugging
	rawResponse := string(respBody)
	fmt.Printf("Raw API response: %s\n", rawResponse)
	
	// First try parsing with generic map to understand structure
	var rawData map[string]interface{}
	if err := json.Unmarshal(respBody, &rawData); err != nil {
		fmt.Printf("Failed to parse raw response: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to parse raw response: %v", err),
		})
		return
	}
	
	// Print structure of the response for debugging
	fmt.Println("API Response Structure:")
	for k, v := range rawData {
		fmt.Printf("Key: %s, Type: %T\n", k, v)
	}
	
	// Create a manual SecurityGroupsResponse with the data
	var securityGroups []SecurityGroup
	
	// Extract securityGroups from the response based on what's actually in the data
	// Attempt to get from a standard format first
	if sgArray, ok := rawData["securityGroups"].([]interface{}); ok {
		// Got the expected array format
		for _, sg := range sgArray {
			if sgMap, ok := sg.(map[string]interface{}); ok {
				newSG := extractSecurityGroup(sgMap)
				securityGroups = append(securityGroups, newSG)
			}
		}
	} else {
		// Try other possible formats in the response
		fmt.Println("Could not find standard securityGroups array, trying alternatives")
		
		// If the response is itself an array of security groups
		if sgArray, ok := rawData["response"].([]interface{}); ok {
			for _, sg := range sgArray {
				if sgMap, ok := sg.(map[string]interface{}); ok {
					newSG := extractSecurityGroup(sgMap)
					securityGroups = append(securityGroups, newSG)
				}
			}
		}
	}
	
	// Construct a SecurityGroupsResponse
	securityGroupsData := SecurityGroupsResponse{
		SecurityGroups: securityGroups,
	}
	
	fmt.Printf("Processed %d security groups\n", len(securityGroupsData.SecurityGroups))
	
	// Compute KPI data
	kpiData := computeKPIData(securityGroupsData)
	c.JSON(http.StatusOK, kpiData)
}

// Trigger a refresh of security groups data
func refreshSecurityGroupsData(c *gin.Context) {
	if tenant == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No tenant linked",
		})
		return
	}

	// Find an active device
	device := findActiveDevice()
	if device == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No active device found",
		})
		return
	}

	now := time.Now()
	
	// Create request body for the API call
	reqBody := "{}" // Empty request to get all security groups
	fmt.Println("Sending refresh request to security groups API")
	
	req, err := http.NewRequest(http.MethodPost, "/pxgrid/ise/config/trustsec/getSecurityGroups", bytes.NewBufferString(reqBody))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to create request: %v", err),
		})
		return
	}

	// Execute the request
	resp, err := device.Query(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to query security groups: %v", err),
		})
		return
	}
	defer resp.Body.Close()

	// Parse the response
	var securityGroupsData SecurityGroupsResponse
	if err := json.NewDecoder(resp.Body).Decode(&securityGroupsData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to parse response: %v", err),
		})
		return
	}

	// Update historical data
	updateHistoricalData(securityGroupsData, now)
	
	// Update last polled time
	lastPolledTime = now
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Security groups data refreshed",
		"timestamp": now.Format(time.RFC3339),
	})
}

// Compute KPI data from security groups response
func computeKPIData(data SecurityGroupsResponse) KPIData {
	totalSecurityGroups := len(data.SecurityGroups)
	
	kpi := KPIData{
		TotalCount:         totalSecurityGroups,
		DeletedCount:       0, // No deleted groups data in the new API format
		StatusDistribution: make(map[string]int),
		TagDistribution:    make(map[string]int),
		CreationTrend:      make(map[string]int),
		DeletionTrend:      make(map[string]int),
		HistoricalData:     historicalKPIData,
	}

	// Get current date for trend data
	currentDate := time.Now().Format("2006-01-02")
	kpi.CreationTrend[currentDate] = totalSecurityGroups
	
	// Calculate tag distribution from security groups
	for _, group := range data.SecurityGroups {
		// Add to active count
		kpi.StatusDistribution["active"]++

		// Tag distribution
		if group.Tag != "" {
			kpi.TagDistribution[group.Tag]++
		}
	}

	return kpi
}

// Helper function to extract security group data from map
func extractSecurityGroup(data map[string]interface{}) SecurityGroup {
	sg := SecurityGroup{}
	
	// Extract fields with proper type conversion
	if id, ok := data["id"].(string); ok {
		sg.ID = id
	}
	if name, ok := data["name"].(string); ok {
		sg.Name = name
	}
	if desc, ok := data["description"].(string); ok {
		sg.Description = desc
	}
	if tag, ok := data["tag"].(string); ok {
		sg.Tag = tag
	} else if value, ok := data["value"].(string); ok {
		// Some APIs use "value" instead of "tag"
		sg.Tag = value
	} else if sgt, ok := data["sgt"].(string); ok {
		// Some APIs use "sgt" for the tag
		sg.Tag = sgt
	}
	
	// Boolean values with type conversion
	if propagate, ok := data["propagateToApic"].(bool); ok {
		sg.Propagate = propagate
	}
	if defaultSGT, ok := data["defaultSGT"].(bool); ok {
		sg.Default = defaultSGT
	}
	
	return sg
}

// Update historical data with new security groups data
func updateHistoricalData(data SecurityGroupsResponse, timestamp time.Time) {
	// Count total groups
	totalCount := len(data.SecurityGroups)
	
	// For this API, we can't determine which groups are new since creation time isn't available
	// We'll just record the total count over time
	
	// Create a new historical data point
	dataPoint := HistoricalDataPoint{
		Timestamp:    timestamp,
		TotalCount:   totalCount,
		NewCount:     0,      // We can't determine new groups from this API
		DeletedCount: 0,      // We can't determine deleted groups from this API
	}

	// Add to historical data
	historicalKPIData = append(historicalKPIData, dataPoint)

	// Limit historical data to last 100 entries
	if len(historicalKPIData) > 100 {
		historicalKPIData = historicalKPIData[len(historicalKPIData)-100:]
	}
}
