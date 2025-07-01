package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Authentication handlers
// These functions handle authentication-related API endpoints

// LoginRequest represents a login request
type LoginRequest struct {
	UsernameOrEmail string `json:"usernameOrEmail"`
	Password        string `json:"password"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// SignupRequest represents a signup request
type SignupRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// loginHandler handles user login
func loginHandler(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fmt.Printf("[Debug] Login request bind error: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}
	
	// Debug login attempt
	fmt.Printf("[Debug] Login attempt for user: %s with password length: %d\n", req.UsernameOrEmail, len(req.Password))
	
	// Validate input
	if req.UsernameOrEmail == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Username/email and password are required",
		})
		return
	}
	
	// For testing/debugging - allow direct admin login
	if req.UsernameOrEmail == "admin" && req.Password == "admin123" {
		fmt.Println("[Debug] Using special admin account override")
		// Create a session directly for admin
		adminUser, err := userManager.GetUserByUsername("admin")
		if err != nil || adminUser == nil {
			// Create admin user if not found
			fmt.Println("[Debug] Admin user not found, creating it")
			adminUser = &User{
				ID:        "admin-1",
				Username:  "admin",
				Email:     "admin@example.com",
				Role:      "admin",
				CreatedAt: time.Now(),
			}
			userManager.users[adminUser.ID] = adminUser
			userManager.SaveUsers()
		}
		
		session := &Session{
			Token:      generateSessionToken(),
			UserID:     adminUser.ID,
			Expiration: time.Now().Add(24 * time.Hour),
		}
		userManager.sessions[session.Token] = session
		userManager.SaveUsers()
		
		c.JSON(http.StatusOK, gin.H{
			"token": session.Token,
			"user": adminUser,
		})
		return
	}
	
	// Regular authentication
	fmt.Println("[Debug] Attempting regular authentication flow")
	session, err := userManager.Authenticate(req.UsernameOrEmail, req.Password)
	if err != nil {
		fmt.Printf("[Debug] Authentication failed: %v\n", err)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid credentials",
		})
		return
	}
	
	// Get user details
	user, err := userManager.GetUserByID(session.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get user details",
		})
		return
	}
	
	// Create a copy without password hash
	userCopy := *user
	userCopy.PasswordHash = ""
	
	c.JSON(http.StatusOK, LoginResponse{
		Token: session.Token,
		User:  userCopy,
	})
}

// signupHandler handles user signup
func signupHandler(c *gin.Context) {
	var req SignupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}
	
	// Validate input
	if req.Username == "" || req.Email == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Username, email, and password are required",
		})
		return
	}
	
	// Create user with regular user role
	user, err := userManager.CreateUser(req.Username, req.Email, req.Password, "user")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}
	
	// Authenticate the new user
	session, err := userManager.Authenticate(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "User created but failed to authenticate",
		})
		return
	}
	
	// Create a copy without password hash
	userCopy := *user
	userCopy.PasswordHash = ""
	
	c.JSON(http.StatusCreated, LoginResponse{
		Token: session.Token,
		User:  userCopy,
	})
}

// logoutHandler handles user logout
func logoutHandler(c *gin.Context) {
	// Get session token from context (set by authMiddleware)
	token := c.GetString("sessionToken")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Not authenticated",
		})
		return
	}
	
	// Invalidate session
	err := userManager.InvalidateSession(token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to logout",
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Logged out successfully",
	})
}

// getCurrentUserHandler gets the current user's information
func getCurrentUserHandler(c *gin.Context) {
	// Get user from context (set by authMiddleware)
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Not authenticated",
		})
		return
	}
	
	user := userInterface.(*User)
	
	// Create a copy without password hash
	userCopy := *user
	userCopy.PasswordHash = ""
	
	c.JSON(http.StatusOK, userCopy)
}

// Authentication Middleware

// authMiddleware authenticates requests using the session token
func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// FOR API ROUTES ONLY - we rely on client-side auth for direct page access
		// Admin HTML pages like /admin/dashboard should NOT use this middleware
		// They should check authentication client-side via JavaScript
		
		// Get Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// Missing auth header for API requests - return 401
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header is required",
			})
			return
		}
		
		// Extract token from header
		// Format should be "Bearer {token}"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authorization format, should be 'Bearer {token}'",
			})
			return
		}
		
		token := parts[1]
		
		// Get user by session token
		user, err := userManager.GetUserBySessionToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired session",
			})
			return
		}
		
		// Store user and token in context for later use
		c.Set("user", user)
		c.Set("sessionToken", token)
		
		c.Next()
	}
}

// adminMiddleware checks if the authenticated user is an admin
func adminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user from context (set by authMiddleware)
		userInterface, exists := c.Get("user")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Not authenticated",
			})
			return
		}
		
		user := userInterface.(*User)
		
		// Check if user is an admin
		if user.Role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "Admin access required",
			})
			return
		}
		
		c.Next()
	}
}
