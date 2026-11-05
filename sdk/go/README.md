# Yily Go SDK

Official lightweight Go SDK for retrieving configuration variables from [Yily](https://github.com/yannick2009/yily).

## Installation

```bash
go get github.com/yannick2009/yily/sdk/go
```

## Quickstart

```go
package main

import (
	"context"
	"log"
	"os"

	"github.com/yannick2009/yily/sdk/go"
)

type AppConfig struct {
	DBHost  string `yily:"DB_HOST"`
	DBPort  int    `yily:"DB_PORT"`
	Debug   bool   `yily:"DEBUG"`
	Timeout float64 `yily:"TIMEOUT"`
}

func main() {
	ctx := context.Background()

	client, err := yily.NewClient(yily.Config{
		BaseURL:       "http://localhost:16529",
		EnvironmentID: "your-environment-uuid",
		Token:         os.Getenv("YILY_TOKEN"),
	})
	if err != nil {
		log.Fatalf("failed to initialize Yily client: %v", err)
	}

	// 1. Load configuration into a struct
	var cfg AppConfig
	if err := client.Load(ctx, &cfg); err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	log.Println("Yily configuration loaded")

	// 2. Retrieve an individual variable by name
	apiKey, err := client.Get(ctx, "API_KEY")
	if err != nil {
		log.Fatalf("failed to get API_KEY: %v", err)
	}
	if apiKey == "" {
		log.Fatal("API_KEY is empty")
	}

	// 3. Retrieve a specific version of a variable
	legacyKey, err := client.Get(ctx, "API_KEY", yily.Version(1))
	if err != nil {
		log.Fatalf("failed to get legacy key: %v", err)
	}
	_ = legacyKey // Use the value in the application; do not log secrets.
}
```

Use a read-scoped PAT for applications that only need to load configuration. PATs are created and managed through Yily's web interface. `Load` makes one request for active values; if fields request historical versions, it makes one request for each distinct version number. A single `version=N` request applies that version to all names in that request.

Use HTTPS for deployed applications. The HTTP URL in the example is intended for local development only.

## Supported Struct Field Types

The loader automatically converts Yily variables into the following Go types:
- `string`
- `int`
- `int64`
- `float64`
- `bool`

Fields tagged with `yily:"-"` or without any `yily` tag are ignored.

## Error Handling

The SDK exposes sentinel errors that can be inspected with standard `errors.Is`:
- `yily.ErrInvalidBaseURL`: the instance URL is malformed or uses a scheme other than HTTP or HTTPS.
- `yily.ErrInvalidToken`: the configured token does not have the Yily PAT prefix.
- `yily.ErrVariableNotFound`: a requested variable does not exist.
- `yily.ErrUnauthorized`: invalid or expired token (HTTP 401).
- `yily.ErrNotFound`: environment or resource not found (HTTP 404).
- `yily.ErrUnsupportedType`: struct field has an unsupported Go type.
- `yily.ErrInvalidTarget`: `Load` was called with an invalid target (must be a non-nil pointer to a struct).
- `yily.ErrInvalidValue`: a value could not be converted to the requested Go type. The error does not include the value.
- `yily.ErrInvalidVariableName`: `Get` was called with an empty variable name after normalization.
- `yily.ErrInvalidVersion`: a requested version is not a positive integer.
- `yily.ErrInvalidTag`: a struct tag is malformed or contains an unsupported option.
- `yily.ErrDuplicateVariable`: one struct requests different versions of the same Yily variable.
