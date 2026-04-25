package main

import (
	"log"
	"os"

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

	repo := repository.NewURLRepo(db)
	svc := service.NewURLService(repo, redisClient)
	h := handler.NewURLHandler(svc)

	worker.StartClickSync(redisClient, repo)

	app := fiber.New()

	app.Use(middleware.RateLimiter(redisClient))

	app.Use(logger.New())

	app.Post("/shorten", h.Shorten)
	app.Get("/:code", h.Redirect)
	app.Get("/alias/check/:code", h.CheckAlias)

	log.Fatal(app.Listen(":8000"))
}
