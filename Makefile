.PHONY: prepare-envtest
prepare-envtest: setup-envtest
	@# Prepare a recent testenv
	@$(SETUP_ENVTEST) use --bin-dir "$(PWD)/testbin" 1.24.2 -v debug -p env > envtest.env

.PHONY: test
test: prepare-envtest
	source envtest.env; \
	unset TEST_ASSET_ETCD && \
	unset TEST_ASSET_KUBECTL && \
	unset TEST_ASSET_KUBE_APISERVER && \
	go test -tags envtest -race -v -covermode=atomic -coverprofile cover.out ./... && \
	go tool cover -html=cover.out -o coverage.html

SETUP_ENVTEST = $(shell pwd)/bin/setup-envtest
setup-envtest:
	$(call go-get-tool,$(SETUP_ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest,v0.0.0-20220722124738-f0351217e9e0)

# go-get-tool will 'go get' any package $2@$3 and install it to $1.
define go-get-tool
@set -e; \
if [ -f $(1) ]; then \
  [ -z $(3) ] && exit 0; \
  install_version=$$(go version -m "$(1)" | grep -E '[[:space:]]+mod[[:space:]]+' | awk '{print $$3}') ; \
  [ "$${install_version}" == "$(3)" ] && exit 0; \
  echo ">> $(1) $(2) $${install_version}==$(3)"; \
fi; \
module=$(2); \
if ! [ -z $(3) ]; then module=$(2)@$(3); fi; \
echo "Downloading $${module}" ;\
GOBIN=$(shell pwd)/bin go install $${module} ;
endef
