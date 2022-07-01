.PHONY: build
build:
	@go mod tidy
	@ginkgo -r --flake-attempts=3

.PHONY: test
test:
	@ginkgo -r
