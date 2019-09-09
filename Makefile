VERSION     ?= 0.0.666
NOW         := $(shell date)
GITCOMMIT   := $(shell git rev-parse --short HEAD 2>/dev/null)
BUILD_FLAGS := -ldflags="-w -X halkyon.io/hal/pkg/hal/cli/version.commit=$(GITCOMMIT) -X halkyon.io/hal/pkg/hal/cli/version.version=$(VERSION) -X 'halkyon.io/hal/pkg/hal/cli/version.date=$(NOW)'"

.PHONY: build
build:
	@echo "> Build go application"
	go build $(BUILD_FLAGS) ./cmd/hal.go

version:
	@echo $(VERSION)