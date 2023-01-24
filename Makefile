APP_IMAGE_NAME=selfservice/ascraper
BUILD_NUMBER=n/a
OUTPUT_DIR=${PWD}/.output
OUTPUT_DIR_APP=${OUTPUT_DIR}/app
OUTPUT_DIR_MANIFESTS=${OUTPUT_DIR}/manifests
GOOS=linux
GOARCH=amd64
RICHGO:=$(shell which richgo 2>/dev/null)

init: ci

clean:
	@rm -Rf $(OUTPUT_DIR)
	@mkdir $(OUTPUT_DIR)
	@mkdir $(OUTPUT_DIR_APP)
	@mkdir $(OUTPUT_DIR_MANIFESTS)

restore:
	@go mod download -x

build:
	@ GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_BUILD=0 go build \
		-v \
		-ldflags='-extldflags=-static -w -s' \
		-tags netgo,osusergo \
		-o $(OUTPUT_DIR_APP)/ascraper

container:
	@docker build -t $(APP_IMAGE_NAME) .

manifests:
	@cp -r ./k8s/. $(OUTPUT_DIR_MANIFESTS)
	@find "$(OUTPUT_DIR_MANIFESTS)" -type f -name '*.yml' | xargs sed -i 's:{{BUILD_NUMBER}}:${BUILD_NUMBER}:g'

deliver:
	@sh ./push-container.sh "${APP_IMAGE_NAME}" "${BUILD_NUMBER}"

ci: clean restore build container manifests
cd: ci deliver

run:
	@go run main.go
