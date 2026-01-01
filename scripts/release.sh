#!/usr/bin/env bash
#
# release.sh - Build release binaries for multiple platforms
#
# Usage:
#   ./scripts/release.sh [VERSION]
#
# Arguments:
#   VERSION    Version string for the release (default: git tag or 'dev')
#
# Environment Variables:
#   PLATFORMS      Space-separated list of OS/ARCH pairs (default: all supported)
#   DIST_DIR       Output directory for release binaries (default: dist)
#   CHECKSUM       Generate checksums (default: true)
#   COMPRESS       Create compressed archives (default: true)
#   INCLUDE_README Include README.md in archives (default: true)
#   BUILD_WEB      Build web UI before binaries (default: false)
#   CGO_ENABLED    Enable CGO for builds (default: 0)
#
# Examples:
#   ./scripts/release.sh
#   ./scripts/release.sh v1.0.0
#   PLATFORMS="linux/amd64 darwin/arm64" ./scripts/release.sh v1.0.0
#   BUILD_WEB=true ./scripts/release.sh v1.0.0

set -euo pipefail

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Configuration
BINARY_NAME="yunt"
MAIN_PATH="./cmd/yunt"
DIST_DIR="${DIST_DIR:-${PROJECT_ROOT}/dist}"

# Default platforms to build
DEFAULT_PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "linux/arm"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
    "windows/arm64"
)

# Use provided platforms or defaults
if [[ -n "${PLATFORMS:-}" ]]; then
    read -ra PLATFORMS_ARRAY <<< "${PLATFORMS}"
else
    PLATFORMS_ARRAY=("${DEFAULT_PLATFORMS[@]}")
fi

# Options
CHECKSUM="${CHECKSUM:-true}"
COMPRESS="${COMPRESS:-true}"
INCLUDE_README="${INCLUDE_README:-true}"
BUILD_WEB="${BUILD_WEB:-false}"
CGO_ENABLED="${CGO_ENABLED:-0}"

# Version information
get_version() {
    local version="${1:-}"
    if [[ -n "${version}" ]]; then
        echo "${version}"
    elif git describe --tags --exact-match 2>/dev/null; then
        return
    elif git describe --tags --always 2>/dev/null; then
        return
    else
        echo "dev"
    fi
}

VERSION="$(get_version "${1:-}")"
COMMIT="${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')}"
BUILD_DATE="${BUILD_DATE:-$(date -u +"%Y-%m-%dT%H:%M:%SZ")}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print functions
print_header() {
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}"
}

print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    print_header "Checking Prerequisites"
    
    if ! command -v go &>/dev/null; then
        print_error "Go is not installed"
        exit 1
    fi
    
    local go_version
    go_version=$(go version | awk '{print $3}' | sed 's/go//')
    print_info "Go version: ${go_version}"
    print_info "Project: ${PROJECT_ROOT}"
    print_info "CGO_ENABLED: ${CGO_ENABLED}"
}

# Build web UI
build_web_ui() {
    if [[ "${BUILD_WEB}" != "true" ]]; then
        return
    fi
    
    print_header "Building Web UI"
    
    if ! command -v npm &>/dev/null; then
        print_error "npm is not installed, skipping web UI build"
        return 1
    fi
    
    cd "${PROJECT_ROOT}/web"
    
    if [[ ! -d "node_modules" ]]; then
        print_info "Installing web dependencies..."
        npm ci --no-audit --no-fund
    fi
    
    print_info "Building web UI..."
    npm run build
    
    if [[ -f "${PROJECT_ROOT}/webui/dist/index.html" ]]; then
        print_info "Web UI built successfully"
    else
        print_error "Web UI build failed"
        return 1
    fi
    
    cd "${PROJECT_ROOT}"
}

# Clean previous builds
clean_dist() {
    print_info "Cleaning previous release builds..."
    rm -rf "${DIST_DIR}"
    mkdir -p "${DIST_DIR}"
}

