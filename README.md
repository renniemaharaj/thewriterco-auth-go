# The Writer Company Backend Implementation in Go (thewriterco-auth-go)

[![GoDoc](https://godoc.org/github.com/qiangxue/go-rest-api?status.png)](http://godoc.org/github.com/qiangxue/go-rest-api)
[![Build Status](https://github.com/qiangxue/go-rest-api/workflows/build/badge.svg)](https://github.com/qiangxue/go-rest-api/actions?query=workflow%3Abuild)
[![Code Coverage](https://codecov.io/gh/qiangxue/go-rest-api/branch/master/graph/badge.svg)](https://codecov.io/gh/qiangxue/go-rest-api)
[![Go Report](https://goreportcard.com/badge/github.com/qiangxue/go-rest-api)](https://goreportcard.com/report/github.com/qiangxue/go-rest-api)

This Go RESTful API starter kit serves as the backbone of The Writer Company's backend implementation. It retains the structure and best practices from the original boilerplate to facilitate a clean, maintainable project for quickly building Go-based RESTful services while aligning with [SOLID principles](https://en.wikipedia.org/wiki/SOLID) and [clean architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html).

Included features:

• RESTful endpoints in a widely accepted format  
• Standard CRUD operations of a database table  
• JWT-based authentication  
• Environment dependent application configuration management  
• Structured logging with contextual information  
• Error handling with proper error response generation  
• Database migration  
• Data validation  
• Full test coverage  
• Live reloading during development  

The following Go packages can be replaced with preferred alternatives because their usage is localized and abstracted:

• Routing: [ozzo-routing](https://github.com/go-ozzo/ozzo-routing)  
• Database access: [ozzo-dbx](https://github.com/go-ozzo/ozzo-dbx)  
• Database migration: [golang-migrate](https://github.com/golang-migrate/migrate)  
• Data validation: [ozzo-validation](https://github.com/go-ozzo/ozzo-validation)  
• Logging: [zap](https://github.com/uber-go/zap)  
• JWT: [jwt-go](https://github.com/dgrijalva/jwt-go)  

## Getting Started

Install [Go 1.13 or above](https://golang.org/doc/install).  
Also install [Docker 17.05 or above](https://www.docker.com/get-started).

Try the kit:

```shell
git clone https://github.com/qiangxue/go-rest-api.git

cd go-rest-api

make db-start

make testdata

make run
```

Or run live:

```shell
make run-live
```

This starts a RESTful API server at http://127.0.0.1:8080 with endpoints:

• GET /healthcheck  
• POST /v1/login  
• GET /v1/albums  
• GET /v1/albums/:id  
• POST /v1/albums  
• PUT /v1/albums/:id  
• DELETE /v1/albums/:id  

Various tests can be performed using cURL or other API tools:

```shell
curl -X POST -H "Content-Type: application/json" -d '{"username": "demo", "password": "pass"}' http://localhost:8080/v1/login
```

Use the returned JWT:

```shell
curl -X GET -H "Authorization: Bearer ...JWT token here..." http://localhost:8080/v1/albums
```

To customize this starter kit for The Writer Company or any other organization, replace `github.com/qiangxue/go-rest-api` with your repository name, for instance `github.com/abc/xyz`.

## Project Layout

```
.
├── cmd
│   └── server
├── config
├── internal
│   ├── album
│   ├── auth
│   ├── config
│   ├── entity
│   ├── errors
│   ├── healthcheck
│   └── test
├── migrations
├── pkg
│   ├── accesslog
│   ├── graceful
│   ├── log
│   └── pagination
└── testdata
```

This layout follows [Standard Go Project Layout](https://github.com/golang-standards/project-layout). Packages in `internal` and `pkg` are structured by features, adhering to [clean architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html).

## Common Development Tasks

### Implementing a New Feature

1. Write the service implementing the business logic (see `internal/album/service.go`).  
2. Develop the RESTful API exposing that service (see `internal/album/api.go`).  
3. Implement the repository persisting data (see `internal/album/repository.go`).  
4. Inject dependencies (see `album.RegisterHandlers()` in `cmd/server/main.go`).

### Working with DB Transactions

Use `dbcontext.DB.Transactional()` for transactional operations in the service layer. Example:

```go
func serviceMethod(ctx context.Context, repo Repository, transactional dbcontext.TransactionFunc) error {
    return transactional(ctx, func(ctx context.Context) error {
        repo.method1(...)
        repo.method2(...)
        return nil
    })
}
```

You can also use `dbcontext.DB.TransactionHandler()` as middleware to wrap multiple service calls from a single API handler.

### Updating Database Schema

```shell
make migrate
make migrate-new
make migrate-down
make migrate-reset
```

### Managing Configurations

The application reads configurations from `internal/config/config.go`. The configuration file path defaults to `./config/local.yml`, and environment variables take precedence (using the `APP_` prefix). Keep secrets in environment variables rather than committing them to the repository.

## Deployment

For production usage, build a Docker image:

```shell
make build-docker
```

The container uses `cmd/server/entryscript.sh` and decides which configuration file to use based on `APP_ENV`. Or build a binary:

```shell
make build
./server -config=./config/prod.yml
```
