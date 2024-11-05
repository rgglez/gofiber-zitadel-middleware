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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestZitadelMiddleware(t *testing.T) {
	// Get the test data
	providerUrl := os.Getenv("ZITADEL_PROVIDER")
	clientId := os.Getenv("ZITADEL_CLIENTID")
	token := os.Getenv("ZITADEL_TOKEN")
	validName := os.Getenv("ZITADEL_NAME")

	// Initialize Fiber app and middleware
	app := fiber.New()
	app.Use(New(Config{ProviderUrl: providerUrl, ClientID: clientId}))

	// Protected route to test the middleware
	app.Get("/", func(c *fiber.Ctx) error {
		claims := c.Locals("claims").(map[string]interface{})
		return c.JSON(claims)
	})

	validToken := "Bearer " + token
	invalidToken := "Bearer " + ""

	testCases := []struct {
		name           string
		token          string
		expectedStatus int
		expectedBody   string
	}{
		{"Valid token", validToken, http.StatusOK, validName},
		{"Invalid token", invalidToken, http.StatusUnauthorized, "Forbidden: Invalid token"},
		{"Missing token", "", http.StatusUnauthorized, "Forbidden: No token provided"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tc.token != "" {
				req.Header.Set("Authorization", tc.token)
			}

			resp, _ := app.Test(req)

			assert.Equal(t, tc.expectedStatus, resp.StatusCode)

			if tc.expectedStatus == http.StatusOK {
				var body map[string]interface{}
				_ = json.NewDecoder(resp.Body).Decode(&body)
				assert.Equal(t, tc.expectedBody, body["name"])
			} else {
				body := make([]byte, resp.ContentLength)
				resp.Body.Read(body)
				assert.Contains(t, string(body), tc.expectedBody)
			}
		})
	}
}
