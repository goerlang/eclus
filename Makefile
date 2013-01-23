GOPATH := $(shell pwd)

all: eclus


eclus:
	GOPATH=$(GOPATH) go get $@
	GOPATH=$(GOPATH) go build $@

clean:
	GOPATH=$(GOPATH) go clean
	${RM} -r pkg/

.PHONY: eclus
