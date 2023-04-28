export GO = go
export GOOS ?= $(shell $(GO) env GOOS)
export GOARCH ?= $(shell $(GO) env GOARCH)

ifeq ($(GOARCH),armv6)
export GOARCH = arm
export GOARM = 6
endif

ifeq ($(GOARCH),armv7)
export GOARCH = arm
export GOARM = 7
endif

# Env Variables
export GO111MODULE = on
export CGO_ENABLED = 0
# export PRECOMMIT = poetry run pre-commit

BUILD_DIR = amplify
BINARY_NAME = amplify

ifeq ($(GOOS),windows)
BINARY_NAME := ${BINARY_NAME}.exe
CC = gcc.exe
endif

BINARY_PATH = bin/${GOOS}_${GOARCH}${GOARM}/${BINARY_NAME}

TAG ?= $(eval TAG := $(shell git describe --tags --always))$(TAG)
COMMIT ?= $(eval COMMIT := $(shell git rev-parse HEAD))$(COMMIT)
REPO ?= $(shell echo $$(cd ../${BUILD_DIR} && git config --get remote.origin.url) | sed 's/git@\(.*\):\(.*\).git$$/https:\/\/\1\/\2/')
BRANCH ?= $(shell cd ../${BUILD_DIR} && git branch | grep '^*' | awk '{print $$2}')
BUILDDATE ?= $(eval BUILDDATE := $(shell date -u +'%Y-%m-%dT%H:%M:%SZ'))$(BUILDDATE)
PACKAGE := $(shell echo "amplify_$(TAG)_${GOOS}_$(GOARCH)${GOARM}")
# PRECOMMIT_HOOKS_INSTALLED ?= $(shell grep -R "pre-commit.com" .git/hooks)
TEST_BUILD_TAGS ?= unit
TEST_PARALLEL_PACKAGES ?= 1
TEST_OUTPUT_FILE_PREFIX ?= test

PRIVATE_KEY_FILE := /tmp/private.pem
PUBLIC_KEY_FILE := /tmp/public.pem

define BUILD_FLAGS
-X github.com/bacalhau-project/amplify/pkg/version.GITVERSION=$(TAG)
endef

all: build

# Run init repo after cloning it
.PHONY: init
init:
# 	@ops/repo_init.sh 1>/dev/null
	@echo "Build environment initialized."

# # Run install pre-commit
# .PHONY: install-pre-commit
# install-pre-commit:
# 	@ops/install_pre_commit.sh 1>/dev/null
# 	@echo "Pre-commit installed."

## Run all pre-commit hooks
################################################################################
# Target: precommit
################################################################################
# .PHONY: precommit
# precommit: buildenvcorrect
# 	${PRECOMMIT} run --all
# 	cd python && make pre-commit

.PHONY: buildenvcorrect
buildenvcorrect:
	@echo "Checking build environment..."
# Checking GO
# @echo "Checking for go..."
# @which go
# @echo "Checking for go version..."
# @go version
# @echo "Checking for go env..."
# @go env
# @echo "Checking for go env GOOS..."
# @go env GOOS
# @echo "Checking for go env GOARCH..."
# @go env GOARCH
# @echo "Checking for go env GO111MODULE..."
# @go env GO111MODULE
# @echo "Checking for go env GOPATH..."
# @go env GOPATH
# @echo "Checking for go env GOCACHE..."
# @go env GOCACHE
# ===============
# Ensure that "pre-commit.com" is in .git/hooks/pre-commit to run all pre-commit hooks
# before each commit.
# Error if it's empty or not found.
# ifeq ($(PRECOMMIT_HOOKS_INSTALLED),)
# 	@echo "Pre-commit is not installed in .git/hooks/pre-commit. Please run 'make install-pre-commit' to install it."
# 	@exit 1
# endif
# 	@echo "Build environment correct."

################################################################################
# Target: build
################################################################################
.PHONY: build
build: build-amplify-ui build-amplify

.PHONY: build-ci
build-ci: build-amplify-ui build-amplify

.PHONY: build-dev
build-dev: build-amplify-ui build-ci
	sudo cp ${BINARY_PATH} /usr/local/bin

################################################################################
# Target: build-amplify
################################################################################
.PHONY: build-amplify
build-amplify: ${BINARY_PATH}

