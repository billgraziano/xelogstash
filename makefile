TARGETDIR=D:\Deploy\xelogstash
#buildTime=$(shell %date%)
#sha1ver=_sha1ver_
# @echo "test"
sha1ver := $(shell git rev-parse HEAD)
test := $(shell date /t)
#buildTime := $(shell @echo %date%)
#j=%date%

all: clean buildEXE copyFiles

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

clean:
# 	del /Q embed_static.go
# 	del /Q /S $(TARGETDIR)\config
#	del /q $(TARGETDIR)\xelogstash.exe

race: clean buildRace copyFiles

 



