.PHONY: gofmt govet test errcheck ensure-errcheck ensure-godep clean testserver golint ensure-golint

default: gofmt golint govet test deployify errcheck

testserver:
	env GOPATH=$$(godep path):$(GOPATH) goconvey

gofmt:
	go fmt ./...

govet:
	go vet ./...

test: ensure-godep
	godep go test ./...

deployify: ensure-godep
	godep go build -o $@ .

errcheck: ensure-errcheck
	env GOPATH="$$(godep path):$(GOPATH)" errcheck github.com/Shopify/deployify

golint: ensure-golint
	find . -name '*.go' | grep -v Godep | xargs golint

ensure-golint:
	if [[ -z "$(shell which golint)" ]]; then go get github.com/golang/lint/golint; fi

ensure-errcheck:
	if [[ -z "$(shell which errcheck)" ]]; then go get github.com/kisielk/errcheck; fi

ensure-godep:
	if [[ -z "$(shell which godep)" ]]; then go get github.com/tools/godep; fi

clean:
	rm -f deployify
