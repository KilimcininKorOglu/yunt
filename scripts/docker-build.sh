#!/usr/bin/env bash
#
# docker-build.sh - Multi-platform Docker image build script for Yunt
#
# Usage:
#   ./scripts/docker-build.sh [options]
#
# Options:
#   -t, --tag TAG           Image tag (default: latest)
#   -r, --registry REG      Registry prefix (default: ghcr.io/yunt)
#   -p, --platforms PLAT    Platforms to build (default: linux/amd64,linux/arm64)
#   -P, --push              Push images to registry
#   -l, --load              Load image locally (single platform only)
#   -c, --cache             Use registry cache for builds
#   -n, --no-cache          Disable build cache
#   -h, --help              Show this help message
#
# Examples:
#   ./scripts/docker-build.sh                          # Build for local testing
#   ./scripts/docker-build.sh -t v1.0.0 -P             # Build and push version
#   ./scripts/docker-build.sh -l                       # Build and load locally
#   ./scripts/docker-build.sh -p linux/amd64 -l        # Single platform local build
#   ./scripts/docker-build.sh -t latest -P -c          # Push with registry cache
#
# Environment Variables:
#   REGISTRY        Override default registry (default: ghcr.io/yunt)
#   IMAGE_NAME      Override image name (default: yunt)
#   DOCKER_BUILDKIT Enable BuildKit (default: 1)

set -euo pipefail

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Default configuration
IMAGE_NAME="${IMAGE_NAME:-yunt}"
REGISTRY="${REGISTRY:-ghcr.io/yunt}"
TAG="latest"
PLATFORMS="linux/amd64,linux/arm64"
PUSH=false
LOAD=false
USE_CACHE=false
NO_CACHE=false

# Enable BuildKit
export DOCKER_BUILDKIT=1

# Version information from git
VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo 'dev')}"
COMMIT="${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')}"
BUILD_DATE="${BUILD_DATE:-$(date -u +"%Y-%m-%dT%H:%M:%SZ")}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print functions
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_header() {
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}"
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
            -t|--tag)
                TAG="$2"
                shift 2
                ;;
            -r|--registry)
                REGISTRY="$2"
                shift 2
                ;;
            -p|--platforms)
                PLATFORMS="$2"
                shift 2
                ;;
            -P|--push)
                PUSH=true
                shift
                ;;
            -l|--load)
                LOAD=true
                shift
                ;;
            -c|--cache)
                USE_CACHE=true
                shift
                ;;
            -n|--no-cache)
                NO_CACHE=true
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

# Check prerequisites
check_prerequisites() {
    print_header "Checking Prerequisites"
    
    # Check Docker
    if ! command -v docker &>/dev/null; then
        print_error "Docker is not installed"
        exit 1
    fi
    
    local docker_version
    docker_version=$(docker version --format '{{.Server.Version}}' 2>/dev/null || echo "unknown")
    print_info "Docker version: ${docker_version}"
    
    # Check if buildx is available
    if ! docker buildx version &>/dev/null; then
        print_error "Docker Buildx is not available. Please install Docker Buildx."
        exit 1
    fi
    
    local buildx_version
    buildx_version=$(docker buildx version | head -1)
    print_info "Buildx version: ${buildx_version}"
}

# Setup buildx builder for multi-platform builds
setup_buildx() {
    local builder_name="yunt-multiplatform"
    
    # Check if builder exists
    if ! docker buildx inspect "${builder_name}" &>/dev/null; then
        print_info "Creating buildx builder: ${builder_name}"
        docker buildx create \
            --name "${builder_name}" \
            --driver docker-container \
            --platform linux/amd64,linux/arm64 \
            --bootstrap
    fi
    
    # Use the builder
    docker buildx use "${builder_name}"
    print_info "Using buildx builder: ${builder_name}"
}

