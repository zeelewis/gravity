VERSION ?= 0.0.1
BUILD_DIR ?=
LENS_IMAGE := gravitational/lens:$(VERSION)
HOOK_IMAGE := gravitational/lens-hook:$(VERSION)

#
# all builds all Lens Docker images.
#
.PHONY: all
all: lens hook

#
# lens builds a Docker image with Lens admission server.
#
.PHONY: lens
lens: check-vars copy-binary
	docker build --pull \
		--tag $(LENS_IMAGE) \
		lens

#
# copy-binary copies Lens binary to Docker build context.
#
.PHONY: copy-binary
copy-binary:
	mkdir -p lens/build && \
		cp $(BUILD_DIR)/lens lens/build/

#
# hook builds a Docker image with Lens hooks.
#
.PHONY: hook
hook: check-vars
	$(eval CHANGESET = $(shell echo $$VERSION | sed -e 's/[\.]//g'))
	if [ -z "$(CHANGESET)" ]; then \
	  echo "CHANGESET is not set"; exit 1; \
	fi;
	docker build --pull \
		--build-arg CHANGESET=lens-$(CHANGESET) \
		--tag $(HOOK_IMAGE) \
		hook

#
# check-vars makes sure that all required variables are set.
#
.PHONY: check-vars
check-vars:
	@if [ -z "$(BUILD_DIR)" ]; then \
		echo "BUILD_DIR is not set"; exit 1; \
	fi;
