.PHONY: all
all: format build test lint

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
example: build
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

.PHONY: test-examples
test-examples: build
	@echo "Running all examples..."
	@failed=0; \
	for config in $$(find examples -name "config.yaml" | sort); do \
		example_name=$$(echo $$config | sed 's|examples/||' | sed 's|/config.yaml||'); \
		echo "Testing example: $$example_name"; \
		if ./bin/cron-workflow-replicator --config $$config; then \
			echo "✓ Example $$example_name passed"; \
		else \
			echo "✗ Example $$example_name failed"; \
			failed=$$((failed + 1)); \
		fi; \
	done; \
	if [ $$failed -eq 0 ]; then \
		echo "All examples passed!"; \
	else \
		echo "$$failed example(s) failed"; \
		exit 1; \
	fi

.PHONY: lint
lint:
	@which golangci-lint > /dev/null || (curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b tools v2.7.2)
	./tools/golangci-lint run