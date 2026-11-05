#!/usr/bin/env bash
set -euo pipefail

# Cligram Installer Script
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/Kumneger0/cligram/main/install.sh | bash
#
# Custom options:
#   BINDIR=/custom/path curl -fsSL ... | bash
#   CLIGRAM_VERSION=v2.7.0 curl -fsSL ... | bash

REPO="Kumneger0/cligram"

# Color helpers
if [ -t 1 ]; then
  GREEN='\033[0;32m'
  CYAN='\033[0;36m'
  YELLOW='\033[1;33m'
  RED='\033[0;31m'
  BOLD='\033[1m'
  RESET='\033[0m'
else
  GREEN=''
  CYAN=''
  YELLOW=''
  RED=''
  BOLD=''
  RESET=''
fi

info() {
  printf "${CYAN}==>${RESET} ${BOLD}%s${RESET}\n" "$1"
}

success() {
  printf "${GREEN}==>${RESET} ${BOLD}%s${RESET}\n" "$1"
}

warn() {
  printf "${YELLOW}WARNING:${RESET} %s\n" "$1"
}

error() {
  printf "${RED}ERROR:${RESET} %s\n" "$1" >&2
  exit 1
}

# 1. Detect OS
OS="$(uname -s)"
case "${OS}" in
  Linux)
    TARGET_OS="Linux"
    ;;
  Darwin)
    TARGET_OS="Darwin"
    ;;
  MINGW*|MSYS*|CYGWIN*)
    error "Windows native is not supported. Please use Windows Subsystem for Linux (WSL): https://learn.microsoft.com/en-us/windows/wsl/"
    ;;
  *)
    error "Unsupported operating system: ${OS}"
    ;;
esac

# 2. Detect Architecture
ARCH="$(uname -m)"
case "${ARCH}" in
  x86_64|amd64)
    TARGET_ARCH="x86_64"
    ;;
  arm64|aarch64)
    TARGET_ARCH="arm64"
    ;;
  *)
    error "Unsupported architecture: ${ARCH}"
    ;;
esac

# 3. Determine Version
VERSION="${CLIGRAM_VERSION:-}"
if [ -z "${VERSION}" ]; then
  info "Fetching latest release version..."
  EFFECTIVE_URL="$(curl -sIL -o /dev/null -w '%{url_effective}' "https://github.com/${REPO}/releases/latest" 2>/dev/null || true)"
  VERSION="$(echo "${EFFECTIVE_URL}" | sed 's#.*/tag/##')"
  if [ -z "${VERSION}" ] || [ "${VERSION}" = "latest" ]; then
    # Fallback to API if redirect failed
    VERSION="$(curl -s "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/' || true)"
  fi
fi

if [ -z "${VERSION}" ]; then
  error "Could not determine latest version. Specify CLIGRAM_VERSION manually, e.g. CLIGRAM_VERSION=v2.7.0"
fi

info "Detected platform: ${TARGET_OS} (${TARGET_ARCH})"
info "Selected version:  ${VERSION}"

# 4. Construct Download URL
TARBALL="cligram_${TARGET_OS}_${TARGET_ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${TARBALL}"

# 5. Download & Extract
TMP_DIR="$(mktemp -d -t cligram-install.XXXXXX)"
cleanup() {
  rm -rf "${TMP_DIR}"
}
trap cleanup EXIT INT TERM

info "Downloading ${DOWNLOAD_URL}..."
if ! curl -fsSL "${DOWNLOAD_URL}" -o "${TMP_DIR}/${TARBALL}"; then
  error "Failed to download ${DOWNLOAD_URL}. Please check your network connection or verify that release ${VERSION} has assets for ${TARGET_OS}_${TARGET_ARCH}."
fi

info "Extracting archive..."
tar -xzf "${TMP_DIR}/${TARBALL}" -C "${TMP_DIR}"

if [ ! -f "${TMP_DIR}/cligram" ]; then
  error "Archive did not contain the 'cligram' binary."
fi

# 6. Determine Target Directory
INSTALL_DIR="${BINDIR:-}"
USE_SUDO=0

if [ -z "${INSTALL_DIR}" ]; then
  if [ -w "/usr/local/bin" ]; then
    INSTALL_DIR="/usr/local/bin"
  elif [ "$(id -u)" -eq 0 ]; then
    INSTALL_DIR="/usr/local/bin"
  elif [ -d "/usr/local/bin" ] && command -v sudo >/dev/null 2>&1 && [ -t 0 ]; then
    # Interactive shell with sudo available
    INSTALL_DIR="/usr/local/bin"
    USE_SUDO=1
  else
    INSTALL_DIR="${HOME}/.local/bin"
  fi
fi

info "Installing to ${INSTALL_DIR}..."

if [ "${USE_SUDO}" -eq 1 ]; then
  sudo mkdir -p "${INSTALL_DIR}"
  sudo mv "${TMP_DIR}/cligram" "${INSTALL_DIR}/cligram"
  sudo chmod +x "${INSTALL_DIR}/cligram"
else
  mkdir -p "${INSTALL_DIR}"
  mv "${TMP_DIR}/cligram" "${INSTALL_DIR}/cligram"
  chmod +x "${INSTALL_DIR}/cligram"
fi

# 7. Verification & PATH Check
success "cligram ${VERSION} installed successfully to ${INSTALL_DIR}/cligram!"

case ":${PATH}:" in
  *":${INSTALL_DIR}:"*)
    ;;
  *)
    printf "\n"
    warn "${INSTALL_DIR} is not in your system PATH."
    printf "To use 'cligram' globally, add it to your PATH by running:\n\n"
    if [ -n "${ZSH_VERSION:-}" ] || [ -f "${HOME}/.zshrc" ]; then
      printf "  echo 'export PATH=\"%s:\$PATH\"' >> ~/.zshrc && source ~/.zshrc\n\n" "${INSTALL_DIR}"
    else
      printf "  echo 'export PATH=\"%s:\$PATH\"' >> ~/.bashrc && source ~/.bashrc\n\n" "${INSTALL_DIR}"
    fi
    ;;
esac

printf "\nRun ${BOLD}cligram${RESET} to start.\n"
