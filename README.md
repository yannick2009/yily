<p align="center">
  <picture>
    <img src="logo.svg" alt="Overmind" />
  </picture>
</p>

**Yily** is a lightweight and fast **open-source secret management application** designed to help you securely store and manage sensitive information such as API keys, passwords, tokens, and other secrets.

The project was born from a simple need: having a **self-hosted secret management solution that is easy to deploy, simple to use, and efficient**, without the complexity that can come with more advanced solutions such as HashiCorp Vault or Infisical.

Yily aims to provide a good balance between **simplicity, performance, security, and self-hosting**. It is designed for developers and teams who want to keep control of their secrets and infrastructure while using a straightforward and lightweight solution.

The project is still in its **early stages of development**, and the goal is to build it together with the open-source community. Contributions, ideas, feedback, and discussions are very welcome.

If you are interested in contributing, please check out the [contributing guidelines](CONTRIBUTING.MD) and feel free to open an issue or submit a pull request.

## Technologies Used

| Technology | Description |
|------------|-------------|
| Go | The core programming language used for building Yily server and CLI. |
| Echo | A web framework for Go, used for building the Yily web interface. |
| SQLite | A lightweight relational database used for storing secrets and metadata. |
| SolidJS | A reactive JavaScript library used for building the Yily web interface. |
| Tailwind CSS | A utility-first CSS framework used for styling the Yily web interface. |
| Nix | A package manager and build system used for building and packaging Yily. |

## Project Structure

For this project, we have a monorepo structure that contains both the server and the web interface. The main directories are:

```text
.
├── client/                 # Interface web
└── server/                 # API, CLI et services backend
```

The `client` contains the web interface, while the `server` contains the API, the command-line interface, and the application's internal packages.

## Installation

### Docker

A `linux/amd64` image is published to the GitHub Container Registry. Generate the master key once and keep it somewhere safe — if it is lost, the stored secrets become unrecoverable:

```bash
echo "YILY_MASTER_KEY=$(openssl rand -base64 32)" > yily.env
chmod 600 yily.env
```

Run the container:

```bash
docker run -d \
  --name yily \
  --restart unless-stopped \
  -p 16529:16529 \
  -v yily-data:/data \
  --env-file yily.env \
  ghcr.io/yannick2009/yily:1.0.0-beta.1
```

The server listens on port `16529` and serves the web interface at `http://localhost:16529`. Its SQLite database lives in the `yily-data` volume under `/data/instance`. Keep both the volume and `yily.env` safe.

With Docker Compose:

```yaml
services:
  yily:
    image: ghcr.io/yannick2009/yily:1.0.0-beta.1
    container_name: yily
    restart: unless-stopped
    ports:
      - "16529:16529"
    volumes:
      - yily-data:/data
    env_file:
      - yily.env

volumes:
  yily-data:
```

```bash
docker compose up -d
```

> The image is currently `linux/amd64` only. On ARM hosts, add `--platform linux/amd64` to `docker run` (or `platform: linux/amd64` to the Compose service); Docker will use emulation. The `latest` tag is only published for stable releases, so pin `1.0.0-beta.1` until a stable version exists.

### Server binary (Linux)

Docker is the recommended way to run the server. If Docker is not available, the release also ships a `linux/amd64` archive containing the `yily` binary and the `dist/` web client. The `dist/` directory must stay next to the binary.

```bash
curl -fsSLO https://github.com/yannick2009/yily/releases/download/v1.0.0-beta.1/yily_1.0.0-beta.1_linux_amd64.tar.gz
curl -fsSLO https://github.com/yannick2009/yily/releases/download/v1.0.0-beta.1/checksums.txt
sha256sum -c checksums.txt --ignore-missing

tar -xzf yily_1.0.0-beta.1_linux_amd64.tar.gz
cd yily_1.0.0-beta.1_linux_amd64
chmod -R u+w dist  # the web client is read-only in the archive
```

Generate the master key once and keep it safe — if it is lost, the stored secrets become unrecoverable:

```bash
echo "YILY_MASTER_KEY=$(openssl rand -base64 32)" > yily.env
chmod 600 yily.env
```

Run the server:

```bash
set -a
. ./yily.env
set +a
./yily
```