CMD_FILES := $(shell bash -c 'comm -23 <(git ls-files cmd) <(git ls-files cmd --deleted)')
PKG_FILES := $(shell bash -c 'comm -23 <(git ls-files pkg) <(git ls-files pkg --deleted)')

${BINARY_PATH}: ${CMD_FILES} ${PKG_FILES} main.go
	${GO} build -ldflags "${BUILD_FLAGS}" -trimpath -o ${BINARY_PATH} .

################################################################################
# Target: build-amplify-ui
################################################################################
RM := rm -fr
UI_DIR := ui

.PHONY: build-amplify-ui
build-amplify-ui: $(UI_DIR)/yarn.lock

$(UI_DIR)/dist:
	@mkdir -p $(UI_DIR)/dist

$(UI_DIR)/node_modules:
	@mkdir -p $(UI_DIR)/node_modules

$(UI_DIR)/yarn.lock: $(UI_DIR)/dist $(UI_DIR)/node_modules
	(cd $(UI_DIR) && yarn install && yarn build)

.PHONY: clean-amplify-ui
clean-amplify-ui:
	$(RM) $(UI_DIR)/node_modules $(UI_DIR)/dist

################################################################################
# Target: build-docker-images
################################################################################
AMPLIFY_IMAGE ?= ghcr.io/bacalhau-project/amplify
AMPLIFY_TAG ?= ${TAG}
.PHONY: build-amplify-image
build-amplify-image:
	docker build --progress=plain \
		--tag ${AMPLIFY_IMAGE}:latest \
		--file containers/amplify/Dockerfile \
		.

.PHONY: push-amplify-image
push-amplify-image:
	docker buildx build --push --progress=plain \
		--platform linux/amd64,linux/arm64 \
		--tag ${AMPLIFY_IMAGE}:${AMPLIFY_TAG} \
		--tag ${AMPLIFY_IMAGE}:latest \
		--label org.opencontainers.artifact.created=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ") \
		--label org.opencontainers.image.version=${AMPLIFY_TAG} \
		--label "org.opencontainers.image.source=https://github.com/bacalhau-project/amplify" \
		--cache-from=type=registry,ref=${AMPLIFY_IMAGE}:latest \
		--file containers/amplify/Dockerfile \
		.


################################################################################
# Target: *-tika-image
################################################################################

TIKA_IMAGE ?= ghcr.io/bacalhau-project/amplify/tika
TIKA_TAG ?= ${TAG}
.PHONY: build-tika-image
build-tika-image:
	docker build --progress=plain \
		--tag ${TIKA_IMAGE}:latest \
		--file containers/tika/Dockerfile \
		.

.PHONY: test-tika-image
test-tika-image: build-tika-image
	bash containers/tika/test.sh

.PHONY: push-tika-image
push-tika-image:
	docker buildx build --push --progress=plain \
		--platform linux/amd64,linux/arm64 \
		--tag ${TIKA_IMAGE}:${TIKA_TAG} \
		--tag ${TIKA_IMAGE}:latest \
		--label org.opencontainers.artifact.created=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ") \
		--label org.opencontainers.image.version=${TIKA_TAG} \
		--label "org.opencontainers.image.source=https://github.com/bacalhau-project/amplify" \
		--cache-from=type=registry,ref=${TIKA_IMAGE}:latest \
		--file containers/tika/Dockerfile \
		.

################################################################################
# Target: *-ffmpeg-image
################################################################################

FFMPEG_IMAGE ?= ghcr.io/bacalhau-project/amplify/ffmpeg
FFMPEG_TAG ?= ${TAG}
.PHONY: build-magick-image
build-ffmpeg-image:
	docker build --progress=plain \
		--tag ${FFMPEG_IMAGE}:latest \
		--file containers/ffmpeg/Dockerfile \
		.

.PHONY: test-ffmpeg-image
test-ffmpeg-image: build-ffmpeg-image
	bash containers/ffmpeg/test.sh

