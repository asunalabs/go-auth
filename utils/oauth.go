package utils

import (
	"api/database/models"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
)

// OAuth configuration
type OAuthConfig struct {
	GoogleConfig *oauth2.Config
	GithubConfig *oauth2.Config
}

var OAuthConfigs *OAuthConfig

// Initialize OAuth configurations
func InitOAuth() {
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:5000"
	}

	OAuthConfigs = &OAuthConfig{
		GoogleConfig: &oauth2.Config{
			ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
			ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
			RedirectURL:  baseURL + "/api/v1/auth/oauth/google/callback",
			Scopes:       []string{"openid", "profile", "email"},
			Endpoint:     google.Endpoint,
		},
		GithubConfig: &oauth2.Config{
			ClientID:     os.Getenv("GITHUB_CLIENT_ID"),
			ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
			RedirectURL:  baseURL + "/api/v1/auth/oauth/github/callback",
			Scopes:       []string{"user:email", "read:user"},
			Endpoint:     github.Endpoint,
		},
	}
}

// GetOAuthConfig returns the OAuth config for a specific provider
func GetOAuthConfig(provider models.OAuthProvider) (*oauth2.Config, error) {
	if OAuthConfigs == nil {
		return nil, errors.New("OAuth not initialized")
	}

	switch provider {
	case models.OAuthProviderGoogle:
		if OAuthConfigs.GoogleConfig.ClientID == "" {
			return nil, errors.New("google OAuth not configured")
		}
		return OAuthConfigs.GoogleConfig, nil
	case models.OAuthProviderGithub:
		if OAuthConfigs.GithubConfig.ClientID == "" {
			return nil, errors.New("github OAuth not configured")
		}
		return OAuthConfigs.GithubConfig, nil
	default:
		return nil, errors.New("unsupported OAuth provider")
	}
}

// GenerateOAuthState creates a cryptographically secure state parameter
func GenerateOAuthState() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// GenerateNonce creates a cryptographically secure nonce
func GenerateNonce() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// GoogleUserInfo represents user information from Google OAuth
type GoogleUserInfo struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Picture  string `json:"picture"`
	Verified bool   `json:"email_verified"`
}

// GitHubUserInfo represents user information from GitHub OAuth
type GitHubUserInfo struct {
	ID        int    `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

// GitHubEmail represents email information from GitHub API
type GitHubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

// FetchGoogleUserInfo retrieves user information from Google using the access token
func FetchGoogleUserInfo(ctx context.Context, token *oauth2.Token) (*GoogleUserInfo, error) {
	config, err := GetOAuthConfig(models.OAuthProviderGoogle)
	if err != nil {
		return nil, err
	}

	client := config.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("failed to get Google user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google API returned status %d", resp.StatusCode)
	}

	var userInfo GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode Google user info: %w", err)
	}

	if !userInfo.Verified {
		return nil, errors.New("google email not verified")
	}

	return &userInfo, nil
}

// FetchGitHubUserInfo retrieves user information from GitHub using the access token
func FetchGitHubUserInfo(ctx context.Context, token *oauth2.Token) (*GitHubUserInfo, error) {
	config, err := GetOAuthConfig(models.OAuthProviderGithub)
	if err != nil {
		return nil, err
	}

	client := config.Client(ctx, token)

	// Get user info
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return nil, fmt.Errorf("failed to get GitHub user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var userInfo GitHubUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode GitHub user info: %w", err)
	}

	// If email is not public, fetch primary email from emails endpoint
	if userInfo.Email == "" {
		email, err := fetchGitHubPrimaryEmail(client)
		if err != nil {
			return nil, fmt.Errorf("failed to get GitHub email: %w", err)
		}
		userInfo.Email = email
	}

	return &userInfo, nil
}

// fetchGitHubPrimaryEmail gets the primary verified email from GitHub
func fetchGitHubPrimaryEmail(client *http.Client) (string, error) {
	resp, err := client.Get("https://api.github.com/user/emails")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub emails API returned status %d", resp.StatusCode)
	}

	var emails []GitHubEmail
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", err
	}

	// Find primary verified email
	for _, email := range emails {
		if email.Primary && email.Verified {
			return email.Email, nil
		}
	}

	// If no primary verified email found, return first verified email
	for _, email := range emails {
		if email.Verified {
			return email.Email, nil
		}
	}

	return "", errors.New("no verified email found in GitHub account")
}

// EncryptToken encrypts OAuth tokens for secure storage
func EncryptToken(token string) (string, error) {
	if token == "" {
		return "", nil
	}
	// TODO: Implement proper encryption (AES-256-GCM)
	// For now, return as-is - in production, encrypt with a key from environment
	return token, nil
}

// DecryptToken decrypts OAuth tokens from storage
func DecryptToken(encryptedToken string) (string, error) {
	if encryptedToken == "" {
		return "", nil
	}
	// TODO: Implement proper decryption
	// For now, return as-is - in production, decrypt with the same key
	return encryptedToken, nil
}

// ValidateOAuthState validates the OAuth state parameter for CSRF protection
func ValidateOAuthState(state, nonce, userAgent, ipAddress string) error {
	if state == "" {
		return errors.New("missing OAuth state parameter")
	}
	if len(state) < 32 {
		return errors.New("OAuth state parameter too short")
	}
	// Additional validations can be added here
	return nil
}
