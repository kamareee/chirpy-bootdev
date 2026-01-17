# Chirpy

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-4169E1?style=flat&logo=postgresql&logoColor=white)
![License](https://img.shields.io/badge/License-MIT-green.svg)

A RESTful API server for a Twitter-like social platform, built with Go. Created as part of the [boot.dev](https://boot.dev) Go server course.

## Features

- User registration and JWT authentication
- Create, read, and delete chirps (posts)
- Refresh token support
- PostgreSQL database with sqlc

## Setup

```bash
# Set environment variables in .env
DB_URL=postgres://...
JWT_SECRET=your-secret
PLATFORM=dev
POLKA_KEY=your-polka-key

# Run the server
go run .
```

## API Examples

**Create User**
```bash
curl -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "secret"}'
```

**Login**
```bash
curl -X POST http://localhost:8080/api/login \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "secret"}'
```

**Create Chirp** (requires auth)
```bash
curl -X POST http://localhost:8080/api/chirps \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"body": "Hello, Chirpy!"}'
```

**List Chirps**
```bash
curl http://localhost:8080/api/chirps
```
