
VERSION="0.1.0"
TITLE=osquery-condition
OUTPUT=./build

$(OUTPUT):
	mkdir -p $@

.PHONY:
binary: $(OUTPUT)/apple/$(TITLE)
$(OUTPUT)/apple/$(TITLE): main.go
	@mkdir -p $(@D)
	GOOS=darwin GOARCH=arm64 go build -o $@ -ldflags "-X main.version=$(VERSION) -X main.gitHash=`git rev-parse HEAD`" ./*.go

build/intel/$(TITLE): cmd/cpe_puppetsync/*.go go.mod
	@mkdir -p $(@D)
	GOOS=darwin GOARCH=amd64 go build -o $@ -ldflags "-X main.version=$(VERSION) -X main.gitHash=`git rev-parse HEAD`" ./*.go

.PHONY:
clean:
	@rm -rf build/
