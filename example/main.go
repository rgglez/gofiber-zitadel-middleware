/*
Copyright 2024 Rodolfo González González

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

// This setups a server listening on port :3000 on all available IPs. 
// You need to call it with something such as RESTing or curl, passing
// a valid (or invalid) JWT in a GET request.

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
