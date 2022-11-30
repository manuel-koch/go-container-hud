build:: build_macos

build_macos::
	-[ -d container-hud.app ] && rm -rf container-hud.app
	go build -ldflags "-s -w" -o build/container-hud .
	fyne package -os darwin -icon Icon.png --name container-hud --executable build/container-hud

bundle::
	@echo "package main\n" > resources.go
	fyne bundle --append --output resources.go --name heartHealthyIconData heart-healthy.png
	fyne bundle --append --output resources.go --name heartUnhealthyIconData heart-unhealthy.png
	fyne bundle --append --output resources.go --name heartUnknownIconData heart-unknown.png
	fyne bundle --append --output resources.go --name restartIconData restart.png
	fyne bundle --append --output resources.go --name stopIconData stop.png
	@echo "You MUST edit resource.go !"