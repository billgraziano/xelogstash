TARGETDIR=.\deploy\xelogstash
sha1ver := $(shell git rev-parse HEAD)
dt := $(shell date /t)
# VERSIONFILE := .\cmd\xelogstash\version.go

all: vet test clean buildEXE copyFiles

vet:
	go vet -all .\cmd\xelogstash
	go vet -all .\applog .\config .\log .\logstash .\seq .\status .\summary .\xe

buildEXE:
	echo "$(dt)"
# 	go build -o "$(TARGETDIR)\xelogstash.exe" -a -ldflags "-X main.sha1ver=$(sha1ver)" .\cmd\xelogstash  
	@echo "Building $(TARGETDIR)\xelogstash.exe ..."
	go build -o "$(TARGETDIR)\xelogstash.exe" -a -ldflags "-X main.sha1ver=$(sha1ver)" ".\cmd\xelogstash"
# #	cd cmd\xelogstash && govvv build -print && govvv build -a -o "..\..\$(TARGETDIR)\xelogstash.exe"

test:
	go test .\cmd\xelogstash .\applog .\config .\eshelper .\log .\logstash .\seq .\sink .\summary .\xe
# buildRace:
# #	go generate 
# 	go build -a -o "$(TARGETDIR)\xelogstash.exe" -race -ldflags "-X main.sha1ver=$(sha1ver)" .\cmd\xelogstash 

copyFiles:
	xcopy .\samples\start.toml $(TARGETDIR) /y
	xcopy .\samples\complete.toml $(TARGETDIR) /y
	xcopy .\samples\minimum.batch $(TARGETDIR) /y
	xcopy .\README.md $(TARGETDIR) /y

clean:
# 	del /Q embed_static.go
# 	del /Q /S $(TARGETDIR)\config
#	del /q $(TARGETDIR)\xelogstash.exe

# race: clean buildRace copyFiles

 



