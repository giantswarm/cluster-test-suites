
##@ Ginkgo

.PHONY: ginkgo-run
ginkgo-run: ## Runs ginkgo against the test suites.
	@echo "====> $@"
	ginkgo --skip-package /X -v ./...

.PHONY: ginkgo-lint
ginkgo-lint: ## Runs ginkgolinter.
	@echo "====> $@"
	ginkgolinter --suppress-async-assertion=true ./...
