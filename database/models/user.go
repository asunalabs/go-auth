package models

import (
	"time"

	"gorm.io/gorm"
)

// AccountType represents how the user account was created
type AccountType string

const (
	AccountTypeEmail  AccountType = "email"  // Traditional email/password account
	AccountTypeOAuth  AccountType = "oauth"  // OAuth-only account (no password)
	AccountTypeHybrid AccountType = "hybrid" // Email account with OAuth providers linked
)

// OAuthProvider represents supported OAuth providers
type OAuthProvider string

const (
	OAuthProviderGoogle OAuthProvider = "google"
	OAuthProviderGithub OAuthProvider = "github"
)

// Currency represents supported currencies
type Currency string

const (
	CurrencyRON Currency = "ron" // Romanian Leu
	CurrencyEUR Currency = "eur" // Euro
	CurrencyGBP Currency = "gbp" // British Pound
	CurrencyUSD Currency = "usd" // US Dollar
)

// Timezone represents supported timezones
type Timezone string

const (
	TimezoneUTC               Timezone = "UTC"
	TimezoneEuropeAmsterdam   Timezone = "Europe/Amsterdam"
	TimezoneEuropeBerlin      Timezone = "Europe/Berlin"
	TimezoneEuropeBucharest   Timezone = "Europe/Bucharest"
	TimezoneEuropeLondon      Timezone = "Europe/London"
	TimezoneEuropeParis       Timezone = "Europe/Paris"
	TimezoneEuropeRome        Timezone = "Europe/Rome"
	TimezoneAmericaNewYork    Timezone = "America/New_York"
	TimezoneAmericaChicago    Timezone = "America/Chicago"
	TimezoneAmericaDenver     Timezone = "America/Denver"
	TimezoneAmericaLosAngeles Timezone = "America/Los_Angeles"
	TimezoneAsiaShanghai      Timezone = "Asia/Shanghai"
	TimezoneAsiaTokyo         Timezone = "Asia/Tokyo"
	TimezoneAsiaKolkata       Timezone = "Asia/Kolkata"
	TimezoneAustraliaSydney   Timezone = "Australia/Sydney"
)

type User struct {
	ID          uint        `gorm:"primaryKey;autoIncrement" json:"id"`
	Username    string      `gorm:"uniqueIndex;size:255" json:"username"`
	Email       string      `gorm:"unique" json:"email"`
	Password    string      `json:"-"` // Nullable for OAuth-only accounts
	AccountType AccountType `gorm:"type:varchar(20);default:'email'" json:"account_type"`

	// Profile fields
	Currency Currency `gorm:"type:varchar(3);default:'usd'" json:"currency"`
	Timezone Timezone `gorm:"type:varchar(50);default:'UTC'" json:"timezone"`

	Sessions   []Session      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	OAuthLinks []OAuthAccount `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	UpdatedAt  time.Time      `gorm:"autoUpdateTime" json:"uat"`
	CreatedAt  time.Time      `gorm:"autoCreateTime" json:"cat"`
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}

// OAuthAccount stores OAuth provider linkage information
type OAuthAccount struct {
	ID           uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       uint           `gorm:"index" json:"user_id"`
	User         User           `gorm:"foreignKey:UserID;references:ID" json:"-"`
	Provider     OAuthProvider  `gorm:"type:varchar(20);index" json:"provider"`
	ProviderID   string         `gorm:"size:255;index" json:"provider_id"`    // OAuth provider's user ID
	Email        string         `gorm:"size:255;index" json:"email"`          // Email from OAuth provider
	Name         string         `gorm:"size:255" json:"name"`                 // Display name from provider
	AvatarURL    string         `gorm:"size:500" json:"avatar_url,omitempty"` // Profile picture URL
	AccessToken  string         `gorm:"type:text" json:"-"`                   // Encrypted OAuth access token
	RefreshToken string         `gorm:"type:text" json:"-"`                   // Encrypted OAuth refresh token
	TokenExpiry  *time.Time     `json:"token_expiry,omitempty"`               // When access token expires
	Scopes       string         `gorm:"type:text" json:"scopes,omitempty"`    // Granted OAuth scopes
	LinkedAt     time.Time      `gorm:"autoCreateTime" json:"linked_at"`
	LastUsedAt   *time.Time     `json:"last_used_at,omitempty"` // Last OAuth login
	CreatedAt    time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

// OAuthState stores CSRF protection state for OAuth flows
type OAuthState struct {
	ID          uint          `gorm:"primaryKey;autoIncrement" json:"id"`
	State       string        `gorm:"uniqueIndex;size:64" json:"state"`       // Random state parameter
	Provider    OAuthProvider `gorm:"type:varchar(20)" json:"provider"`       // OAuth provider
	Nonce       string        `gorm:"size:64" json:"nonce"`                   // Additional CSRF protection
	RedirectURL string        `gorm:"size:500" json:"redirect_url,omitempty"` // Post-auth redirect
	UserAgent   string        `gorm:"size:500" json:"user_agent,omitempty"`   // Security: track requesting UA
	IPAddress   string        `gorm:"size:45" json:"ip_address,omitempty"`    // Security: track requesting IP
	ExpiresAt   time.Time     `json:"expires_at"`                             // State expiration (5-10 minutes)
	CreatedAt   time.Time     `gorm:"autoCreateTime" json:"created_at"`
}

// Unique constraint to prevent duplicate OAuth accounts per provider per user
func (OAuthAccount) TableName() string {
	return "oauth_accounts"
}

type Session struct {
	ID           uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	JTI          string    `gorm:"unique" json:"jti"`
	UserID       uint      `json:"uid"`
	User         User      `gorm:"foreignKey:UserID;references:ID" json:"-"`
	RefreshToken string    `json:"-"`
	Revoked      bool      `gorm:"default:false" json:"revoked"`
	IssuedAt     time.Time `gorm:"autoCreateTime" json:"iat"`
	ExpiresAt    time.Time `json:"exp"`
}

type PasswordReset struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Email     string    `gorm:"index" json:"email"`
	Token     string    `gorm:"unique" json:"-"`
	Used      bool      `gorm:"default:false" json:"used"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}
