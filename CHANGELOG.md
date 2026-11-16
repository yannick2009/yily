# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- `yily-cli environments export` now writes a new timestamped file (`export_<name>_<YYYYMMDD-HHMMSS>.<ext>`) when no output file is given, in the current directory or in the `-o <directory>`. It never overwrites an existing file: on a name collision it adds `-2`, `-3`, and so on. An explicit `-o <file>` still overwrites that file.

### Security

- Password recovery now revokes every active session **and every unexpired personal access token**, together with the password and recovery code change, in a single transaction. A failed reset changes nothing, and old sessions can no longer survive a reset that reported success. CLI and CI integrations need new tokens after a reset.
- `yily-cli environments export` no longer lets the server choose which file it writes. The server's filename is used only when it matches `export_<name>.<ext>` for the requested format; otherwise the CLI falls back to `export_<environment id>`. A compromised server can no longer overwrite files such as `Makefile` or `.bashrc`.
- Export filenames replace every character other than letters, digits, `.`, `-` and `_` with `_` (e.g. `prod/eu-west` → `export_prod_eu-west.env`). An environment name containing `/` could previously make the CLI write a file other than `export_*`, such as `.env`.
- Audit log CSV export prefixes cells starting with `=`, `+`, `-`, `@`, a tab or a carriage return with `'`, preventing spreadsheet formula injection.
- Environment descriptions reject non-printable characters, so they cannot carry terminal escape sequences into CLI output (new error code `ENVIRONMENT_DESCRIPTION_INVALID_CHARS`).

## [1.0.0-beta.2] - 2026-11-15

### Added

- Password reset via a one-time recovery code downloaded as a file at account setup (`/recover` and `/recovery-code`).
- Deployment templates under `deploy/`: Docker Compose (direct), Caddy, and nginx reverse-proxy setups.
- Complete translations for all languages (en, fr, es, de, it, pt, ru, zh, ja, ko).

### Changed

- Personal access tokens shortened from 256 to 192 bits of entropy (existing tokens keep working).
- Server port moved to `52015`.
- Stronger password policy: 12–128 characters with upper/lower/digit/special complexity.

### Security

- CSRF protection for state-changing requests.
- First-run setup gated by an out-of-band setup token (`YILY_SETUP_TOKEN`).
- Session cookies set to `SameSite=Strict`.
- Rate limiting now keys on the direct peer IP instead of forwarding headers.
- Variable version activation scoped to its owning variable (fixes an IDOR).
- Name validation for projects, environments, variables, and tags.

### Fixed

- `SETUP_TOKEN_NOT_CONFIGURED` now returns `400` instead of `500`.
