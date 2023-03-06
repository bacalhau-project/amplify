.PHONY: push-tika-image build-images test

TAG ?= $(eval TAG := $(shell git describe --tags --always))$(TAG)

build-images:
	docker build -t ghcr.io/bacalhau-project/amplify/tika:latest -f ./containers/tika/Dockerfile .

TIKA_IMAGE ?= ghcr.io/bacalhau-project/amplify/tika
TIKA_TAG ?= ${TAG}
push-tika-image:
	docker buildx build --push --progress=plain \
		--platform linux/amd64,linux/arm64 \
		--tag ${TIKA_IMAGE}:${TIKA_TAG} \
		--tag ${TIKA_IMAGE}:latest \
		--label org.opencontainers.artifact.created=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ") \
		--label org.opencontainers.image.version=${TIKA_TAG} \
		--cache-from=type=registry,ref=${TIKA_IMAGE}:latest \
		--file containers/tika/Dockerfile \
		.

test: build-images
	bash ./containers/tika/test.sh

generate:
	oapi-codegen --config=api/cfg.yaml api/openapi.yaml