.PHONY: push-ffmpeg-image
push-ffmpeg-image:
	docker buildx build --push --progress=plain \
		--platform linux/amd64,linux/arm64 \
		--tag ${FFMPEG_IMAGE}:${FFMPEG_TAG} \
		--tag ${FFMPEG_IMAGE}:latest \
		--label org.opencontainers.artifact.created=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ") \
		--label org.opencontainers.image.version=${FFMPEG_TAG} \
		--label "org.opencontainers.image.source=https://github.com/bacalhau-project/amplify" \
		--cache-from=type=registry,ref=${FFMPEG_IMAGE}:latest \
		--file containers/ffmpeg/Dockerfile \
		.

################################################################################
# Target: *-frictionless-image
################################################################################

FRICTIONLESS_IMAGE ?= ghcr.io/bacalhau-project/amplify/frictionless
FRICTIONLESS_TAG ?= ${TAG}
.PHONY: build-frictionless-image
build-frictionless-image:
	docker build --progress=plain \
		--tag ${FRICTIONLESS_IMAGE}:latest \
		--file containers/frictionless/Dockerfile \
		.

.PHONY: test-frictionless-image
test-frictionless-image: build-frictionless-image
	bash containers/frictionless/test.sh

.PHONY: push-frictionless-image
push-frictionless-image:
	docker buildx build --push --progress=plain \
		--platform linux/amd64,linux/arm64 \
		--tag ${FRICTIONLESS_IMAGE}:${FRICTIONLESS_TAG} \
		--tag ${FRICTIONLESS_IMAGE}:latest \
		--label org.opencontainers.artifact.created=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ") \
		--label org.opencontainers.image.version=${FRICTIONLESS_TAG} \
		--label "org.opencontainers.image.source=https://github.com/bacalhau-project/amplify" \
		--cache-from=type=registry,ref=${FRICTIONLESS_IMAGE}:latest \
		--file containers/frictionless/Dockerfile \
		.

################################################################################
# Target: *-frictionless-extract-image
################################################################################

FRICTIONLESS_EXTRACT_IMAGE ?= ghcr.io/bacalhau-project/amplify/frictionless-extract
FRICTIONLESS_EXTRACT_TAG ?= ${TAG}
.PHONY: build-frictionless-extract-image
build-frictionless-extract-image:
	docker build --progress=plain \
		--tag ${FRICTIONLESS_EXTRACT_IMAGE}:latest \
		--file containers/frictionless-extract/Dockerfile \
		.

.PHONY: test-frictionless-extract-image
test-frictionless-extract-image: build-frictionless-extract-image
	bash containers/frictionless-extract/test.sh

.PHONY: push-frictionless-extract-image
push-frictionless-extract-image:
	docker buildx build --push --progress=plain \
		--platform linux/amd64,linux/arm64 \
		--tag ${FRICTIONLESS_EXTRACT_IMAGE}:${FRICTIONLESS_EXTRACT_TAG} \
		--tag ${FRICTIONLESS_EXTRACT_IMAGE}:latest \
		--label org.opencontainers.artifact.created=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ") \
		--label org.opencontainers.image.version=${FRICTIONLESS_EXTRACT_TAG} \
		--cache-from=type=registry,ref=${FRICTIONLESS_EXTRACT_IMAGE}:latest \
		--file containers/frictionless-extract/Dockerfile \
		.

################################################################################
# Target: *-ydata-profiling-image
################################################################################

YDATA_PROFILING_IMAGE ?= ghcr.io/bacalhau-project/amplify/ydata-profiling
YDATA_PROFILING_TAG ?= ${TAG}
.PHONY: build-ydata-profiling-image
build-ydata-profiling-image:
	docker build --progress=plain \
		--tag ${YDATA_PROFILING_IMAGE}:latest \
		--file containers/ydata-profiling/Dockerfile \
		.

.PHONY: test-ydata-profiling-image
test-ydata-profiling-image: build-ydata-profiling-image
	bash containers/ydata-profiling/test.sh

.PHONY: push-ydata-profiling-image
push-ydata-profiling-image:
	docker buildx build --push --progress=plain \
		--platform linux/amd64,linux/arm64 \
		--tag ${YDATA_PROFILING_IMAGE}:${YDATA_PROFILING_TAG} \
		--tag ${YDATA_PROFILING_IMAGE}:latest \
		--label org.opencontainers.artifact.created=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ") \
		--label org.opencontainers.image.version=${YDATA_PROFILING_TAG} \
		--label "org.opencontainers.image.source=https://github.com/bacalhau-project/amplify" \
		--cache-from=type=registry,ref=${YDATA_PROFILING_IMAGE}:latest \
		--file containers/ydata-profiling/Dockerfile \
		.