# Build for a specific platform
build_platform() {
    local platform="$1"
    local goos goarch
    
    IFS='/' read -r goos goarch <<< "${platform}"
    
    local output_name="${BINARY_NAME}-${VERSION}-${goos}-${goarch}"
    local output_file="${DIST_DIR}/${output_name}"
    local ext=""
    
    # Add .exe extension for Windows
    if [[ "${goos}" == "windows" ]]; then
        ext=".exe"
        output_file="${output_file}${ext}"
    fi
    
    print_info "Building ${goos}/${goarch}..."
    
    # Build with optimizations
    local ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildDate=${BUILD_DATE}"
    
    if CGO_ENABLED="${CGO_ENABLED}" GOOS="${goos}" GOARCH="${goarch}" go build \
        -ldflags "${ldflags}" \
        -trimpath \
        -o "${output_file}" \
        "${MAIN_PATH}" 2>/dev/null; then
        
        local size
        size=$(du -h "${output_file}" | cut -f1)
        print_info "  Built: ${output_name}${ext} (${size})"
        
        # Compress if enabled
        if [[ "${COMPRESS}" == "true" ]]; then
            compress_binary "${output_file}" "${output_name}" "${goos}"
        fi
        
        return 0
    else
        print_warn "  Failed to build ${goos}/${goarch}"
        return 1
    fi
}

# Compress binary into archive
compress_binary() {
    local binary="$1"
    local name="$2"
    local goos="$3"
    
    local archive_name
    
    # Prepare files to include in archive
    local files_to_archive=("$(basename "${binary}")")
    if [[ "${INCLUDE_README}" == "true" ]] && [[ -f "${PROJECT_ROOT}/README.md" ]]; then
        cp "${PROJECT_ROOT}/README.md" "${DIST_DIR}/README.md"
        files_to_archive+=("README.md")
    fi
    
    if [[ "${goos}" == "windows" ]]; then
        # Create zip for Windows
        archive_name="${DIST_DIR}/${name}.zip"
        if command -v zip &>/dev/null; then
            (cd "${DIST_DIR}" && zip -q "${name}.zip" "${files_to_archive[@]}")
            print_info "  Compressed: ${name}.zip"
        fi
    else
        # Create tar.gz for Unix systems
        archive_name="${DIST_DIR}/${name}.tar.gz"
        if command -v tar &>/dev/null; then
            tar -czf "${archive_name}" -C "${DIST_DIR}" "${files_to_archive[@]}"
            print_info "  Compressed: ${name}.tar.gz"
        fi
    fi
    
    # Remove README copy after archiving
    if [[ "${INCLUDE_README}" == "true" ]]; then
        rm -f "${DIST_DIR}/README.md"
    fi
}

# Generate checksums
generate_checksums() {
    print_header "Generating Checksums"
    
    local checksum_file="${DIST_DIR}/checksums.txt"
    
    # Try different checksum tools
    local checksum_cmd=""
    if command -v sha256sum &>/dev/null; then
        checksum_cmd="sha256sum"
    elif command -v shasum &>/dev/null; then
        checksum_cmd="shasum -a 256"
    else
        print_warn "No checksum tool found, skipping checksums"
        return
    fi
    
    print_info "Using ${checksum_cmd}..."
    
    # Generate checksums for all files in dist
    (cd "${DIST_DIR}" && find . -maxdepth 1 -type f ! -name "checksums.txt" -exec ${checksum_cmd} {} \; | sort) > "${checksum_file}"
    
    print_info "Checksums written to: checksums.txt"
}

# Print release summary
print_summary() {
    print_header "Release Summary"
    
    print_info "Version: ${VERSION}"
    print_info "Commit: ${COMMIT}"
    print_info "Build Date: ${BUILD_DATE}"
    print_info ""
    print_info "Release files:"
    
    ls -lh "${DIST_DIR}" | tail -n +2 | while read -r line; do
        echo "  ${line}"
    done
    
    echo ""
    print_info "Release build complete!"
}

# Main function
main() {
    print_header "Yunt Release Build"
    print_info "Version: ${VERSION}"
    print_info "Platforms: ${PLATFORMS_ARRAY[*]}"
    
    check_prerequisites
    clean_dist
    
    # Change to project root
    cd "${PROJECT_ROOT}"
    
    # Build web UI if requested
    build_web_ui
    
    print_header "Building Binaries"
    
    local success_count=0
    local total_count=${#PLATFORMS_ARRAY[@]}
    
    for platform in "${PLATFORMS_ARRAY[@]}"; do
        if build_platform "${platform}"; then
            ((success_count++))
        fi
    done
    
    print_info ""
    print_info "Built ${success_count}/${total_count} platforms successfully"
    
    # Generate checksums if enabled
    if [[ "${CHECKSUM}" == "true" ]] && [[ ${success_count} -gt 0 ]]; then
        generate_checksums
    fi
    
    print_summary
    
    # Exit with error if no builds succeeded
    if [[ ${success_count} -eq 0 ]]; then
        print_error "No platforms built successfully"
        exit 1
    fi
}

main "$@"
