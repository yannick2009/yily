# Yily Go SDK — Architecture Proposal

## 1. Objective

Introduce a lightweight Go SDK that allows applications to retrieve configuration variables from Yily and use them directly in their Go applications.

The SDK aims to provide a simple and concise developer experience, following Yily's philosophy: **simple and effective**.

## 2. Scope

The SDK will support:

- Connecting to a Yily instance and selecting an environment.
- Authenticating with a pre-provisioned read-scoped PAT.
- Retrieving multiple active variables in a single request.
- Retrieving individual variables.
- Loading values directly into Go structs.
- Automatically converting values into Go types.
- Optionally selecting a specific version for individual reads or Go struct fields.

It will not support variable management, environment administration, Yily's project or variable tag management, or access token lifecycle operations. Go struct tags are used only to map configuration fields to Yily variable names.

## 3. Project Structure

The SDK will be integrated into the existing repository as an independent Go module under `sdk/go/`.

```text
sdk/
└── go/
    ├── go.mod
    ├── client.go
    ├── config.go
    ├── loader.go
    ├── decoder.go
    ├── errors.go
    ├── README.md
    ├── examples/
    │   └── main.go
    ├── client_test.go
    ├── config_test.go
    └── loader_test.go
```

The module will expose a single public package named `yily`.

| File | Responsibility |
|---|---|
| `client.go` | HTTP communication and individual value reads. |
| `config.go` | Client configuration, URL validation, and defaults. |
| `loader.go` | Struct inspection and variable retrieval. |
| `decoder.go` | Go type conversion. |
| `errors.go` | SDK error definitions. |
| `*_test.go` | Client, configuration, loading, and conversion tests. |

## 4. Public API

The SDK will provide three main capabilities:

- `NewClient(config Config) (*Client, error)` to initialize the client.
- `(*Client).Load(ctx context.Context, target any) error` to load multiple variables into a Go struct.
- `(*Client).Get(ctx context.Context, name string, opts ...GetOption) (string, error)` to retrieve an individual variable.

`Config` contains the instance `BaseURL`, `EnvironmentID`, a read-scoped PAT, an optional `HTTPClient`, and an optional request timeout (5 seconds by default). The SDK does not load credentials from environment variables itself.

Example struct:

```go
type AppConfig struct {
    DBHost    string `yily:"DB_HOST"`
    DBPort    int    `yily:"DB_PORT"`
    JWTSecret string `yily:"JWT_SECRET"`
}
```

Example usage:

```go
var cfg AppConfig

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

For the common case where all fields use active versions, `Load()` retrieves the requested values through one HTTP `GET` request. If fields request historical versions, it makes one request for each distinct version number.

The read-only endpoint is:

```http
GET /api/v1/environments/{environmentID}/variables/values?name=DB_HOST&name=DB_PORT
```

An optional `version` query parameter selects that version for every name in the request:

```http
GET /api/v1/environments/{environmentID}/variables/values?name=DB_HOST&name=DB_PORT&version=2
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

The `name` parameter is repeatable and at least one name is required. The response includes only variables that exist in the requested environment. A variable without an active version is returned with an empty string; a requested name that does not exist is omitted so the SDK can return `ErrVariableNotFound`. If an explicitly requested version is missing for an existing variable, the API returns the existing variable-version-not-found error.

## 6. Variable Version Selection

The SDK supports retrieving a specific positive version number.

An individual read can select a specific version:

```go
value, err := client.Get(ctx, "DB_HOST", yily.Version(2))
```

For struct loading, version selection could be expressed through the `yily` tag:

```go
type AppConfig struct {
    DBHost string `yily:"DB_HOST,version=2"`
}
```

For struct loading, fields are grouped by version. A request without a version returns the active version; each positive version group is sent in a separate request. The endpoint does not mix different version numbers in one request.

If a struct maps the same Yily variable to fields with different requested versions, `Load` returns an error rather than making the assignment ambiguous.

When no version is specified, the SDK will retrieve the active version by default.

## 7. Loading and Type Conversion

The loader will:

1. Inspect exported struct fields containing a `yily` tag.
2. Collect the requested variable names.
3. Retrieve active values in one request, or make one request per distinct explicitly selected version.
4. Convert values into the corresponding Go types.
5. Assign values only if all conversions succeed.

Supported types will include `string`, `int`, `int64`, `float64`, and `bool`, using Go's standard `strconv` package.

Missing variables will return explicit errors. Existing variables with empty values will be accepted. Fields without `yily` tags and undeclared variables will be ignored. The target struct is updated only after every requested value has been retrieved and converted successfully. Conversion errors identify the variable and target type without including the secret value.

Tags may use the variable name alone or one supported option, such as `version=2`; unknown or repeated options will return an error instead of silently changing the selection.

## 8. HTTP and Security

The SDK will use Go's standard `net/http` package without additional HTTP dependencies.

It will support:

- Configurable base URL and environment ID.
- Bearer token authentication.
- `context.Context` for request cancellation.
- A configurable timeout (5 seconds by default).
- Explicit HTTP and authentication errors.

Read-scoped PATs are provisioned outside the SDK through the web interface. The server enforces read and write scopes on routes; token creation and management are session-only. The SDK exposes only read operations.

Use HTTPS outside local development so PATs and retrieved values are protected in transit. Sensitive values must never appear in logs or error messages. The SDK returns values to the caller and does not log or persist them.

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
