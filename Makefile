NOW                     := $(shell date -u +'%Y-%m-%d_%TZ')
HEAD_SHA1               := $(shell git rev-parse HEAD)
HEAD_TAG                := $(shell git describe --tags | grep -e "^v" | sort | tail -1 | cut -b2-)
MACAPP_GO               := ~/workspace/macapp.go/macapp.go
CODE_SIGN_CERT          := Manuel Koch Code Sign
APP_NAME                := Container-HUD

DARWIN_APP_ID           := com.manuel-koch.container-hud

DARWIN_ARM64_EXE_NAME   := container-hud.darwin-arm64
DARWIN_ARM64_DIST_DIR   := dist/darwin_arm64
DARWIN_ARM64_APP_BUNDLE := $(DARWIN_ARM64_DIST_DIR)/$(APP_NAME).app
DARWIN_ARM64_DMG        := $(DARWIN_ARM64_DIST_DIR)/$(APP_NAME)_darwin-arm64_$(HEAD_TAG).dmg

DARWIN_AMD64_EXE_NAME   := container-hud.darwin-amd64
DARWIN_AMD64_DIST_DIR   := dist/darwin_amd64
DARWIN_AMD64_APP_BUNDLE := $(DARWIN_AMD64_DIST_DIR)/$(APP_NAME).app
DARWIN_AMD64_DMG        := $(DARWIN_AMD64_DIST_DIR)/$(APP_NAME)_darwin-amd64_$(HEAD_TAG).dmg

build/container-hud.%::
	@echo "*** Building $@"
	env GOOS=$(firstword $(subst -, ,$*)) GOARCH=$(lastword $(subst -, ,$*)) go build \
		-ldflags "-s -w -X main.versionTag=$(HEAD_TAG) -X main.versionSha1=$(HEAD_SHA1) -X main.buildDate=$(NOW)" \
		-o $@ \
		.
	@echo "*** Built $@"

build/macapp.go:
	@echo "*** Fetching $@"
	curl -o build/macapp.go https://gist.githubusercontent.com/mholt/11008646c95d787c30806d3f24b2c844/raw/0c07883ba937f2d066d125ce3efd731adfd899d7/macapp.go
	@echo "*** Fetched $@"

%.app:
	@echo "*** Building $@ from $<"
	go run build/macapp.go \
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

$(DARWIN_AMD64_APP_BUNDLE): build/$(DARWIN_AMD64_EXE_NAME) build/macapp.go
$(DARWIN_AMD64_APP_BUNDLE).signed: $(DARWIN_AMD64_APP_BUNDLE)
$(DARWIN_AMD64_DMG): $(DARWIN_AMD64_APP_BUNDLE) $(DARWIN_AMD64_APP_BUNDLE).signed

darwin_amd64_dmg: $(DARWIN_AMD64_DMG)

$(DARWIN_ARM64_APP_BUNDLE): build/$(DARWIN_ARM64_EXE_NAME) build/macapp.go
$(DARWIN_ARM64_APP_BUNDLE).signed: $(DARWIN_ARM64_APP_BUNDLE)
$(DARWIN_ARM64_DMG): $(DARWIN_ARM64_APP_BUNDLE) $(DARWIN_ARM64_APP_BUNDLE).signed

darwin_arm64_dmg: $(DARWIN_ARM64_DMG)

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