VERSION ?= dev
LDFLAGS := -ldflags="-X main.Version=$(VERSION)"

.PHONY: dev build-linux build-windows build-macos build-all
.PHONY: build-activator-windows build-activator-linux build-activator

dev:
	cd backend && go run main.go

build-linux:
	cd backend && GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o ../dist/tpdroid-linux . && \
	cd ../dist && tar czf tpdroid-linux.tar.gz tpdroid-linux && rm tpdroid-linux

build-windows:
	cd backend && GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o ../dist/tpdroid.exe .

build-macos:
	cd backend && GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o ../dist/tpdroid-macos-amd64 . && \
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o ../dist/tpdroid-macos-arm64 . && \
	cd ../dist && tar czf tpdroid-macos-amd64.tar.gz tpdroid-macos-amd64 && \
	tar czf tpdroid-macos-arm64.tar.gz tpdroid-macos-arm64 && \
	rm tpdroid-macos-amd64 tpdroid-macos-arm64

build-all: build-linux build-windows build-macos

# ── Activator ──────────────────────────────────────────
build-activator-windows:
	cd activator && GOOS=windows GOARCH=amd64 go build \
		-ldflags="-X main.defaultWorkerURL=https://licencias.tpdroid.workers.dev" \
		-o ../dist/activator.exe .

build-activator-linux:
	cd activator && GOOS=linux GOARCH=amd64 go build \
		-ldflags="-X main.defaultWorkerURL=https://licencias.tpdroid.workers.dev" \
		-o ../dist/activator-linux .

build-activator: build-activator-windows build-activator-linux

# ──────────────────────────────────────────────────────────────
# dist-windows
#
# LO QUE HACE:
#   Compila tpdroid.exe para Windows x64 y genera el instalador
#   TPDroid-Setup.exe listo para entregar al cliente.
#
# REQUISITO:
#   nsis instalado en Linux: sudo apt install nsis
#
# RESULTADO:
#   dist/TPDroid-Setup.exe  <- este archivo le das al cliente
#
# USO:
#   make dist-windows
# ──────────────────────────────────────────────────────────────
dist-windows:
	./script/build-installer.sh
