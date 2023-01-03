NOW                     := $(shell date -u +'%Y-%m-%d_%TZ')
HEAD_SHA1               := $(shell git rev-parse HEAD)
HEAD_TAG                := $(shell git describe --tags | grep -e "^v" | sort | tail -1 | cut -b2-)
CODE_SIGN_CERT          := Manuel Koch Code Sign
APP_NAME                := Container-HUD

MACAPP_GO               := build/macapp.go

SPACE                   := $(subst ,, )
DARWIN_OS_VERSION       := $(subst $(SPACE),.,$(wordlist 1,2,$(subst ., ,$(shell sw_vers -productVersion))))
DARWIN_APP_ID           := com.manuel-koch.container-hud

DARWIN_ARM64_BINARY     := build/darwin-$(DARWIN_OS_VERSION)-arm64/container-hud.darwin-$(DARWIN_OS_VERSION)-arm64
DARWIN_ARM64_DIST_DIR   := dist/darwin-$(DARWIN_OS_VERSION)-arm64
DARWIN_ARM64_APP_BUNDLE := $(DARWIN_ARM64_DIST_DIR)/$(APP_NAME).app
DARWIN_ARM64_DMG        := $(DARWIN_ARM64_DIST_DIR)/$(APP_NAME)_v$(HEAD_TAG)_darwin_$(DARWIN_OS_VERSION)_arm64.dmg

DARWIN_AMD64_BINARY     := build/darwin-$(DARWIN_OS_VERSION)-amd64/container-hud.darwin-$(DARWIN_OS_VERSION)-amd64
DARWIN_AMD64_DIST_DIR   := dist/darwin-$(DARWIN_OS_VERSION)-amd64
DARWIN_AMD64_APP_BUNDLE := $(DARWIN_AMD64_DIST_DIR)/$(APP_NAME).app
DARWIN_AMD64_DMG        := $(DARWIN_AMD64_DIST_DIR)/$(APP_NAME)_v$(HEAD_TAG)_darwin_$(DARWIN_OS_VERSION)_amd64.dmg


container-hud.%:
	@echo "*** Building $@"
	env GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=1 go build \
		-ldflags "-s -w -X main.versionTag=$(HEAD_TAG) -X main.versionSha1=$(HEAD_SHA1) -X main.buildDate=$(NOW)" \
		-o $@ \
		.
	@echo "*** Built $@"

$(MACAPP_GO):
	@echo "*** Fetching $@"
	curl -o $(MACAPP_GO) https://gist.githubusercontent.com/mholt/11008646c95d787c30806d3f24b2c844/raw/0c07883ba937f2d066d125ce3efd731adfd899d7/macapp.go
	@echo "*** Fetched $@"

%.app:
	@echo "*** Building $@ from $<"
	env GOOS=$(GOOS) GOARCH=$(GOARCH) go run $(MACAPP_GO) \
    		-assets $(dir $<) \
    		-bin $(notdir $<) \
			-icon ./Icon.png \
			-identifier $(DARWIN_APP_ID) \
			-name $(APP_NAME) \
			-o $(dir $@)
	plutil -replace CFBundleShortVersionString -string $(HEAD_TAG) $@/Contents/Info.plist
	@echo "*** Built $@ from $<"

%.dmg:
	@echo "*** Creating $@"
	create-dmg --volname $(notdir $@) --volicon $</Contents/Resources/icon.icns \
             --icon $(notdir $<) 110 150 \
             --app-drop-link 380 150 \
             --background ./dmg_bg.png \
             $@ \
             $<
	cd $(dir $@) && shasum -a 256 $(notdir $@) > $(notdir $@).sha256
	@echo "*** Created $@"

.PHONY: %.signed
%.signed:
	@echo "*** Signing $*"
	security find-certificate -c "$(CODE_SIGN_CERT)" -p | openssl x509 -noout -text  -inform pem | grep -E "Validity|(Not (Before|After)\s*:)"
	codesign --verbose=4 --force --deep --sign "$(CODE_SIGN_CERT)" $*
	codesign --verbose=4 --display $*
	@echo "*** Signed $*"

$(DARWIN_AMD64_APP_BUNDLE): $(DARWIN_AMD64_BINARY) $(MACAPP_GO)
$(DARWIN_AMD64_APP_BUNDLE).signed: $(DARWIN_AMD64_APP_BUNDLE)
$(DARWIN_AMD64_DMG): $(DARWIN_AMD64_APP_BUNDLE) $(DARWIN_AMD64_APP_BUNDLE).signed

darwin_amd64: GOOS=darwin
darwin_amd64: GOARCH=amd64
darwin_amd64: $(DARWIN_AMD64_DMG)

$(DARWIN_ARM64_APP_BUNDLE): $(DARWIN_ARM64_BINARY) $(MACAPP_GO)
$(DARWIN_ARM64_APP_BUNDLE).signed: $(DARWIN_ARM64_APP_BUNDLE)
$(DARWIN_ARM64_DMG): $(DARWIN_ARM64_APP_BUNDLE) $(DARWIN_ARM64_APP_BUNDLE).signed

darwin_arm64: GOOS=darwin
darwin_arm64: GOARCH=arm64
darwin_arm64: $(DARWIN_ARM64_DMG)

.PHONY: clean
clean::
	-rm -rf build/*
	-rm -rf dist/*
	@echo "*** Clean"

bundle::
	@echo "package main\n" > resources.go
	fyne bundle --append --output resources.go --name heartHealthyIconData heart-healthy.png
	fyne bundle --append --output resources.go --name heartUnhealthyIconData heart-unhealthy.png
	fyne bundle --append --output resources.go --name heartUnknownIconData heart-unknown.png
	fyne bundle --append --output resources.go --name restartIconData restart.png
	fyne bundle --append --output resources.go --name stopIconData stop.png
	@echo "You MUST edit resource.go !"