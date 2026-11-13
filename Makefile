APP      := build/bin/Sayki.app
SHERPA   := $(shell go list -m -f '{{.Dir}}' github.com/k2-fsa/sherpa-onnx-go-macos)/lib/$(if $(filter arm64,$(shell uname -m)),aarch64,x86_64)-apple-darwin
# A stable signing identity keeps macOS privacy grants across rebuilds; ad-hoc ("-") loses them.
SIGN_ID  ?= $(or $(shell security find-identity -v -p codesigning | awk -F'"' 'NR==1 && /"/ {print $$2}'),-)

SIGN_FLAGS ?=
ENTITLEMENTS ?=
# Developer ID + notarization lets anyone open the DMG without Gatekeeper warnings.
DEV_ID   ?= $(shell security find-identity -v -p codesigning | awk -F'"' '/Developer ID Application/ {print $$2; exit}')
NOTARY   ?= sayki-notary
VERSION  := $(shell /usr/libexec/PlistBuddy -c "Print :CFBundleShortVersionString" build/darwin/Info.plist)
RELEASE  := build/release
DMG      := $(RELEASE)/Sayki-$(VERSION)-$(shell uname -m).dmg
ZIP      := $(RELEASE)/Sayki-$(VERSION)-$(shell uname -m).zip
# dmgbuild lays out the installer window without scripting Finder; kept in a local venv.
DMGBUILD := build/.venv/bin/dmgbuild
DMG_BG   := build/darwin/dmg-background.png

.PHONY: app frontend test run clean dmg zip

app: frontend build/darwin/icon.icns
	rm -rf $(APP) && mkdir -p $(APP)/Contents/MacOS $(APP)/Contents/Frameworks $(APP)/Contents/Resources
	go build -trimpath -ldflags "-s -w" -o $(APP)/Contents/MacOS/sayki ./cmd/sayki
	cp build/darwin/Info.plist $(APP)/Contents/
	cp build/darwin/icon.icns $(APP)/Contents/Resources/
	cp $(SHERPA)/libsherpa-onnx-c-api.dylib $(SHERPA)/libonnxruntime.dylib $(APP)/Contents/Frameworks/
	chmod u+w $(APP)/Contents/Frameworks/*
	install_name_tool -delete_rpath $(SHERPA) $(APP)/Contents/MacOS/sayki
	install_name_tool -add_rpath @executable_path/../Frameworks $(APP)/Contents/MacOS/sayki
	codesign --force $(SIGN_FLAGS) -s "$(SIGN_ID)" $(APP)/Contents/Frameworks/*.dylib
	codesign --force $(SIGN_FLAGS) $(ENTITLEMENTS) -s "$(SIGN_ID)" $(APP)
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

# Release builds go to their own folder, so your local build/bin app keeps its
# privacy grants. With a Developer ID identity the DMG is signed with the hardened
# runtime, notarized (keychain profile $(NOTARY)) and stapled. Without one it is
# ad-hoc signed (never "Apple Development", which embeds its owner's email) and
# testers need System Settings → Privacy & Security → Open Anyway.
dmg: $(DMGBUILD) $(DMG_BG)
ifeq ($(DEV_ID),)
	@echo "No Developer ID Application identity found: building an ad-hoc signed test DMG."
	$(MAKE) app SIGN_ID=- APP=$(RELEASE)/Sayki.app
else
	$(MAKE) app SIGN_ID="$(DEV_ID)" APP=$(RELEASE)/Sayki.app SIGN_FLAGS="--options runtime --timestamp" ENTITLEMENTS="--entitlements build/darwin/entitlements.plist"
endif
	$(DMGBUILD) -s build/darwin/dmg-settings.py -D app=$(RELEASE)/Sayki.app -D background=$(DMG_BG) -D icon=build/darwin/icon.icns Sayki $(DMG)
ifneq ($(DEV_ID),)
	codesign --force --timestamp -s "$(DEV_ID)" $(DMG)
	xcrun notarytool submit $(DMG) --keychain-profile $(NOTARY) --wait
	xcrun stapler staple $(DMG)
	spctl --assess --type open --context context:primary-signature -v $(DMG)
endif
	rm -rf $(RELEASE)/Sayki.app # only the DMG is the release; a loose copy shows up as a second Sayki in Spotlight
	@echo "built $(DMG)"

# The ZIP download skips the disk image: unzip, open, and Sayki offers to move
# itself into Applications. It must be notarized (with the ticket stapled to the
# app inside) or Gatekeeper blocks it, so a Developer ID identity is required.
zip:
ifeq ($(DEV_ID),)
	$(error make zip needs a "Developer ID Application" signing identity)
endif
	$(MAKE) app SIGN_ID="$(DEV_ID)" APP=$(RELEASE)/Sayki.app SIGN_FLAGS="--options runtime --timestamp" ENTITLEMENTS="--entitlements build/darwin/entitlements.plist"
	rm -f $(ZIP) && ditto -c -k --keepParent $(RELEASE)/Sayki.app $(ZIP)
	xcrun notarytool submit $(ZIP) --keychain-profile $(NOTARY) --wait
	xcrun stapler staple $(RELEASE)/Sayki.app
	rm -f $(ZIP) && ditto -c -k --keepParent $(RELEASE)/Sayki.app $(ZIP)
	spctl --assess --type execute -v $(RELEASE)/Sayki.app
	rm -rf $(RELEASE)/Sayki.app
	@echo "built $(ZIP)"

$(DMGBUILD):
	python3 -m venv build/.venv && build/.venv/bin/pip install -q dmgbuild==1.6.7

$(DMG_BG): build/darwin/dmg-background.swift
	swift $< $@ 1
	swift $< $(@:.png=@2x.png) 2

clean:
	rm -rf build/bin build/release frontend/dist/assets
