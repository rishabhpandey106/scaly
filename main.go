package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("env file not loaded")
	}

	db, err := ConnectDB(os.Getenv("DATABASE_URL"))
	if err != nil {
		panic(err)
	}

	if db == nil {
		panic("DB is nil")
	}

	err = CreateTable(db)
	if err != nil {
		panic(err)
	}

	app := fiber.New()
	app.Use(logger.New())

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("db", db)
		return c.Next()
	})

	app.Post("/shorten", ShortenHandler)
	app.Get("/:code", RedirectHandler)

	log.Fatal(app.Listen(":8000"))
}
