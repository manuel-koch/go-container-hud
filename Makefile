NOW       := $(shell date -u +'%Y-%m-%d_%TZ')
HEAD_SHA1 := $(shell git rev-parse HEAD)
HEAD_TAG  := $(shell git tag --points-at HEAD | grep -e "^v" | sort | tail -1 | cut -b2-)

build_macos_binary::
	go build -ldflags "-s -w -X main.versionTag=$(HEAD_TAG) -X main.versionSha1=$(HEAD_SHA1) -X main.buildDate=$(NOW)" -o build/container-hud .

build_macos_app:: build_macos_binary
	-[ -d container-hud.app ] && rm -rf container-hud.app
	fyne package -os darwin -icon Icon.png --name container-hud --executable build/container-hud

build:: build_macos_app

bundle::
	@echo "package main\n" > resources.go
	fyne bundle --append --output resources.go --name heartHealthyIconData heart-healthy.png
	fyne bundle --append --output resources.go --name heartUnhealthyIconData heart-unhealthy.png
	fyne bundle --append --output resources.go --name heartUnknownIconData heart-unknown.png
	fyne bundle --append --output resources.go --name restartIconData restart.png
	fyne bundle --append --output resources.go --name stopIconData stop.png
	@echo "You MUST edit resource.go !"