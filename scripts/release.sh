#!/bin/bash

# Lettersmith Release Script
# Helps create semantic version releases

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if we're on main branch
current_branch=$(git branch --show-current)
if [ "$current_branch" != "main" ]; then
    print_error "You must be on the main branch to create a release"
    print_status "Current branch: $current_branch"
    exit 1
fi

# Check if working directory is clean
if ! git diff-index --quiet HEAD --; then
    print_error "Working directory is not clean. Please commit or stash changes."
    git status --short
    exit 1
fi

# Get current version
current_version=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
print_status "Current version: $current_version"

# Remove 'v' prefix for version comparison
current_version_number=${current_version#v}

# Parse version components
IFS='.' read -r major minor patch <<< "$current_version_number"

# Show version bump options
echo ""
echo "Choose version bump type:"
echo "1) Patch (${major}.${minor}.$((patch + 1))) - Bug fixes"
echo "2) Minor (${major}.$((minor + 1)).0) - New features"
echo "3) Major ($((major + 1)).0.0) - Breaking changes"
echo "4) Custom version"
echo "5) Exit"

read -p "Enter choice [1-5]: " choice

case $choice in
    1)
        new_version="v${major}.${minor}.$((patch + 1))"
        ;;
    2)
        new_version="v${major}.$((minor + 1)).0"
        ;;
    3)
        new_version="v$((major + 1)).0.0"
        ;;
    4)
        read -p "Enter custom version (e.g., v1.2.3): " custom_version
        if [[ ! $custom_version =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.-]+)?$ ]]; then
            print_error "Invalid version format. Use format: v1.2.3 or v1.2.3-beta.1"
            exit 1
        fi
        new_version="$custom_version"
        ;;
    5)
        print_status "Exiting without creating release"
        exit 0
        ;;
    *)
        print_error "Invalid choice"
        exit 1
        ;;
esac

print_status "New version will be: $new_version"

# Get release notes
print_status "Generating changelog since last release..."
changelog=$(git log --pretty=format:"- %s" ${current_version}..HEAD 2>/dev/null || git log --pretty=format:"- %s")

echo ""
echo "Changelog:"
echo "$changelog"
echo ""

read -p "Continue with release? [y/N]: " confirm
if [[ ! $confirm =~ ^[Yy]$ ]]; then
    print_status "Release cancelled"
    exit 0
fi

# Create and push tag
print_status "Creating git tag: $new_version"
git tag -a "$new_version" -m "Release $new_version

$changelog"

print_status "Pushing tag to origin..."
git push origin "$new_version"

print_success "Release $new_version created!"
print_status "GitHub Actions will now build and publish the Docker image"
print_status "Check the Actions tab for build progress: https://github.com/$(git config --get remote.origin.url | sed 's/.*github.com[:/]\([^.]*\).*/\1/')/actions"

# Show next steps
echo ""
print_status "Next steps:"
echo "1. Wait for GitHub Actions to complete the build"
echo "2. The following Docker images will be available:"
echo "   - ghcr.io/$(git config --get remote.origin.url | sed 's/.*github.com[:/]\([^.]*\).*/\1/' | tr '[:upper:]' '[:lower:]')/lettersmith:$new_version"
echo "   - ghcr.io/$(git config --get remote.origin.url | sed 's/.*github.com[:/]\([^.]*\).*/\1/' | tr '[:upper:]' '[:lower:]')/lettersmith:latest"
echo "3. A GitHub release will be created automatically" 