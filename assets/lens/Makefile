REPOSITORY := gravitational.io
NAME := lens-app
VERSION ?= 0.0.1
BUILD_DIR ?=
OUT ?= $(BUILD_DIR)/$(NAME)-$(VERSION).tar.gz
GRAVITY ?= gravity
export

DEBIAN_VERSION := buster
LENS_VERSION := $(VERSION)
HOOK_VERSION := $(VERSION)

GRAVITY_EXTRA_FLAGS ?=
GRAVITY_IMAGE_FLAGS := --set-image=quay.io/gravitational/debian-tall:$(DEBIAN_VERSION) \
	--set-image=gravitational/lens:$(VERSION) \
	--set-image=gravitational/lens-hook:$(VERSION)

#
# import imports Lens as a Gravity application.
#
.PHONY: import
import: images
	-$(GRAVITY) app delete --force --insecure  \
		$(GRAVITY_EXTRA_FLAGS) \
		$(REPOSITORY)/$(NAME):$(VERSION)
	$(GRAVITY) app import --vendor --version=$(VERSION) \
		$(GRAVITY_IMAGE_FLAGS) $(GRAVITY_EXTRA_FLAGS) \
		--include=resources --include=registry .

#
# images builds all Lens Docker images.
#
.PHONY: images
images:
	$(MAKE) -C images

#
# check-vars makes sure all required variables are set.
#
.PHONY: check-vars
check-vars:
	@if [ -z "$(BUILD_DIR)" ]; then \
		echo "BUILD_DIR is not set"; exit 1; \
	fi;
