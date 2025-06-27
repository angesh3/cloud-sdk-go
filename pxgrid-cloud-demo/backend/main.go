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
	App    AppConfig    `yaml:"app"`
	Tenant TenantConfig `yaml:"tenant"`
}

// Message store
type MessageStore struct {
	sync.RWMutex
	Messages []Message
}

// Message format for UI
type Message struct {
	Time       time.Time `json:"time"`
	TenantName string    `json:"tenantName"`
	DeviceName string    `json:"deviceName"`
	Stream     string    `json:"stream"`
	Content    string    `json:"content"`
}

// Device info for UI
type DeviceInfo struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
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
	tenant       *sdk.Tenant
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

	c := Config{}
	err = yaml.Unmarshal(data, &c)
	if err != nil {
		return nil, err
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
	msg := Message{
		Time:       time.Now(),
		TenantName: d.Tenant().Name(),
		DeviceName: d.Name(),
		Stream:     stream,
		Content:    string(p),
	}

	messageStore.Lock()
	messageStore.Messages = append(messageStore.Messages, msg)
	if len(messageStore.Messages) > 100 {
		messageStore.Messages = messageStore.Messages[1:]
	}
	messageStore.Unlock()

	// Broadcast to all WebSocket clients
	broadcastToClients(msg)

	log.Printf("Message received: %s, %s, %s\n", d.Tenant().Name(), d.Name(), stream)
}

// SDK Activation handler
func activationHandler(d *sdk.Device) {
	deviceMutex.Lock()
	deviceStore[d.ID()] = DeviceInfo{
		ID:         d.ID(),
		Name:       d.Name(),
		TenantName: d.Tenant().Name(),
		Status:     "active",
	}
	deviceMutex.Unlock()

	// Store the device reference for security groups API
	storeActiveDevice(d)

	broadcastDeviceUpdate()
	log.Printf("Device activated: %s (%s)\n", d.Name(), d.ID())
}

// SDK Deactivation handler
func deactivationHandler(d *sdk.Device) {
	deviceMutex.Lock()
	if info, ok := deviceStore[d.ID()]; ok {
		info.Status = "inactive"
		deviceStore[d.ID()] = info
	}
	deviceMutex.Unlock()

	// Remove the device reference for security groups API
	removeInactiveDevice(d)

	broadcastDeviceUpdate()
	log.Printf("Device deactivated: %s (%s)\n", d.Name(), d.ID())
}

// SDK Tenant unlinked handler
func tenantUnlinkedHandler(t *sdk.Tenant) {
	log.Printf("Tenant unlinked: %s (%s)\n", t.Name(), t.ID())

	// Update devices for this tenant
	deviceMutex.Lock()
	for id, device := range deviceStore {
		if device.TenantName == t.Name() {
			device.Status = "inactive"
			deviceStore[id] = device
		}
	}
	deviceMutex.Unlock()

	broadcastDeviceUpdate()
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
	deviceMutex.RLock()
	devices := make([]DeviceInfo, 0, len(deviceStore))
	for _, device := range deviceStore {
		devices = append(devices, device)
	}
	deviceMutex.RUnlock()

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

	// Handle tenant configuration
	tc := &config.Tenant
	if tc.Otp != "" {
		tenant, err = app.LinkTenant(tc.Otp)
		if err != nil {
			log.Printf("Failed to link tenant with OTP: %v", err)
		} else {
			// Update config with new tenant info
			tc.Otp = ""
			tc.ID = tenant.ID()
			tc.Name = tenant.Name()
			tc.Token = tenant.ApiToken()
			storeConfig(configFile, &config)
			log.Printf("Linked with tenant: %s", tenant.Name())
		}
	} else if tc.ID != "" && tc.Name != "" && tc.Token != "" {
		tenant, err = app.SetTenant(tc.ID, tc.Name, tc.Token)
		if err != nil {
			log.Printf("Failed to set tenant to app: %v", err)
		} else {
			log.Printf("Linked with existing tenant: %s", tenant.Name())
		}
	} else {
		log.Println("No tenant configured. Use the API to link a tenant with OTP.")
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
		
		// Register security groups API endpoints
		registerSecurityGroupsAPI(api)

		api.GET("/status", func(c *gin.Context) {
			var tenantName string
			if tenant != nil {
				tenantName = tenant.Name()
			}

			c.JSON(http.StatusOK, gin.H{
				"status":  "running",
				"app":     config.App.Id,
				"tenant":  tenantName,
				"devices": len(deviceStore),
			})
		})

		api.GET("/messages", func(c *gin.Context) {
			messageStore.RLock()
			msgs := messageStore.Messages
			messageStore.RUnlock()

			c.JSON(http.StatusOK, msgs)
		})

		api.GET("/devices", func(c *gin.Context) {
			deviceMutex.RLock()
			devices := make([]DeviceInfo, 0, len(deviceStore))
			for _, device := range deviceStore {
				devices = append(devices, device)
			}
			deviceMutex.RUnlock()

			c.JSON(http.StatusOK, devices)
		})

		api.POST("/tenant/link", func(c *gin.Context) {
			var request struct {
				OTP string `json:"otp" binding:"required"`
			}

			if err := c.ShouldBindJSON(&request); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			tenant, err = app.LinkTenant(request.OTP)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to link tenant: %v", err)})
				return
			}

			// Update config
			config.Tenant.Otp = ""
			config.Tenant.ID = tenant.ID()
			config.Tenant.Name = tenant.Name()
			config.Tenant.Token = tenant.ApiToken()
			storeConfig(configFile, &config)

			c.JSON(http.StatusOK, gin.H{
				"id":   tenant.ID(),
				"name": tenant.Name(),
			})
		})

		api.POST("/tenant/unlink", func(c *gin.Context) {
			if tenant == nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "No tenant linked"})
				return
			}

			tenantName := tenant.Name()
			err = app.UnlinkTenant(tenant)
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
