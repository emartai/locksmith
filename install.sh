#!/bin/sh

set -eu

REPO="emartai/locksmith"
VERSION=""

usage() {
  cat <<'EOF'
Usage: install.sh [--version <tag>]

Installs the Locksmith CLI from GitHub Releases.
EOF
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --version)
      if [ "$#" -lt 2 ]; then
        echo "missing value for --version" >&2
        exit 1
      fi
      VERSION="$2"
      shift 2
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

need_cmd() {
  if command -v "$1" >/dev/null 2>&1; then
    return 0
  fi
  return 1
}

download_file() {
  url="$1"
  dest="$2"

  if need_cmd curl; then
    curl -fsSL "$url" -o "$dest"
    return 0
  fi

  if need_cmd wget; then
    wget -qO "$dest" "$url"
    return 0
  fi

  echo "curl or wget is required to install locksmith" >&2
  exit 1
}

fetch_text() {
  url="$1"

  if need_cmd curl; then
    curl -fsSL "$url"
    return 0
  fi

  if need_cmd wget; then
    wget -qO- "$url"
    return 0
  fi

  echo "curl or wget is required to install locksmith" >&2
  exit 1
}

detect_os() {
  raw_os="$(uname -s)"
  case "$raw_os" in
    Linux)
      echo "linux"
      ;;
    Darwin)
      echo "darwin"
      ;;
    MINGW*|MSYS*|CYGWIN*)
      echo "windows"
      ;;
    *)
      echo "unsupported operating system: $raw_os" >&2
      exit 1
      ;;
  esac
}

detect_arch() {
  raw_arch="$(uname -m)"
  case "$raw_arch" in
    x86_64|amd64)
      echo "amd64"
      ;;
    arm64|aarch64)
      echo "arm64"
      ;;
    *)
      echo "unsupported architecture: $raw_arch" >&2
      exit 1
      ;;
  esac
}

resolve_version() {
  if [ -n "$VERSION" ]; then
    echo "$VERSION"
    return 0
  fi

  latest_json="$(fetch_text "https://api.github.com/repos/$REPO/releases/latest")"
  latest_version="$(printf '%s\n' "$latest_json" | awk -F'"' '/"tag_name"/ { print $4; exit }')"
  if [ -z "$latest_version" ]; then
    echo "failed to resolve latest release version" >&2
    exit 1
  fi

  echo "$latest_version"
}

verify_checksum() {
  checksums_file="$1"
  archive_file="$2"

  archive_name="$(basename "$archive_file")"
  expected="$(awk -v file="$archive_name" '$2 == file { print $1 }' "$checksums_file")"
  if [ -z "$expected" ]; then
    echo "failed to find checksum for $archive_name" >&2
    exit 1
  fi

  if need_cmd sha256sum; then
    actual="$(sha256sum "$archive_file" | awk '{print $1}')"
  elif need_cmd shasum; then
    actual="$(shasum -a 256 "$archive_file" | awk '{print $1}')"
  else
    echo "sha256sum or shasum is required to verify downloads" >&2
    exit 1
  fi

  if [ "$expected" != "$actual" ]; then
    echo "checksum verification failed for $archive_name" >&2
    exit 1
  fi
}

install_dir() {
  if [ -w "/usr/local/bin" ]; then
    echo "/usr/local/bin"
    return 0
  fi

  if need_cmd sudo; then
    echo "/usr/local/bin"
    return 0
  fi

  echo "$HOME/.local/bin"
}

extract_archive() {
  archive_file="$1"
  destination_dir="$2"
  os_name="$3"

  mkdir -p "$destination_dir"

  if [ "$os_name" = "windows" ]; then
    if ! need_cmd unzip; then
      echo "unzip is required to install the Windows archive" >&2
      exit 1
    fi
    unzip -oq "$archive_file" -d "$destination_dir"
    return 0
  fi

  tar -xzf "$archive_file" -C "$destination_dir"
}

move_binary() {
  extracted_dir="$1"
  target_dir="$2"
  os_name="$3"

  binary_name="locksmith"
  if [ "$os_name" = "windows" ]; then
    binary_name="locksmith.exe"
  fi

  extracted_bin="$extracted_dir/$binary_name"
  if [ ! -f "$extracted_bin" ]; then
    echo "expected binary $binary_name not found in archive" >&2
    exit 1
  fi

  target_bin="$target_dir/$binary_name"

  if [ "$target_dir" = "/usr/local/bin" ] && [ ! -w "$target_dir" ] && need_cmd sudo; then
    sudo install "$extracted_bin" "$target_bin"
  else
    install "$extracted_bin" "$target_bin"
  fi

  echo "$target_bin"
}

OS_NAME="$(detect_os)"
ARCH_NAME="$(detect_arch)"

if [ "$OS_NAME" = "windows" ]; then
  echo "Windows users should use 'go install github.com/emartai/locksmith@latest'." >&2
  exit 1
fi

RESOLVED_VERSION="$(resolve_version)"
TMP_DIR="$(mktemp -d 2>/dev/null || mktemp -d -t locksmith-install)"
ARCHIVE_EXT="tar.gz"
ARCHIVE_NAME="locksmith_${OS_NAME}_${ARCH_NAME}.${ARCHIVE_EXT}"
ARCHIVE_URL="https://github.com/$REPO/releases/download/${RESOLVED_VERSION}/${ARCHIVE_NAME}"
CHECKSUMS_URL="https://github.com/$REPO/releases/download/${RESOLVED_VERSION}/checksums.txt"
ARCHIVE_PATH="$TMP_DIR/$ARCHIVE_NAME"
CHECKSUMS_PATH="$TMP_DIR/checksums.txt"

trap 'rm -rf "$TMP_DIR"' EXIT HUP INT TERM

download_file "$ARCHIVE_URL" "$ARCHIVE_PATH"
download_file "$CHECKSUMS_URL" "$CHECKSUMS_PATH"
verify_checksum "$CHECKSUMS_PATH" "$ARCHIVE_PATH"

EXTRACT_DIR="$TMP_DIR/extracted"
extract_archive "$ARCHIVE_PATH" "$EXTRACT_DIR" "$OS_NAME"

TARGET_DIR="$(install_dir)"
if [ "$TARGET_DIR" = "$HOME/.local/bin" ]; then
  mkdir -p "$TARGET_DIR"
fi

TARGET_BIN="$(move_binary "$EXTRACT_DIR" "$TARGET_DIR" "$OS_NAME")"

if ! "$TARGET_BIN" --version >/dev/null 2>&1; then
  echo "installation verification failed" >&2
  exit 1
fi

echo "locksmith ${RESOLVED_VERSION} installed to ${TARGET_BIN}"

if [ "$TARGET_DIR" = "$HOME/.local/bin" ]; then
  echo "Add $HOME/.local/bin to your PATH if it is not already present."
fi
