# Auth Service

Microservice for user authentication and authorization. Supports registration, login, session and token management.

## Tech Stack

- **Go** 1.21+
- **PostgreSQL** 15+ - main database
- **Redis** 7+ - token blacklists and sessions
- **Gin** - HTTP web framework
- **JWT** - authentication tokens

## Features

### MVP (Current Version)

- ✅ User registration
- ✅ User login
- ✅ Token refresh
- ✅ User logout
- ✅ Get current user profile
- ✅ Token validation (middleware)

### Planned

- 🔄 Password recovery
- 🔄 Email verification
- 🔄 OAuth2 integration (Google, Apple, Facebook)
- 🔄 Two-factor authentication (2FA)
- 🔄 Active session management

## Quick Start

### Requirements

- Go 1.21+
- Docker and Docker Compose
- Make (optional)

### Installation

1. Clone the repository:
```bash
git clone https://github.com/PeremyshlevPR/auth-service-2
cd auth-service-2
```

2. Copy the environment variables file:
```bash
cp .env.example .env
```

3. Start Docker containers (PostgreSQL and Redis):
```bash
make docker-up
# or
docker-compose up -d
```

4. Run database migrations:
```bash
make migrate-up
```

5. Install dependencies:
```bash
make deps
# or
go mod download
```

6. Start the service:
```bash
make run
# or
go run ./cmd/server
```

The service will be available at: `http://localhost:8080`

## Configuration

All settings are configured through environment variables. See `.env.example` for a list of available variables.

### Main variables:

- `SERVER_PORT` - server port (default: 8080)
- `JWT_SECRET` - secret key for JWT (required, minimum 32 characters)
- `POSTGRES_HOST`, `POSTGRES_PORT`, `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB` - PostgreSQL settings
- `REDIS_HOST`, `REDIS_PORT` - Redis settings

### Main endpoints:

- `POST /api/v1/auth/register` - Registration
- `POST /api/v1/auth/login` - Login
- `POST /api/v1/auth/refresh` - Token refresh
- `POST /api/v1/auth/logout` - Logout
- `GET /api/v1/auth/me` - Get profile (requires authorization)

### Make Commands

```bash
make help           # Show all available commands
make deps           # Install dependencies
make build          # Build the application
make run            # Run the application
make test           # Run tests
make lint           # Run linter
make fmt            # Format code
make docker-up      # Start Docker containers
make docker-down    # Stop Docker containers
make migrate-up     # Apply migrations
make migrate-down   # Rollback migrations
make migrate-create NAME=create_table  # Create a new migration
```

Run tests with coverage:
```bash
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```
