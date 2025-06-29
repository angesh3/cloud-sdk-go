package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	sdk "github.com/cisco-pxgrid/cloud-sdk-go"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gopkg.in/yaml.v2"
)

// Configuration structures
type AppConfig struct {
	Id            string   `yaml:"id"`
	ApiKey        string   `yaml:"apiKey"`
	GlobalFQDN    string   `yaml:"globalFQDN"`
	RegionalFQDN  string   `yaml:"regionalFQDN"`
	RegionalFQDNs []string `yaml:"regionalFQDNs"`
	ReadStream    string   `yaml:"readStream"`
	WriteStream   string   `yaml:"writeStream"`
	GroupId       string   `yaml:"groupId"`
}

type TenantConfig struct {
	Otp   string `yaml:"otp"`
	ID    string `yaml:"id"`
	Name  string `yaml:"name"`
	Token string `yaml:"token"`
}

type Config struct {
	App           AppConfig                `yaml:"app"`
	Tenants       map[string]TenantConfig  `yaml:"tenants"`
	DefaultTenantId string                 `yaml:"defaultTenantId"`
	// For backward compatibility
	Tenant        TenantConfig `yaml:"tenant"` 
}

// Message store
type MessageStore struct {
	sync.RWMutex
	Messages []Message
}

// Message format for UI
type Message struct {
	Time       time.Time `json:"time"`
	TenantID   string    `json:"tenantId"`
	TenantName string    `json:"tenantName"`
	DeviceName string    `json:"deviceName"`
	Stream     string    `json:"stream"`
	Content    string    `json:"content"`
}

// Device info for UI
type DeviceInfo struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	TenantID   string `json:"tenantId"`
	TenantName string `json:"tenantName"`
	Status     string `json:"status"` // "active" or "inactive"
}

var (
	config       Config
	messageStore = MessageStore{Messages: make([]Message, 0)}
	deviceStore  = make(map[string]DeviceInfo)
	deviceMutex  sync.RWMutex
	clients      = make(map[*websocket.Conn]bool)
	clientsMutex sync.Mutex
	app          *sdk.App
	tenantManager *TenantManager
	upgrader     = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for demo purposes
		},
	}
)

func loadConfig(file string) (*Config, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	c := Config{
		Tenants: make(map[string]TenantConfig),
	}
	err = yaml.Unmarshal(data, &c)
	if err != nil {
		return nil, err
	}
	
	// Handle backward compatibility - if there's a single tenant in the old format
	if len(c.Tenants) == 0 && c.Tenant.ID != "" {
		c.Tenants[c.Tenant.ID] = c.Tenant
		c.DefaultTenantId = c.Tenant.ID
	}
	
	return &c, nil
}

func storeConfig(file string, c *Config) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(file, data, 0644)
}

// SDK Message handler
func messageHandler(id string, d *sdk.Device, stream string, p []byte) {
	tenantID := d.Tenant().ID()

	msg := Message{
		Time:       time.Now(),
		TenantID:   tenantID,
		TenantName: d.Tenant().Name(),
		DeviceName: d.Name(),
		Stream:     stream,
		Content:    string(p),
	}

	messageStore.Lock()
	messageStore.Messages = append(messageStore.Messages, msg)
	if len(messageStore.Messages) > 200 { // Increased for multi-tenant support
		messageStore.Messages = messageStore.Messages[len(messageStore.Messages)-200:]
	}
	messageStore.Unlock()

	// Broadcast to all WebSocket clients
	broadcastToClients(msg)

	log.Printf("[Tenant: %s] Message received: %s, %s\n", d.Tenant().Name(), d.Name(), stream)
}

// SDK Activation handler
func activationHandler(d *sdk.Device) {
	tenantID := d.Tenant().ID()

	// Add to tenant manager
	tenantManager.AddDeviceToTenant(d)

	// Also maintain global device store for backward compatibility
	deviceMutex.Lock()
	deviceStore[d.ID()] = DeviceInfo{
		ID:         d.ID(),
		Name:       d.Name(),
		TenantID:   tenantID,
		TenantName: d.Tenant().Name(),
		Status:     "active",
	}
	deviceMutex.Unlock()

	// Store the device reference for security groups API
	storeActiveDevice(d)

	broadcastDeviceUpdate()
	log.Printf("[Tenant: %s] Device activated: %s (%s)\n", d.Tenant().Name(), d.Name(), d.ID())
}

