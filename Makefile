BIN=hookah
HEAD=$(shell git describe --tags 2> /dev/null  || git rev-parse --short HEAD)

default: setup test install

setup:
ifeq ($(shell echo $$CI),true)
	go get -u -v
endif

test:
	go test ./...

install:
	go install ./cmd/hookah

.PHONY: clean
clean:
	-rm -rf release
	mkdir release

.PHONY: release
release: clean darwin64 linux64
	cd release/darwin64 && zip -9 ../$(BIN).darwin64.$(HEAD).zip $(BIN)
	cd release/linux64 && zip -9 ../$(BIN).linux64.$(HEAD).zip $(BIN)

darwin64:
	env GOOS=darwin GOARCH=amd64 go build -o release/darwin64/$(BIN) ./cmd/hookah

linux64:
	env GOOS=linux GOARCH=amd64 go build -o release/linux64/$(BIN) ./cmd/hookah
