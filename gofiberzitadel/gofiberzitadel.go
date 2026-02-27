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
	"fmt"
	"slices"
	"log"
	"net/http"
	"net/url"
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

	// ClientSecret is the client secret used for token introspection (RFC 7662).
	// Required when handling opaque access tokens (non-JWT).
	//
	// Optional. Default: ""
	ClientSecret string

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
	//   "opaque"       - always use token introspection (for non-JWT tokens)
	//
	// Optional. Default: "auto"
	TokenType string
}

// Set the default configuration.
var ConfigDefault = Config{
	Next:                    nil,
	ProviderUrl:             "",
	ClientID:                "",
	ClientSecret:            "",
	StoreClaimsIndividually: false,
	TokenType:               "auto",
}

// detectTokenType inspects the unverified JWT payload to determine the token type.
// For non-JWT (opaque) tokens it returns "opaque" immediately.
//
// JWT detection heuristics (in order):
//  1. "scope" claim present → "access_token"
//  2. "nonce" or "at_hash" claim present → "id_token"
//  3. "aud" claim present but does not contain clientID → "access_token"
//  4. Default → "id_token"
func detectTokenType(rawToken, clientID string) string {
	parts := strings.Split(rawToken, ".")
	if len(parts) != 3 {
		return "opaque"
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "opaque"
	}

	var claims map[string]json.RawMessage
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "opaque"
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

	// If aud is present but doesn't contain clientID, it's an access token
	// whose audience is the resource server, not the application.
	if audRaw, ok := claims["aud"]; ok && clientID != "" {
		if !audContains(audRaw, clientID) {
			return "access_token"
		}
	}

	return "id_token"
}

// audContains checks whether a raw JSON "aud" claim (string or []string)
// contains the given clientID.
func audContains(audRaw json.RawMessage, clientID string) bool {
	var single string
	if json.Unmarshal(audRaw, &single) == nil {
		return single == clientID
	}
	var multi []string
	if json.Unmarshal(audRaw, &multi) == nil {
		return slices.Contains(multi, clientID)
	}
	return false
}

// introspectToken calls the RFC 7662 introspection endpoint with basic auth and
// returns the active token's claims. Returns an error if the token is inactive
// or the request fails.
func introspectToken(introspectionURL, clientID, clientSecret, token string) (map[string]any, error) {
	form := url.Values{}
	form.Set("token", token)

	req, err := http.NewRequest(http.MethodPost, introspectionURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("building introspection request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(clientID, clientSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling introspection endpoint: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding introspection response: %w", err)
	}

	active, _ := result["active"].(bool)
	if !active {
		return nil, fmt.Errorf("token is not active")
	}

	return result, nil
}

// New returns a Fiber middleware handler that validates tokens issued by a
// Zitadel OIDC provider. It supports ID tokens, JWT access tokens, and opaque
// access tokens (via token introspection), selected via Config.TokenType.
func New(config ...Config) fiber.Handler {
	cfg := ConfigDefault

	var idTokenVerifier *oidc.IDTokenVerifier
	var accessTokenVerifier *oidc.IDTokenVerifier
	var introspectionEndpoint string

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

		// Fetch the introspection endpoint from the OIDC discovery document.
		var disc struct {
			IntrospectionEndpoint string `json:"introspection_endpoint"`
		}
		if err := provider.Claims(&disc); err == nil {
			introspectionEndpoint = disc.IntrospectionEndpoint
		}
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
			effectiveType = detectTokenType(strToken, cfg.ClientID)
		}

		// Opaque token: use token introspection.
		if effectiveType == "opaque" {
			if cfg.ClientSecret == "" || introspectionEndpoint == "" {
				log.Printf("opaque token received but introspection is not configured (missing ClientSecret or introspection endpoint)")
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "Forbidden: Invalid token",
				})
			}

			claims, err := introspectToken(introspectionEndpoint, cfg.ClientID, cfg.ClientSecret, strToken)
			if err != nil {
				log.Printf("token introspection failed: %v", err)
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "Forbidden: Invalid token",
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

		// JWT token: use the appropriate verifier.
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

		var claims map[string]any
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
