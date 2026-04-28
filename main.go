package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"

	"url-shortener/config"
	"url-shortener/handler"
	"url-shortener/middleware"
	"url-shortener/repository"
	"url-shortener/service"
	"url-shortener/worker"
)

func main() {

	godotenv.Load()

	db, err := config.ConnectDB(os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}

	err = config.CreateTable(db)
	if err != nil {
		log.Fatal(err)
	}
	// log.Println("Database ready")
	redisClient := config.InitRedis(os.Getenv("REDIS_URL"))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// init repo
	urlRepo := repository.NewURLRepo(db)
	userRepo := repository.NewUserRepo(db)

	// init service
	urlSvc := service.NewURLService(urlRepo, redisClient)
	authSvc := service.NewAuthService(userRepo, os.Getenv("JWT_SECRET"), 24*time.Hour)

	// init handler
	urlHandler := handler.NewURLHandler(urlSvc)
	authHandler := handler.NewAuthHandler(authSvc)

	worker.StartClickSync(ctx, redisClient, urlRepo)
	worker.StartExpiryCleanup(ctx, urlRepo)

	app := fiber.New(fiber.Config{
		ProxyHeader: fiber.HeaderXForwardedFor,
	})

	app.Use(middleware.RateLimiter(redisClient))
	app.Use(logger.New())

	// auth := app.Group("/auth")
	app.Post("/auth/signup", authHandler.Signup)
	app.Post("/auth/login", authHandler.Login)
	app.Post("/auth/logout", authHandler.Logout)

	protected := app.Group("/", middleware.AuthMiddleware(authSvc))
	protected.Post("/shorten", urlHandler.Shorten)
	protected.Get("/:code", urlHandler.Redirect)
	protected.Get("/alias/check/:code", urlHandler.CheckAlias)
	protected.Get("/user/urls", urlHandler.GetUserURLs)
	protected.Delete("/:code", urlHandler.DeleteURL)

	log.Fatal(app.Listen(":8000"))
}
