# Yily SDK Guidelines — Cross-Language Specification

## 1. Objective

Define how every Yily SDK must be written, regardless of the programming language. Each SDK lets applications retrieve configuration variables from Yily and use them directly in their own code.

Every SDK must provide a simple and concise developer experience, following Yily's philosophy: **simple and effective**.

## 2. Scope

Every SDK must support:

- Connecting to a Yily instance and selecting an environment.
- Authenticating with a pre-provisioned read-scoped PAT.
- Retrieving every variable of an environment.
- Retrieving individual variables by name.
- Loading values directly into native data structures (structs, classes, records, or the language's equivalent).
- Automatically converting values into the language's native types.
- Optionally selecting a specific version for individual reads or per-field mappings.

SDKs must **not** support variable management, environment administration, Yily's project or variable tag management, or access token lifecycle operations. Mapping metadata (struct tags, decorators, annotations, attributes, or the language's equivalent) is used only to map configuration fields to Yily variable names and to optionally select a version.

## 3. Repository and Structure

Each SDK lives in a dedicated repository named after its language:

```text
github.com/yannick2009/yily-sdk-<language>
```

Regardless of language, every SDK exposes a single public package, module, or namespace named `yily`, organized around the same responsibilities:

| Component | Responsibility |
|---|---|
| Client | Client type, public retrieval methods, and HTTP internals. |
| Configuration | Client configuration, base-URL validation, and defaults. |
| Loader | Structure inspection and value decoding used by the load operation. |
| Errors | SDK error definitions. |
| Utilities | Variable-name and base-URL normalization. |
| Documentation | Package documentation and a file/component map. |

File and module names follow the language's conventions; the responsibilities above must remain identifiable.

## 4. Public API

Every SDK must expose the same set of capabilities, expressed idiomatically in its language:

- `NewClient(config)` to initialize the client.
- `Load(ctx, target)` to load multiple variables into a native data structure.
- `Get(ctx, name)` to retrieve an individual variable's active value.
- `GetVersion(ctx, name, version)` to retrieve an individual variable at a specific version.
- `GetAll(ctx)` to retrieve every variable of the environment and its active value as a map or dictionary.

`Config` (or the equivalent configuration type) contains the instance `BaseURL`, `EnvironmentID`, a read-scoped PAT, an optional HTTP client, and an optional request timeout (5 seconds by default). SDKs must **not** load credentials from environment variables themselves.

Example structure mapping (idiomatic to the language):

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
```

Individual retrieval:

```go
value, err := client.Get(context.Background(), "DB_HOST")
```

These operations retrieve configuration values without requiring applications to handle HTTP communication directly.

## 5. Batch Retrieval

The load and single-get operations retrieve values through one HTTP `POST` batch request. The get-all operation first lists the environment's variables, then retrieves their values through the same batch endpoint. The JSON body selects a version per variable, so a single request can mix active and historical versions.

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

The `POST` body accepts between 1 and 1,000 variable selections, and a name may appear only once per request; duplicate selections are rejected with a 400 error. An omitted or zero version selects the active version. The response includes only variables that exist in the requested environment. A variable without an active version is returned with an empty string; a requested name that does not exist is omitted so the SDK can return a variable-not-found error. If an explicitly requested version is missing for an existing variable, the API returns the existing variable-version-not-found error.

## 6. Variable Version Selection

Specific versions are selected per field through the language's mapping metadata:

```go
type AppConfig struct {
    DBHost string `yily:"DB_HOST,version=2"`
}
```

An omitted version loads the active version. Active and versioned fields can be mixed in one structure.

For an individual read, use the get-version operation:

```go
value, err := client.GetVersion(ctx, "DB_HOST", 2)
```

The loader sends every field in one `POST` batch request, each with its own optional version. If a structure maps the same Yily variable to fields with different requested versions, the load operation returns a conflicting-variable-version error rather than making the assignment ambiguous. The single-get operation reads the active version.

## 7. Loading and Type Conversion

The loader, in every language:

1. Inspects exported fields containing Yily mapping metadata.
2. Collects the requested variable names and versions.
3. Retrieves all requested values in one batch request, selecting each variable's version individually.
4. Converts values into the language's native types.
5. Assigns values only if all conversions succeed.

Supported types map to each language's native equivalents: strings, integers (including 64-bit), floating-point numbers, booleans, and durations.

Missing variables return explicit errors. Existing variables with empty values are accepted. Fields without Yily mapping metadata are ignored, as are environment variables not mapped to a field. The target structure is updated only after every requested value has been retrieved and converted successfully. Conversion errors identify the variable and target type without including the secret value.

Metadata may use the variable name alone or one supported option, such as `version=2`; unknown or repeated options return an error instead of silently changing the selection.

## 8. HTTP and Security

Each SDK uses its language's standard HTTP client without additional HTTP dependencies.

Every SDK must support:

- Configurable base URL and environment ID.
- Bearer token authentication.
- Request cancellation through the language's context or cancellation mechanism.
- A configurable timeout (5 seconds by default).
- Explicit HTTP and authentication errors.

Read-scoped PATs are provisioned outside the SDK through the web interface. The server enforces read and write scopes on routes; token creation and management are session-only. SDKs expose only read operations.

Use HTTPS outside local development so PATs and retrieved values are protected in transit. Sensitive values must never appear in logs or error messages. SDKs return values to the caller and must not log or persist them.

## 9. Testing

Every SDK must include test coverage for:

- Client configuration.
- Individual and batch variable retrieval.
- Structure loading and type conversion.
- Missing variables and empty values.
- Version selection.
- HTTP and authentication errors.
- Request cancellation and timeout handling.

Tests should use the language's standard HTTP test server to simulate API responses.

## 10. Design Principles

All SDKs must follow the same principles:

- **Simplicity:** Prefer the language's standard library.
- **Minimalism:** Avoid unnecessary dependencies and abstractions.
- **Idiomatic:** Follow the conventions of the target language.
- **Read-only:** Focus exclusively on configuration retrieval.
- **Reliability:** Provide predictable behavior and explicit errors.

The goal is a lightweight SDK that makes retrieving and loading configuration from Yily straightforward in any language.