The server listens on `http://localhost:16529`. Its SQLite database is created in `instance/` under the working directory. To upgrade, extract the new archive elsewhere and copy `yily` and `dist/` into the existing directory, keeping `instance/` and `yily.env`. If a `dist/` directory is read-only (older archives), run `chmod -R u+w dist` first. For a long-running service, run it under systemd or another process manager with the extracted directory as `WorkingDirectory`.

### CLI

Install `yily-cli` from the latest GitHub release on Linux or macOS:

```bash
curl -fsSL https://raw.githubusercontent.com/yannick2009/yily/main/scripts/install.sh | bash -s -- --pre-release
```

> There is no stable release yet, so `--pre-release` is required. Once a stable version exists, omit it to install the latest stable release.

The binary is installed in `~/.local/bin`. Run the script as your user (it refuses to run as root). When `~/.local/bin` is not in your `PATH`, the script prints the line to add; pass `--update-path` to update `~/.bashrc` and `~/.zshrc` automatically.

| Option | Description |
|--------|-------------|
| `--version <v>` | Install a specific version (e.g. `1.0.0-beta.1`). |
| `--pre-release` | Include prereleases when resolving the latest version. |
| `--update-path` | Add `~/.local/bin` to `PATH` in `~/.bashrc` and `~/.zshrc`. |

The CLI can also update or remove itself:

```bash
yily-cli upgrade --pre-release      # upgrade to the latest release (prereleases included)
yily-cli upgrade --version 1.0.0-beta.1
yily-cli uninstall                  # remove the binary (asks for confirmation)
yily-cli uninstall --yes --purge    # also remove the saved login configuration
```

`uninstall` leaves the `PATH` entry added by the install script untouched.

To review the script before running it:

```bash
curl -fsSL https://raw.githubusercontent.com/yannick2009/yily/main/scripts/install.sh -o install.sh
less install.sh
bash install.sh --pre-release
```

## Development environment

The project pins its system toolchain (Go, Node.js, linters, task runner…) through its Nix flake, and its JavaScript toolchain — including **vite-plus** — through `client/package.json`. To enter the dev shell:

```bash
nix develop
```

Then `npm install` (run automatically by the shell hook) pulls the pinned `vite-plus` into `client/node_modules/`. The npm scripts (`npm run dev`, `npm run build`, `npm run preview`) and `just lint` resolve `vp` from there — no global install needed.

> **Do not run the official vite-plus installer** (`curl -fsSL https://viteplus.dev/install.sh | sh` or `vp self install`). It writes to `~/.local/share/vite-plus/`, prepends itself to your `PATH` via `~/.config/vite-plus/env`, and installs a `vp` shell function — all of which is unnecessary (the version in `client/package.json` is used by every project command) and can cause confusion if you type `vp` manually.
>
> If you had already run the installer, remove it with:
>
> ```bash
> rm -rf ~/.local/share/vite-plus ~/.config/vite-plus
> # then remove the `. "$HOME/.config/vite-plus/env"` line from
> # ~/.zshrc, ~/.zshenv, ~/.bashrc, ~/.profile
> ```

## Configuration

### Master Key (`YILY_MASTER_KEY`)

Yily encrypts secret values at rest using AES-256-GCM. The server requires a 32-byte Base64-encoded master key provided through the `YILY_MASTER_KEY` environment variable.

1. **Generate a key:**
   ```bash
   openssl rand -base64 32
   ```

2. **Configure the key:**
   - **Environment variable:** set it in your environment or process manager:
     ```bash
     export YILY_MASTER_KEY="<your-generated-base64-key>"
     ```
   **Local development with Air:** from the `server` directory, copy `.env.example` to `.env` and set `YILY_MASTER_KEY`. Air loads `.env` into the process environment when running `air`.
   > **Note:** The Go binary reads `YILY_MASTER_KEY` directly from process environment variables via `os.LookupEnv`. If running without Air (e.g. `go run ./cmd/server` or `./tmp/server`), export the variable in your shell.

3. **Important precautions:**
   - **Keep it secret:** Never commit the master key or a populated `.env` file to version control.
   - **Backup securely:** Store `YILY_MASTER_KEY` in a secure location separate from the SQLite database.
   - **Unrecoverable on loss:** If the master key is lost, all encrypted secrets become permanently unrecoverable.
   - **Persistence:** Do not generate a new key on every restart; reuse the same key to retain access to existing data.

## License

Yily is licensed under the MIT License. See the [LICENSE](LICENSE) file for more information.
