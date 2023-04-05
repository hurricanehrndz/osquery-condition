.DEFAULT_GOAL := all
VERSION := 0.1.0
TITLE := osquery-condition
OUTPUT := ./build

$(OUTPUT):
	mkdir -p $@

.PHONY:
binary: $(OUTPUT)/apple/$(TITLE)
$(OUTPUT)/apple/$(TITLE): main.go go.mod
	@mkdir -p $(@D)
	GOOS=darwin GOARCH=arm64 go build -o $@ -ldflags "-X main.version=$(VERSION) -X main.gitHash=`git rev-parse HEAD`" ./*.go

build/intel/$(TITLE): main.go go.mod
	@mkdir -p $(@D)
	GOOS=darwin GOARCH=amd64 go build -o $@ -ldflags "-X main.version=$(VERSION) -X main.gitHash=`git rev-parse HEAD`" ./*.go

.PHONY:
build: build/$(TITLE)
build/$(TITLE): build/intel/$(TITLE) build/apple/$(TITLE)
	@lipo -create -output $(@) build/intel/$(TITLE) build/apple/$(TITLE)
	$(eval SIGNING_ID=$(shell security find-identity -p basic -v | sed -n -r 's#.*Developer ID Application: (.*)"#\1#p'|head -1))
	@echo "Using signing id: $(SIGNING_ID)"
	@codesign --deep --force --options=runtime -i com.yelpcorp.osquery-condition --sign "Developer ID Application: $(SIGNING_ID)" --timestamp build/$(TITLE)

build-info.json:
	@sed  -E 's#(.*name.: ).*#\1"$(TITLE)-$(VERSION).pkg",#g;s#(.*version.: ).*#\1"$(VERSION)"#' \
		build-info.src.json > build-info.json

payload/usr/local/bin/$(TITLE): build/$(TITLE)
	@mkdir -p $(@D)
	@cp ./build/$(TITLE) $@

build/$(TITLE)-$(VERSION).pkg: build-info.json payload/usr/local/bin/$(TITLE)
	@munkipkg .

build/$(TITLE)-$(VERSION)_signed.pkg: build/$(TITLE)-$(VERSION).pkg
	$(eval SIGNING_ID=$(shell security find-identity -p basic -v | sed -n -r 's#.*Developer ID Installer: (.*)"#\1#p'|head -1))
	@echo "Using $(SIGNING_ID)"
	@productbuild --sign "Developer ID Installer: $(SIGNING_ID)" \
		--package $< $(@)

.PHONY:
all: build/$(TITLE)-$(VERSION)_signed.pkg

.PHONY:
clean:
	@rm -rf build/
	@rm -rf payload
	@rm -f build-info.json
