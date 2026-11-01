#!/usr/bin/env bash
set -euo pipefail

VERSION="v2.2.5"
BASE_URL="https://github.com/pytgcalls/ntgcalls/releases/download/${VERSION}"

# Resolve script directory and repository root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Detect OS
OS_TYPE="${TARGET_OS:-$(uname -s)}"
case "${OS_TYPE}" in
    Linux*|linux*)     GOOS="linux" ;;
    Darwin*|darwin*)   GOOS="darwin" ;;
    MINGW*|MSYS*|CYGWIN*|windows*) GOOS="windows" ;;
    *)          echo "Unsupported OS: ${OS_TYPE}" >&2; exit 1 ;;
esac

# Detect Architecture
ARCH_TYPE="${TARGET_ARCH:-$(uname -m)}"
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
    ZIP_NAME="ntgcalls.${ASSET_PLATFORM}-static_libs.zip"
elif [ "${GOOS}" = "darwin" ]; then
    TARGET_LIB="${TARGET_DIR}/libntgcalls.dylib"
    ZIP_NAME="ntgcalls.${ASSET_PLATFORM}-shared_libs.zip"
else
    TARGET_LIB="${TARGET_DIR}/libntgcalls.a"
    ZIP_NAME="ntgcalls.${ASSET_PLATFORM}-static_libs.zip"
fi

if [ -s "${TARGET_LIB}" ]; then
    echo "--> [ntgcalls] Library already exists at ${TARGET_LIB}"
    exit 0
fi

DOWNLOAD_URL="${BASE_URL}/${ZIP_NAME}"

echo "--> [ntgcalls] Downloading ${VERSION} library for ${ASSET_PLATFORM}..."
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

curl -fsSL --retry 3 --retry-connrefused -o "${TMP_DIR}/${ZIP_NAME}" "${DOWNLOAD_URL}"

echo "--> [ntgcalls] Extracting library to ${TARGET_DIR}..."
unzip -q -o "${TMP_DIR}/${ZIP_NAME}" -d "${TMP_DIR}/extracted"

if [ "${GOOS}" = "darwin" ]; then
    if [ -f "${TMP_DIR}/extracted/lib/libntgcalls.dylib" ]; then
        mv "${TMP_DIR}/extracted/lib/libntgcalls.dylib" "${TARGET_LIB}"
    elif [ -f "${TMP_DIR}/extracted/libntgcalls.dylib" ]; then
        mv "${TMP_DIR}/extracted/libntgcalls.dylib" "${TARGET_LIB}"
    else
        DYLIB_FOUND="$(find "${TMP_DIR}/extracted" -name "*.dylib" | head -n 1)"
        if [ -n "${DYLIB_FOUND}" ]; then
            mv "${DYLIB_FOUND}" "${TARGET_LIB}"
        else
            echo "Error: shared library not found in extracted archive" >&2
            ls -la "${TMP_DIR}/extracted" >&2
            exit 1
        fi
    fi
else
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
fi

echo "--> [ntgcalls] Successfully installed library at ${TARGET_LIB}"
