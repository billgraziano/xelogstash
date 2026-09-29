Param (
    [string]$version = "dev",
    [switch]$Sign = $false
)
$ErrorActionPreference = "Stop"

function Test-AzureLogin {
    try {
        az account show --output none 2>$null
        return $true
    }
    catch {
        return $false
    }
}


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

if ($Sign -and -not (Test-AzureLogin)) {
    try {
        az login `
            --use-device-code `
            --scope "https://codesigning.azure.net/.default" | Out-Null
        if ($LASTEXITCODE -ne 0) {
            throw "Azure login failed."
        }
    }
    catch {
        Write-Error $_
        exit 1
    }
    Write-Host "Azure login successful"
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

go vet -all .\pkg\...
if ($LastExitCode -ne 0) {
    exit
}

Write-Output "Running go test..."
go test -count=1 .\cmd\xelogstash ./cmd/sqlxewriter .\pkg\...
if ($LastExitCode -ne 0) {
    exit
}

Write-Output "Building sqlxewriter.exe..."
go build -o "$($target)\sqlxewriter.exe" -a -ldflags "-X main.sha1ver=$sha1 -X main.buildTime=$now -X main.version=$version" ".\cmd\sqlxewriter"
if ($LastExitCode -ne 0) {
    exit
}


if ($Sign) {
    # Signing code
    Write-Host "Signing sqlxewriter.exe..."
    sign.exe code artifact-signing `
        -b "C:\dev\github.com\xelogstash\deploy\windows\sqlxewriter" `
        --artifact-signing-endpoint "https://cus.codesigning.azure.net/" `
        --artifact-signing-certificate-profile "sign-cert" `
        --artifact-signing-account "acct-codesign" `
        sqlxewriter.exe `
        -v Warning `
        --azure-credential-type azure-cli

    $signature = Get-AuthenticodeSignature "$($target)\sqlxewriter.exe"

    if ($signature.Status -ne 'Valid') {
        Write-Error "Signature validation failed: $($signature.Status)"
        exit 1
    }

    Write-Host "Signed by: $($signature.SignerCertificate.Subject)"
    Write-Host "Issuer:    $($signature.SignerCertificate.Issuer)"
    Write-Host "Expires :  $($signature.SignerCertificate.NotAfter)"
}


.\Deploy\Windows\SQLXEWriter\SQLXEWriter.exe -version 

Write-Output "Copying Files..."
blackfriday-tool -css .\docs\style.css   -embed README.md "README.html"
blackfriday-tool -css .\docs\style.css   -embed LICENSE.md "LICENSE.html"
Copy-Item -Path README.html -Destination $target
Copy-Item -Path LICENSE.html -Destination $target
Copy-Item -Path ".\samples\sqlxewriter.toml" -Destination $target
Copy-Item -Path ".\samples" -Destination $target -Recurse

$stdZip = "$($deploy)\sqlxewriter_$($version)_windows_x64.zip"
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
