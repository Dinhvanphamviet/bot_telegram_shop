package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"telegram-shop/internal/bot"
	"telegram-shop/internal/config"
	"telegram-shop/internal/handler"
	"telegram-shop/internal/middleware"
	"telegram-shop/internal/repository"
	"telegram-shop/internal/service"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Println("Starting Telegram Shop Bot...")

	ctx := context.Background()
	dbPool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbPool.Close()

	if err := dbPool.Ping(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("✅ Connected to database")

	if err := runMigrations(ctx, dbPool); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("✅ Migrations completed")

	userRepo := repository.NewUserRepo(dbPool)
	productRepo := repository.NewProductRepo(dbPool)
	itemRepo := repository.NewItemRepo(dbPool)
	productLinkRepo := repository.NewProductLinkRepo(dbPool)
	orderRepo := repository.NewOrderRepo(dbPool)
	paymentRepo := repository.NewPaymentRepo(dbPool)
	walletRepo := repository.NewWalletRepo(dbPool)

	userService := service.NewUserService(userRepo)
	productService := service.NewProductService(productRepo, itemRepo, productLinkRepo)
	orderService := service.NewOrderService(dbPool, cfg, orderRepo, userRepo, itemRepo, productLinkRepo, paymentRepo, walletRepo)
	paymentService := service.NewPaymentService(dbPool, paymentRepo, orderRepo, userRepo, productLinkRepo, walletRepo)
	walletService := service.NewWalletService(dbPool, cfg, userRepo, walletRepo, paymentRepo)

	stateManager := bot.NewStateManager()
	telegramBot := bot.NewBot(cfg.TelegramBotToken)
	commandHandler := bot.NewCommandHandler(telegramBot, cfg, userService, productService, orderService, walletService, paymentService, stateManager)
	callbackHandler := bot.NewCallbackHandler(telegramBot, cfg, userService, productService, orderService, walletService, paymentService, stateManager)

	telegramHandler := handler.NewTelegramHandler(telegramBot, cfg, commandHandler, callbackHandler, paymentService, userService)
	adminHandler := handler.NewAdminHandler(telegramBot, productService, orderService, walletService, userService)

	r := chi.NewRouter()
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(chiMiddleware.Timeout(30 * time.Second))

	r.Post("/webhook/telegram", telegramHandler.HandleTelegramWebhook)
	r.Post("/webhook/sepay", telegramHandler.HandleSepayWebhook)
	r.Get("/health", telegramHandler.HandleHealth)

	r.Route("/api/admin", func(r chi.Router) {
		r.Use(middleware.AdminAPIKey(cfg.AdminAPIKey))
		adminHandler.RegisterRoutes(r)
	})

	webhookURL := fmt.Sprintf("%s/webhook/telegram", cfg.WebhookURL)
	if err := telegramBot.SetWebhook(webhookURL); err != nil {
		log.Printf("⚠️ Failed to set webhook: %v", err)
	} else {
		log.Printf("✅ Webhook set: %s", webhookURL)
	}

	// Background job: expire pending payments every minute and notify users
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			expiredList, err := paymentService.ExpirePendingPayments(context.Background())
			if err != nil {
				log.Printf("Error expiring payments: %v", err)
				continue
			}
			for _, exp := range expiredList {
				if exp.ChatID != 0 && exp.MessageID != 0 {
					_ = telegramBot.EditMessageCaption(exp.ChatID, exp.MessageID, bot.MsgPaymentExpiredCaption(exp.TransferContent, exp.Amount), bot.KbBackToMenu())
				}
				telegramBot.SendMessage(exp.TelegramID, bot.MsgPaymentExpired(exp.TransferContent), bot.KbBackToMenu())
			}
		}
	}()

	addr := fmt.Sprintf(":%s", cfg.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	log.Printf("🚀 Server starting on %s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
	log.Println("Server stopped")
}

// runMigrations executes SQL migration files.
func runMigrations(ctx context.Context, db *pgxpool.Pool) error {
	migrationSQL, err := os.ReadFile("migrations/000001_init_schema.up.sql")
	if err != nil {
		return fmt.Errorf("read migration file: %w", err)
	}

	_, err = db.Exec(ctx, string(migrationSQL))
	if err != nil {
		// Migration may fail if tables already exist — that's OK
		log.Printf("Migration note (may be normal if tables exist): %v", err)
	}

	// Always ensure recent schema alterations are applied to live DB
	if _, err := db.Exec(ctx, "ALTER TABLE orders ADD COLUMN IF NOT EXISTS quantity INT NOT NULL DEFAULT 1;"); err != nil {
		log.Printf("Warning executing schema update (quantity column): %v", err)
	}
	return nil
}
