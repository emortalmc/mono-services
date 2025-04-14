MODULES := $(shell find . -name "go.mod" -type f -print0 | xargs -0 -I {} dirname {} )
MAKEFILE_DIR := $(shell pwd)

lint:
	@for module in $(MODULES); do \
		echo "Linting module: $$module"; \
		cd "$(MAKEFILE_DIR)"; \
		cd "$$module" && golangci-lint run ./...; \
	done

go-tidy:
	go work sync
	go mod tidy
	# Loop through all directories containing a go.mod file and run go mod tidy in each
	find . -type f -name "go.mod" -execdir go mod tidy \;

test:
	@for module in $(MODULES); do \
		echo "Testing module: $$module"; \
		cd "$(MAKEFILE_DIR)"; \
		cd "$$module" && go test ./...; \
	done

pre-commit:
	go-tidy
	lint
