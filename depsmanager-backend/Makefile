.PHONY: all build test generate_docs test_e2e
ALLDIRS=$(shell ls -d */ | grep -Fv -e 'vendor' -e 'mocks' -e 'pkg' -e'.git' -e 'client')
PACKAGES=$(shell GO111MODULE=on go list -mod=vendor ./... | grep -Fv -e '/vendor/')

all: build test tests

build:
	@echo Version: ${VERSION}
	@GO111MODULE=on go build -mod=vendor -v -ldflags "-X main.VERSION=${VERSION}" ${PACKAGES}

test:
	@echo ${PACKAGES}
	@GO111MODULE=on go test -v ${PACKAGES}

test_e2e:
	@GO111MODULE=on go test -mod=vendor -tags=test_e2e ./tests -count=1

generate_docs:
	go generate ./...

