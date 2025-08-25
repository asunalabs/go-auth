# Go Authentication API

A production-ready, enterprise-grade authentication API built with Go, Fiber, and PostgreSQL. Features comprehensive authentication flows including email/password, OAuth (Google & GitHub), password reset, and session management.

## 🚀 Features

### Core Authentication

- ✅ **Email/Password Registration & Login** - Traditional authentication with secure password hashing
- ✅ **JWT Authentication** - Secure token-based authentication with refresh tokens
- ✅ **Session Management** - Persistent sessions with token revocation
- ✅ **Password Reset** - Email-based password reset functionality

### OAuth Integration

- ✅ **Google OAuth** - Seamless Google account integration
- ✅ **GitHub OAuth** - GitHub account authentication
- ✅ **Account Linking** - Link multiple OAuth providers to existing accounts
- ✅ **Hybrid Accounts** - Support for email + OAuth provider combinations

### Enterprise Security

- 🔒 **CSRF Protection** - State validation with nonce for OAuth flows
- 🔒 **Encrypted OAuth Tokens** - OAuth access/refresh tokens encrypted at rest
- 🔒 **Request ID Tracking** - Request tracing for debugging and monitoring
- 🔒 **Error Handling** - Comprehensive error handling with structured responses
- 🔒 **Input Validation** - Strict validation for all API inputs

### Developer Experience

- 📊 **Built-in Monitoring** - `/metrics` endpoint for application monitoring
- 📝 **Structured Logging** - Request ID correlation and structured error logging
- 🧪 **Test-Ready** - Architecture designed for easy unit and integration testing
- 📚 **Documentation** - Comprehensive API documentation and setup guides

## 🛠️ Tech Stack

