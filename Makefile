NAME=sv-rollout
PACKAGE=github.com/Shopify/ejson
VERSION=$(shell cat VERSION)
DEB=pkg/$(NAME)_$(VERSION)_amd64.deb

GOFILES=$(shell find . -type f -name '*.go')
MANFILES=$(shell find man -name '*.ronn' -exec echo build/{} \; | sed 's/\.ronn/\.gz/')

GODEP_PATH=$(shell pwd)/Godeps/_workspace

BUNDLE_EXEC=bundle exec

.PHONY: default all binaries man clean dev_bootstrap

default: all
all: deb
binaries: build/bin/linux-amd64
deb: $(DEB)
man: $(MANFILES)

build/man/%.gz: man/%.ronn
	mkdir -p "$(@D)"
	$(BUNDLE_EXEC) ronn -r --pipe "$<" | gzip > "$@"

build/bin/linux-amd64: $(GOFILES) version.go
	GOPATH=$(GODEP_PATH):$$GOPATH gox -osarch="linux/amd64" -output="$@" .

version.go: VERSION
	echo 'package main\n\nconst VERSION string = "$(VERSION)"' > $@

$(DEB): build/bin/linux-amd64 man
	mkdir -p $(@D)
	$(BUNDLE_EXEC) fpm \
		-t deb \
		-s dir \
		--name="$(NAME)" \
		--version="$(VERSION)" \
		--package="$@" \
		--license=MIT \
		--category=admin \
		--no-depends \
		--no-auto-depends \
		--architecture=amd64 \
		--maintainer="Burke Libbey <burke.libbey@shopify.com>" \
		--description="utility to restart multiple runit services concurrently" \
		--url="https://github.com/Shopify/sv-rollout" \
		./build/man/=/usr/share/man/ \
		./$<=/usr/bin/$(NAME)

clean:
	rm -rf build pkg

dev_bootstrap: version.go
	GOPATH=$(GODEP_PATH):$$GOPATH go get github.com/mitchellh/gox
	GOPATH=$(GODEP_PATH):$$GOPATH go install github.com/mitchellh/gox
	GOPATH=$(GODEP_PATH):$$GOPATH gox -build-toolchain -osarch="linux/amd64"
	bundle install
