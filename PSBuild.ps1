Param (
    [string]$version = "dev"
)
$ErrorActionPreference = "Stop"

Write-Output "Running PSBuild.ps1..."
Write-Output "" 
$deploy=".\deploy"
$target="$($deploy)\windows\sqlxewriter"
Write-Output "Deploy:  $deploy"
Write-Output "Target:  $target"
If ($Version -eq "") {
    Write-Output "Missing Version"
    Exit
}
Write-Output "Version: $($version)"

# Clean deploy directory
If (Test-Path $target) {
    Remove-Item $target -Recurse
}

# $now = Get-Date -UFormat "%Y-%m-%d_%T_%Z"
$now = Get-Date -Format "yyyy'-'MM'-'dd'T'HH':'mm':'sszzz"
$sha1 = (git describe --tags --dirty --always).Trim()
Write-Output "Git:     $sha1"
Write-Output "Build:   $now"

Write-Output "" 
Write-Output "Running go vet..."
go vet -all .\cmd\sqlxewriter
if ($LastExitCode -ne 0) {
    exit
}

go vet -all .\config .\log .\logstash .\seq .\status .\summary .\xe .\sink .\pkg\...
if ($LastExitCode -ne 0) {
    exit
}

Write-Output "Running go test..."
go test .\cmd\xelogstash ./cmd/sqlxewriter .\config .\seq .\xe .\sink .\status .\pkg\...
if ($LastExitCode -ne 0) {
    exit
}

Write-Output "Building sqlxewriter.exe..."
go build -o "$($target)\sqlxewriter.exe" -a -ldflags "-X main.sha1ver=$sha1 -X main.buildTime=$now -X main.version=$version" ".\cmd\sqlxewriter"
if ($LastExitCode -ne 0) {
    exit
}

Write-Output "Copying Files..."
blackfriday-tool -css .\docs\style.css   -embed README.md "README.html"
Copy-Item -Path README.html -Destination $target
Copy-Item -Path LICENSE.txt -Destination $target
Copy-Item -Path ".\samples\sqlxewriter.toml" -Destination $target
Copy-Item -Path ".\samples" -Destination $target -Recurse

$stdZip = "$($deploy)\sqlxewriter_$($sha1)_windows_x64.zip"
If (Test-Path $stdZip) {
    Remove-Item $stdZip
}
Write-Host "Writing $($stdZip)..."
$stdCompress = @{
    Path = $target
    CompressionLevel = "Fastest"
    DestinationPath = $stdZip
    Update = $true
}
Compress-Archive @stdCompress

Write-Output "Done."
