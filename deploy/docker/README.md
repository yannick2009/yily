# Yily — direct access (Docker)

Local deployment without a reverse proxy: port `52015` is exposed directly.

## 1. Generate the secrets

Create a `.env` file in this folder:

```bash
cd deploy/docker
printf 'YILY_MASTER_KEY=%s\nYILY_SETUP_TOKEN=%s\n' \
  "$(openssl rand -base64 32)" "$(openssl rand -base64 32)" > .env
```

## 2. Start

```bash
docker compose up -d
```

## 3. Initialize

Open <http://localhost:52015> and create the Super Admin, pasting the
`YILY_SETUP_TOKEN` from `.env`.

## Notes

- The SQLite database is persisted in the `yily-data` volume (`/data/instance`).
- This mode serves plain HTTP: use it only on a trusted network, or behind a
  tunnel/VPN. For public HTTPS access, see [`../caddy`](../caddy) or
  [`../nginx`](../nginx).
- `.env` is ignored by git (see the root `.gitignore`).
