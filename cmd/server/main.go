package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	dbx "github.com/go-ozzo/ozzo-dbx"
	routing "github.com/go-ozzo/ozzo-routing/v2"
	"github.com/go-ozzo/ozzo-routing/v2/content"
	"github.com/go-ozzo/ozzo-routing/v2/cors"
	_ "github.com/lib/pq"

	// "github.com/qiangxue/go-rest-api/internal/album"
	"github.com/qiangxue/go-rest-api/internal/album"
	"github.com/qiangxue/go-rest-api/internal/auth"
	"github.com/qiangxue/go-rest-api/internal/config"
	"github.com/qiangxue/go-rest-api/internal/errors"
	"github.com/qiangxue/go-rest-api/internal/gemini"
	"github.com/qiangxue/go-rest-api/internal/healthcheck"

	"github.com/qiangxue/go-rest-api/pkg/accesslog"
	"github.com/qiangxue/go-rest-api/pkg/dbcontext"
	"github.com/qiangxue/go-rest-api/pkg/log"
)

// Version indicates the current version of the application.
var Version = "1.0.0"

var flagConfig = flag.String("config", "./config/local.yml", "path to the config file")

func main() {
	flag.Parse()
	// create root logger tagged with server version
	logger := log.New().With(context.Background(), "version", Version)

	// load application configurations
	cfg, err := config.Load(*flagConfig, logger)
	if err != nil {
		logger.Errorf("failed to load application configuration: %s", err)
		os.Exit(-1)
	}

	// connect to the database
	db, err := dbx.MustOpen("postgres", cfg.DSN)

	// close the database when the application exits
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error(err)
		}
	}()

	// check the database connection
	if err != nil {
		logger.Error(err)
		os.Exit(-1)
	}

	// set up database connection pool
	db.QueryLogFunc = logDBQuery(logger)
	db.ExecLogFunc = logDBExec(logger)

	// build HTTP server
	address := fmt.Sprintf(":%v", cfg.ServerPort)
	hs := &http.Server{
		Addr:    address,
		Handler: buildHandler(logger, dbcontext.New(db), cfg),
	}

	// start the HTTP server with graceful shutdown
	go routing.GracefulShutdown(hs, 10*time.Second, logger.Infof)
	logger.Infof("server %v is running at %v", Version, address)
	if err := hs.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error(err)
		os.Exit(-1)
	}
}

// buildHandler sets up the HTTP routing and builds an HTTP handler.
func buildHandler(logger log.Logger, db *dbcontext.DB, cfg *config.Config) http.Handler {
	router := routing.New()
	// Define allowed origins
	allowedOrigins := []string{
		"http://localhost:5173",
		"https://www.thewriterco.com",
		"https://thewriterco.com",
		"thewriterco.pages.dev",
	}

	router.Use(
		accesslog.Handler(logger),
		errors.Handler(logger),
		content.TypeNegotiator(content.JSON),
		cors.Handler(cors.Options{
			AllowOrigins:     strings.Join(allowedOrigins, ","),
			AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
			AllowHeaders:     "Authorization,Content-Type",
			AllowCredentials: true,
			MaxAge:           3600, // seconds
		}),
	)

	healthcheck.RegisterHandlers(router, Version)

	rg := router.Group("/v1")

	authHandler := auth.Handler(cfg.JWTSigningKey)

	album.RegisterHandlers(rg.Group(""),
		album.NewService(album.NewRepository(db, logger), logger),
		authHandler, logger,
	)

	auth.RegisterHandlers(rg.Group(""),
		auth.NewService(cfg.JWTSigningKey, cfg.JWTExpiration, logger),
		logger,
	)

	// prefrably, this should be in .env file
	genaicfg := gemini.Config{
		APIKey:           "AIzaSyB1rbyr2HsV6EXOMqbFw28sqjH0uPoKHO4",
		ModelName:        "gemini-2.0-flash-exp",
		Temperature:      1.0,
		TopK:             40,
		TopP:             0.95,
		MaxOutputTokens:  8192,
		ResponseMIMEType: "text/plain",
	}

	genaiService, err := gemini.NewService(context.Background(), genaicfg)
	if err != nil {
		logger.Error("failed to create GenAI service")
		os.Exit(-1)
	}
	gemini.RegisterHandlers(router, genaiService)
	return router
}

// logDBQuery returns a logging function that can be used to log SQL queries.
func logDBQuery(logger log.Logger) dbx.QueryLogFunc {
	return func(ctx context.Context, t time.Duration, sql string, rows *sql.Rows, err error) {
		if err == nil {
			logger.With(ctx, "duration", t.Milliseconds(), "sql", sql).Info("DB query successful")
		} else {
			logger.With(ctx, "sql", sql).Errorf("DB query error: %v", err)
		}
	}
}

// logDBExec returns a logging function that can be used to log SQL executions.
func logDBExec(logger log.Logger) dbx.ExecLogFunc {
	return func(ctx context.Context, t time.Duration, sql string, result sql.Result, err error) {
		if err == nil {
			logger.With(ctx, "duration", t.Milliseconds(), "sql", sql).Info("DB execution successful")
		} else {
			logger.With(ctx, "sql", sql).Errorf("DB execution error: %v", err)
		}
	}
}
