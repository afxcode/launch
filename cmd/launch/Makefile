.PHONY = all clean windows_amd64 compile

compile: windows_amd64

windows_amd64:
	@echo "Compiling for windows AMD64"
	go-winres make
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o bin/launch-windows-amd64.exe -trimpath

clean:
	@echo "Cleaning up..."
	@rm -rvf bin

all: clean windows_amd64