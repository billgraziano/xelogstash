TARGETDIR=.\deploy\xelogstash
sha1ver := $(shell git rev-parse HEAD)
test := $(shell date /t)
VERSIONFILE := .\cmd\xelogstash\version.go

all: vet clean buildEXE copyFiles

vet:
	go vet -all -shadow .\cmd\xelogstash
	go vet -all -shadow .\applog .\config .\log .\logstash .\seq .\status .\summary .\xe

buildEXE:
#	go build -o "$(TARGETDIR)\xelogstash.exe" -a -ldflags "-X main.sha1ver=$(sha1ver)" .\cmd\xelogstash  
	cd cmd\xelogstash && govvv build -a -o "..\..\$(TARGETDIR)\xelogstash.exe"

buildRace:
#	go generate 
	go build -a -o "$(TARGETDIR)\xelogstash.exe" -race -ldflags "-X main.sha1ver=$(sha1ver)" .\cmd\xelogstash 

copyFiles:
	copy .\samples\*.toml $(TARGETDIR)
	copy .\samples\*.batch $(TARGETDIR)
	copy .\README.md $(TARGETDIR)

clean:
# 	del /Q embed_static.go
# 	del /Q /S $(TARGETDIR)\config
#	del /q $(TARGETDIR)\xelogstash.exe

race: clean buildRace copyFiles

 



