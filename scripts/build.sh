#!/usr/bin/env bash
#
# build.sh - Build script for Yunt mail server
#
# Usage:
#   ./scripts/build.sh [options]
#
# Options:
#   -o, --output DIR    Output directory (default: bin)
#   -v, --version VER   Version string (default: git describe or 'dev')
#   -p, --platform OS/ARCH  Target platform (default: current)
#   -d, --debug         Build with debug symbols
#   -w, --with-web      Build web UI before Go binary
#   -h, --help          Show this help message
#
# Examples:
#   ./scripts/build.sh
#   ./scripts/build.sh -v 1.0.0
#   ./scripts/build.sh -p linux/amd64
#   ./scripts/build.sh -o dist -v 1.0.0 -p darwin/arm64
#   ./scripts/build.sh -w  # Build with embedded web UI

set -euo pipefail

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Default values
BINARY_NAME="yunt"
OUTPUT_DIR="${PROJECT_ROOT}/bin"
MAIN_PATH="./cmd/yunt"
DEBUG_BUILD=false
BUILD_WEB=false
TARGET_OS=""
TARGET_ARCH=""

# Version information
get_version() {
    if git describe --tags --always 2>/dev/null; then
        return
    fi
    echo "dev"
}

VERSION="${VERSION:-$(get_version)}"
COMMIT="${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')}"
BUILD_DATE="${BUILD_DATE:-$(date -u +"%Y-%m-%dT%H:%M:%SZ")}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Print colored output
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Show help message
show_help() {
    sed -n '/^#/!q;s/^# \?//p' "$0" | tail -n +2
    exit 0
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -o|--output)
                OUTPUT_DIR="$2"
                shift 2
                ;;
            -v|--version)
                VERSION="$2"
                shift 2
                ;;
            -p|--platform)
                IFS='/' read -r TARGET_OS TARGET_ARCH <<< "$2"
                shift 2
                ;;
            -d|--debug)
                DEBUG_BUILD=true
                shift
                ;;
            -w|--with-web)
                BUILD_WEB=true
                shift
                ;;
            -h|--help)
                show_help
                ;;
            *)
                print_error "Unknown option: $1"
                show_help
                ;;
        esac
    done
}

# Validate Go installation
check_go() {
    if ! command -v go &>/dev/null; then
        print_error "Go is not installed. Please install Go 1.22 or higher."
        exit 1
    fi
    
    local go_version
    go_version=$(go version | awk '{print $3}' | sed 's/go//')
    print_info "Using Go version: ${go_version}"
}

# Build web UI
build_web() {
    print_info "Building web UI..."
    
    # Check if npm is installed
    if ! command -v npm &>/dev/null; then
        print_error "npm is not installed. Please install Node.js and npm."
        exit 1
    fi
    
    # Change to web directory
    cd "${PROJECT_ROOT}/web"
    
    # Check if node_modules exists
    if [[ ! -d "node_modules" ]]; then
        print_info "Installing web dependencies..."
        npm install
    fi
    
    # Clean previous build in webui/dist
    print_info "Cleaning previous web build..."
    find "${PROJECT_ROOT}/webui/dist" -type f ! -name '.gitkeep' -delete 2>/dev/null || true
    
    # Build the web UI
    npm run build
    
    # Verify build output
    if [[ -f "${PROJECT_ROOT}/webui/dist/index.html" ]]; then
        print_info "Web UI built successfully to webui/dist/"
    else
        print_error "Web UI build failed: index.html not found in webui/dist/"
        exit 1
    fi
    
    # Return to project root
    cd "${PROJECT_ROOT}"
}

# Build the binary
build() {
    local output_file="${OUTPUT_DIR}/${BINARY_NAME}"
    local ldflags="-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildDate=${BUILD_DATE}"
    
    # Add strip flags for release builds
    if [[ "${DEBUG_BUILD}" == "false" ]]; then
        ldflags="-s -w ${ldflags}"
    fi
    
    # Determine target platform
    local goos="${TARGET_OS:-$(go env GOOS)}"
    local goarch="${TARGET_ARCH:-$(go env GOARCH)}"
    
    # Add .exe extension for Windows
    if [[ "${goos}" == "windows" ]]; then
        output_file="${output_file}.exe"
    fi
    
    # Create output directory
    mkdir -p "${OUTPUT_DIR}"
    
    print_info "Building ${BINARY_NAME}..."
    print_info "  Version: ${VERSION}"
    print_info "  Commit: ${COMMIT}"
    print_info "  Build Date: ${BUILD_DATE}"
    print_info "  Platform: ${goos}/${goarch}"
    print_info "  Output: ${output_file}"
    
    # Change to project root
    cd "${PROJECT_ROOT}"
    
    # Build
    GOOS="${goos}" GOARCH="${goarch}" go build \
        -ldflags "${ldflags}" \
        -o "${output_file}" \
        "${MAIN_PATH}"
    
    # Verify the binary was created
    if [[ -f "${output_file}" ]]; then
        local size
        size=$(du -h "${output_file}" | cut -f1)
        print_info "Build successful!"
        print_info "  Binary: ${output_file}"
        print_info "  Size: ${size}"
    else
        print_error "Build failed: binary not created"
        exit 1
    fi
}

# Main function
main() {
    parse_args "$@"
    check_go
    
    # Build web UI if requested
    if [[ "${BUILD_WEB}" == "true" ]]; then
        build_web
    fi
    
    build
}

main "$@"
