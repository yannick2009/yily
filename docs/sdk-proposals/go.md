# Yily Go SDK — Architecture Proposal

## 1. Objective

Introduce a lightweight Go SDK that allows applications to retrieve configuration variables from Yily and use them directly in their Go applications.

The SDK aims to provide a simple and concise developer experience, following Yily's philosophy: **simple and effective**.

## 2. Scope

The SDK will support:

- Connecting to a Yily instance.
- Authenticating with a pre-provisioned access token.
- Retrieving multiple variables in a single request.
- Retrieving individual variables.
- Loading values directly into Go structs.
- Automatically converting values into Go types.
- Selecting specific variable versions.

It will not support variable management, environment administration, or access token lifecycle operations.

## 3. Project Structure

The SDK will be integrated into the existing repository as an independent Go module under `sdk/go/`.

```text
sdk/
└── go/
    ├── go.mod
    ├── client.go
    ├── loader.go
    ├── decoder.go
    ├── errors.go
    └── loader_test.go
```

The module will expose a single public package named `yily`.

| File | Responsibility |
|---|---|
| `client.go` | Client configuration and HTTP communication. |
| `loader.go` | Struct inspection and variable retrieval. |
| `decoder.go` | Go type conversion. |
| `errors.go` | SDK error definitions. |
| `loader_test.go` | Loading and conversion tests. |

## 4. Public API

The SDK will provide three main capabilities:

- `NewClient(config Config)` to initialize the client.
- `Load(ctx context.Context, target any) error` to load multiple variables into a Go struct.
- `Get(ctx context.Context, name string)` to retrieve an individual variable.

Example struct:

```go
type Config struct {
    DBHost    string `yily:"DB_HOST"`
    DBPort    int    `yily:"DB_PORT"`
    JWTSecret string `yily:"JWT_SECRET"`
}
```

Example usage:

```go
var cfg Config

err := client.Load(context.Background(), &cfg)
if err != nil {
    log.Fatal(err)
}
```

Individual retrieval:

```go
value, err := client.Get(context.Background(), "DB_HOST")
if err != nil {
    log.Fatal(err)
}
```

Both operations will support retrieving configuration values without requiring applications to handle HTTP communication directly.

## 5. Batch Retrieval

To avoid individual requests for each variable, `Load()` will retrieve all requested values through a single HTTP `GET` request.

Proposed endpoint:

```http
GET /api/v1/environments/{environmentID}/variables/values?name=DB_HOST&name=DB_PORT
```

Authentication:

```http
Authorization: Bearer <TOKEN>
```

The response will follow Yily's existing JSON envelope:

```json
{
  "status": 200,
  "message": "Variable values retrieved successfully",
  "data": {
    "DB_HOST": "localhost",
    "DB_PORT": "5432"
  }
}
```

## 6. Variable Version Selection

The SDK will also support retrieving a specific version of a variable.

For example, individual retrieval could allow specifying a version:

```go
value, err := client.Get(ctx, "DB_HOST", yily.Version(2))
```

For struct loading, version selection could be expressed through the `yily` tag:

```go
type Config struct {
    DBHost string `yily:"DB_HOST,version=2"`
}
```

These examples illustrate possible API designs. The final mechanism for selecting versions, including its integration with struct tags, will be agreed upon separately.

When no version is specified, the SDK will retrieve the active version by default.

## 7. Loading and Type Conversion

The loader will:

1. Inspect exported struct fields containing a `yily` tag.
2. Collect the requested variable names.
3. Retrieve their values in a single request.
4. Convert values into the corresponding Go types.
5. Assign values only if all conversions succeed.

Supported types will include `string`, `int`, `int64`, `float64`, and `bool`, using Go's standard `strconv` package.

Missing variables will return explicit errors. Existing variables with empty values will be accepted. Fields without `yily` tags and undeclared variables will be ignored.

## 8. HTTP and Security

The SDK will use Go's standard `net/http` package without additional HTTP dependencies.

It will support:

- Configurable base URL and environment ID.
- Bearer token authentication.
- `context.Context` for request cancellation.
- A configurable timeout (5 seconds by default).
- Explicit HTTP and authentication errors.

Access tokens will be provisioned outside the SDK. Read-only permissions will be enforced server-side.

Sensitive values must never appear in logs or error messages.

## 9. Testing

Tests will cover:

- Client configuration.
- Individual and batch variable retrieval.
- Struct loading and type conversion.
- Missing variables and empty values.
- Version selection.
- HTTP and authentication errors.
- Request cancellation and timeout handling.

The standard `net/http/httptest` package will be used to simulate API responses.

## 10. Design Principles

- **Simplicity:** Prefer Go's standard library.
- **Minimalism:** Avoid unnecessary dependencies and abstractions.
- **Idiomatic Go:** Follow Go conventions.
- **Read-only:** Focus exclusively on configuration retrieval.
- **Reliability:** Provide predictable behavior and explicit errors.

The goal is a lightweight SDK that makes retrieving and loading configuration from Yily straightforward for Go applications.