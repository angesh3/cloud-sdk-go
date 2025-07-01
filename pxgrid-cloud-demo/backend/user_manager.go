package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"time"
	
	"golang.org/x/crypto/bcrypt"
)

// User represents a user in the system
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"passwordHash"` // Store in JSON but don't expose in API responses
	Role         string    `json:"role"` // "admin" or "user"
	CreatedAt    time.Time `json:"createdAt"`
	LastLogin    time.Time `json:"lastLogin"`
	TenantAccess []string  `json:"tenantAccess"` // List of tenant IDs this user can access
}

// Session represents a user session
type Session struct {
	Token      string    `json:"token"`
	UserID     string    `json:"userId"`
	Expiration time.Time `json:"expiration"`
}

// UserManager handles user-related operations
type UserManager struct {
	users    map[string]*User      // Map of user ID to user
	usersDB  string                // Path to users database file
	sessions map[string]*Session   // Map of session tokens to sessions
	mutex    sync.RWMutex
}

// UsersData is the structure stored in the users database file
type UsersData struct {
	Users    []*User    `json:"users"`
	Sessions []*Session `json:"sessions"`
}

// NewUserManager creates a new user manager instance
func NewUserManager(usersDBPath string) (*UserManager, error) {
	um := &UserManager{
		users:    make(map[string]*User),
		sessions: make(map[string]*Session),
		usersDB:  usersDBPath,
	}
	
	// Create admin user if no users exist
	err := um.LoadUsers()
	if err != nil {
		if os.IsNotExist(err) {
			// Create initial admin user
			adminUser := &User{
				ID:        generateID(),
				Username:  "admin",
				Email:     "admin@example.com",
				Role:      "admin",
				CreatedAt: time.Now(),
			}
			
			// Default admin password is "admin123" - should be changed on first login
			passwordHash, err := hashPassword("admin123")
			if err != nil {
				return nil, fmt.Errorf("failed to hash admin password: %v", err)
			}
			adminUser.PasswordHash = passwordHash
			
			um.users[adminUser.ID] = adminUser
			err = um.SaveUsers()
			if err != nil {
				return nil, fmt.Errorf("failed to save users: %v", err)
			}
		} else {
			return nil, fmt.Errorf("failed to load users: %v", err)
		}
	}
	
	return um, nil
}

// LoadUsers loads users from the database file
func (um *UserManager) LoadUsers() error {
	data, err := ioutil.ReadFile(um.usersDB)
	if err != nil {
		return err
	}
	
	var usersData UsersData
	err = json.Unmarshal(data, &usersData)
	if err != nil {
		return fmt.Errorf("failed to parse users data: %v", err)
	}
	
	um.mutex.Lock()
	defer um.mutex.Unlock()
	
	// Load users
	um.users = make(map[string]*User)
	for _, user := range usersData.Users {
		um.users[user.ID] = user
	}
	
	// Load sessions and filter out expired ones
	um.sessions = make(map[string]*Session)
	now := time.Now()
	for _, session := range usersData.Sessions {
		if session.Expiration.After(now) {
			um.sessions[session.Token] = session
		}
	}
	
	return nil
}

// SaveUsers saves users to the database file
func (um *UserManager) SaveUsers() error {
	um.mutex.RLock()
	defer um.mutex.RUnlock()
	
	var usersData UsersData
	
	// Collect users
	for _, user := range um.users {
		usersData.Users = append(usersData.Users, user)
	}
	
	// Collect non-expired sessions
	now := time.Now()
	for _, session := range um.sessions {
		if session.Expiration.After(now) {
			usersData.Sessions = append(usersData.Sessions, session)
		}
	}
	
	data, err := json.MarshalIndent(usersData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal users data: %v", err)
	}
	
	err = ioutil.WriteFile(um.usersDB, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write users data: %v", err)
	}
	
	return nil
}

// CreateUser creates a new user
func (um *UserManager) CreateUser(username, email, password, role string) (*User, error) {
	um.mutex.Lock()
	defer um.mutex.Unlock()
	
	// Check if username or email already exists
	for _, user := range um.users {
		if user.Username == username {
			return nil, fmt.Errorf("username already exists")
		}
		if user.Email == email {
			return nil, fmt.Errorf("email already exists")
		}
	}
	
	// Create new user
	user := &User{
		ID:        generateID(),
		Username:  username,
		Email:     email,
		Role:      role,
		CreatedAt: time.Now(),
	}
	
	// Hash password
	passwordHash, err := hashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %v", err)
	}
	user.PasswordHash = passwordHash
	
	// Add user to map
	um.users[user.ID] = user
	
	// Save changes
	err = um.SaveUsers()
	if err != nil {
		return nil, fmt.Errorf("failed to save user: %v", err)
	}
	
	return user, nil
}

// Authenticate authenticates a user with username/email and password
func (um *UserManager) Authenticate(usernameOrEmail, password string) (*Session, error) {
	um.mutex.RLock()
	defer um.mutex.RUnlock()
	
	// Find user by username or email
	var user *User
	for _, u := range um.users {
		if u.Username == usernameOrEmail || u.Email == usernameOrEmail {
			user = u
			break
		}
	}
	
	if user == nil {
		return nil, fmt.Errorf("invalid credentials")
	}
	
	// Verify password
	err := verifyPassword(password, user.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}
	
	// Update last login time
	user.LastLogin = time.Now()
	
	// Create session
	session := &Session{
		Token:      generateSessionToken(),
		UserID:     user.ID,
		Expiration: time.Now().Add(24 * time.Hour), // 24 hour session
	}
	
	um.sessions[session.Token] = session
	
	// Save changes
	err = um.SaveUsers()
	if err != nil {
		return nil, fmt.Errorf("failed to save session: %v", err)
	}
	
	return session, nil
}

// GetUserBySessionToken gets a user by session token
func (um *UserManager) GetUserBySessionToken(token string) (*User, error) {
	um.mutex.RLock()
	defer um.mutex.RUnlock()
	
	session, exists := um.sessions[token]
	if !exists {
		return nil, fmt.Errorf("invalid session")
	}
	
	// Check if session is expired
	if session.Expiration.Before(time.Now()) {
		delete(um.sessions, token)
		return nil, fmt.Errorf("session expired")
	}
	
	user, exists := um.users[session.UserID]
	if !exists {
		delete(um.sessions, token)
		return nil, fmt.Errorf("user not found")
	}
	
	return user, nil
}

// GetUserByID gets a user by ID
func (um *UserManager) GetUserByID(id string) (*User, error) {
	um.mutex.RLock()
	defer um.mutex.RUnlock()
	
	user, exists := um.users[id]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}
	
	return user, nil
}

// GetUserByUsername gets a user by username
func (um *UserManager) GetUserByUsername(username string) (*User, error) {
	um.mutex.RLock()
	defer um.mutex.RUnlock()
	
	// Search for user with matching username
	for _, user := range um.users {
		if user.Username == username {
			return user, nil
		}
	}
	
	return nil, fmt.Errorf("user not found")
}

// UpdateUser updates a user's information
func (um *UserManager) UpdateUser(id string, updates map[string]interface{}) (*User, error) {
	um.mutex.Lock()
	defer um.mutex.Unlock()
	
	user, exists := um.users[id]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}
	
	// Apply updates
	if username, ok := updates["username"].(string); ok {
		// Check if username is already taken by another user
		for uid, u := range um.users {
			if u.Username == username && uid != id {
				return nil, fmt.Errorf("username already exists")
			}
		}
		user.Username = username
	}
	
	if email, ok := updates["email"].(string); ok {
		// Check if email is already taken by another user
		for uid, u := range um.users {
			if u.Email == email && uid != id {
				return nil, fmt.Errorf("email already exists")
			}
		}
		user.Email = email
	}
	
	if password, ok := updates["password"].(string); ok {
		passwordHash, err := hashPassword(password)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %v", err)
		}
		user.PasswordHash = passwordHash
	}
	
	if role, ok := updates["role"].(string); ok {
		user.Role = role
	}
	
	if tenantAccess, ok := updates["tenantAccess"].([]string); ok {
		user.TenantAccess = tenantAccess
	}
	
	// Save changes
	err := um.SaveUsers()
	if err != nil {
		return nil, fmt.Errorf("failed to save user updates: %v", err)
	}
	
	return user, nil
}

// DeleteUser deletes a user
func (um *UserManager) DeleteUser(id string) error {
	um.mutex.Lock()
	defer um.mutex.Unlock()
	
	_, exists := um.users[id]
	if !exists {
		return fmt.Errorf("user not found")
	}
	
	// Delete user
	delete(um.users, id)
	
	// Delete all sessions for this user
	for token, session := range um.sessions {
		if session.UserID == id {
			delete(um.sessions, token)
		}
	}
	
	// Save changes
	err := um.SaveUsers()
	if err != nil {
		return fmt.Errorf("failed to save after user deletion: %v", err)
	}
	
	return nil
}

// ListUsers lists all users
func (um *UserManager) ListUsers() []*User {
	um.mutex.RLock()
	defer um.mutex.RUnlock()
	
	var users []*User
	for _, user := range um.users {
		// Create a copy without password hash
		userCopy := *user
		userCopy.PasswordHash = ""
		users = append(users, &userCopy)
	}
	
	return users
}

// AssignTenantToUser assigns a tenant to a user
func (um *UserManager) AssignTenantToUser(userID, tenantID string) error {
	um.mutex.Lock()
	defer um.mutex.Unlock()
	
	user, exists := um.users[userID]
	if !exists {
		return fmt.Errorf("user not found")
	}
	
	// Check if tenant is already assigned
	for _, tid := range user.TenantAccess {
		if tid == tenantID {
			return nil // Already assigned
		}
	}
	
	// Add tenant to user's access list
	user.TenantAccess = append(user.TenantAccess, tenantID)
	
	// Save changes
	err := um.SaveUsers()
	if err != nil {
		return fmt.Errorf("failed to save tenant assignment: %v", err)
	}
	
	return nil
}

// RemoveTenantFromUser removes a tenant from a user
func (um *UserManager) RemoveTenantFromUser(userID, tenantID string) error {
	um.mutex.Lock()
	defer um.mutex.Unlock()
	
	user, exists := um.users[userID]
	if !exists {
		return fmt.Errorf("user not found")
	}
	
	// Find and remove tenant from user's access list
	found := false
	var newTenantAccess []string
	for _, tid := range user.TenantAccess {
		if tid != tenantID {
			newTenantAccess = append(newTenantAccess, tid)
		} else {
			found = true
		}
	}
	
	if !found {
		return fmt.Errorf("tenant not assigned to user")
	}
	
	user.TenantAccess = newTenantAccess
	
	// Save changes
	err := um.SaveUsers()
	if err != nil {
		return fmt.Errorf("failed to save tenant removal: %v", err)
	}
	
	return nil
}

// GetUserTenants gets all tenants assigned to a user
func (um *UserManager) GetUserTenants(userID string) ([]string, error) {
	um.mutex.RLock()
	defer um.mutex.RUnlock()
	
	user, exists := um.users[userID]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}
	
	return user.TenantAccess, nil
}

// InvalidateSession invalidates a session (logout)
func (um *UserManager) InvalidateSession(token string) error {
	um.mutex.Lock()
	defer um.mutex.Unlock()
	
	_, exists := um.sessions[token]
	if !exists {
		return fmt.Errorf("session not found")
	}
	
	delete(um.sessions, token)
	
	// Save changes
	err := um.SaveUsers()
	if err != nil {
		return fmt.Errorf("failed to save session invalidation: %v", err)
	}
	
	return nil
}

// Utility functions

// generateID generates a unique ID
func generateID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("%x", b)
}

// generateSessionToken generates a random session token
func generateSessionToken() string {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return base64.StdEncoding.EncodeToString(b)
}

// hashPassword hashes a password using bcrypt
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// verifyPassword verifies a password against a hash
func verifyPassword(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
