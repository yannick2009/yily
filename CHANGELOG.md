# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
