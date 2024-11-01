# gofiber-zitadel-middleware

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
![GitHub all releases](https://img.shields.io/github/downloads/rgglez/gofiber-zitadel-middleware/total)
![GitHub issues](https://img.shields.io/github/issues/rgglez/gofiber-zitadel-middleware)
![GitHub commit activity](https://img.shields.io/github/commit-activity/y/rgglez/gofiber-zitadel-middleware)
[![Go Report Card](https://goreportcard.com/badge/github.com/rgglez/gofiber-zitadel-middleware)](https://goreportcard.com/report/github.com/rgglez/gofiber-zitadel-middleware)
[![GitHub release](https://img.shields.io/github/release/rgglez/gofiber-zitadel-middleware.svg)](https://github.com/rgglez/gofiber-zitadel-middleware/releases/)


**gofiber-zitadel-middleware** is a [gofiber](https://gofiber.io/) [middleware](https://docs.gofiber.io/category/-middleware/) to be used along with the [Zitadel](https://zitadel.com/) (and perhaps other [OIDC](https://auth0.com/es/intro-to-iam/what-is-openid-connect-oidc) servers) security manager to verify the [JWT token](https://jwt.io/) provided by it in the corresponding flows.

## Installation

```bash
go get github.com/rgglez/gofiber-zitadel-middleware
```

## Usage

```go
import gofiberzitadel

// Initialize Fiber app and middleware
app := fiber.New()
app.Use(gofiberzitadel.New(gofiberzitadel.Config{ProviderUrl: providerUrl, ClientID: clientId}))
```

## Configuration

There are some configuration options available in the ```Config``` struct:

* **```Next```** defines a function to skip this middleware when returned true. Optional. Default: nil
* **```ProviderUrl```** a string which defines the URL of the Zitadel instance. Required.
* **```ClientID```** a string which defines the ```client_id``` of the application to be used in the validation. Required.
* **```StoreClaimsIndividually```** a boolean which defines if the claims should be stored as key:value pairs in the fiber context. Optional. Default: false
The claims are stored in the fiber context as "claims" by default. 


## Testing

A test is included. To run the test you must:

1. Setup a working Zitadel instance, either self-hosted of as SaaS. You will need the URL of this instance, as the **Provider URL**.
1. Setup a Zitadel application in your instance. You will need the [**Client ID**](https://zitadel.com/docs/guides/manage/console/applications#application-settings) of this application.
1. Create a human user and write down the user's **name**. You will need it for the assertion of the claims.
1. Optionally, create an application which will be using the Zitadel provider for authentication. You can use [this](https://github.com/rvs1257/svelte-zitadel-pkce) Svelte application as the basis. You will need to login into a real or sample application in order to get the **```id_token```** field from the JSON returned by the ```/token``` endpoint.
Otherwise you would need to use the Zitadel API to get this token manually.
1. Set the test data in the enviroment. An example bash script is provided in ```tests/test_data.sh``` as a guide. You must fill in the values with your own data accordingly:

    ```bash
    # The full URL including trailing / of your Zitadel instance
    export ZITADEL_PROVIDER=
    # The client_id of the Zitadel application
    export ZITADEL_CLIENTID=
    # A token got from a valid login 
    export ZITADEL_TOKEN=
    # The "name" of the logged user
    export ZITADEL_NAME=
    ```
    If you use this script, you should need to [source](https://www.geeksforgeeks.org/source-command-in-linux-with-examples/) it.

1. Run
    ```bash
    go test
    ```
    inside the ```src/``` directory.

## Dependencies

* [github.com/coreos/go-oidc](https://github.com/coreos/go-oidc)
* [github.com/gofiber/fiber/v2](https://github.com/gofiber/fiber/v2)

## License

Copyright (c) 2024 Rodolfo González González

Licensed under the [Apache 2.0](LICENSE) license. Read the [LICENSE](LICENSE) file.
