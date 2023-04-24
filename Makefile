BIN=hookah
HEAD=$(shell git describe --tags 2> /dev/null  || git rev-parse --short HEAD)

default: setup test install

setup:
ifeq ($(shell echo $$CI),true)
	cd cmd/hookah && go get -u -v
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
release: clean release/darwin_amd64 release/darwin_arm64 release/linux_amd64
	cd release/darwin_amd64 && zip -9 ../$(BIN).darwin_amd64.$(HEAD).zip $(BIN)
	cd release/darwin_arm64 && zip -9 ../$(BIN).darwin_arm64.$(HEAD).zip $(BIN)
	cd release/linux_amd64 && zip -9 ../$(BIN).linux_amd64.$(HEAD).zip $(BIN)

release/darwin_amd64:
	env GOOS=darwin GOARCH=amd64 go build -o release/darwin_amd64/$(BIN) ./cmd/hookah

release/darwin_arm64:
	env GOOS=darwin GOARCH=arm64 go build -o release/darwin_arm64/$(BIN) ./cmd/hookah

release/linux_amd64:
	env GOOS=linux GOARCH=amd64 go build -o release/linux_amd64/$(BIN) ./cmd/hookah
