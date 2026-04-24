package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"

	"url-shortener/config"
	"url-shortener/handler"
	"url-shortener/repository"
	"url-shortener/service"
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

	repo := repository.NewURLRepo(db)
	svc := service.NewURLService(repo)
	h := handler.NewURLHandler(svc)

	app := fiber.New()
	app.Use(logger.New())

	app.Post("/shorten", h.Shorten)
	app.Get("/:code", h.Redirect)

	log.Fatal(app.Listen(":8000"))
}
