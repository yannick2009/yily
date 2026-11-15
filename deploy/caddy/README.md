# Yily + Caddy (automatic HTTPS)

Caddy obtains and renews TLS certificates on its own: no certificate handling is
required.

## 1. Generate the secrets and set the domain

Create a `.env` file in this folder:

```bash
cd deploy/caddy
printf 'YILY_MASTER_KEY=%s\nYILY_SETUP_TOKEN=%s\nYILY_DOMAIN=%s\n' \
  "$(openssl rand -base64 32)" "$(openssl rand -base64 32)" \
  "yily.example.com" > .env
```

Replace `yily.example.com` with your domain, pointing its DNS record (A/AAAA) at
this machine.

## 2. Start

```bash
docker compose up -d
```

## 3. Initialize

Open `https://yily.example.com` and paste the `YILY_SETUP_TOKEN` from `.env`.

## Notes

- The [`Caddyfile`](Caddyfile) uses `{$YILY_DOMAIN}`, injected from `.env`.
- Ports: Caddy publishes `80`/`443`; Yily is not exposed on the host.
- Certificates and Caddy state are persisted in the `caddy_data` and
  `caddy_config` volumes.
- `.env` is ignored by git.
