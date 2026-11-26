# Contributing to Yily

Thank you for your interest in contributing to Yily. Bug reports, feature ideas, documentation improvements, and code contributions are welcome.

## Who can contribute?

Yily is especially looking for contributors interested in:

- **Backend development** with Go, APIs, and data storage.
- **Frontend development** with SolidJS and Tailwind CSS.
- **Security**, authentication, and secret management.
- **Nix and developer tooling** to improve the development experience.
- **Documentation, testing, and bug reports**.

You do not need to be an expert or cover all these areas. Choose a subject you know, or start with a small issue to learn the project.

## Before you start

- Open an issue before working on a large change.
- Keep changes focused and easy to review.
- Do not include secrets, credentials, or personal data in commits.

## Development

Nix provides the development environment and all required tools. First, fork the Yily repository on GitHub. Then install Nix and clone your fork:

```bash
git clone https://github.com/<your-github-username>/yily.git
cd yily
git remote add upstream https://github.com/yannick2009/yily.git
nix develop
```

The `nix develop` command makes Go, Node.js, Just, Hivemind, and the Go development tools available. It also installs the client dependencies and prepares the server dependencies automatically.

Create your branch from the up-to-date default branch before making changes:

```bash
git fetch upstream
git switch -c <type>/<short-description> upstream/main
```

Start the client and server from the repository root:

```bash
just start
```

Run the linters often while you work, and always before pushing. This formats the client, formats and vets the server, and catches most issues before CI does:

```bash
just lint   # or the alias: just l
```

Before opening a pull request, run the server checks:

```bash
just test
```

## Code style

### Backend (Go)

- Keep packages small and named after what they provide (`httputil`, `authctx`, `env`). Avoid catch-all packages such as `utils`, `common` or `helpers`.
- Order declarations by importance: **the major exported types, functions and methods come first; the unexported helpers that serve them go below**, in the order they are used. A reader should meet the public surface before the plumbing.
- `internal/domain/model` describes persisted data only. Infrastructure concerns such as schema migration live in `internal/adapter`.
- `internal/testutil` is test-only: production packages must never import it.
- Every exported identifier gets a doc comment starting with its name. Comments explain why, not what the code already says.
- Return sentinel errors from the package that owns them and wrap them with `%w` or `errors.Join`, so callers can keep using `errors.Is`.

### Frontend (client)

- **Layers.** All HTTP goes through `src/api/client.ts`. Each resource has one module in `src/api/`, re-exported as a namespace by `src/api/index.ts`. Routes and components never call `fetch` directly.
- **Routes and components.** One page per `src/routes/<Name>Route.tsx`, exported as a named function. Generic primitives live in `src/components/ui/`, cross-feature components in `src/components/shared/`, feature components in `src/components/<feature>/`; each folder exposes an `index.ts` barrel and imports go through it.
- **State.** `src/store/index.ts` holds app-wide boot, auth and version state only. Page data is fetched in the route that needs it with `createResource`.
- **Types.** `src/types/index.ts` mirrors the server JSON exactly (snake_case, no renaming). Main entities come first and composite types at the bottom, following the same order-of-importance rule as Go.
- **Naming and exports.** Component and route files are `PascalCase.tsx` with named exports; `App` is the only default export because `src/index.tsx` requires it. Props are `<Component>Props`, helpers in `src/lib/` are `camelCase.ts`.
- **Solid rules.** Use `createSignal`, `createMemo` and `createResource`. Do not destructure props (it breaks reactivity); use `splitProps` when a component forwards or merges them. Inside a file, the main component comes first and local helpers go below it.
- **TypeScript.** `erasableSyntaxOnly` is enabled: no `enum` and no `namespace`. Use string-literal unions or `as const`, and import types with `import type`.
- **Text and i18n.** Every user-facing string goes through `t()` or `plural()` from `src/lib/i18n.ts`, with a flat `<feature>.<key>` key present in all 10 languages. Display API errors with `errorMessage(err)` so stable server codes are translated.
- **Styling.** Use Tailwind utilities and the design tokens defined in `src/index.css` (`bg-surface`, `text-ink`, `border-line`, `brand`, `danger`, …). Keep inline `style` for dynamic values that utilities cannot express.
- **Tooling.** Run `just lint` (`vp check --fix`) to format and lint the client, and `cd client && npm run build` to type-check and build the SPA.

## Testing

Tests are part of the review: a bug fix or a feature comes with the test that proves it, and CI (`just check`) must stay green.

### Test file layout

- Put tests next to the code in `<file>_test.go`, using the same package (`package repository`) so they can exercise unexported helpers. Use an external `<pkg>_test` package only when you intentionally test the public API from the outside.
- Name tests `Test_<Subject>_<Behavior>` (for example `Test_GeneratePAT` or `Test_VariableService_Create_PreservesWhitespace`). Use `t.Run` subtests for table-driven cases.
- Prefer `require` from `testify` when a failure makes the rest of the test meaningless, and `assert` for independent checks.

### Database-backed tests

Use `testutil.NewDB(t)` instead of opening a database yourself:

```go
package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yannick2009/yily/internal/domain/model"
	"github.com/yannick2009/yily/internal/testutil"
)

func Test_UserRepository_Create(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	repo := NewUserRepository(db)

	user := &model.User{Name: "Admin", Email: "admin@example.com", PasswordHash: "hash"}
	require.NoError(t, repo.Create(context.Background(), user))

	got, err := repo.GetByID(context.Background(), user.ID)
	require.NoError(t, err)
	require.Equal(t, user.Email, got.Email)
}
```

`NewDB(t)` returns an isolated in-memory SQLite database migrated with the production schema, with foreign keys enabled and `TranslateError` on; it registers a `t.Cleanup` that closes the connection. Each call gets its own database, so `t.Parallel()` is safe. Never read or write `instance/database.db` from a test.

### Rules

- Do not add a mocking framework. Exercise real repositories against SQLite; hand-write small fakes for time and audit when a test needs them.
- Tests must be deterministic: no `time.Sleep`, no network, no fixed ports. Use `t.TempDir()` for files and a bounded retry loop (deadline + backoff) when waiting for a condition.
- Tests must leave the working tree clean: no database files, exports or temporary artifacts outside `t.TempDir()`.
- Close what you open and register cleanup with `t.Cleanup`. Test helpers take `t *testing.T`, call `t.Helper()`, and use `t.Fatalf` instead of `panic`.
- Coverage is a diagnostic, not a target. Test behavior and security invariants, not lines.

### Running tests

```bash
just test                                                              # full suite, -race + coverage
cd server && go test -race ./internal/crypto                           # one package
cd server && go test -race ./internal/crypto -run Test_GeneratePAT -v  # one test
```

After `just test`, open `server/coverage/coverage.html` for the line-by-line report.

End-to-end tests that need the built binary use the `e2e` build tag so they stay out of `just test`, and run in a dedicated CI step (see issue #91).

## Branches

Every branch must use a type prefix and a short kebab-case description:

```text
<type>/<short-description>
```

Use one of these types:

- `feat`: new feature
- `fix`: bug fix
- `chore`: maintenance or configuration
- `docs`: documentation
- `refactor`: code change without behavior change
- `test`: tests

Examples: `feat/secret-sharing`, `fix/login-error`, `docs/update-readme`.

## Commits

Every commit must use the same type prefix:

```text
<type>: <short description>
```

Examples: `feat: add secret sharing` or `chore: update dependencies`.

## Pull requests

- Explain what changed and why.
- Mention the issue related to the change, if there is one.
- Include the checks you ran.
- Keep the pull request focused on one subject.
