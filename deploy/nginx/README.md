# Yily + nginx (reverse proxy)

nginx does not generate certificates: you must provide them yourself.

## 1. Generate the secrets

Create a `.env` file in this folder:

```bash
cd deploy/nginx
printf 'YILY_MASTER_KEY=%s\nYILY_SETUP_TOKEN=%s\n' \
  "$(openssl rand -base64 32)" "$(openssl rand -base64 32)" > .env
```

## 2. Provide the TLS certificates

Place `fullchain.pem` and `privkey.pem` in the `certs/` folder:

```bash
cp /etc/letsencrypt/live/yily.example.com/fullchain.pem certs/
cp /etc/letsencrypt/live/yily.example.com/privkey.pem  certs/
```

Obtain them for example with certbot:

```bash
certbot certonly --standalone -d yily.example.com
```

## 3. Adjust the configuration

In [`nginx.conf`](nginx.conf), replace `server_name yily.example.com` with your
domain (in both places).

## 4. Start

```bash
docker compose up -d
```

## 5. Initialize

Open `https://yily.example.com` and paste the `YILY_SETUP_TOKEN` from `.env`.

## Notes

- `nginx.conf` is mounted at `/etc/nginx/conf.d/default.conf` (it replaces the
  image's default config).
- `.env` and `certs/` are ignored by git.