// SDK Deactivation handler
func deactivationHandler(d *sdk.Device) {
	// Update in tenant manager
	tenantManager.RemoveDeviceFromTenant(d)

	// Update global device store for backward compatibility
	deviceMutex.Lock()
	if info, ok := deviceStore[d.ID()]; ok {
		info.Status = "inactive"
		deviceStore[d.ID()] = info
	}
	deviceMutex.Unlock()

	// Remove the device reference for security groups API
	removeInactiveDevice(d)

	broadcastDeviceUpdate()
	log.Printf("[Tenant: %s] Device deactivated: %s (%s)\n", d.Tenant().Name(), d.Name(), d.ID())
}

// SDK Tenant unlinked handler
func tenantUnlinkedHandler(t *sdk.Tenant) {
	log.Printf("Tenant unlinked: %s (%s)\n", t.Name(), t.ID())

	// Remove tenant from manager (this might be redundant if UnlinkTenant was used)
	tenantManager.UnlinkTenant(t.ID())

	// Update global devices for backward compatibility
	deviceMutex.Lock()
	for id, device := range deviceStore {
		if device.TenantID == t.ID() {
			device.Status = "inactive"
			deviceStore[id] = device
		}
	}
	deviceMutex.Unlock()

	broadcastDeviceUpdate()
	
	// Also broadcast tenant update
	broadcastTenantUpdate()
}

// Broadcast message to all WebSocket clients
func broadcastToClients(msg Message) {
	msgJSON, err := json.Marshal(map[string]interface{}{
		"type":    "message",
		"message": msg,
	})
	if err != nil {
		log.Printf("Error marshalling message: %v", err)
		return
	}

	clientsMutex.Lock()
	defer clientsMutex.Unlock()
	for client := range clients {
		err := client.WriteMessage(websocket.TextMessage, msgJSON)
		if err != nil {
			log.Printf("Error sending to client: %v", err)
			client.Close()
			delete(clients, client)
		}
	}
}

// Broadcast device updates to all WebSocket clients
func broadcastDeviceUpdate() {
	devices := tenantManager.GetAllDevices()

	msgJSON, err := json.Marshal(map[string]interface{}{
		"type":    "devices",
		"devices": devices,
	})
	if err != nil {
		log.Printf("Error marshalling devices: %v", err)
		return
	}

	clientsMutex.Lock()
	defer clientsMutex.Unlock()
	for client := range clients {
		err := client.WriteMessage(websocket.TextMessage, msgJSON)
		if err != nil {
			log.Printf("Error sending to client: %v", err)
			client.Close()
			delete(clients, client)
		}
	}
}

// Broadcast tenant list updates when tenants change
func broadcastTenantUpdate() {
	tenants := tenantManager.GetAllTenants()

	msgJSON, err := json.Marshal(map[string]interface{}{
		"type":    "tenants",
		"tenants": tenants,
	})
	if err != nil {
		log.Printf("Error marshalling tenants: %v", err)
		return
	}

	clientsMutex.Lock()
	defer clientsMutex.Unlock()
	for client := range clients {
		err := client.WriteMessage(websocket.TextMessage, msgJSON)
		if err != nil {
			log.Printf("Error sending to client: %v", err)
			client.Close()
			delete(clients, client)
		}
	}
}

