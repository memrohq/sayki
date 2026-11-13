APP      := build/bin/Sayki.app
SHERPA   := $(shell go list -m -f '{{.Dir}}' github.com/k2-fsa/sherpa-onnx-go-macos)/lib/$(if $(filter arm64,$(shell uname -m)),aarch64,x86_64)-apple-darwin
# A stable signing identity keeps macOS privacy grants across rebuilds; ad-hoc ("-") loses them.
SIGN_ID  ?= $(or $(shell security find-identity -v -p codesigning | awk -F'"' 'NR==1 && /"/ {print $$2}'),-)

VERSION  := $(shell /usr/libexec/PlistBuddy -c "Print :CFBundleShortVersionString" build/darwin/Info.plist)
RELEASE  := build/release
DMG      := $(RELEASE)/Sayki-$(VERSION)-$(shell uname -m).dmg

.PHONY: app frontend test run clean dmg

app: frontend build/darwin/icon.icns
	rm -rf $(APP) && mkdir -p $(APP)/Contents/MacOS $(APP)/Contents/Frameworks $(APP)/Contents/Resources
	go build -trimpath -ldflags "-s -w" -o $(APP)/Contents/MacOS/sayki ./cmd/sayki
	cp build/darwin/Info.plist $(APP)/Contents/
	cp build/darwin/icon.icns $(APP)/Contents/Resources/
	cp $(SHERPA)/libsherpa-onnx-c-api.dylib $(SHERPA)/libonnxruntime.dylib $(APP)/Contents/Frameworks/
	chmod u+w $(APP)/Contents/Frameworks/*
	install_name_tool -delete_rpath $(SHERPA) $(APP)/Contents/MacOS/sayki
	install_name_tool -add_rpath @executable_path/../Frameworks $(APP)/Contents/MacOS/sayki
	codesign --force -s "$(SIGN_ID)" $(APP)/Contents/Frameworks/*.dylib
	codesign --force -s "$(SIGN_ID)" $(APP)
	@echo "built $(APP) (signed: $(SIGN_ID))"

frontend:
	cd frontend && npm ci --silent && npm run build

build/darwin/icon.icns: build/darwin/icon.swift
	swift build/darwin/icon.swift build/icon.png
	rm -rf build/icon.iconset && mkdir build/icon.iconset
	for s in 16 32 128 256 512; do \
	  sips -z $$s $$s build/icon.png --out build/icon.iconset/icon_$${s}x$${s}.png >/dev/null; \
	  sips -z $$((s*2)) $$((s*2)) build/icon.png --out build/icon.iconset/icon_$${s}x$${s}@2x.png >/dev/null; \
	done
	iconutil -c icns build/icon.iconset -o $@
	rm -rf build/icon.iconset build/icon.png

test:
	go test -race ./...

run: app
	open $(APP)

# Release builds are ad-hoc signed into their own folder: an "Apple Development"
# identity would embed its owner's email, and only Developer ID + notarization
# avoids Gatekeeper's "Open Anyway" step. Your local build/bin app is untouched.
dmg:
	$(MAKE) app SIGN_ID=- APP=$(RELEASE)/Sayki.app
	rm -rf $(RELEASE)/dmg && mkdir -p $(RELEASE)/dmg
	cp -R $(RELEASE)/Sayki.app $(RELEASE)/dmg/ && ln -s /Applications $(RELEASE)/dmg/Applications
	hdiutil create -volname Sayki -srcfolder $(RELEASE)/dmg -ov -format UDZO $(DMG)
	rm -rf $(RELEASE)/dmg
	@echo "built $(DMG)"

clean:
	rm -rf build/bin build/release frontend/dist/assets
