package handlers

import (
	"api/database/models"
	"api/utils"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

// OAuthInitiateRequest represents the request to initiate OAuth flow
type OAuthInitiateRequest struct {
	Provider    string `json:"provider"`
	RedirectURL string `json:"redirect_url,omitempty"`
}

// OAuthCallbackQuery represents OAuth callback query parameters
type OAuthCallbackQuery struct {
	Code  string `query:"code"`
	State string `query:"state"`
	Error string `query:"error"`
}

// OAuthInitiate starts the OAuth flow for a given provider
func OAuthInitiate(c *fiber.Ctx) error {
	var req OAuthInitiateRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(400, "Invalid request body")
	}

	// Validate provider
	provider := models.OAuthProvider(strings.ToLower(req.Provider))
	if provider != models.OAuthProviderGoogle && provider != models.OAuthProviderGithub {
		return fiber.NewError(400, "Unsupported OAuth provider")
	}

	// Get OAuth config
	config, err := utils.GetOAuthConfig(provider)
	if err != nil {
		return fiber.NewError(500, "OAuth provider not configured")
	}

	// Generate secure state and nonce
	state, err := utils.GenerateOAuthState()
	if err != nil {
		return fiber.NewError(500, "Failed to generate OAuth state")
	}

	nonce, err := utils.GenerateNonce()
	if err != nil {
		return fiber.NewError(500, "Failed to generate OAuth nonce")
	}

	// Store OAuth state in database for validation
	oauthState := models.OAuthState{
		State:       state,
		Provider:    provider,
		Nonce:       nonce,
		RedirectURL: req.RedirectURL,
		UserAgent:   c.Get("User-Agent"),
		IPAddress:   c.IP(),
		ExpiresAt:   time.Now().Add(10 * time.Minute), // 10-minute expiry
	}

	if err := db.Create(&oauthState).Error; err != nil {
		return fiber.NewError(500, "Failed to store OAuth state")
	}

	// Generate authorization URL
	authURL := config.AuthCodeURL(state, oauth2.AccessTypeOffline)

	return c.JSON(utils.Response{
		Success: true,
		Code:    200,
		Message: "OAuth flow initiated",
		Data: fiber.Map{
			"auth_url": authURL,
			"state":    state,
		},
	})
}

