package main

import (
	"fmt"
	"log"
	"os"
	"sync"

	sdk "github.com/cisco-pxgrid/cloud-sdk-go"
	"gopkg.in/yaml.v2"
)

// TenantContext holds tenant-specific data and state
type TenantContext struct {
	Config   TenantConfig
	Instance *sdk.Tenant
	Devices  map[string]DeviceInfo
	mu       sync.RWMutex
}

// TenantManager handles multiple tenant contexts
type TenantManager struct {
	tenants       map[string]*TenantContext
	app           *sdk.App
	defaultTenant string
	mu            sync.RWMutex
	configFile    string
}

// TenantSummary represents a summary of tenant data for API responses
type TenantSummary struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	DeviceCount int    `json:"devices"`
	IsDefault  bool   `json:"isDefault"`
}

// NewTenantManager creates a new tenant manager
func NewTenantManager(app *sdk.App, configFile string) *TenantManager {
	return &TenantManager{
		tenants:    make(map[string]*TenantContext),
		app:        app,
		configFile: configFile,
	}
}

// LoadTenantsFromConfig initializes tenants from config
func (tm *TenantManager) LoadTenantsFromConfig(config *Config) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	
	tm.defaultTenant = config.DefaultTenantId
	
	// Process the regular tenants map first
	for id, tenantConfig := range config.Tenants {
		if tenantConfig.ID == "" || tenantConfig.Name == "" || tenantConfig.Token == "" {
			continue // Skip invalid tenants
		}
		
		tenant, err := tm.app.SetTenant(tenantConfig.ID, tenantConfig.Name, tenantConfig.Token)
		if err != nil {
			log.Printf("Failed to set tenant %s: %v\n", tenantConfig.Name, err)
			continue
		}
		
		tm.tenants[id] = &TenantContext{
			Config:   tenantConfig,
			Instance: tenant,
			Devices:  make(map[string]DeviceInfo),
		}
		
		log.Printf("Linked with tenant: %s (%s)\n", tenant.Name(), tenant.ID())
	}
	
	// Handle backward compatibility with single tenant configuration
	if config.Tenant.ID != "" && config.Tenant.Name != "" && config.Tenant.Token != "" {
		tenant, err := tm.app.SetTenant(config.Tenant.ID, config.Tenant.Name, config.Tenant.Token)
		if err != nil {
			log.Printf("Failed to set backward compatibility tenant %s: %v\n", config.Tenant.Name, err)
		} else {
			tm.tenants[config.Tenant.ID] = &TenantContext{
				Config:   config.Tenant,
				Instance: tenant,
				Devices:  make(map[string]DeviceInfo),
			}
			
			log.Printf("Linked with backward compatibility tenant: %s (%s)\n", tenant.Name(), tenant.ID())
			
			// If no default tenant is set, use this one as the default
			if tm.defaultTenant == "" {
				tm.defaultTenant = config.Tenant.ID
				log.Printf("Setting backward compatibility tenant as default: %s (%s)\n", tenant.Name(), tenant.ID())
			}
		}
	}
	
	return nil
}

// GetTenant returns a tenant context by ID
func (tm *TenantManager) GetTenant(id string) (*TenantContext, bool) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	
	tenant, exists := tm.tenants[id]
	return tenant, exists
}

// GetDefaultTenant returns the default tenant context
func (tm *TenantManager) GetDefaultTenant() (*TenantContext, bool) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	
	if tm.defaultTenant == "" {
		return nil, false
	}
	
	tenant, exists := tm.tenants[tm.defaultTenant]
	return tenant, exists
}

// GetAllTenants returns all tenant contexts
func (tm *TenantManager) GetAllTenants() []TenantSummary {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	
	tenants := make([]TenantSummary, 0, len(tm.tenants))
	for id, tenant := range tm.tenants {
		tenant.mu.RLock()
		tenants = append(tenants, TenantSummary{
			ID:          id,
			Name:        tenant.Config.Name,
			DeviceCount: len(tenant.Devices),
			IsDefault:   id == tm.defaultTenant,
		})
		tenant.mu.RUnlock()
	}
	
	return tenants
}

// LinkTenant links a new tenant with OTP
func (tm *TenantManager) LinkTenant(otp string) (*sdk.Tenant, error) {
	tenant, err := tm.app.LinkTenant(otp)
	if err != nil {
		return nil, err
	}
	
	tenantConfig := TenantConfig{
		ID:    tenant.ID(),
		Name:  tenant.Name(),
		Token: tenant.ApiToken(),
		Otp:   "",
	}
	
	tm.mu.Lock()
	tm.tenants[tenant.ID()] = &TenantContext{
		Config:   tenantConfig,
		Instance: tenant,
		Devices:  make(map[string]DeviceInfo),
	}
	tm.mu.Unlock()
	
	// Update config file
	tm.updateConfigFile(tenant.ID(), false)
	
	return tenant, nil
}

// SetDefaultTenant sets a tenant as the default
func (tm *TenantManager) SetDefaultTenant(id string) error {
	tm.mu.Lock()
	_, exists := tm.tenants[id]
	if !exists {
		tm.mu.Unlock()
		return fmt.Errorf("tenant not found: %s", id)
	}
	
	tm.defaultTenant = id
	tm.mu.Unlock()
	
	// Update config file
	return tm.updateConfigFile(id, true)
}

