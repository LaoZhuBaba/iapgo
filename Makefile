iapgo:
	@echo Building executables/iapgo
	go build -o executables/iapgo cmd/main.go

test:
	go test --count=1 --cover ./...
clean:
	@echo Deleting all files in executables directory
	rm -f executables/*

all: mac linux-amd64 linux-arm windows

windows:
	@echo Building executable for Windows
	GOOS=windows GOARCH=amd64 go build -o executables/iapgo.exe cmd/main.go

windows-arm64: windows-arm
windows-arm:
	@echo Building executable for Windows
	GOOS=windows GOARCH=arm64 go build -o executables/iapgo-arm64.exe cmd/main.go

mac: macos
mac-arm: macos
macos-arm: macos
macos-arm64: macos
mac-arm64: macos
macos:
	@echo Building executable for MacOS
	GOOS=darwin GOARCH=arm64 go build -o executables/iapgo-mac cmd/main.go

linux-arm: linux-arm64
linux-arm64:
	@echo Building executable for MacOS
	GOOS=linux GOARCH=arm64 go build -o executables/iapgo-linux-arm cmd/main.go

linux-amd64:
	@echo Building executable for MacOS
	GOOS=linux GOARCH=amd64 go build -o executables/iapgo-linux-amd64 cmd/main.go