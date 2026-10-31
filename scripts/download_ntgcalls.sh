#!/usr/bin/env bash
set -euo pipefail

VERSION="v2.2.1-beta03"
BASE_URL="https://github.com/pytgcalls/ntgcalls/releases/download/${VERSION}"

# Resolve script directory and repository root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Detect OS
OS_TYPE="$(uname -s)"
case "${OS_TYPE}" in
    Linux*)     GOOS="linux" ;;
    Darwin*)    GOOS="darwin" ;;
    MINGW*|MSYS*|CYGWIN*) GOOS="windows" ;;
    *)          echo "Unsupported OS: ${OS_TYPE}" >&2; exit 1 ;;
esac

# Detect Architecture
ARCH_TYPE="$(uname -m)"
case "${ARCH_TYPE}" in
    x86_64|amd64)   GOARCH="amd64"; NTG_ARCH="x86_64" ;;
    aarch64|arm64)  GOARCH="arm64"; NTG_ARCH="arm64" ;;
    *)              echo "Unsupported architecture: ${ARCH_TYPE}" >&2; exit 1 ;;
esac

# Map to pytgcalls release asset platform name
if [ "${GOOS}" = "darwin" ]; then
    ASSET_PLATFORM="macos-${NTG_ARCH}"
else
    ASSET_PLATFORM="${GOOS}-${NTG_ARCH}"
fi

TARGET_DIR="${REPO_ROOT}/ntg/lib/${GOOS}_${GOARCH}"
mkdir -p "${TARGET_DIR}"

if [ "${GOOS}" = "windows" ]; then
    TARGET_LIB="${TARGET_DIR}/ntgcalls.lib"
else
    TARGET_LIB="${TARGET_DIR}/libntgcalls.a"
fi

if [ -s "${TARGET_LIB}" ]; then
    echo "--> [ntgcalls] Static library already exists at ${TARGET_LIB}"
    exit 0
fi

ZIP_NAME="ntgcalls.${ASSET_PLATFORM}-static_libs.zip"
DOWNLOAD_URL="${BASE_URL}/${ZIP_NAME}"

echo "--> [ntgcalls] Downloading ${VERSION} static library for ${ASSET_PLATFORM}..."
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

curl -fsSL -o "${TMP_DIR}/${ZIP_NAME}" "${DOWNLOAD_URL}"

echo "--> [ntgcalls] Extracting static library to ${TARGET_DIR}..."
unzip -q -o "${TMP_DIR}/${ZIP_NAME}" -d "${TMP_DIR}/extracted"

if [ -f "${TMP_DIR}/extracted/lib/libntgcalls.a" ]; then
    mv "${TMP_DIR}/extracted/lib/libntgcalls.a" "${TARGET_LIB}"
elif [ -f "${TMP_DIR}/extracted/lib/ntgcalls.lib" ]; then
    mv "${TMP_DIR}/extracted/lib/ntgcalls.lib" "${TARGET_LIB}"
elif [ -f "${TMP_DIR}/extracted/libntgcalls.a" ]; then
    mv "${TMP_DIR}/extracted/libntgcalls.a" "${TARGET_LIB}"
else
    echo "Error: static library not found in extracted archive" >&2
    ls -la "${TMP_DIR}/extracted" >&2
    exit 1
fi

echo "--> [ntgcalls] Successfully installed static library at ${TARGET_LIB}"
