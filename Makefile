GOPATH := $(shell pwd)

all: eclus


eclus:
	GOPATH=$(GOPATH) go get $@
	GOPATH=$(GOPATH) go build $@

.PHONY: eclus
