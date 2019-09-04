VERSION     ?= 0.0.666
NOW         := $(shell date)
GITCOMMIT   := $(shell git rev-parse --short HEAD 2>/dev/null)
BUILD_FLAGS := -ldflags="-w -X main.commit=$(GITCOMMIT) -X main.version=$(VERSION) -X 'main.date=$(NOW)'"

build:
	@echo "> Build go application"
	go build $(BUILD_FLAGS) ./cmd/hal.go

version:
	@echo $(VERSION)