all:
	@echo Building executables/iapgo
	go build -o executables/iapgo main.go

clean:
	@echo Deleting all files in executables directory
	rm -f executables/*

windows:
	@echo Building executable for Windows
	GOOS=windows GOARCH=amd64 go build -o executables/iapgo.exe main.go

windows-arm64: windows-arm
windows-arm:
	@echo Building executable for Windows
	GOOS=windows GOARCH=arm64 go build -o executables/iapgo-arm64.exe main.go

mac: macos
mac-arm: macos
macos-arm: macos
macos-arm64: macos
mac-arm64: macos
macos:
	@echo Building executable for MacOS
	GOOS=darwin GOARCH=arm64 go build -o executables/iapgo-mac main.go

linux-arm: linux-arm64
linux-arm64:
	@echo Building executable for MacOS
	GOOS=linux GOARCH=arm64 go build -o executables/iapgo-linux-arm main.go

linux-amd64:
	@echo Building executable for MacOS
	GOOS=linux GOARCH=amd64 go build -o executables/iapgo-mac main.go