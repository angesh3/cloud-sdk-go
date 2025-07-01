package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// User represents a user in the system
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"passwordHash"` // Include for setup script
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"createdAt"`
	LastLogin    time.Time `json:"lastLogin"`
	TenantAccess []string  `json:"tenantAccess"`
}

// Session represents a user session
type Session struct {
	Token      string    `json:"token"`
	UserID     string    `json:"userId"`
	Expiration time.Time `json:"expiration"`
}

// UsersData is the structure stored in the users database file
type UsersData struct {
	Users    []*User    `json:"users"`
	Sessions []*Session `json:"sessions"`
}

func generateID() string {
	// Simple ID generation - not for production use
	return fmt.Sprintf("user-%d", time.Now().UnixNano())
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func main() {
	// Get path to config directory
	configDir := "../config"
	absConfigDir, err := filepath.Abs(configDir)
	if err != nil {
		log.Fatalf("Failed to get absolute path: %v", err)
	}
	
	// Path to users.json file
	usersDBPath := filepath.Join(absConfigDir, "users.json")
	
	// Check if the file exists
	var usersData UsersData
	
	_, err = os.Stat(usersDBPath)
	if err == nil {
		// File exists, read it
		data, err := os.ReadFile(usersDBPath)
		if err != nil {
			log.Fatalf("Failed to read users file: %v", err)
		}
		
		err = json.Unmarshal(data, &usersData)
		if err != nil {
			log.Printf("Failed to parse users file: %v", err)
			// Continue with empty usersData
			usersData = UsersData{
				Users:    []*User{},
				Sessions: []*Session{},
			}
		}
	} else if os.IsNotExist(err) {
		// File doesn't exist, create new data structure
		usersData = UsersData{
			Users:    []*User{},
			Sessions: []*Session{},
		}
	} else {
		log.Fatalf("Failed to check users file: %v", err)
	}
	
	// Check if admin user already exists
	adminExists := false
	for _, user := range usersData.Users {
		if user.Username == "admin" {
			adminExists = true
			break
		}
	}
	
	if !adminExists {
		// Create admin user
		passwordHash, err := hashPassword("admin123")
		if err != nil {
			log.Fatalf("Failed to hash password: %v", err)
		}
		
		adminUser := &User{
			ID:           generateID(),
			Username:     "admin",
			Email:        "admin@example.com",
			PasswordHash: passwordHash,
			Role:         "admin",
			CreatedAt:    time.Now(),
			LastLogin:    time.Time{},
			TenantAccess: []string{},
		}
		
		usersData.Users = append(usersData.Users, adminUser)
		
		// Save updated users file
		data, err := json.MarshalIndent(usersData, "", "  ")
		if err != nil {
			log.Fatalf("Failed to marshal users data: %v", err)
		}
		
		err = os.WriteFile(usersDBPath, data, 0644)
		if err != nil {
			log.Fatalf("Failed to write users file: %v", err)
		}
		
		fmt.Println("Admin user created successfully!")
	} else {
		fmt.Println("Admin user already exists!")
	}
}