// OAuthCallback handles OAuth provider callbacks
func OAuthCallback(c *fiber.Ctx) error {
	provider := models.OAuthProvider(c.Params("provider"))
	if provider != models.OAuthProviderGoogle && provider != models.OAuthProviderGithub {
		return fiber.NewError(400, "Invalid OAuth provider")
	}

	var query OAuthCallbackQuery
	if err := c.QueryParser(&query); err != nil {
		return fiber.NewError(400, "Invalid callback parameters")
	}

	// Check for OAuth errors
	if query.Error != "" {
		return fiber.NewError(400, fmt.Sprintf("OAuth error: %s", query.Error))
	}

	if query.Code == "" || query.State == "" {
		return fiber.NewError(400, "Missing OAuth code or state")
	}

	// Validate and retrieve OAuth state
	var oauthState models.OAuthState
	err := db.Where("state = ? AND provider = ? AND expires_at > ?",
		query.State, provider, time.Now()).First(&oauthState).Error
	if err != nil {
		return fiber.NewError(400, "Invalid or expired OAuth state")
	}

	// Additional state validation
	if err := utils.ValidateOAuthState(query.State, oauthState.Nonce,
		c.Get("User-Agent"), c.IP()); err != nil {
		return fiber.NewError(400, "OAuth state validation failed")
	}

	// Clean up used state
	db.Delete(&oauthState)

	// Exchange code for token
	config, err := utils.GetOAuthConfig(provider)
	if err != nil {
		return fiber.NewError(500, "OAuth provider not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	token, err := config.Exchange(ctx, query.Code)
	if err != nil {
		return fiber.NewError(400, "Failed to exchange OAuth code")
	}

	// Fetch user info from OAuth provider
	var userInfo OAuthUserInfo
	switch provider {
	case models.OAuthProviderGoogle:
		googleInfo, err := utils.FetchGoogleUserInfo(ctx, token)
		if err != nil {
			return fiber.NewError(400, fmt.Sprintf("Failed to fetch Google user info: %v", err))
		}
		userInfo = OAuthUserInfo{
			ID:        googleInfo.ID,
			Email:     googleInfo.Email,
			Name:      googleInfo.Name,
			AvatarURL: googleInfo.Picture,
		}
	case models.OAuthProviderGithub:
		githubInfo, err := utils.FetchGitHubUserInfo(ctx, token)
		if err != nil {
			return fiber.NewError(400, fmt.Sprintf("Failed to fetch GitHub user info: %v", err))
		}
		userInfo = OAuthUserInfo{
			ID:        fmt.Sprintf("%d", githubInfo.ID),
			Email:     githubInfo.Email,
			Name:      githubInfo.Name,
			AvatarURL: githubInfo.AvatarURL,
		}
	}

	// Process OAuth login/registration
	result, err := processOAuthLogin(provider, userInfo, token)
	if err != nil {
		return err
	}

	return c.JSON(result)
}

// OAuthUserInfo represents normalized user information from OAuth providers
type OAuthUserInfo struct {
	ID        string
	Email     string
	Name      string
	AvatarURL string
}

// OAuthLoginResult represents the result of OAuth login processing
type OAuthLoginResult struct {
	Success     bool   `json:"success"`
	Code        int    `json:"code"`
	Message     string `json:"message"`
	Action      string `json:"action"` // "login", "register", "link_required"
	Token       string `json:"token,omitempty"`
	RedirectURL string `json:"redirect_url,omitempty"`
}

// processOAuthLogin implements the enterprise OAuth flow logic
func processOAuthLogin(provider models.OAuthProvider, userInfo OAuthUserInfo, token *oauth2.Token) (*utils.Response, error) {
	// Start database transaction for consistency
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Check if OAuth account already exists
	var existingOAuth models.OAuthAccount
	err := tx.Where("provider = ? AND provider_id = ?", provider, userInfo.ID).First(&existingOAuth).Error

	if err == nil {
		// OAuth account exists - proceed with login
		return handleExistingOAuthLogin(tx, &existingOAuth, userInfo, token)
	}

	if err != gorm.ErrRecordNotFound {
		tx.Rollback()
		return nil, fiber.NewError(500, "Database error during OAuth lookup")
	}

	// OAuth account doesn't exist - check if email user exists
	var existingUser models.User
	err = tx.Where("email = ?", userInfo.Email).First(&existingUser).Error

	if err == gorm.ErrRecordNotFound {
		// No user with this email - create new OAuth user
		return handleNewOAuthUser(tx, provider, userInfo, token)
	}

	if err != nil {
		tx.Rollback()
		return nil, fiber.NewError(500, "Database error during user lookup")
	}

	// User exists with this email
	switch existingUser.AccountType {
	case models.AccountTypeEmail:
		// Email account exists - require explicit linking
		tx.Rollback()
		return &utils.Response{
			Success: false,
			Code:    409,
			Message: "Account with this email already exists. Please log in with your email and link your OAuth account in settings.",
			Data: fiber.Map{
				"action":           "link_required",
				"existing_account": "email",
				"provider":         string(provider),
				"email":            userInfo.Email,
			},
		}, nil

	case models.AccountTypeOAuth:
		// OAuth-only account exists - link new provider
		return handleOAuthAccountLinking(tx, &existingUser, provider, userInfo, token)

	case models.AccountTypeHybrid:
		// Hybrid account exists - link new provider
		return handleOAuthAccountLinking(tx, &existingUser, provider, userInfo, token)

	default:
		tx.Rollback()
		return nil, fiber.NewError(500, "Unknown account type")
	}
}

// handleExistingOAuthLogin processes login for existing OAuth accounts
func handleExistingOAuthLogin(tx *gorm.DB, oauthAccount *models.OAuthAccount, userInfo OAuthUserInfo, token *oauth2.Token) (*utils.Response, error) {
	// Load the associated user
	var user models.User
	if err := tx.First(&user, oauthAccount.UserID).Error; err != nil {
		tx.Rollback()
		return nil, fiber.NewError(500, "Failed to load user")
	}

	// Update OAuth account with latest info
	encryptedAccess, _ := utils.EncryptToken(token.AccessToken)
	encryptedRefresh, _ := utils.EncryptToken(token.RefreshToken)

	updates := map[string]interface{}{
		"email":         userInfo.Email,
		"name":          userInfo.Name,
		"avatar_url":    userInfo.AvatarURL,
		"access_token":  encryptedAccess,
		"refresh_token": encryptedRefresh,
		"token_expiry":  token.Expiry,
		"last_used_at":  time.Now(),
	}

	if err := tx.Model(oauthAccount).Updates(updates).Error; err != nil {
		tx.Rollback()
		return nil, fiber.NewError(500, "Failed to update OAuth account")
	}

	// Create JWT session
	jti, jwt, err := utils.GetSignedKey(user.ID)
	if err != nil {
		tx.Rollback()
		return nil, fiber.NewError(500, "Failed to generate JWT")
	}

	_, hashedToken := utils.GenerateRefreshToken()
	session := models.Session{
		JTI:          jti,
		UserID:       user.ID,
		RefreshToken: hashedToken,
		Revoked:      false,
		ExpiresAt:    time.Now().Add(30 * 24 * time.Hour),
	}

	if err := tx.Create(&session).Error; err != nil {
		tx.Rollback()
		return nil, fiber.NewError(500, "Failed to create session")
	}

	tx.Commit()

	return &utils.Response{
		Success: true,
		Code:    200,
		Message: fmt.Sprintf("Logged in successfully with %s", string(oauthAccount.Provider)),
		Data: fiber.Map{
			"action": "login",
			"token":  jwt,
			"user": fiber.Map{
				"id":           user.ID,
				"username":     user.Username,
				"email":        user.Email,
				"account_type": user.AccountType,
			},
		},
	}, nil
}

// handleNewOAuthUser creates a new OAuth-only user account
func handleNewOAuthUser(tx *gorm.DB, provider models.OAuthProvider, userInfo OAuthUserInfo, token *oauth2.Token) (*utils.Response, error) {
	// Generate username from email or name
	username := generateUsernameFromOAuth(userInfo)

	// Ensure username uniqueness
	username, err := ensureUniqueUsername(tx, username)
	if err != nil {
		tx.Rollback()
		return nil, fiber.NewError(500, "Failed to generate unique username")
	}

	// Create new OAuth user
	user := models.User{
		Username:    username,
		Email:       userInfo.Email,
		AccountType: models.AccountTypeOAuth,
		// Password is null for OAuth-only accounts
	}

	if err := tx.Create(&user).Error; err != nil {
		tx.Rollback()
		return nil, fiber.NewError(400, "Failed to create user account")
	}

	// Create OAuth account record
	encryptedAccess, _ := utils.EncryptToken(token.AccessToken)
	encryptedRefresh, _ := utils.EncryptToken(token.RefreshToken)

	// Extract scopes safely
	scopes := ""
	if scopeValue := token.Extra("scope"); scopeValue != nil {
		if scopeStr, ok := scopeValue.(string); ok {
			scopes = scopeStr
		}
	}

	oauthAccount := models.OAuthAccount{
		UserID:       user.ID,
		Provider:     provider,
		ProviderID:   userInfo.ID,
		Email:        userInfo.Email,
		Name:         userInfo.Name,
		AvatarURL:    userInfo.AvatarURL,
		AccessToken:  encryptedAccess,
		RefreshToken: encryptedRefresh,
		TokenExpiry:  &token.Expiry,
		Scopes:       scopes,
		LinkedAt:     time.Now(),
		LastUsedAt:   &[]time.Time{time.Now()}[0],
	}

	if err := tx.Create(&oauthAccount).Error; err != nil {
		tx.Rollback()
		return nil, fiber.NewError(500, "Failed to create OAuth account")
	}

	// Create JWT session
	jti, jwt, err := utils.GetSignedKey(user.ID)
	if err != nil {
		tx.Rollback()
		return nil, fiber.NewError(500, "Failed to generate JWT")
	}

	_, hashedToken := utils.GenerateRefreshToken()
	session := models.Session{
		JTI:          jti,
		UserID:       user.ID,
		RefreshToken: hashedToken,
		Revoked:      false,
		ExpiresAt:    time.Now().Add(30 * 24 * time.Hour),
	}

	if err := tx.Create(&session).Error; err != nil {
		tx.Rollback()
		return nil, fiber.NewError(500, "Failed to create session")
	}

	tx.Commit()

	return &utils.Response{
		Success: true,
		Code:    201,
		Message: fmt.Sprintf("Account created successfully with %s", string(provider)),
		Data: fiber.Map{
			"action": "register",
			"token":  jwt,
			"user": fiber.Map{
				"id":           user.ID,
				"username":     user.Username,
				"email":        user.Email,
				"account_type": user.AccountType,
			},
		},
	}, nil
}

// handleOAuthAccountLinking links a new OAuth provider to existing user
func handleOAuthAccountLinking(tx *gorm.DB, user *models.User, provider models.OAuthProvider, userInfo OAuthUserInfo, token *oauth2.Token) (*utils.Response, error) {
	// Check if this provider is already linked
	var existingLink models.OAuthAccount
	err := tx.Where("user_id = ? AND provider = ?", user.ID, provider).First(&existingLink).Error
	if err == nil {
		tx.Rollback()
		return nil, fiber.NewError(409, fmt.Sprintf("%s account already linked to this user", string(provider)))
	}

	if err != gorm.ErrRecordNotFound {
		tx.Rollback()
		return nil, fiber.NewError(500, "Database error checking existing OAuth links")
	}

	// Create new OAuth account link
	encryptedAccess, _ := utils.EncryptToken(token.AccessToken)
	encryptedRefresh, _ := utils.EncryptToken(token.RefreshToken)

	oauthAccount := models.OAuthAccount{
		UserID:       user.ID,
		Provider:     provider,
		ProviderID:   userInfo.ID,
		Email:        userInfo.Email,
		Name:         userInfo.Name,
		AvatarURL:    userInfo.AvatarURL,
		AccessToken:  encryptedAccess,
		RefreshToken: encryptedRefresh,
		TokenExpiry:  &token.Expiry,
		LinkedAt:     time.Now(),
		LastUsedAt:   &[]time.Time{time.Now()}[0],
	}

	if err := tx.Create(&oauthAccount).Error; err != nil {
		tx.Rollback()
		return nil, fiber.NewError(500, "Failed to link OAuth account")
	}

	// Update user account type to hybrid if it was OAuth-only
	if user.AccountType == models.AccountTypeOAuth {
		user.AccountType = models.AccountTypeHybrid
		if err := tx.Save(user).Error; err != nil {
			tx.Rollback()
			return nil, fiber.NewError(500, "Failed to update account type")
		}
	}

	// Create JWT session
	jti, jwt, err := utils.GetSignedKey(user.ID)
	if err != nil {
		tx.Rollback()
		return nil, fiber.NewError(500, "Failed to generate JWT")
	}

	_, hashedToken := utils.GenerateRefreshToken()
	session := models.Session{
		JTI:          jti,
		UserID:       user.ID,
		RefreshToken: hashedToken,
		Revoked:      false,
		ExpiresAt:    time.Now().Add(30 * 24 * time.Hour),
	}

	if err := tx.Create(&session).Error; err != nil {
		tx.Rollback()
		return nil, fiber.NewError(500, "Failed to create session")
	}

	tx.Commit()

	return &utils.Response{
		Success: true,
		Code:    200,
		Message: fmt.Sprintf("%s account linked and logged in successfully", string(provider)),
		Data: fiber.Map{
			"action": "login",
			"token":  jwt,
			"user": fiber.Map{
				"id":           user.ID,
				"username":     user.Username,
				"email":        user.Email,
				"account_type": user.AccountType,
			},
		},
	}, nil
}

// Helper functions

func generateUsernameFromOAuth(userInfo OAuthUserInfo) string {
	// Try to use the part before @ in email
	if userInfo.Email != "" {
		parts := strings.Split(userInfo.Email, "@")
		if len(parts) > 0 && parts[0] != "" {
			return strings.ToLower(parts[0])
		}
	}

	// Fallback to name
	if userInfo.Name != "" {
		return strings.ToLower(strings.ReplaceAll(userInfo.Name, " ", "_"))
	}

	// Final fallback
	return "user"
}

func ensureUniqueUsername(tx *gorm.DB, baseUsername string) (string, error) {
	username := baseUsername
	suffix := 1

	for {
		var count int64
		if err := tx.Model(&models.User{}).Where("username = ?", username).Count(&count).Error; err != nil {
			return "", err
		}

		if count == 0 {
			return username, nil
		}

		username = fmt.Sprintf("%s_%d", baseUsername, suffix)
		suffix++

		// Prevent infinite loops
		if suffix > 1000 {
			return "", fmt.Errorf("unable to generate unique username")
		}
	}
}
