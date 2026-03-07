VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BINARY_NAME := mpd-discplayer
DIST_DIR := dist

ARCHS := amd64 arm64 armv6 armv7

.PHONY: all
all: build-all

.PHONY: clean
clean:
	rm -rf $(DIST_DIR)
	docker buildx rm multiarch-builder 2>/dev/null || true

.PHONY: setup
setup:
	@echo "Setting up buildx..."
	@docker buildx create --name multiarch-builder --use --bootstrap 2>/dev/null || \
		docker buildx use multiarch-builder

define build_arch
	@mkdir -p $(DIST_DIR)
	@echo "Building for $(3)..."
	@docker buildx build \
		--platform $(2) \
		--target export \
		--build-arg VERSION=$(VERSION) \
		--output type=local,dest=$(DIST_DIR)/tmp-$(1) \
		-f Dockerfile .
	@mv $(DIST_DIR)/tmp-$(1)/mpd-discplayer $(DIST_DIR)/$(BINARY_NAME)-$(1)
	@rm -rf $(DIST_DIR)/tmp-$(1)
endef

# $(1)=label $(2)=platform $(3)=desc $(4)=deb-arch
# armv6 keeps native armhf name; armv7 is renamed from armhf → armv7hf
define build_deb
	@mkdir -p $(DIST_DIR)
	@echo "Building .deb for $(3)..."
	@docker buildx build \
		--platform $(2) \
		--target deb-export \
		--build-arg VERSION=$(VERSION) \
		--output type=local,dest=$(DIST_DIR)/tmp-deb-$(1) \
		-f Dockerfile .
	@for f in $(DIST_DIR)/tmp-deb-$(1)/*.deb; do \
		mv "$$f" "$(DIST_DIR)/$$(basename "$$f" | sed 's/_armhf\.deb/_$(4).deb/')"; \
	done
	@rm -rf $(DIST_DIR)/tmp-deb-$(1)
endef

.PHONY: build-all
build-all: setup
	$(call build_arch,amd64,linux/amd64,amd64)
	$(call build_arch,arm64,linux/arm64,arm64 (RPi 3/4/5 64-bit))
	$(call build_arch,armv6,linux/arm/v6,armv6 (RPi 1/Zero))
	$(call build_arch,armv7,linux/arm/v7,armv7 (RPi 2/3 32-bit))
	@echo "Build complete! Binaries in $(DIST_DIR)/"
	@ls -lh $(DIST_DIR)/

.PHONY: build-amd64
build-amd64: setup
	$(call build_arch,amd64,linux/amd64,amd64)

.PHONY: build-arm64
build-arm64: setup
	$(call build_arch,arm64,linux/arm64,arm64 (RPi 3/4/5 64-bit))

.PHONY: build-armv6
build-armv6: setup
	$(call build_arch,armv6,linux/arm/v6,armv6 (RPi 1/Zero))

.PHONY: build-armv7
build-armv7: setup
	$(call build_arch,armv7,linux/arm/v7,armv7 (RPi 2/3 32-bit))

.PHONY: deb-all
deb-all: setup
	$(call build_deb,amd64,linux/amd64,amd64,amd64)
	$(call build_deb,arm64,linux/arm64,arm64 (RPi 3/4/5 64-bit),arm64)
	$(call build_deb,armv6,linux/arm/v6,armv6 (RPi 1/Zero),armhf)
	$(call build_deb,armv7,linux/arm/v7,armv7 (RPi 2/3 32-bit),armv7hf)
	@echo "Deb packages in $(DIST_DIR)/"
	@ls -lh $(DIST_DIR)/*.deb

.PHONY: deb-amd64
deb-amd64: setup
	$(call build_deb,amd64,linux/amd64,amd64,amd64)

.PHONY: deb-arm64
deb-arm64: setup
	$(call build_deb,arm64,linux/arm64,arm64 (RPi 3/4/5 64-bit),arm64)

.PHONY: deb-armv6
deb-armv6: setup
	$(call build_deb,armv6,linux/arm/v6,armv6 (RPi 1/Zero),armhf)

.PHONY: deb-armv7
deb-armv7: setup
	$(call build_deb,armv7,linux/arm/v7,armv7 (RPi 2/3 32-bit),armv7hf)

.PHONY: release
release: build-all
	@echo "Creating release archives..."
	@cd $(DIST_DIR) && for arch in $(ARCHS); do \
		echo "Packaging $(BINARY_NAME)-$$arch..."; \
		tar czf $(BINARY_NAME)-$$arch-$(VERSION).tar.gz $(BINARY_NAME)-$$arch; \
		sha256sum $(BINARY_NAME)-$$arch-$(VERSION).tar.gz > $(BINARY_NAME)-$$arch-$(VERSION).tar.gz.sha256; \
	done
	@echo "Release archives created in $(DIST_DIR)/"
	@ls -lh $(DIST_DIR)/*.tar.gz

.PHONY: test-local
test-local:
	@echo "Testing local build (native arch)..."
	go test -v ./...

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  make build-all      - Build for all architectures (amd64, arm64, armv6, armv7)"
	@echo "  make build-amd64    - Build for amd64 only"
	@echo "  make build-arm64    - Build for arm64 only (RPi 3/4/5 64-bit)"
	@echo "  make build-armv6    - Build for armv6 only (RPi 1/Zero)"
	@echo "  make build-armv7    - Build for armv7 only (RPi 2/3 32-bit)"
	@echo "  make deb-all        - Build .deb packages for all architectures"
	@echo "  make deb-amd64      - Build .deb for amd64 only"
	@echo "  make deb-arm64      - Build .deb for arm64 only"
	@echo "  make deb-armv6      - Build .deb for armv6 only (RPi 1/Zero) → _armhf.deb"
	@echo "  make deb-armv7      - Build .deb for armv7 only (RPi 2/3)    → _armv7hf.deb"
	@echo "  make release        - Build all binaries + create tar.gz archives"
	@echo "  make clean          - Remove dist/ and buildx builder"
	@echo "  make test-local     - Run tests locally"
	@echo "  make setup          - Setup Docker buildx"