################################################################################
# Target: *-amplify-image
# Target: *-magick-image
################################################################################

MAGICK_IMAGE ?= ghcr.io/bacalhau-project/amplify/magick
MAGICK_TAG ?= ${TAG}
.PHONY: build-magick-image
build-magick-image:
	docker build --progress=plain \
		--tag ${MAGICK_IMAGE}:latest \
		--file containers/magick/Dockerfile \
		.

.PHONY: test-magick-image
test-magick-image: build-magick-image
	bash containers/magick/test.sh

.PHONY: push-magick-image
push-magick-image:
	docker buildx build --push --progress=plain \
		--platform linux/amd64,linux/arm64 \
		--tag ${MAGICK_IMAGE}:${MAGICK_TAG} \
		--tag ${MAGICK_IMAGE}:latest \
		--label org.opencontainers.artifact.created=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ") \
		--label org.opencontainers.image.version=${MAGICK_TAG} \
		--label "org.opencontainers.image.source=https://github.com/bacalhau-project/amplify" \
		--cache-from=type=registry,ref=${MAGICK_IMAGE}:latest \
		--file containers/magick/Dockerfile \
		.

################################################################################
# Target: *-detection-image
################################################################################

DETECTION_IMAGE ?= ghcr.io/bacalhau-project/amplify/detection
DETECTION_TAG ?= ${TAG}
.PHONY: build-detection-image
build-detection-image:
	docker build --progress=plain \
		--tag ${DETECTION_IMAGE}:latest \
		--file containers/detection/Dockerfile \
		.

.PHONY: test-detection-image
test-detection-image: build-detection-image
	bash containers/detection/test.sh

.PHONY: push-detection-image
push-detection-image:
	docker buildx build --push --progress=plain \
		--platform linux/amd64,linux/arm64 \
		--tag ${DETECTION_IMAGE}:${DETECTION_TAG} \
		--tag ${DETECTION_IMAGE}:latest \
		--label org.opencontainers.artifact.created=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ") \
		--label org.opencontainers.image.version=${DETECTION_TAG} \
		--cache-from=type=registry,ref=${DETECTION_IMAGE}:latest \
		--file containers/detection/Dockerfile \
		.

################################################################################
# Target: *-docker-images
################################################################################

.PHONY: build-docker-images
build-docker-images: build-amplify-image build-tika-image build-ffmpeg-image build-magick-image build-frictionless-image build-detection-image build-frictionless-extract-image

.PHONY: test-docker-images
test-docker-images: test-tika-image test-ffmpeg-image test-magick-image test-frictionless-image test-detection-image test-frictionless-extract-image

.PHONY: push-docker-images
push-docker-images: push-amplify-image push-tika-image push-ffmpeg-image push-magick-image push-frictionless-image push-detection-image push-frictionless-extract-image

# Release tarballs suitable for upload to GitHub release pages
################################################################################
# Target: build-amplify-tgz
################################################################################
.PHONY: build-amplify-tgz
build-amplify-tgz: dist/${PACKAGE}.tar.gz dist/${PACKAGE}.tar.gz.signature.sha256

dist/:
	mkdir -p $@

dist/${PACKAGE}.tar.gz: ${BINARY_PATH} | dist/
	tar cvzf $@ -C $(dir $(BINARY_PATH)) $(notdir ${BINARY_PATH})

dist/${PACKAGE}.tar.gz.signature.sha256: dist/${PACKAGE}.tar.gz | dist/
	openssl dgst -sha256 -sign $(PRIVATE_KEY_FILE) -passin pass:"$(PRIVATE_KEY_PASSPHRASE)" $^ | openssl base64 -out $@


################################################################################
# Target: images
################################################################################
IMAGE_REGEX := 'Image ?(:|=)\s*"[^"]+"'
FILES_WITH_IMAGES := $(shell grep -Erl ${IMAGE_REGEX} pkg cmd)

