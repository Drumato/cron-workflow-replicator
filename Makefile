.PHONY: all
all: format build test

.PHONY: format
format:
	go fmt ./...

.PHONY: build
build:
	go build -o bin/cron-workflow-replicator .

.PHONY: test
test:
	go test ./...

.PHONY: example
example:
	@if [ -z "$(NAME)" ]; then \
		echo "Error: NAME is required. Usage: NAME=v1alpha1/novalue make example"; \
		exit 1; \
	fi
	@echo "Generating expected output for example: $(NAME)"
	@mkdir -p test/$(NAME)
	@rm -rf test/$(NAME)/*
	./bin/cron-workflow-replicator --config examples/$(NAME)/config.yaml
	@if [ -d "output" ]; then \
		cp -r output/* test/$(NAME)/; \
		echo "Expected output files generated in test/$(NAME)/"; \
	else \
		echo "Warning: No output directory found"; \
	fi
