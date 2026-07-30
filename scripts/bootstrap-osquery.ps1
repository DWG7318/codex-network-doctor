[CmdletBinding()]
param(
    [string]$DestinationDirectory = (Join-Path (Split-Path -Parent $PSScriptRoot) '.tools'),
    [switch]$Force
)

$ErrorActionPreference = 'Stop'
$version = '5.23.1'
$archiveName = "osquery-$version.windows_x86_64.zip"
$downloadUri = "https://github.com/osquery/osquery/releases/download/$version/$archiveName"
$expectedSHA256 = '7bd411050ef6b5aae1b23956aec0dc5ce6e800c5656f0cd463ac70a6e1bdf30b'
$expectedBinarySHA256 = 'a5be97a7cf3fcf58276f619f4519c2d3fbdf5ade58c1cb51e37cf4215c5beee5'
$archiveRootName = "osquery-$version.windows_x86_64"
$expandedRoot = Join-Path $DestinationDirectory $archiveRootName
$binaryRelativePath = 'Program Files\osquery\osqueryi.exe'
$binaryPath = Join-Path $expandedRoot $binaryRelativePath

if (Test-Path -LiteralPath $binaryPath) {
    $existingBinarySHA256 = (Get-FileHash -Algorithm SHA256 -LiteralPath $binaryPath).Hash.ToLowerInvariant()
    if ($existingBinarySHA256 -eq $expectedBinarySHA256 -and -not $Force) {
        Write-Output $binaryPath
        return
    }
    if (-not $Force) {
        throw "Existing osqueryi.exe SHA-256 mismatch. Expected $expectedBinarySHA256, got $existingBinarySHA256. Re-run with -Force to restore it."
    }
}
if ((Test-Path -LiteralPath $expandedRoot) -and -not $Force) {
    throw "Expanded directory exists but a verified osqueryi.exe is unavailable: $expandedRoot. Re-run with -Force to repair it."
}

New-Item -ItemType Directory -Force -Path $DestinationDirectory | Out-Null
$archivePath = Join-Path ([System.IO.Path]::GetTempPath()) ("cnd-$archiveName-" + [System.Guid]::NewGuid().ToString('N'))

try {
    Invoke-WebRequest -Uri $downloadUri -OutFile $archivePath
    $actualSHA256 = (Get-FileHash -Algorithm SHA256 -LiteralPath $archivePath).Hash.ToLowerInvariant()
    if ($actualSHA256 -ne $expectedSHA256) {
        throw "osquery archive SHA-256 mismatch. Expected $expectedSHA256, got $actualSHA256."
    }
    Expand-Archive -LiteralPath $archivePath -DestinationPath $DestinationDirectory -Force
    if (-not (Test-Path -LiteralPath $binaryPath)) {
        throw "Archive passed SHA-256 validation but osqueryi.exe was not found at $binaryPath."
    }
    $actualBinarySHA256 = (Get-FileHash -Algorithm SHA256 -LiteralPath $binaryPath).Hash.ToLowerInvariant()
    if ($actualBinarySHA256 -ne $expectedBinarySHA256) {
        throw "Extracted osqueryi.exe SHA-256 mismatch. Expected $expectedBinarySHA256, got $actualBinarySHA256."
    }
    Write-Output $binaryPath
}
finally {
    if (Test-Path -LiteralPath $archivePath) {
        Remove-Item -LiteralPath $archivePath -Force
    }
}
