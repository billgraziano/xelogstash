$ErrorActionPreference = "Stop"
Write-Output "Running PSBuild.ps1..."
$target=".\deploy\xelogstash"
Write-Output "Target: $target"

$now = Get-Date -UFormat "%Y-%m-%d_%T"
# $sha1 = (git rev-parse HEAD).Trim()
$sha1 = (git describe --tags --dirty --always).Trim()
Write-Output "Git: $sha1"

Write-Output "Running go vet..."
go vet -all .\cmd\xelogstash
if ($LastExitCode -ne 0) {
    exit
}

go vet -all .\applog .\config .\log .\logstash .\seq .\status .\summary .\xe
if ($LastExitCode -ne 0) {
    exit
}

Write-Output "Running go test..."
go test .\cmd\xelogstash .\config .\seq .\xe
if ($LastExitCode -ne 0) {
    exit
}

Write-Output "Building..."
go build -o "$($target)\xelogstash.exe" -a -ldflags "-X main.sha1ver=$sha1 -X main.buildTime=$now" ".\cmd\xelogstash"
if ($LastExitCode -ne 0) {
    exit
}

Write-Output "Copying Files..."
Copy-Item -Path ".\samples\*.toml"          -Destination $target
Copy-Item -Path ".\samples\*.sql"           -Destination $target
Copy-Item -Path ".\samples\minimum.batch"   -Destination $target
Copy-Item -Path ".\README.md"               -Destination $target

Write-Output "Done."