- **Framework**: [Fiber v2](https://gofiber.io/) - Express-inspired web framework
- **Database**: PostgreSQL with [GORM](https://gorm.io/) ORM
- **Authentication**: JWT with refresh token rotation
- **OAuth**: Google & GitHub OAuth 2.0 integration
- **Security**: bcrypt password hashing, encrypted token storage
- **Email**: SMTP email delivery for notifications
- **Monitoring**: Built-in metrics endpoint

## 📋 Prerequisites

- Go 1.21+
- PostgreSQL 12+
- SMTP server (for password reset emails)
- OAuth provider credentials (Google/GitHub)

## ⚡ Quick Start

### 1. Clone and Setup

```bash
git clone https://github.com/yourusername/go-auth.git
cd go-auth

# Install dependencies
go mod download

# Copy environment template
cp .env.example .env
```

### 2. Configure Environment

Edit `.env` with your configuration:

```bash
# Database
DB_URI=postgresql://user:password@localhost:5432/go_auth?sslmode=disable

# JWT Secret (generate a secure random string)
JWT_SECRET=your_super_secure_jwt_secret_here

# Server Configuration
PORT=5000
ENV=development

# Email Configuration (for password reset)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your-email@gmail.com
SMTP_PASSWORD=your-app-password

# OAuth Configuration (see OAuth Setup section)
BASE_URL=http://localhost:5000
GOOGLE_CLIENT_ID=your_google_client_id
GOOGLE_CLIENT_SECRET=your_google_client_secret
GITHUB_CLIENT_ID=your_github_client_id
GITHUB_CLIENT_SECRET=your_github_client_secret
```

### 3. Database Setup

```bash
# Create your PostgreSQL database
createdb go_auth

# Run the application (auto-migrates database)
go run .
```

### 4. Test the API

```bash
# Health check
curl http://localhost:5000

# Register a new user
curl -X POST http://localhost:5000/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "johndoe",
    "email": "john@example.com",
    "password": "securepassword123"
  }'
```

## 🔧 OAuth Provider Setup

### Google OAuth

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create/select a project → **APIs & Services** → **Credentials**
3. Click **Create Credentials** → **OAuth 2.0 Client ID**
4. Choose **Web application**
5. Add authorized redirect URI: `http://localhost:5000/api/v1/auth/oauth/google/callback`
6. Copy your **Client ID** and **Client Secret** to `.env`

### GitHub OAuth

1. Go to [GitHub Settings](https://github.com/settings/developers)
2. Click **OAuth Apps** → **New OAuth App**
3. Fill in application details
4. Set **Authorization callback URL**: `http://localhost:5000/api/v1/auth/oauth/github/callback`
5. Copy your **Client ID** and **Client Secret** to `.env`

## 📖 API Documentation

### Authentication Endpoints

#### Register User

```http
POST /api/v1/auth/register
Content-Type: application/json

{
  "username": "johndoe",
  "email": "john@example.com",
  "password": "securepassword123"
}
```

#### Login User

```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "email": "john@example.com",
  "password": "securepassword123"
}
```

#### Refresh Token

```http
POST /api/v1/auth/refresh
Cookie: refresh_token=your_refresh_token
```

#### Logout (Revoke Token)

```http
POST /api/v1/auth/revoke
Cookie: refresh_token=your_refresh_token
```

### OAuth Endpoints

#### Initiate OAuth Flow

```http
POST /api/v1/auth/oauth/initiate
Content-Type: application/json

{
  "provider": "google",
  "redirect_url": "https://yourapp.com/dashboard"
}
```

#### OAuth Callback (Automatic)

```http
GET /api/v1/auth/oauth/{provider}/callback?code=xxx&state=xxx
```

### Protected Endpoints (Require JWT)

#### Get User Profile

```http
GET /api/v1/user/me
Authorization: Bearer your_jwt_token
```

#### Update User Profile

```http
PUT /api/v1/user/me
Authorization: Bearer your_jwt_token
Content-Type: application/json

{
  "username": "newusername",
  "currency": "usd",
  "timezone": "America/New_York"
}
```

#### Get Linked OAuth Accounts

```http
GET /api/v1/user/oauth/accounts
Authorization: Bearer your_jwt_token
```

#### Unlink OAuth Account

```http
DELETE /api/v1/user/oauth/accounts/{provider}
Authorization: Bearer your_jwt_token
```

### Password Reset

#### Request Password Reset

```http
POST /api/v1/auth/password-reset/request
Content-Type: application/json

{
  "email": "john@example.com"
}
```

#### Confirm Password Reset

```http
POST /api/v1/auth/password-reset/confirm
Content-Type: application/json

{
  "token": "reset_token_from_email",
  "new_password": "newpassword123"
}
```

## 🏗️ Project Structure

```
go-auth/
├── main.go                 # Application entry point
├── handlers/              # HTTP request handlers
│   ├── auth.go           # Authentication handlers
│   ├── oauth.go          # OAuth flow handlers
│   ├── me.go             # User profile handlers
│   └── password_reset.go # Password reset handlers
├── routes/               # Route definitions
│   ├── auth.go          # Auth route registration
│   └── user.go          # User route registration
├── database/            # Database configuration
│   ├── db.go           # Database connection
│   └── models/         # Data models
│       └── user.go     # User, Session, OAuth models
├── middleware/          # HTTP middleware
│   ├── error_handler.go # Global error handling
│   └── request_id.go    # Request ID injection
├── utils/              # Utility functions
│   ├── crypto.go      # Password hashing
│   ├── jwt.go         # JWT token handling
│   ├── oauth.go       # OAuth utilities
│   ├── email.go       # Email sending
│   └── response.go    # API response formatting
└── .env.example       # Environment configuration template
```

## 🧪 Testing

### Run Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test ./... -cover

# Run tests with verbose output
go test ./... -v
```

### Manual Testing with curl

See the included `test-endpoints.sh` script for comprehensive API testing examples.

## 🚀 Deployment

### Production Environment Variables

```bash
# Set production environment
ENV=production

# Use production database
DB_URI=postgresql://user:pass@prod-db:5432/go_auth?sslmode=require

# Generate strong JWT secret
JWT_SECRET=$(openssl rand -base64 64)

# Configure production OAuth redirect URLs
BASE_URL=https://yourdomain.com
```

### Docker Deployment

```dockerfile
# Example Dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o main .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
EXPOSE 5000
CMD ["./main"]
```

### Docker Compose

```yaml
# docker-compose.yml
version: "3.8"
services:
  app:
    build: .
    ports:
      - "5000:5000"
    environment:
      - DB_URI=postgresql://postgres:password@db:5432/go_auth?sslmode=disable
      - JWT_SECRET=your_production_jwt_secret
    depends_on:
      - db

  db:
    image: postgres:15-alpine
    environment:
      - POSTGRES_DB=go_auth
      - POSTGRES_USER=postgres
      - POSTGRES_PASSWORD=password
    volumes:
      - postgres_data:/var/lib/postgresql/data

volumes:
  postgres_data:
```

## 🤝 Contributing

We welcome contributions! Please follow these guidelines:

### Development Setup

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Make your changes following the coding standards
4. Add tests for new functionality
5. Ensure all tests pass: `go test ./...`
6. Run linting: `go vet ./...` and `gofmt -w .`
7. Commit with a descriptive message
8. Push and create a Pull Request

### Coding Standards

- Follow Go conventions and use `gofmt` for formatting
- Add unit tests for all new business logic
- Include integration tests for API endpoints
- Update documentation for API changes
- Use structured logging for debugging
- Handle errors explicitly and return appropriate HTTP status codes

### Pull Request Requirements

- [ ] All tests pass (`go test ./...`)
- [ ] Code is formatted (`go vet ./...` and `gofmt`)
- [ ] New functionality includes tests
- [ ] API changes are documented
- [ ] No breaking changes without approval
- [ ] Commit messages follow conventional format

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 💡 Features Roadmap

- [ ] **Two-Factor Authentication (2FA)** - TOTP and SMS-based 2FA
- [ ] **Social Logins** - Discord, Twitter, LinkedIn OAuth
- [ ] **Rate Limiting** - API rate limiting and abuse prevention
- [ ] **Audit Logging** - User action audit trails
- [ ] **Admin Panel** - Web interface for user management
- [ ] **API Keys** - Generate and manage API keys for integrations
- [ ] **Webhooks** - User event notifications via webhooks

## 🆘 Support

- **Documentation**: Check this README and inline code comments
- **Issues**: [GitHub Issues](https://github.com/asunalabs/go-auth/issues)
- **Discussions**: [GitHub Discussions](https://github.com/asunalabs/go-auth/discussions)

## 👨‍💻 Authors

- **Asuna Labs** - [@asunalabs](https://github.com/asunalabs)

## 🙏 Acknowledgments

- [Fiber](https://gofiber.io/) - Fast HTTP framework
- [GORM](https://gorm.io/) - Fantastic ORM for Go
- [JWT-Go](https://github.com/golang-jwt/jwt) - JWT implementation
- [OAuth2](https://pkg.go.dev/golang.org/x/oauth2) - OAuth 2.0 client

---

**⭐ If this project helped you, please consider giving it a star!**
