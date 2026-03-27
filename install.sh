#!/usr/bin/env bash
set -e

REPO="tesh254/lissh"
INSTALL_DIR="${HOME}/.local/bin"
TMP_DIR=$(mktemp -d)

cleanup() {
    rm -rf "$TMP_DIR"
}
trap cleanup EXIT

detect_os() {
    case "$(uname -s)" in
        Linux*)  echo "linux" ;;
        Darwin*) echo "darwin" ;;
        *)       echo " unsupported OS" >&2; exit 1 ;;
    esac
}

detect_arch() {
    case "$(uname -m)" in
        x86_64)  echo "x86_64" ;;
        aarch64|arm64) echo "arm64" ;;
        i386|i686) echo "i386" ;;
        *)        echo " unsupported architecture" >&2; exit 1 ;;
    esac
}

download() {
    local os=$1
    local arch=$2
    local version=$3

    local ext="tar.gz"
    if [ "$os" = "windows" ]; then
        ext="zip"
    fi

    local first=$(printf '%s' "$os" | cut -c1 | tr '[:lower:]' '[:upper:]')
    local rest=$(printf '%s' "$os" | cut -c2-)
    local filename="lissh_${first}${rest}_${arch}.${ext}"
    local url="https://github.com/${REPO}/releases/download/${version}/${filename}"

    echo "Downloading ${filename}..."
    curl -fsSL "$url" -o "${TMP_DIR}/${filename}"

    if [ "$ext" = "zip" ]; then
        unzip -j "${TMP_DIR}/${filename}" -d "${TMP_DIR}"
    else
        tar xzf "${TMP_DIR}/${filename}" -C "${TMP_DIR}"
    fi

    mv "${TMP_DIR}/lissh" "${TMP_DIR}/lissh_new"
    rm -f "${TMP_DIR}/${filename}"
}

echo "Installing lissh..."

if [ -d "$INSTALL_DIR" ]; then
    echo "Install directory exists: $INSTALL_DIR"
else
    echo "Creating install directory: $INSTALL_DIR"
    mkdir -p "$INSTALL_DIR"
fi

VERSION="${1:-latest}"
if [ "$VERSION" = "latest" ]; then
    VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | sed -n 's/.*"tag_name": "v\([^"]*\)".*/\1/p')
    echo "Latest version: v$VERSION"
fi

OS=$(detect_os)
ARCH=$(detect_arch)

download "$OS" "$ARCH" "$VERSION"

chmod +x "${TMP_DIR}/lissh_new"
mv "${TMP_DIR}/lissh_new" "${INSTALL_DIR}/lissh"

echo ""
echo "Installed to: ${INSTALL_DIR}/lissh"
echo ""

if echo "$PATH" | grep -q "${INSTALL_DIR}"; then
    echo "${INSTALL_DIR} is already in your PATH"
else
    echo "Add to your PATH by running:"
    echo ""
    echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
    echo ""
    echo "Add this line to your ~/.bashrc or ~/.zshrc to persist across sessions."
fi

echo ""
echo "Run 'lissh --help' to get started!"
echo ""
echo "To update to a new version, run: lissh update --install"
