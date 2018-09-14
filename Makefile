#.PHONY: .all
GO_IMAGE=golang:1.10-alpine
PLUGIN_SCOPE=edimarlnx
PLUGIN_NAME=docker-ebs-volume
PLUGIN_TAG ?= latest
BUILD_PATH=plugin-build

all: 

clean:
	@rm -rf $(BUILD_PATH)

# build-go:
# 	@docker run -it --rm \
# 		-v ~/go:/go \
# 		-v $$(pwd):/go/src/$(PLUGIN_NAME) \
# 		-w /go/src/$(PLUGIN_NAME) \
# 		$(GO_IMAGE) \
# 		/bin/sh -c \ "go build -ldflags=\"-s -w\" -o ./plugin-out/$(PLUGIN_NAME) *.go "
plugin-build: plugin-pre-build

plugin-pre-build:
	@echo "## Build image from Dockerfile"
	@docker build -q -t $(PLUGIN_SCOPE)/$(PLUGIN_NAME):$(PLUGIN_TAG) .
	@echo "## Create rootfs directory"
	@mkdir -p $(BUILD_PATH)/rootfs
	@echo "## Create temp container from image"
	@docker create --name plugin-tmp $(PLUGIN_SCOPE)/$(PLUGIN_NAME):$(PLUGIN_TAG)
	@echo "## Create rootfs plugin"
	@docker export plugin-tmp | tar -x -C $(BUILD_PATH)/rootfs
	@echo "## Copy config to builder plugin "
	@cp config.json $(BUILD_PATH)/
	@echo "## Remover temp container "
	@docker rm -vf plugin-tmp

plugin-create: plugin-pre-build
	@echo "## Remove plugin if exists"
	@docker plugin rm -f $(PLUGIN_SCOPE)/$(PLUGIN_NAME):$(PLUGIN_TAG) || true
	@echo "## Create plugin"
	@docker plugin create $(PLUGIN_SCOPE)/$(PLUGIN_NAME):$(PLUGIN_TAG) $(BUILD_PATH)

plugin-push: plugin-create
	@echo "## Push plugin"
	@docker plugin push $(PLUGIN_SCOPE)/$(PLUGIN_NAME):$(PLUGIN_TAG)

