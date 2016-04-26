NAME=sv-rollout
VERSION=$(shell cat VERSION)
DEB=pkg/$(NAME)_$(VERSION)_amd64.deb

MANFILES=$(shell find man -name '*.ronn' -exec echo build/{} \; | sed 's/\.ronn/\.gz/')

BUNDLE_EXEC=bundle exec

.PHONY: default all binaries man clean dependencies deb

default: all
all: clean deb
binaries: build/bin/linux-amd64
deb: $(DEB)
man: $(MANFILES)

mkfile_path := $(abspath $(lastword $(MAKEFILE_LIST)))
mkfile_dir  := $(abspath $(dir $(mkfile_path)))

build/man/%.gz: man/%.ronn
	mkdir -p "$(@D)"
	$(BUNDLE_EXEC) ronn -r --pipe "$<" | gzip > "$@"

build/bin/linux-amd64:
		script/compile

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
		--maintainer="borg@shopify.com" \
		--description="utility to restart multiple runit services concurrently" \
		--url="https://github.com/Shopify/sv-rollout" \
		./build/man/=/usr/share/man/ \
		./bin/sv-rollout=/usr/bin/$(NAME)

clean:
	rm -rf bin pkg build

dependencies:
	script/setup