// UnlinkTenant unlinks a tenant
func (tm *TenantManager) UnlinkTenant(id string) error {
	tm.mu.Lock()
	tenantCtx, exists := tm.tenants[id]
	tm.mu.Unlock()
	
	if !exists {
		return fmt.Errorf("tenant not found: %s", id)
	}
	
	err := tm.app.UnlinkTenant(tenantCtx.Instance)
	if err != nil {
		return err
	}
	
	tm.mu.Lock()
	delete(tm.tenants, id)
	
	// If we just removed the default tenant, set a new default if possible
	if tm.defaultTenant == id {
		tm.defaultTenant = ""
		for newDefault := range tm.tenants {
			tm.defaultTenant = newDefault
			break
		}
	}
	tm.mu.Unlock()
	
	// Update config file to remove the tenant
	return tm.removeFromConfigFile(id)
}

// AddDeviceToTenant adds a device to a tenant context
func (tm *TenantManager) AddDeviceToTenant(device *sdk.Device) {
	tenantID := device.Tenant().ID()
	
	tm.mu.RLock()
	tenantCtx, exists := tm.tenants[tenantID]
	tm.mu.RUnlock()
	
	if !exists {
		log.Printf("Warning: Device from unknown tenant: %s\n", tenantID)
		return
	}
	
	deviceInfo := DeviceInfo{
		ID:         device.ID(),
		Name:       device.Name(),
		TenantName: device.Tenant().Name(),
		Status:     "active",
	}
	
	tenantCtx.mu.Lock()
	tenantCtx.Devices[device.ID()] = deviceInfo
	tenantCtx.mu.Unlock()
}

// RemoveDeviceFromTenant updates device status to inactive
func (tm *TenantManager) RemoveDeviceFromTenant(device *sdk.Device) {
	tenantID := device.Tenant().ID()
	
	tm.mu.RLock()
	tenantCtx, exists := tm.tenants[tenantID]
	tm.mu.RUnlock()
	
	if !exists {
		return
	}
	
	tenantCtx.mu.Lock()
	if deviceInfo, ok := tenantCtx.Devices[device.ID()]; ok {
		deviceInfo.Status = "inactive"
		tenantCtx.Devices[device.ID()] = deviceInfo
	}
	tenantCtx.mu.Unlock()
}

// GetAllDevices returns all devices across all tenants
func (tm *TenantManager) GetAllDevices() []DeviceInfo {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	
	devices := []DeviceInfo{}
	for _, tenantCtx := range tm.tenants {
		tenantCtx.mu.RLock()
		for _, device := range tenantCtx.Devices {
			devices = append(devices, device)
		}
		tenantCtx.mu.RUnlock()
	}
	
	return devices
}

// GetTenantDevices returns devices for a specific tenant
func (tm *TenantManager) GetTenantDevices(tenantID string) ([]DeviceInfo, error) {
	tm.mu.RLock()
	tenantCtx, exists := tm.tenants[tenantID]
	tm.mu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("tenant not found: %s", tenantID)
	}
	
	tenantCtx.mu.RLock()
	defer tenantCtx.mu.RUnlock()
	
	devices := make([]DeviceInfo, 0, len(tenantCtx.Devices))
	for _, device := range tenantCtx.Devices {
		devices = append(devices, device)
	}
	
	return devices, nil
}

// updateConfigFile updates the config file with current tenant data
func (tm *TenantManager) updateConfigFile(newTenantID string, setAsDefault bool) error {
	// Read existing config
	data, err := os.ReadFile(tm.configFile)
	if err != nil {
		return err
	}
	
	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return err
	}
	
	// Create tenants map if it doesn't exist
	if config.Tenants == nil {
		config.Tenants = make(map[string]TenantConfig)
	}
	
	// Update tenants in config
	tm.mu.RLock()
	for id, tenant := range tm.tenants {
		tenant.mu.RLock()
		config.Tenants[id] = tenant.Config
		tenant.mu.RUnlock()
	}
	
	// Update default tenant ID if requested
	if setAsDefault {
		config.DefaultTenantId = newTenantID
	} else if tm.defaultTenant != "" {
		config.DefaultTenantId = tm.defaultTenant
	}
	tm.mu.RUnlock()
	
	// Save config
	updatedData, err := yaml.Marshal(&config)
	if err != nil {
		return err
	}
	
	return os.WriteFile(tm.configFile, updatedData, 0644)
}

// removeFromConfigFile removes a tenant from the config file
func (tm *TenantManager) removeFromConfigFile(tenantID string) error {
	// Read existing config
	data, err := os.ReadFile(tm.configFile)
	if err != nil {
		return err
	}
	
	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return err
	}
	
	// Remove tenant from config
	delete(config.Tenants, tenantID)
	
	// Update default tenant ID if needed
	tm.mu.RLock()
	if config.DefaultTenantId == tenantID {
		config.DefaultTenantId = tm.defaultTenant
	}
	tm.mu.RUnlock()
	
	// Save config
	updatedData, err := yaml.Marshal(&config)
	if err != nil {
		return err
	}
	
	return os.WriteFile(tm.configFile, updatedData, 0644)
}
