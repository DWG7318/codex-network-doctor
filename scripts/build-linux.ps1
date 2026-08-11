[CmdletBinding()]
param(
    [string]$Version,
    [string]$OutputDirectory
)

$ErrorActionPreference = 'Stop'
$repositoryRoot = Split-Path -Parent $PSScriptRoot
$go = Join-Path $repositoryRoot '.tools\go\bin\go.exe'

if (-not (Test-Path -LiteralPath $go)) {
    $go = (Get-Command go -ErrorAction Stop).Source
}
if ([string]::IsNullOrWhiteSpace($Version)) {
    $Version = (Get-Content -Raw -LiteralPath (Join-Path $repositoryRoot 'VERSION')).Trim()
}
if ([string]::IsNullOrWhiteSpace($OutputDirectory)) {
    $OutputDirectory = Join-Path $repositoryRoot 'dist\linux-amd64'
}

New-Item -ItemType Directory -Force -Path $OutputDirectory | Out-Null
$oldGOOS = $env:GOOS
$oldGOARCH = $env:GOARCH
$oldCGO = $env:CGO_ENABLED
Push-Location $repositoryRoot
try {
    $env:GOOS = 'linux'
    $env:GOARCH = 'amd64'
    $env:CGO_ENABLED = '0'
    $networkExecutable = Join-Path $OutputDirectory 'cnd-network'
    & $go build -trimpath -ldflags "-s -w -X main.version=$Version" -o $networkExecutable .\cmd\cnd-network
    if ($LASTEXITCODE -ne 0) {
        throw 'linux cnd-network build failed'
    }
}
finally {
    $env:GOOS = $oldGOOS
    $env:GOARCH = $oldGOARCH
    $env:CGO_ENABLED = $oldCGO
    Pop-Location
}

Write-Output (Join-Path $OutputDirectory 'cnd-network')