func main() {
	configFile := "/app/config/config.yaml"
	if os.Getenv("CONFIG_FILE") != "" {
		configFile = os.Getenv("CONFIG_FILE")
	}

	var err error
	cfg, err := loadConfig(configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	config = *cfg

	// Setup SDK app
	getCredentials := func() (*sdk.Credentials, error) {
		return &sdk.Credentials{
			ApiKey: []byte(config.App.ApiKey),
		}, nil
	}

	appConfig := sdk.Config{
		ID:                        config.App.Id,
		GetCredentials:            getCredentials,
		GlobalFQDN:                config.App.GlobalFQDN,
		RegionalFQDN:              config.App.RegionalFQDN,
		RegionalFQDNs:             config.App.RegionalFQDNs,
		DeviceActivationHandler:   activationHandler,
		DeviceDeactivationHandler: deactivationHandler,
		TenantUnlinkedHandler:     tenantUnlinkedHandler,
		DeviceMessageHandler:      messageHandler,
		ReadStreamID:              config.App.ReadStream,
		WriteStreamID:             config.App.WriteStream,
		GroupID:                   config.App.GroupId,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false, // Set to true only in dev/testing
			},
			Proxy: http.ProxyFromEnvironment,
		},
	}

	// Create SDK app
	app, err = sdk.New(appConfig)
	if err != nil {
		log.Fatalf("Failed to create app: %v", err)
	}
	defer app.Close()

	// Create tenant manager
	tenantManager = NewTenantManager(app, configFile)
	
	// Load tenants from config
	err = tenantManager.LoadTenantsFromConfig(&config)
	if err != nil {
		log.Printf("Warning: Failed to load tenants from config: %v", err)
	}

	// Set up Gin HTTP server
	r := gin.Default()
	r.Use(cors.Default())

	// Serve HTML files for each route
	r.StaticFile("/", "/app/frontend/index.html")
	r.StaticFile("/index.html", "/app/frontend/index.html")
	r.StaticFile("/settings.html", "/app/frontend/settings.html")
	r.StaticFile("/dashboard.html", "/app/frontend/dashboard.html")
	// Serve CSS and JS files (only enable this if we're not using r.Static below)
	r.StaticFile("/css/bootstrap.min.css", "/app/frontend/css/bootstrap.min.css")
	r.StaticFile("/css/style.css", "/app/frontend/css/style.css")
	r.StaticFile("/js/bootstrap.bundle.min.js", "/app/frontend/js/bootstrap.bundle.min.js")

	// API routes
	api := r.Group("/api")
	{
		// Register settings API endpoints
		registerSettingsAPI(api)
		
		// Register tenant-aware security groups API endpoints
		sgAPI := NewSecurityGroupsAPI(tenantManager)
		sgAPI.RegisterAPI(api)

		api.GET("/status", func(c *gin.Context) {
			tenants := tenantManager.GetAllTenants()
		
			c.JSON(http.StatusOK, gin.H{
				"status":  "running",
				"app":     config.App.Id,
				"tenants": len(tenants),
				"devices": len(tenantManager.GetAllDevices()),
			})
		})

		// Get all tenants
		api.GET("/tenants", func(c *gin.Context) {
			tenants := tenantManager.GetAllTenants()
			c.JSON(http.StatusOK, tenants)
		})

		// Get specific tenant
		api.GET("/tenants/:id", func(c *gin.Context) {
			tenantID := c.Param("id")
			
			tenantCtx, exists := tenantManager.GetTenant(tenantID)
			if !exists {
				c.JSON(http.StatusNotFound, gin.H{"error": "Tenant not found"})
				return
			}
			
			c.JSON(http.StatusOK, gin.H{
				"id":     tenantCtx.Config.ID,
				"name":   tenantCtx.Config.Name,
			})
		})

		// Get devices for specific tenant
		api.GET("/tenants/:id/devices", func(c *gin.Context) {
			tenantID := c.Param("id")
			
			devices, err := tenantManager.GetTenantDevices(tenantID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			
			c.JSON(http.StatusOK, devices)
		})

		// Set default tenant
		api.POST("/tenants/:id/setDefault", func(c *gin.Context) {
			tenantID := c.Param("id")
			
			err := tenantManager.SetDefaultTenant(tenantID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			
			c.JSON(http.StatusOK, gin.H{
				"message": fmt.Sprintf("Tenant %s set as default", tenantID),
			})
			
			// Broadcast tenant update
			broadcastTenantUpdate()
		})

		api.GET("/messages", func(c *gin.Context) {
			// Check for tenant filter
			tenantID := c.Query("tenant_id")
			
			messageStore.RLock()
			msgs := messageStore.Messages
			messageStore.RUnlock()
			
			// Filter by tenant if requested
			if tenantID != "" {
				filteredMsgs := []Message{}
				for _, msg := range msgs {
					if msg.TenantID == tenantID {
						filteredMsgs = append(filteredMsgs, msg)
					}
				}
				c.JSON(http.StatusOK, filteredMsgs)
			} else {
				c.JSON(http.StatusOK, msgs)
			}
		})

		api.GET("/devices", func(c *gin.Context) {
			// Check for tenant filter
			tenantID := c.Query("tenant_id")
			
			var devices []DeviceInfo
			if tenantID != "" {
				// Get devices for specific tenant
				devices, _ = tenantManager.GetTenantDevices(tenantID)
			} else {
				// Get all devices
				devices = tenantManager.GetAllDevices()
			}

			c.JSON(http.StatusOK, devices)
		})

		api.POST("/tenant/link", func(c *gin.Context) {
			var request struct {
				OTP string `json:"otp" binding:"required"`
				SetAsDefault bool `json:"setAsDefault"`
			}

			if err := c.ShouldBindJSON(&request); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			tenant, err := tenantManager.LinkTenant(request.OTP)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to link tenant: %v", err)})
				return
			}

			// Set as default if requested
			if request.SetAsDefault {
				tenantManager.SetDefaultTenant(tenant.ID())
			}
			
			// Broadcast tenant update
			broadcastTenantUpdate()

			c.JSON(http.StatusOK, gin.H{
				"id":   tenant.ID(),
				"name": tenant.Name(),
			})
		})

		// Unlink tenant (multi-tenant support)
		api.POST("/tenants/:id/unlink", func(c *gin.Context) {
			tenantID := c.Param("id")
			
			// Check if tenant exists
			tenantCtx, exists := tenantManager.GetTenant(tenantID)
			if !exists {
				c.JSON(http.StatusNotFound, gin.H{"error": "Tenant not found"})
				return
			}

			tenantName := tenantCtx.Config.Name
			err := tenantManager.UnlinkTenant(tenantID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to unlink tenant: %v", err)})
				return
			}

			// Update config
			config.Tenant.ID = ""
			config.Tenant.Name = ""
			config.Tenant.Token = ""
			storeConfig(configFile, &config)

			c.JSON(http.StatusOK, gin.H{
				"message": fmt.Sprintf("Tenant %s successfully unlinked", tenantName),
			})
		})
	}

	// WebSocket endpoint
	r.GET("/ws", func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("Failed to set websocket upgrade: %+v", err)
			return
		}

		// Register client
		clientsMutex.Lock()
		clients[conn] = true
		clientsMutex.Unlock()

		// Send initial data to client
		messageStore.RLock()
		msgs := messageStore.Messages
		messageStore.RUnlock()

		msgJSON, _ := json.Marshal(map[string]interface{}{
			"type":     "messages",
			"messages": msgs,
		})
		conn.WriteMessage(websocket.TextMessage, msgJSON)

		deviceMutex.RLock()
		devices := make([]DeviceInfo, 0, len(deviceStore))
		for _, device := range deviceStore {
			devices = append(devices, device)
		}
		deviceMutex.RUnlock()

		devJSON, _ := json.Marshal(map[string]interface{}{
			"type":    "devices",
			"devices": devices,
		})
		conn.WriteMessage(websocket.TextMessage, devJSON)

		// Handle disconnects
		go func() {
			for {
				_, _, err := conn.ReadMessage()
				if err != nil {
					clientsMutex.Lock()
					delete(clients, conn)
					clientsMutex.Unlock()
					conn.Close()
					break
				}
			}
		}()
	})

	// Serve additional static files from the frontend directory
	// Instead of using r.Static("/static", "/app/frontend") which causes conflicts,
	// serve individual files as needed
	r.StaticFile("/app.js", "/app/frontend/app.js")
	r.StaticFile("/styles.css", "/app/frontend/styles.css")
	
	// Add mappings for the JS files used by the app
	r.StaticFile("/js/utils.js", "/app/frontend/js/utils.js")
	r.StaticFile("/js/error-handler.js", "/app/frontend/js/error-handler.js")
	r.StaticFile("/js/tenant-manager.js", "/app/frontend/js/tenant-manager.js")
	
	// API proxy was causing route conflicts with specific API endpoints
	// The proxy code has been removed since we're already registering individual API handlers

	// Start HTTP server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Error handling goroutine for SDK app
	go func() {
		for err := range app.Error {
			log.Printf("SDK App error: %v", err)
		}
	}()

	// Start HTTP server
	log.Printf("Starting server on port %s...", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
