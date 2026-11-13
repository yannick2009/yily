# Yily Go SDK — Architecture Proposal

## 1. Objective

Introduce a lightweight Go SDK that allows applications to retrieve configuration variables from Yily and use them directly in their Go applications.

The SDK aims to provide a simple and concise developer experience, following Yily's philosophy: **simple and effective**.

## 2. Scope

The SDK supports:

- Connecting to a Yily instance and selecting an environment.
- Authenticating with a pre-provisioned read-scoped PAT.
- Retrieving every variable of an environment.
- Retrieving individual variables by name.
- Loading values directly into Go structs.
- Automatically converting values into Go types.
- Optionally selecting a specific version for individual reads or Go struct fields.

It does not support variable management, environment administration, Yily's project or variable tag management, or access token lifecycle operations. Go struct tags are used only to map configuration fields to Yily variable names and to optionally select a version.

## 3. Project Structure

The SDK is maintained in a dedicated repository: [`github.com/yannick2009/yily-sdk-go`](https://github.com/yannick2009/yily-sdk-go).

```text
.
├── go.mod
├── go.sum
├── client.go
├── config.go
├── loader.go
├── errors.go
├── utils.go
├── doc.go
├── README.md
├── examples/
│   └── main.go
```

The module exposes a single public package named `yily`.

| File | Responsibility |
|---|---|
| `client.go` | Client type, public retrieval methods, and HTTP internals. |
| `config.go` | Client configuration, URL validation, and defaults. |
| `loader.go` | Struct inspection and value decoding used by `Load`. |
| `errors.go` | SDK error definitions. |
| `utils.go` | Variable-name and base-URL normalization. |
| `doc.go` | Package documentation and file map. |

## 4. Public API

The SDK provides four main capabilities:

- `NewClient(config Config) (*Client, error)` to initialize the client.
- `(*Client).Load(ctx context.Context, target any) error` to load multiple variables into a Go struct.
- `(*Client).Get(ctx context.Context, name string) (string, error)` to retrieve an individual variable's active value.
- `(*Client).GetVersion(ctx context.Context, name string, version int) (string, error)` to retrieve an individual variable at a specific version.
- `(*Client).GetAll(ctx context.Context) (map[string]string, error)` to retrieve every variable of the environment and its active value as a map.

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

These operations retrieve configuration values without requiring applications to handle HTTP communication directly.

## 5. Batch Retrieval

`Load` and `Get` retrieve values through one HTTP `POST` batch request. `GetAll` first lists the environment's variables, then retrieves their values through the same batch endpoint. The JSON body selects a version per variable, so a single request can mix active and historical versions.

The read-only (non-mutating) endpoint is:

```http
POST /api/v1/environments/{environmentID}/variables/values
Content-Type: application/json

{
  "variables": [
    { "name": "DB_HOST", "version": 2 },
    { "name": "DB_PORT" }
  ]
}
```

Authentication:

```http
Authorization: Bearer <TOKEN>
```

The response follows Yily's existing JSON envelope:

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

The `POST` body accepts between 1 and 1,000 variable selections, and a name may appear only once per request; duplicate selections are rejected with a 400 error. An omitted or zero version selects the active version. The response includes only variables that exist in the requested environment. A variable without an active version is returned with an empty string; a requested name that does not exist is omitted so the SDK can return `ErrVariableNotFound`. If an explicitly requested version is missing for an existing variable, the API returns the existing variable-version-not-found error.

## 6. Variable Version Selection

Specific versions are selected per struct field through the `yily` tag:

```go
type AppConfig struct {
    DBHost string `yily:"DB_HOST,version=2"`
}
```

An omitted version loads the active version. Active and versioned fields can be mixed in one struct.

For an individual read, use `GetVersion`:

```go
value, err := client.GetVersion(ctx, "DB_HOST", 2)
```

The loader sends every field in one `POST` batch request, each with its own optional version. If a struct maps the same Yily variable to fields with different requested versions, `Load` returns `ErrConflictingVariableVersion` rather than making the assignment ambiguous. `Get` reads the active version.

## 7. Loading and Type Conversion

The loader:

1. Inspects exported struct fields containing a `yily` tag.
2. Collects the requested variable names and versions.
3. Retrieves all requested values in one batch request, selecting each variable's version individually.
4. Converts values into the corresponding Go types.
5. Assigns values only if all conversions succeed.

Supported types include `string`, `int`, `int64`, `float64`, and `bool` (via `strconv`), plus `time.Duration` (via `time.ParseDuration`).

Missing variables return explicit errors. Existing variables with empty values are accepted. Fields without a `yily` tag are ignored, as are environment variables not mapped to a field. The target struct is updated only after every requested value has been retrieved and converted successfully. Conversion errors identify the variable and target type without including the secret value.

Tags may use the variable name alone or one supported option, such as `version=2`; unknown or repeated options return an error instead of silently changing the selection.

## 8. HTTP and Security

The SDK uses Go's standard `net/http` package without additional HTTP dependencies.

It supports:

- Configurable base URL and environment ID.
- Bearer token authentication.
- `context.Context` for request cancellation.
- A configurable timeout (5 seconds by default).
- Explicit HTTP and authentication errors.

Read-scoped PATs are provisioned outside the SDK through the web interface. The server enforces read and write scopes on routes; token creation and management are session-only. The SDK exposes only read operations.

Use HTTPS outside local development so PATs and retrieved values are protected in transit. Sensitive values must never appear in logs or error messages. The SDK returns values to the caller and does not log or persist them.

## 9. Testing

Planned test coverage:

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
