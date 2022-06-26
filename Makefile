.PHONY: build
build:
	@go mod tidy
	@ginkgo -r

.PHONY: test
test:
	@ginkgo -r
