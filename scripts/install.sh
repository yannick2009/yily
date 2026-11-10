#!/usr/bin/env bash
# Install the yily CLI from GitHub releases on Linux and macOS.
#
# Quick install:
#   curl -fsSL https://raw.githubusercontent.com/yannick2009/yily/main/scripts/install.sh | bash -s -- --pre-release
set -euo pipefail

REPO="yannick2009/yily"
BIN_NAME="yily-cli"
API_URL="https://api.github.com/repos/${REPO}"
RELEASES_URL="https://github.com/${REPO}/releases/download"

VERSION="${YILY_VERSION:-}"
PRE_RELEASE=0
UPDATE_PATH=0

TMP_DIR=""

log() { printf '==> %s\n' "$*"; }

die() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

cleanup() {
  if [ -n "$TMP_DIR" ]; then
    rm -rf "$TMP_DIR"
  fi
}

trap cleanup EXIT

usage() {
  cat <<'EOF'
Install the yily CLI from GitHub releases.

Usage:
  install.sh [options]

Options:
  --version <v>    install a specific version (default: latest stable)
  --pre-release    include prereleases when resolving the latest version
  --update-path    add ~/.local/bin to PATH in ~/.bashrc and ~/.zshrc
  --help           show this help

Environment:
  YILY_VERSION     same as --version

The binary is installed in ~/.local/bin.
EOF
}

parse_args() {
  while [ $# -gt 0 ]; do
    case "$1" in
      --version)
        [ $# -ge 2 ] || die "--version requires a value"
        VERSION="$2"
        shift 2
        ;;
      --pre-release)
        PRE_RELEASE=1
        shift
        ;;
      --update-path)
        UPDATE_PATH=1
        shift
        ;;
      -h|--help)
        usage
        exit 0
        ;;
      *)
        die "unknown option: $1 (try --help)"
        ;;
    esac
  done
}

# Map the host platform to the release archive naming scheme.
detect_platform() {
  case "$(uname -s)" in
    Linux) OS="linux" ;;
    Darwin) OS="darwin" ;;
    *) die "unsupported operating system: $(uname -s)" ;;
  esac

  case "$(uname -m)" in
    x86_64|amd64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    armv7l|armv7|armhf) ARCH="armv7" ;;
    *) die "unsupported architecture: $(uname -m)" ;;
  esac

  TARGET="${OS}_${ARCH}"
  case "$TARGET" in
    linux_amd64|linux_arm64|linux_armv7|darwin_amd64|darwin_arm64) ;;
    *) die "unsupported platform: ${TARGET}" ;;
  esac
}

# Pick the HTTP and checksum tools available on this system.
detect_tools() {
  if command -v curl >/dev/null 2>&1; then
    HTTP_CLIENT="curl"
  elif command -v wget >/dev/null 2>&1; then
    HTTP_CLIENT="wget"
  else
    die "curl or wget is required"
  fi

  if command -v sha256sum >/dev/null 2>&1; then
    SHA256_CMD="sha256sum"
  elif command -v shasum >/dev/null 2>&1; then
    SHA256_CMD="shasum"
  else
    die "sha256sum or shasum is required"
  fi

  command -v tar >/dev/null 2>&1 || die "tar is required"
  command -v install >/dev/null 2>&1 || die "install is required"
}

github_get() {
  local url="$1"
  if [ "$HTTP_CLIENT" = "curl" ]; then
    curl -fsSL --retry 3 \
      -H "Accept: application/vnd.github+json" \
      -H "User-Agent: yily-install" \
      "$url"
  else
    wget -qO- \
      --header="Accept: application/vnd.github+json" \
      --header="User-Agent: yily-install" \
      "$url"
  fi
}

download() {
  local url="$1" dest="$2"
  if [ "$HTTP_CLIENT" = "curl" ]; then
    curl -fsSL --retry 3 --proto '=https' -o "$dest" "$url"
  else
    wget -q -O "$dest" "$url"
  fi
}

