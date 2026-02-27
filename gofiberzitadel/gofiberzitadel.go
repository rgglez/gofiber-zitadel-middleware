/*
	Copyright 2026 Rodolfo González González

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
	"encoding/base64"
	"encoding/json"
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

	// TokenType controls which verifier is used for incoming tokens.
	// Valid values:
	//   "auto"         - detect token type from payload claims (default)
	//   "id_token"     - always use the ID token verifier (validates aud == ClientID)
	//   "access_token" - always use the access token verifier (skips aud check)
	//
	// Optional. Default: "auto"
	TokenType string
}

// Set the default configuration.
var ConfigDefault = Config{
	Next:                    nil,
	ProviderUrl:             "",
	ClientID:                "",
	StoreClaimsIndividually: false,
	TokenType:               "auto",
}

// detectTokenType inspects the unverified JWT payload to determine whether the
// raw token is an access token or an ID token. It does NOT verify the signature.
//
// Detection heuristics (in order):
//  1. "scope" claim present → access token
//  2. "nonce" or "at_hash" claim present → ID token
//  3. Default → "id_token"
func detectTokenType(rawToken string) string {
	parts := strings.Split(rawToken, ".")
	if len(parts) != 3 {
		return "id_token"
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "id_token"
	}

	var claims map[string]json.RawMessage
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "id_token"
	}

	if _, ok := claims["scope"]; ok {
		return "access_token"
	}
	if _, ok := claims["nonce"]; ok {
		return "id_token"
	}
	if _, ok := claims["at_hash"]; ok {
		return "id_token"
	}

	return "id_token"
}

// New returns a Fiber middleware handler that validates JWTs issued by a
// Zitadel OIDC provider. It supports both ID tokens and access tokens,
// selected via Config.TokenType ("auto", "id_token", or "access_token").
func New(config ...Config) fiber.Handler {
	cfg := ConfigDefault

	var idTokenVerifier *oidc.IDTokenVerifier
	var accessTokenVerifier *oidc.IDTokenVerifier

	if len(config) > 0 {
		cfg = config[0]

		if cfg.TokenType == "" {
			cfg.TokenType = "auto"
		}

		provider, err := oidc.NewProvider(context.Background(), cfg.ProviderUrl)
		if err != nil {
			panic("gofiber-zitadel-middleware: cannot obtain the OIDC provider: " + err.Error())
		}

		// ID token verifier: audience must contain ClientID.
		idTokenVerifier = provider.Verifier(&oidc.Config{ClientID: cfg.ClientID})

		// Access token verifier: skip audience check — access tokens carry the
		// resource server identifier in aud, not the ClientID.
		accessTokenVerifier = provider.Verifier(&oidc.Config{SkipClientIDCheck: true})
	} else {
		panic("gofiber-zitadel-middleware: misconfigured middleware")
	}

	if idTokenVerifier == nil || accessTokenVerifier == nil {
		panic("gofiber-zitadel-middleware: misconfigured middleware")
	}

	return func(c *fiber.Ctx) error {
		if cfg.Next != nil && cfg.Next(c) {
			return c.Next()
		}

		authValues, ok := c.GetReqHeaders()["Authorization"]
		if !ok || len(authValues) == 0 || authValues[0] == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authorization header missing",
			})
		}
		authHeader := authValues[0]

		if len(authHeader) < 7 || strings.ToUpper(authHeader[0:6]) != "BEARER" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authorization header malformed",
			})
		}

		strToken := strings.TrimSpace(authHeader[7:])
		if strToken == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Forbidden: No token provided",
			})
		}

		effectiveType := cfg.TokenType
		if effectiveType == "auto" {
			effectiveType = detectTokenType(strToken)
		}

		var verifier *oidc.IDTokenVerifier
		switch effectiveType {
		case "access_token":
			verifier = accessTokenVerifier
		default:
			verifier = idTokenVerifier
		}

		token, err := verifier.Verify(context.Background(), strToken)
		if err != nil {
			log.Printf("can not verify the token: %v", err)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Forbidden: Invalid token",
			})
		}

		var claims map[string]interface{}
		if err := token.Claims(&claims); err != nil {
			log.Printf("can not get claims")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error: Can not obtain claims",
			})
		}

		c.Locals("claims", claims)

		if cfg.StoreClaimsIndividually {
			for key, value := range claims {
				c.Locals(key, value)
			}
		}

		return c.Next()
	}
}
