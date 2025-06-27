package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"syscall"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v2"
)

// The full configuration structure from main.go
type SettingsConfig struct {
	App    AppConfig    `json:"app" yaml:"app"`
	Tenant TenantConfig `json:"tenant" yaml:"tenant"`
}

// Settings API request
type SettingsRequest struct {
	AppID      string `json:"appId"`
	APIKey     string `json:"apiKey"`
	ReadStream string `json:"readStream"`
	WriteStream string `json:"writeStream"`
	GroupID    string `json:"groupId"`
	// Tenant config is managed separately through tenant/link endpoint
}

// Register Settings APIs
func registerSettingsAPI(r *gin.RouterGroup) {
	r.GET("settings", getSettings)
	r.POST("settings", updateSettings)
	r.POST("restart", restartApp)
}

// Get the current settings
func getSettings(c *gin.Context) {
	configFile := os.Getenv("CONFIG_FILE")
	if configFile == "" {
		configFile = "/app/config/config.yaml"
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to read config: %v", err),
		})
		return
	}

	// Parse the YAML
	var settings SettingsConfig
	err = yaml.Unmarshal(data, &settings)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to parse config: %v", err),
		})
		return
	}

	// Mask the API key for security
	if len(settings.App.ApiKey) > 4 {
		maskedKey := settings.App.ApiKey[:2] + "****" + settings.App.ApiKey[len(settings.App.ApiKey)-2:]
		settings.App.ApiKey = maskedKey
	}

	c.JSON(http.StatusOK, settings)
}

// Update settings and save to config file
func updateSettings(c *gin.Context) {
	var request SettingsRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get current config
	configFile := os.Getenv("CONFIG_FILE")
	if configFile == "" {
		configFile = "/app/config/config.yaml"
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to read config: %v", err),
		})
		return
	}

	// Parse the YAML
	var settings SettingsConfig
	err = yaml.Unmarshal(data, &settings)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to parse config: %v", err),
		})
		return
	}

	// Update the settings
	if request.AppID != "" {
		settings.App.Id = request.AppID
	}
	if request.APIKey != "" {
		settings.App.ApiKey = request.APIKey
	}
	if request.ReadStream != "" {
		settings.App.ReadStream = request.ReadStream
	}
	if request.WriteStream != "" {
		settings.App.WriteStream = request.WriteStream
	}
	if request.GroupID != "" {
		settings.App.GroupId = request.GroupID
	}

	// Save back to the file
	updatedData, err := yaml.Marshal(settings)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to convert config: %v", err),
		})
		return
	}

	err = os.WriteFile(configFile, updatedData, 0644)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to save config: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Settings updated successfully. Restart the application for changes to take effect.",
	})
}

// Restart the application
func restartApp(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Application is restarting...",
	})

	go func() {
		// Small delay to allow the response to be sent
		execPath, _ := os.Executable()
		cmd := exec.Command(execPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		// Start new process
		if err := cmd.Start(); err != nil {
			fmt.Printf("Failed to restart: %v\n", err)
			return
		}

		// Wait a bit to ensure new process starts
		// Then exit the current process
		syscall.Kill(os.Getpid(), syscall.SIGTERM)
	}()
}
