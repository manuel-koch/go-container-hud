NOW                     := $(shell date -u +'%Y-%m-%d_%TZ')
HEAD_SHA1               := $(shell git rev-parse HEAD)
HEAD_TAG                := $(shell git tag --points-at HEAD | grep -e "^v" | sort | tail -1 | cut -b2-)
MACAPP_GO               := ~/workspace/macapp.go/macapp.go
CODE_SIGN_CERT          := Manuel Koch Code Sign
APP_NAME                := Container-HUD
DARWIN_APP_ID           := com.manuel-koch.container-hud
DARWIN_ARM64_EXE_NAME   := container-hud.darwin-arm64
DARWIN_ARM64_DIST_DIR   := ./dist/darwin_arm64
DARWIN_ARM64_APP_BUNDLE := $(DARWIN_ARM64_DIST_DIR)/$(APP_NAME).app
DARWIN_ARM64_DMG        := $(DARWIN_ARM64_DIST_DIR)/$(APP_NAME)_darwin-arm64_$(HEAD_TAG).dmg

build_darwin_arm64::
	env GOOS=darwin GOARCH=arm64 go build \
		-ldflags "-s -w -X main.versionTag=$(HEAD_TAG) -X main.versionSha1=$(HEAD_SHA1) -X main.buildDate=$(NOW)" \
		-o build/$(DARWIN_ARM64_EXE_NAME) \
		.

build_darwin_arm64_app:: build_darwin_arm64
	@echo ===========================================================
	@echo == Cleaning dist artifacts darwin_arm64
	echo -[ -d $(DARWIN_ARM64_APP_BUNDLE) ] && rm -rf $(DARWIN_ARM64_APP_BUNDLE)
	echo -[ -f $(DARWIN_ARM64_DMG) ] && rm -rf $(DARWIN_ARM64_DMG)
	@echo ===========================================================
	@echo == Building app bundle darwin_arm64
	go run $(MACAPP_GO) \
    		-assets ./build \
    		-bin $(DARWIN_ARM64_EXE_NAME) \
			-icon ./Icon.png \
			-identifier $(DARWIN_APP_ID) \
			-name $(APP_NAME) \
			-o $(DARWIN_ARM64_DIST_DIR)
	plutil -replace CFBundleShortVersionString -string $(HEAD_TAG) $(DARWIN_ARM64_APP_BUNDLE)/Contents/Info.plist
	@echo ===========================================================
	@echo == Signing app bundle
	#security find-certificate -c "$(CODE_SIGN_CERT)" -p | openssl x509 -noout -text  -inform pem | grep -E "Validity|(Not (Before|After)\s*:)"
	#codesign --verbose=4 --force --deep --sign "$(CODE_SIGN_CERT)" $(DARWIN_ARM64_APP_BUNDLE)
	#codesign --verbose=4 --display $(DARWIN_ARM64_APP_BUNDLE)
	@echo ===========================================================
	@echo == Building app disk image
	create-dmg --volname $(notdir $(DARWIN_ARM64_DMG)) --volicon $(DARWIN_ARM64_APP_BUNDLE)/Contents/Resources/icon.icns \
             --icon $(notdir $(DARWIN_ARM64_APP_BUNDLE)) 110 150 \
             --app-drop-link 380 150 \
             --background ./dmg_bg.png \
             $(DARWIN_ARM64_DMG) \
             $(DARWIN_ARM64_APP_BUNDLE)
	cd $(dir $(DARWIN_ARM64_DMG)) && shasum -a 256 $(notdir $(DARWIN_ARM64_DMG)) > $(notdir $(DARWIN_ARM64_DMG)).sha256

bundle::
	@echo "package main\n" > resources.go
	fyne bundle --append --output resources.go --name heartHealthyIconData heart-healthy.png
	fyne bundle --append --output resources.go --name heartUnhealthyIconData heart-unhealthy.png
	fyne bundle --append --output resources.go --name heartUnknownIconData heart-unknown.png
	fyne bundle --append --output resources.go --name restartIconData restart.png
	fyne bundle --append --output resources.go --name stopIconData stop.png
	@echo "You MUST edit resource.go !"