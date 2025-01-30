package main

import (
	"context"
	// "database/sql"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	// dbx "github.com/go-ozzo/ozzo-dbx"
	// dbx "github.com/go-ozzo/ozzo-dbx"
	routing "github.com/go-ozzo/ozzo-routing/v2"
	"github.com/go-ozzo/ozzo-routing/v2/content"
	"github.com/go-ozzo/ozzo-routing/v2/cors"
	_ "github.com/lib/pq"
	"golang.org/x/time/rate"

	// "golang.org/x/oauth2/authhandler"

	// "github.com/qiangxue/go-rest-api/internal/album"

	"github.com/qiangxue/go-rest-api/internal/auth"
	"github.com/qiangxue/go-rest-api/internal/config"
	"github.com/qiangxue/go-rest-api/internal/errors"
	"github.com/qiangxue/go-rest-api/internal/gemini"
	"github.com/qiangxue/go-rest-api/internal/healthcheck"
	"github.com/qiangxue/go-rest-api/internal/middleware"

	"github.com/qiangxue/go-rest-api/pkg/accesslog"
	// "github.com/qiangxue/go-rest-api/pkg/dbcontext"
	"github.com/qiangxue/go-rest-api/pkg/log"
	// "github.com/qiangxue/go-rest-api/internal/middleware"
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
	// db, err := dbx.MustOpen("postgres", cfg.DSN)

	// close the database when the application exits
	// defer func() {
	// 	if err := db.Close(); err != nil {
	// 		logger.Error(err)
	// 	}
	// }()

	// check the database connection
	if err != nil {
		logger.Error(err)
		os.Exit(-1)
	}

	// set up database connection pool
	// db.QueryLogFunc = logDBQuery(logger)
	// db.ExecLogFunc = logDBExec(logger)

	// build HTTP server
	address := fmt.Sprintf(":%v", cfg.ServerPort)
	hs := &http.Server{
		Addr:    address,
		Handler: buildHandler(logger, cfg),
	}

	// Start health check goroutine
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		client := &http.Client{}
		for range ticker.C {
			apiURL := os.Getenv("STAY_ALIVE_API_URL")
			_, err := client.Get(fmt.Sprintf("%s/healthcheck", apiURL))
			if err != nil {
				logger.Errorf("health check failed: %v", err)
			} else {
				logger.Infof("health check passed")
			}
		}
	}()

	// start the HTTP server with graceful shutdown
	go routing.GracefulShutdown(hs, 10*time.Second, logger.Infof)
	logger.Infof("server %v is running at %v", Version, address)
	if err := hs.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error(err)
		os.Exit(-1)
	}
}

// buildHandler sets up the HTTP routing and builds an HTTP handler.
// func buildHandler(logger log.Logger, db *dbcontext.DB, cfg *config.Config) http.Handler {
func buildHandler(logger log.Logger, cfg *config.Config) http.Handler {
	router := routing.New()
	// Define allowed origins
	allowedOrigins := []string{
		"http://localhost:5173",
		"https://www.thewriterco.com",
		"https://thewriterco.com",
		"thewriterco.pages.dev",
	}

	// 10 requests per 60 seconds → 1 request every 6 seconds
	rl := middleware.NewRateLimiter(rate.Every(6*time.Second), 10)

	// GET,POST,PUT,DELETE,OPTIONS

	router.Use(
		middleware.RateLimitMiddleware(rl),
		accesslog.Handler(logger),
		errors.Handler(logger),
		content.TypeNegotiator(content.JSON),
		cors.Handler(cors.Options{
			AllowOrigins:     strings.Join(allowedOrigins, ","),
			AllowMethods:     "POST",
			AllowHeaders:     "Authorization,Content-Type",
			AllowCredentials: true,
			MaxAge:           3600, // seconds
		}),
	)

	// Register GenAI service
	genaicfg := gemini.Config{
		APIKey:           os.Getenv("GEMINI_API_KEY"),
		ModelName:        "gemini-2.0-flash-exp",
		Temperature:      1.0,
		TopK:             40,
		TopP:             0.95,
		MaxOutputTokens:  8192,
		ResponseMIMEType: "text/plain",
	}

	healthcheck.RegisterHandlers(router, Version)

	rg := router.Group("/v1")

	// authHandler := auth.Handler(cfg.JWTSigningKey)

	// album.RegisterHandlers(rg.Group(""),
	// 	album.NewService(album.NewRepository(db, logger), logger),
	// 	authHandler, logger,
	// )

	auth.RegisterHandlers(rg.Group(""),
		auth.NewService(cfg.JWTSigningKey, cfg.JWTExpiration, logger),
		logger,
	)

	gemini.RegisterHandlers(rg.Group(""),
		gemini.NewService(context.Background(), genaicfg),
		// authHandler,
	)

	// genaiService, err := gemini.NewService(context.Background(), genaicfg)

	// if err != nil {
	// 	logger.Error("failed to create GenAI service")
	// 	os.Exit(-1)
	// }
	return router
}

// logDBQuery returns a logging function that can be used to log SQL queries.
// func logDBQuery(logger log.Logger) dbx.QueryLogFunc {
// 	return func(ctx context.Context, t time.Duration, sql string, rows *sql.Rows, err error) {
// 		if err == nil {
// 			logger.With(ctx, "duration", t.Milliseconds(), "sql", sql).Info("DB query successful")
// 		} else {
// 			logger.With(ctx, "sql", sql).Errorf("DB query error: %v", err)
// 		}
// 	}
// }

// logDBExec returns a logging function that can be used to log SQL executions.
// func logDBExec(logger log.Logger) dbx.ExecLogFunc {
// 	return func(ctx context.Context, t time.Duration, sql string, result sql.Result, err error) {
// 		if err == nil {
// 			logger.With(ctx, "duration", t.Milliseconds(), "sql", sql).Info("DB execution successful")
// 		} else {
// 			logger.With(ctx, "sql", sql).Errorf("DB execution error: %v", err)
// 		}
// 	}
// }