# Use an explicit version when provided, otherwise ask the GitHub API.
resolve_version() {
  if [ -n "$VERSION" ]; then
    VERSION="${VERSION#v}"
    return
  fi

  local response
  if [ "$PRE_RELEASE" -eq 1 ]; then
    log "Resolving the latest release (prereleases included)"
    response="$(github_get "${API_URL}/releases?per_page=1")" \
      || die "could not reach the GitHub API; retry with --version <v>"
  else
    log "Resolving the latest stable release"
    response="$(github_get "${API_URL}/releases/latest")" \
      || die "no stable release found; retry with --pre-release or --version <v>"
  fi

  VERSION="$(printf '%s\n' "$response" | sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p')"
  VERSION="${VERSION#v}"
  [ -n "$VERSION" ] || die "could not read the version from the GitHub response"
}

# Reject versions that could break the download URL.
validate_version() {
  case "$VERSION" in
    [0-9]*) ;;
    *) die "invalid version: ${VERSION}" ;;
  esac
  case "$VERSION" in
    *[!0-9A-Za-z.-]*) die "invalid version: ${VERSION}" ;;
  esac
}

# Add ~/.local/bin to PATH in the user's shell configuration.
update_path() {
  local block rc
  block="$(cat <<'EOF'
# >>> yily-cli >>>
case ":$PATH:" in
  *":$HOME/.local/bin:"*) ;;
  *) export PATH="$HOME/.local/bin:$PATH" ;;
esac
# <<< yily-cli <<<
EOF
)"

  for rc in "${HOME}/.bashrc" "${HOME}/.zshrc"; do
    touch "$rc"
    if grep -qF '# >>> yily-cli >>>' "$rc"; then
      log "PATH already configured in ${rc}"
      continue
    fi
    printf '\n%s\n' "$block" >> "$rc"
    log "Added PATH entry to ${rc}"
  done
}

main() {
  parse_args "$@"

  # The install directory is under $HOME, so root would end up in /root.
  [ "$(id -u)" -ne 0 ] || die "do not run this script as root; it installs into \$HOME/.local/bin"
  [ -n "${HOME:-}" ] || die "HOME is not set"

  detect_platform
  detect_tools
  resolve_version
  validate_version

  INSTALL_DIR="${HOME}/.local/bin"
  local asset archive_url checksums_url expected actual binary

  asset="${BIN_NAME}_${VERSION}_${TARGET}.tar.gz"
  archive_url="${RELEASES_URL}/v${VERSION}/${asset}"
  checksums_url="${RELEASES_URL}/v${VERSION}/checksums.txt"

  TMP_DIR="$(mktemp -d)"

  log "Downloading ${asset}"
  download "$archive_url" "${TMP_DIR}/${asset}" || die "download failed: ${archive_url}"
  download "$checksums_url" "${TMP_DIR}/checksums.txt" || die "download failed: ${checksums_url}"

  # The checksums file lists entries as "./<archive>".
  expected="$(awk -v name="$asset" '$2 == name || $2 == "./" name { print $1; exit }' "${TMP_DIR}/checksums.txt")"
  [ -n "$expected" ] || die "no checksum for ${asset} in checksums.txt"

  if [ "$SHA256_CMD" = "sha256sum" ]; then
    actual="$(sha256sum "${TMP_DIR}/${asset}" | awk '{print $1}')"
  else
    actual="$(shasum -a 256 "${TMP_DIR}/${asset}" | awk '{print $1}')"
  fi
  [ "$expected" = "$actual" ] || die "checksum mismatch for ${asset}"
  log "Checksum verified"

  tar -xzf "${TMP_DIR}/${asset}" -C "$TMP_DIR"

  binary="${TMP_DIR}/${asset%.tar.gz}/${BIN_NAME}"
  [ -f "$binary" ] || die "unexpected archive layout: ${BIN_NAME} not found"

  mkdir -p "$INSTALL_DIR"
  install -m 0755 "$binary" "${INSTALL_DIR}/${BIN_NAME}"
  log "Installed ${BIN_NAME} ${VERSION} in ${INSTALL_DIR}"

  if [ "$UPDATE_PATH" -eq 1 ]; then
    update_path
  else
    case ":$PATH:" in
      *":${INSTALL_DIR}:"*) ;;
      *)
        log "${INSTALL_DIR} is not in your PATH; add this line to your shell configuration:"
        printf '  export PATH="%s:$PATH"\n' "$INSTALL_DIR"
        ;;
    esac
  fi

  "${INSTALL_DIR}/${BIN_NAME}" --version || log "warning: ${BIN_NAME} was installed but could not be executed"
}

main "$@"
