package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/traigalo/traigalo-core/internal/platform/cache"
	"github.com/traigalo/traigalo-core/internal/platform/database"
	"github.com/traigalo/traigalo-core/internal/platform/middleware"
	"github.com/traigalo/traigalo-core/internal/slices/auth"
	"github.com/traigalo/traigalo-core/internal/slices/categories"
	"github.com/traigalo/traigalo-core/internal/slices/dispatch"
	"github.com/traigalo/traigalo-core/internal/slices/drivers"
	"github.com/traigalo/traigalo-core/internal/slices/health"
	"github.com/traigalo/traigalo-core/internal/slices/orders"
	"github.com/traigalo/traigalo-core/internal/slices/payments"
	"github.com/traigalo/traigalo-core/internal/slices/products"
	"github.com/traigalo/traigalo-core/internal/slices/rates"
	"github.com/traigalo/traigalo-core/internal/slices/reviews"
	"github.com/traigalo/traigalo-core/internal/slices/stores"
)

func main() {
	port := getEnv("PORT", "8080")
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "traigalo_admin")
	dbPass := getEnv("DB_PASSWORD", "traigalo_secure_dev_2026")
	dbName := getEnv("DB_NAME", "traigalo")
	redisHost := getEnv("REDIS_HOST", "localhost")
	redisPort := getEnv("REDIS_PORT", "6379")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dbPool, err := database.NewPool(ctx, database.Config{
		Host:     dbHost,
		Port:     dbPort,
		User:     dbUser,
		Password: dbPass,
		Database: dbName,
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbPool.Close()

	if err := database.RunMigrations(ctx, dbPool); err != nil {
		log.Fatalf("Failed to execute migrations: %v", err)
	}

	redisClient, err := cache.NewClient(ctx, cache.Config{
		Host: redisHost,
		Port: redisPort,
	})
	if err != nil {
		log.Fatalf("Failed to connect to redis: %v", err)
	}
	defer redisClient.Close()

	authRepo := auth.NewRepository(dbPool)
	authHandler := auth.NewHandler(authRepo)

	ratesService := rates.NewService(redisClient)
	ratesHandler := rates.NewHandler(ratesService)

	storesRepo := stores.NewRepository(dbPool)
	storesHandler := stores.NewHandler(storesRepo)

	categoriesRepo := categories.NewRepository(dbPool)
	categoriesHandler := categories.NewHandler(categoriesRepo)

	productsRepo := products.NewRepository(dbPool)
	productsHandler := products.NewHandler(productsRepo)

	paymentsRepo := payments.NewRepository(dbPool)
	paymentsHandler := payments.NewHandler(paymentsRepo)

	reviewsRepo := reviews.NewRepository(dbPool)
	reviewsHandler := reviews.NewHandler(reviewsRepo)

	ordersRepo := orders.NewRepository(dbPool, ratesService, redisClient)
	ordersHandler := orders.NewHandler(ordersRepo)

	dispatchRepo := dispatch.NewRepository(dbPool, redisClient)
	dispatchHandler := dispatch.NewHandler(dispatchRepo)

	driversHub := drivers.NewHub(redisClient)
	go driversHub.Run(ctx)

	driversRepo := drivers.NewRepository(redisClient)
	driversHandler := drivers.NewHandler(driversRepo, driversHub)

	healthHandler := health.NewHandler(dbPool, redisClient)

	r := chi.NewRouter()
	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(middleware.CORS)
	r.Use(middleware.GatewayAuth)

	r.Route("/v2", func(r chi.Router) {
		r.Get("/health", healthHandler.Check)
		r.Mount("/auth", authHandler.Routes())
		r.Mount("/rates", ratesHandler.Routes())
		r.Mount("/stores", storesHandler.Routes())
		r.Mount("/categories", categoriesHandler.Routes())
		r.Mount("/products", productsHandler.Routes())
		r.Mount("/payments", paymentsHandler.Routes())
		r.Mount("/reviews", reviewsHandler.Routes())
		r.Mount("/orders", ordersHandler.Routes())
		r.Mount("/dispatch", dispatchHandler.Routes())
		r.Mount("/drivers/locations", driversHandler.Routes())
		r.Get("/drivers/telemetry", driversHub.HandleWebSocket)
	})

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Traigalo Core API v2 listening on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)
	<-shutdownChan

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Graceful shutdown error: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
