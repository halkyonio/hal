VERSION     ?= $(shell git describe --tags)
NOW         := $(shell date)
GITCOMMIT   := $(shell git rev-parse --short HEAD 2>/dev/null)
VERSION_FLAGS := -X halkyon.io/hal/pkg/hal/cli/version.commit=$(GITCOMMIT) -X halkyon.io/hal/pkg/hal/cli/version.version=$(VERSION) -X 'halkyon.io/hal/pkg/hal/cli/version.date=$(NOW)'
BUILD_FLAGS := -ldflags="-w $(VERSION_FLAGS)"
DEBUG_FLAGS := -ldflags="$(VERSION_FLAGS)"

.PHONY: build
build:
	@echo "> Build hal"
	go build $(BUILD_FLAGS) ./cmd/hal/hal.go

debug:
	@echo "> Build hal with debugging symbols"
	go build $(DEBUG_FLAGS) ./cmd/hal/hal.go

reference:
	@echo "> Generate hal command reference"
	go run $(BUILD_FLAGS) ./cmd/hal-doc/hal-doc.go reference >cli-reference.adoc

version:
	@echo $(VERSION)