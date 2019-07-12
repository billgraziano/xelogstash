TARGETDIR=.\deploy\xelogstash
#buildTime=$(shell %date%)
#sha1ver=_sha1ver_
# @echo "test"
sha1ver := $(shell git rev-parse HEAD)
test := $(shell date /t)
VERSIONFILE := .\cmd\xelogstash\version.go
#buildTime := $(shell @echo %date%)
#j=%date%

all: vet clean buildEXE copyFiles

vet:
	go vet -all -shadow .\cmd\xelogstash
	go vet -all -shadow .\applog .\config .\log .\logstash .\seq .\status .\summary .\xe

buildEXE:
#	@echo $(buildTime)
#	$(info "bang" $(sha1ver))
#	$(info $(test))
# 	go generate 
	go build -o "$(TARGETDIR)\xelogstash.exe" -a -ldflags "-X main.sha1ver=$(sha1ver)" .\cmd\xelogstash  

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

#gensrc:
#	echo $(test)
#	@echo package main > $(VERSIONFILE)
#	@echo const ( >> $(VERSIONFILE)
#	@echo 	sha1ver = "$(sha1ver)" >> $(VERSIONFILE)
#	@echo ) >> $(VERSIONFILE)


race: clean buildRace copyFiles

 



