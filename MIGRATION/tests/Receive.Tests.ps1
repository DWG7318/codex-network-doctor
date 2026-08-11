[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'
$receiveScript = Join-Path (Split-Path -Parent $PSScriptRoot) 'RECEIVE.ps1'

if (-not (Test-Path -LiteralPath $receiveScript -PathType Leaf)) {
    throw 'RECEIVE.ps1 is missing'
}

$testRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("lcnd-receive-test-{0}" -f ([guid]::NewGuid().ToString('N')))
$packageRoot = Join-Path $testRoot 'package'
$workspaceRoot = Join-Path $packageRoot 'workspace'
$destinationRoot = Join-Path $testRoot 'restored'

try {
    New-Item -ItemType Directory -Path $workspaceRoot -Force | Out-Null
    Set-Content -LiteralPath (Join-Path $workspaceRoot 'README.md') -Value 'LCnd test workspace' -Encoding UTF8

    $relativePath = 'workspace/README.md'
    $hash = (Get-FileHash -LiteralPath (Join-Path $workspaceRoot 'README.md') -Algorithm SHA256).Hash.ToLowerInvariant()
    Set-Content -LiteralPath (Join-Path $packageRoot 'manifest.sha256') -Value ("{0}  {1}" -f $hash, $relativePath) -Encoding UTF8

    $packageMigration = Join-Path $packageRoot 'MIGRATION'
    New-Item -ItemType Directory -Path $packageMigration -Force | Out-Null
    $packagedReceive = Join-Path $packageMigration 'RECEIVE.ps1'
    Copy-Item -LiteralPath $receiveScript -Destination $packagedReceive
    & $packagedReceive -VerifyOnly
    if ($LASTEXITCODE -ne 0) {
        throw "Packaged RECEIVE.ps1 default PackageRoot returned exit code $LASTEXITCODE"
    }

    & $receiveScript -PackageRoot $packageRoot -DestinationRoot $destinationRoot -SkipTests
    if ($LASTEXITCODE -ne 0) {
        throw "RECEIVE.ps1 returned exit code $LASTEXITCODE"
    }
    if ((Get-Content -Raw -LiteralPath (Join-Path $destinationRoot 'README.md')).Trim() -ne 'LCnd test workspace') {
        throw 'restored workspace content mismatch'
    }

    Set-Content -LiteralPath (Join-Path $destinationRoot 'owner-file.txt') -Value 'must survive' -Encoding UTF8
    $refusedOverwrite = $false
    try {
        & $receiveScript -PackageRoot $packageRoot -DestinationRoot $destinationRoot -SkipTests
    }
    catch {
        $refusedOverwrite = $_.Exception.Message -match '非空'
    }
    if (-not $refusedOverwrite) {
        throw 'RECEIVE.ps1 did not refuse a non-empty destination'
    }

    New-Item -ItemType Directory -Path (Join-Path $workspaceRoot 'LCSP') -Force | Out-Null
    $lcspRejected = $false
    try {
        & $receiveScript -PackageRoot $packageRoot -DestinationRoot (Join-Path $testRoot 'lcsp-destination') -SkipTests
    }
    catch {
        $lcspRejected = $_.Exception.Message -match 'LCSP'
    }
    if (-not $lcspRejected) {
        throw 'RECEIVE.ps1 did not reject an LCSP-contaminated package'
    }

    Write-Output 'PASS: RECEIVE.ps1 verifies, restores, refuses overwrite, and rejects LCSP.'
}
finally {
    if (Test-Path -LiteralPath $testRoot) {
        Remove-Item -LiteralPath $testRoot -Recurse -Force
    }
}
