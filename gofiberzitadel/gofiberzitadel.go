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

package gofiberzitadel

import (
	"context"
	"log"
	"strings"

	oidc "github.com/coreos/go-oidc"
	fiber "github.com/gofiber/fiber/v2"
)

type Config struct {
	// Next defines a function to skip this middleware when returned true.
	//
	// Optional. Default: nil
	Next func(c *fiber.Ctx) bool

	// ProviderUrl defines the URL of the Zitadel instance
	//
	// Required
	ProviderUrl string

	// ClientID defines the client_id of the application to be used in the
	// validation.
	//
	// Required
	ClientID string

	// StoreClaimsIndividually defines if the claims should be stored
	// as key:value pairs in the fiber context.
	//
	// Optional. Default: false
	StoreClaimsIndividually bool
}

// Set the default configuration.
var ConfigDefault = Config{
	Next:                    nil,
	ProviderUrl:             "",
	ClientID:                "",
	StoreClaimsIndividually: false,
}

// Middleware.
func New(config ...Config) fiber.Handler {
	cfg := ConfigDefault

	var verifier *oidc.IDTokenVerifier

	if len(config) > 0 {
		cfg = config[0]

		// Obtain the provider
		provider, err := oidc.NewProvider(context.Background(), cfg.ProviderUrl)
		if err != nil {
			log.Fatalf("can not obtain the OIDC provider: %v", err)
		}
		// Obtain the verifier
		verifier = provider.Verifier(&oidc.Config{ClientID: cfg.ClientID})
	} else {
		log.Fatalf("missconfigured middleware")
	}
	if verifier == nil {
		log.Fatalf("missconfigured middleware")
	}

	return func(c *fiber.Ctx) error {
		// Should we pass?
		if cfg.Next != nil && cfg.Next(c) {
			return c.Next()
		}

		// Get the token from the Authorization header
		bearer := c.Get("Authorization")
		strToken := strings.TrimPrefix(bearer, "Bearer ")

		// If the token is not provided, return a 401 status
		if strToken == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Forbidden: No token provided"})
		}

		// Verify the token using a OIDC IDTokenVerifier
		token, err := verifier.Verify(context.Background(), strToken)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Forbidden: Invalid token"})
		}

		// Obtain the claims
		var claims map[string]interface{}
		if err := token.Claims(&claims); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error: Can not obtain claims"})
		}

		// Store all the claims (by default)
		c.Locals("claims", claims)

		// Store individual claims if needed
		if cfg.StoreClaimsIndividually {
			for key, value := range claims {
				c.Locals(key, value)
			}
		}

		return c.Next()
	}
}
