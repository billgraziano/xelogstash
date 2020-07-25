Param (
    [string]$version = "dev"
)
$ErrorActionPreference = "Stop"

Write-Output "Running PSBuild.ps1..."
Write-Output "" 
$deploy=".\deploy"
$target="$($deploy)\sqlxewriter"
Write-Output "Target:  $target"

Write-Output "Version: $($version)"

# $now = Get-Date -UFormat "%Y-%m-%d_%T_%Z"
$now = Get-Date -Format "yyyy'-'MM'-'dd'T'HH':'mm':'sszzz"
$sha1 = (git describe --tags --dirty --always).Trim()
Write-Output "Git:     $sha1"
Write-Output "Build:   $now"

Write-Output "" 
Write-Output "Running go vet..."
go vet -all .\cmd\xelogstash
if ($LastExitCode -ne 0) {
    exit
}

go vet -all .\cmd\sqlxewriter
if ($LastExitCode -ne 0) {
    exit
}

go vet -all .\config .\log .\logstash .\seq .\status .\summary .\xe .\sink .\pkg\...
if ($LastExitCode -ne 0) {
    exit
}

Write-Output "Running go test..."
go test .\cmd\xelogstash .\config .\seq .\xe .\sink .\status .\pkg\...
if ($LastExitCode -ne 0) {
    exit
}

Write-Output "Building xelogstash.exe..."
go build -o "$($target)\xelogstash.deprecated.exe" -a -ldflags "-X main.sha1ver=$sha1 -X main.buildTime=$now -X main.version=$version" ".\cmd\xelogstash"
if ($LastExitCode -ne 0) {
    exit
}

Write-Output "Building sqlxewriter.exe..."
go build -o "$($target)\sqlxewriter.exe" -a -ldflags "-X main.sha1ver=$sha1 -X main.buildTime=$now -X main.version=$version" ".\cmd\sqlxewriter"
if ($LastExitCode -ne 0) {
    exit
}

Write-Output "Building sqlxewriter_linux..."
$Env:GOARCH="amd64"
$Env:GOOS="linux"
go build -o "$($target)\sqlxewriter_linux" -a -ldflags "-X main.sha1ver=$sha1 -X main.buildTime=$now -X main.version=$version" ".\cmd\sqlxewriter"
if ($LastExitCode -ne 0) {
    exit
}

$Env:GOARCH="amd64"
$Env:GOOS="darwin"
Write-Output "Building sqlxewriter_darwin..."
go build -o "$($target)\sqlxewriter_darwin" -a -ldflags "-X main.sha1ver=$sha1 -X main.buildTime=$now -X main.version=$version" ".\cmd\sqlxewriter"
if ($LastExitCode -ne 0) {
    exit
}

$Env:GOARCH=""
$Env:GOOS=""

Write-Output "Copying Files..."
Copy-Item -Path ".\samples\*.toml"          -Destination $target
Copy-Item -Path ".\samples\*.sql"           -Destination $target
Copy-Item -Path ".\samples\minimum.batch"   -Destination $target
blackfriday-tool -css .\samples\style.css   -embed README-xelogstash.md "$($target)\README-xelogstash.html"
blackfriday-tool -css .\samples\style.css   -embed README.md "$($target)\README.html"

$stdZip = "$($deploy)\sqlxewriter.$($sha1).zip"
Write-Host "Writing $($stdZip)..."
$stdCompress = @{
    Path = $target
    CompressionLevel = "Fastest"
    DestinationPath = $stdZip
    Update = $true
}
Compress-Archive @stdCompress

Write-Output "Done."
