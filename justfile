default:
    @just --list

# build for the host into build/
build:
    go build -o build/osquery-condition .

# build the universal macOS binary that the release packages into dist/
dist:
    #!/usr/bin/env bash
    set -euo pipefail
    mkdir -p build dist
    for arch in arm64 amd64; do
        CGO_ENABLED=0 GOOS=darwin GOARCH=$arch \
            go build -trimpath -ldflags="-s -w" -o "build/osquery-condition-darwin-$arch" .
    done
    lipo -create -output dist/osquery-condition \
        build/osquery-condition-darwin-arm64 build/osquery-condition-darwin-amd64
    lipo dist/osquery-condition -verify_arch arm64 x86_64

# vet + run all tests
test:
    go vet ./...
    go test ./...

# lint, and check formatting and go.mod without changing files
lint:
    golangci-lint run ./...
    golangci-lint fmt --diff
    go mod tidy -diff

# format Go sources
fmt:
    golangci-lint fmt

# install the git hooks
hooks:
    prek install --force

# remove build artifacts
clean:
    rm -rf build dist release

# bump the version in the tracked files (patch|minor|major|...)
# --no-prompt: versionbump's confirmation loops forever when stdin isn't a TTY.
bump part="patch":
    go tool versionbump {{part}} --no-prompt
