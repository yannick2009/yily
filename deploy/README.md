# Deployment — Yily

Ready-to-use deployment templates. Each subfolder is self-contained: it ships
its own `docker-compose.yml` and a dedicated README.

| Folder | Use case | Reverse proxy |
| --- | --- | --- |
| [`docker/`](docker/) | Direct access on local/LAN, no proxy | — |
| [`caddy/`](caddy/) | Automatic HTTPS (Let's Encrypt) | Caddy |
| [`nginx/`](nginx/) | Classic reverse proxy (manual TLS/certbot) | nginx |

## Generate the secrets (once)

```bash
YILY_MASTER_KEY=$(openssl rand -base64 32)   # AES-256 key — keep it safe
YILY_SETUP_TOKEN=$(openssl rand -base64 32)  # first-run setup token
```

> ⚠️ If `YILY_MASTER_KEY` is lost, all encrypted secrets become unrecoverable.

Each `docker-compose.yml` reads these variables from a `.env` file placed in its
folder (or from the shell environment). Each folder's README gives the exact
command. `.env` files are ignored by git.