# Build Docker image
build_image() {
    print_header "Building Docker Image"
    
    local full_image="${REGISTRY}/${IMAGE_NAME}"
    local tags=()
    local build_args=()
    local cache_args=()
    
    # Add version tag
    tags+=("-t" "${full_image}:${TAG}")
    
    # Add 'latest' tag if this is a version tag (starts with 'v')
    if [[ "${TAG}" =~ ^v[0-9] ]]; then
        tags+=("-t" "${full_image}:latest")
    fi
    
    # Add short commit tag for traceability
    if [[ "${TAG}" != "${COMMIT}" ]] && [[ "${COMMIT}" != "unknown" ]]; then
        tags+=("-t" "${full_image}:sha-${COMMIT}")
    fi
    
    print_info "Image: ${full_image}"
    print_info "Tags: ${TAG}"
    print_info "Platforms: ${PLATFORMS}"
    print_info "Version: ${VERSION}"
    print_info "Commit: ${COMMIT}"
    print_info "Build Date: ${BUILD_DATE}"
    
    # Build arguments
    build_args+=(
        "--build-arg" "VERSION=${VERSION}"
        "--build-arg" "COMMIT=${COMMIT}"
        "--build-arg" "BUILD_DATE=${BUILD_DATE}"
    )
    
    # Cache configuration
    if [[ "${USE_CACHE}" == "true" ]]; then
        cache_args+=(
            "--cache-from" "type=registry,ref=${full_image}:buildcache"
            "--cache-to" "type=registry,ref=${full_image}:buildcache,mode=max"
        )
        print_info "Using registry cache: ${full_image}:buildcache"
    fi
    
    # No cache option
    if [[ "${NO_CACHE}" == "true" ]]; then
        cache_args+=("--no-cache")
        print_info "Build cache disabled"
    fi
    
    # Output configuration
    local output_args=()
    if [[ "${PUSH}" == "true" ]]; then
        output_args+=("--push")
        print_info "Push enabled: images will be pushed to registry"
    elif [[ "${LOAD}" == "true" ]]; then
        # Load only works with single platform
        local platform_count
        platform_count=$(echo "${PLATFORMS}" | tr ',' '\n' | wc -l)
        if [[ ${platform_count} -gt 1 ]]; then
            print_warn "Loading multi-platform images is not supported"
            print_warn "Using only first platform for local load"
            PLATFORMS=$(echo "${PLATFORMS}" | cut -d',' -f1)
        fi
        output_args+=("--load")
        print_info "Load enabled: image will be loaded locally"
    else
        output_args+=("--output" "type=image,push=false")
        print_info "Building without push or load (verification only)"
    fi
    
    # Labels for OCI compliance
    local labels=(
        "--label" "org.opencontainers.image.title=Yunt Mail Server"
        "--label" "org.opencontainers.image.description=Lightweight development mail server"
        "--label" "org.opencontainers.image.version=${VERSION}"
        "--label" "org.opencontainers.image.revision=${COMMIT}"
        "--label" "org.opencontainers.image.created=${BUILD_DATE}"
        "--label" "org.opencontainers.image.source=https://github.com/yunt/yunt"
        "--label" "org.opencontainers.image.vendor=Yunt"
    )
    
    print_info ""
    print_info "Starting build..."
    
    # Run the build
    docker buildx build \
        --platform "${PLATFORMS}" \
        "${tags[@]}" \
        "${build_args[@]}" \
        "${cache_args[@]}" \
        "${output_args[@]}" \
        "${labels[@]}" \
        --progress=plain \
        --file "${PROJECT_ROOT}/Dockerfile" \
        "${PROJECT_ROOT}"
    
    print_info ""
    print_info "Build completed successfully!"
}

# Print build summary
print_summary() {
    print_header "Build Summary"
    
    local full_image="${REGISTRY}/${IMAGE_NAME}"
    
    print_info "Image: ${full_image}"
    print_info "Tags:"
    print_info "  - ${full_image}:${TAG}"
    if [[ "${TAG}" =~ ^v[0-9] ]]; then
        print_info "  - ${full_image}:latest"
    fi
    if [[ "${COMMIT}" != "unknown" ]]; then
        print_info "  - ${full_image}:sha-${COMMIT}"
    fi
    print_info "Platforms: ${PLATFORMS}"
    
    if [[ "${PUSH}" == "true" ]]; then
        print_info ""
        print_info "Images have been pushed to the registry."
        print_info "Pull with: docker pull ${full_image}:${TAG}"
    elif [[ "${LOAD}" == "true" ]]; then
        print_info ""
        print_info "Image loaded locally."
        print_info "Run with: docker run -p 1025:1025 -p 1143:1143 -p 8025:8025 ${full_image}:${TAG}"
    fi
}

# Cleanup buildx builder (optional)
cleanup_buildx() {
    local builder_name="yunt-multiplatform"
    if docker buildx inspect "${builder_name}" &>/dev/null; then
        print_info "Removing buildx builder: ${builder_name}"
        docker buildx rm "${builder_name}" || true
    fi
}

# Main function
main() {
    parse_args "$@"
    
    # Change to project root
    cd "${PROJECT_ROOT}"
    
    check_prerequisites
    setup_buildx
    build_image
    print_summary
}

main "$@"
