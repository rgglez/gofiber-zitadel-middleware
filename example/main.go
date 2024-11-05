package main

import (
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/rgglez/gofiber-zitadel-middleware/gofiberzitadel"
)

func main() {
	providerUrl := os.Getenv("ZITADEL_PROVIDER")
	clientId := os.Getenv("ZITADEL_CLIENTID")

	app := fiber.New()
	app.Use(gofiberzitadel.New(gofiberzitadel.Config{ProviderUrl: providerUrl, ClientID: clientId}))

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello world")
	})

	app.Listen(":3000")
}