docker/.images: ${FILES_WITH_IMAGES}
	grep -Eroh ${IMAGE_REGEX} $^ | cut -d'"' -f2 | sort | uniq > $@

docker/.pulled: docker/.images
	- cat $^ | xargs -n1 docker pull
	touch $@

.PHONY: images
images: docker/.pulled

################################################################################
# Target: clean
################################################################################
.PHONY: clean
clean: clean-amplify-ui
	${GO} clean
	${RM} -r bin/*
	${RM} dist/amplify_*
	${RM} docker/.images
	${RM} docker/.pulled


################################################################################
# Target: test
################################################################################
.PHONY: test
test:
# unittests parallelize well (default go test behavior is to parallelize)
	CGO_ENABLED=1 go test ./... -v -race --tags=unit

.PHONY: integration-test
integration-test:
# integration tests parallelize less well (hence -p 1)
	export AMPLIFY_DB_URI=postgres://postgres:mysecretpassword@localhost/amplify?sslmode=disable
	go test ./... -v --tags=integration -p 1

.PHONY: grc-test
grc-test:
	grc go test ./... -v
.PHONY: grc-test-short
grc-test-short:
	grc go test ./... -test.short -v

.PHONY: test-debug
test-debug:
	LOG_LEVEL=debug go test ./... -v

.PHONY: grc-test-debug
grc-test-debug:
	LOG_LEVEL=debug grc go test ./... -v

.PHONY: test-one
test-one:
	go test -v -count 1 -timeout 3000s -run ^$(TEST)$$ github.com/bacalhau-project/amplify/cmd/amplify/

################################################################################
# Target: lint
################################################################################
.PHONY: lint
lint: build-amplify-ui
	golangci-lint run --timeout 10m

.PHONY: lint-fix
lint-fix:
	golangci-lint run --timeout 10m --fix

################################################################################
# Target: modtidy
################################################################################
.PHONY: modtidy
modtidy:
	go mod tidy

################################################################################
# Target: check-diff
################################################################################
.PHONY: check-diff
check-diff:
	git diff --exit-code ./go.mod # check no changes
	git diff --exit-code ./go.sum # check no changes

# Run the unittests and output results for recording
################################################################################
# Target: test-test-and-report
################################################################################
COMMA = ,
COVER_FILE := coverage/${PACKAGE}_$(subst ${COMMA},_,${TEST_BUILD_TAGS}).coverage

.PHONY: test-and-report
test-and-report: unittests.xml ${COVER_FILE}

${COVER_FILE} unittests.xml ${TEST_OUTPUT_FILE_PREFIX}_unit.json: ${BINARY_PATH} $(dir ${COVER_FILE})
	gotestsum \
		--jsonfile ${TEST_OUTPUT_FILE_PREFIX}_unit.json \
		--junitfile unittests.xml \
		--format testname \
		-- \
			-p ${TEST_PARALLEL_PACKAGES} \
			./pkg/... ./cmd/... \
			-coverpkg=./... -coverprofile=${COVER_FILE} \
			--tags=${TEST_BUILD_TAGS}

################################################################################
# Target: coverage-report
################################################################################
.PHONY:
coverage-report: build-amplify-ui coverage/coverage.html

coverage/coverage.out: $(wildcard coverage/*.coverage)
	gocovmerge $^ > $@

coverage/coverage.html: coverage/coverage.out coverage/
	go tool cover -html=$< -o $@

coverage/:
	mkdir -p $@

.PHONY: security
security:
	gosec -exclude=G204,G304 -exclude-dir=test ./...
	echo "[OK] Go security check was completed!"

release: build-amplify
	cp bin/amplify .

OAPI_CODEGEN := $(shell command -v oapi-codegen 2> /dev/null)
generate:
	@echo "Merging OpenAPI specs"
	(cd ui && yarn generate)
ifndef OAPI_CODEGEN
	$(error "Please install oapi-codegen")
endif
	@echo "Building models"
	oapi-codegen --config=api/cfg.yaml api/openapi.yaml
	sqlc generate

.PHONY: deploy-production
deploy-production:
	@echo "Deploying ${AMPLIFY_VERSION}"
	cd ops/terraform ; terraform init && terraform apply -auto-approve -var-file production.tfvars -var="amplify_version=${AMPLIFY_VERSION}"