package main

import (
	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()
	app.Use(gofiberzitadel.New(gofiberzitadel.Config{ProviderUrl: providerUrl, ClientID: clientId}))

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello world")
	})

	app.Listen(":3000")
}
