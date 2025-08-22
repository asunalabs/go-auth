# OAuth Authentication System

## Overview

This Go authentication system implements enterprise-grade OAuth functionality with support for Google and GitHub providers. The system handles three key OAuth flows:

1. **New OAuth User**: Creates OAuth-only account when email doesn't exist
2. **Account Linking Required**: Prevents automatic linking when email account exists without OAuth
3. **OAuth Login**: Logs in existing users with linked OAuth accounts

## OAuth Flows

### Flow 1: New OAuth User (Email Not in DB)
```
User OAuth → Email not found → Create OAuth-only account → Login successful
```
- Creates new user with `account_type: "oauth"`
- No password stored
- Username generated from email/name
- Full session created with JWT

### Flow 2: Account Linking Required (Email Exists, OAuth Not Linked)  
```
User OAuth → Email exists (email type) → Return link_required error
```
- Returns `409` status with action: `link_required`
- User must login with email/password first
- Then use settings to link OAuth account

### Flow 3: OAuth Login (Email Has OAuth Linked)
```
User OAuth → Email exists → OAuth linked → Login successful
```
- Updates OAuth account info (name, avatar, tokens)
- Creates new session with JWT
- Handles both `oauth` and `hybrid` account types

## API Endpoints

### OAuth Authentication

#### Initiate OAuth Flow
```http
POST /api/v1/auth/oauth/initiate
Content-Type: application/json

{
  "provider": "google|github",
  "redirect_url": "https://yourapp.com/dashboard" // optional
}
```

Response:
```json
{
  "success": true,
  "code": 200,
  "message": "OAuth flow initiated",
  "data": {
    "auth_url": "https://accounts.google.com/oauth/authorize?...",
    "state": "secure_random_state"
  }
}
```

#### OAuth Callback (automatic)
```http
GET /api/v1/auth/oauth/{provider}/callback?code=xxx&state=xxx
```

Responses:

**Success (Login/Register):**
```json
{
  "success": true,
  "code": 200,
  "message": "Logged in successfully with google",
  "data": {
    "action": "login|register",
    "token": "jwt_token_here",
    "user": {
      "id": 1,
      "username": "john_doe",
      "email": "john@gmail.com",
      "account_type": "oauth|hybrid"
    }
  }
}
```

**Account Linking Required:**
```json
{
  "success": false,
  "code": 409,
  "message": "Account with this email already exists. Please log in with your email and link your OAuth account in settings.",
  "data": {
    "action": "link_required",
    "existing_account": "email",
    "provider": "google",
    "email": "john@gmail.com"
  }
}
```

### User OAuth Management (Protected Routes)

#### Get Linked OAuth Accounts
```http
GET /api/v1/user/oauth/accounts
Authorization: Bearer {jwt_token}
```

Response:
```json
{
  "success": true,
  "code": 200,
  "message": "Success",
  "data": [
    {
      "id": 1,
      "provider": "google",
      "email": "john@gmail.com",
      "name": "John Doe",
      "avatar_url": "https://...",
      "linked_at": "2025-08-22T20:00:00Z",
      "last_used_at": "2025-08-22T20:30:00Z"
    }
  ]
}
```

#### Unlink OAuth Account
```http
DELETE /api/v1/user/oauth/accounts/{provider}
Authorization: Bearer {jwt_token}
```

Response:
```json
{
  "success": true,
  "code": 200,
  "message": "google account unlinked successfully",
  "data": null
}
```

## Setup Instructions

### 1. OAuth Provider Configuration

#### Google OAuth Setup
1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create/select project → APIs & Services → Credentials
3. Create OAuth 2.0 Client ID
4. Add authorized redirect URI: `http://localhost:5000/api/v1/auth/oauth/google/callback`
5. Add your `GOOGLE_CLIENT_ID` and `GOOGLE_CLIENT_SECRET` to `.env`

#### GitHub OAuth Setup  
1. Go to [GitHub Settings](https://github.com/settings/developers)
2. OAuth Apps → New OAuth App
3. Authorization callback URL: `http://localhost:5000/api/v1/auth/oauth/github/callback`
4. Add your `GITHUB_CLIENT_ID` and `GITHUB_CLIENT_SECRET` to `.env`

### 2. Environment Variables
Copy `.env.example` to `.env` and configure:

```bash
# OAuth Configuration
BASE_URL=http://localhost:5000
GOOGLE_CLIENT_ID=your_google_client_id
GOOGLE_CLIENT_SECRET=your_google_client_secret  
GITHUB_CLIENT_ID=your_github_client_id
GITHUB_CLIENT_SECRET=your_github_client_secret

# Existing configuration
DB_URI=postgresql://user:pass@localhost/dbname
JWT_SECRET=your_secret_key
PORT=5000
ENV=development
```

### 3. Database Migration
The OAuth models will be automatically migrated when you start the application:

```bash
go run .
```

New tables created:
- `oauth_accounts` - OAuth provider linkages
- `oauth_states` - CSRF protection for OAuth flows

## Security Features

### CSRF Protection
- Cryptographically secure state parameters
- State validation with nonce  
- User-Agent and IP tracking
- 10-minute state expiration

### Token Security
- OAuth tokens encrypted before storage
- Refresh tokens properly handled
- Token expiry tracking
- Secure session management

### Account Protection
- Prevents unauthorized account linking
- Validates email verification from providers
- Business rule enforcement (can't unlink last auth method)
- Transaction-based operations for consistency

## Account Types

- **`email`**: Traditional email/password accounts
- **`oauth`**: OAuth-only accounts (no password)
- **`hybrid`**: Email accounts with OAuth providers linked

## Error Handling

All OAuth operations include proper error handling:
- Invalid provider configurations
- Network failures during OAuth exchange
- Database transaction rollbacks
- Comprehensive validation

## Frontend Integration Example

```javascript
// 1. Initiate OAuth flow
const response = await fetch('/api/v1/auth/oauth/initiate', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ 
    provider: 'google',
    redirect_url: '/dashboard' 
  })
});

const { data } = await response.json();

// 2. Redirect user to OAuth provider
window.location.href = data.auth_url;

// 3. Handle callback (OAuth provider redirects back)
// Your backend processes the callback automatically and:
// - Creates account (new user)  
// - Returns link_required error (existing email account)
// - Logs in user (existing OAuth account)
```

This OAuth implementation follows enterprise security standards and provides a seamless user experience while maintaining strict account security.
