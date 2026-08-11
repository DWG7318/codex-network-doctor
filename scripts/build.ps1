[CmdletBinding()]
param(
    [string]$Version,
    [string]$OutputDirectory,
    [switch]$SkipRace
)

$ErrorActionPreference = 'Stop'
$repositoryRoot = Split-Path -Parent $PSScriptRoot
$localGo = Join-Path $repositoryRoot '.tools\go\bin\go.exe'

if ([string]::IsNullOrWhiteSpace($OutputDirectory)) {
    $OutputDirectory = Join-Path $repositoryRoot 'bin'
}

if ([string]::IsNullOrWhiteSpace($Version)) {
    $Version = (Get-Content -Raw -LiteralPath (Join-Path $repositoryRoot 'VERSION')).Trim()
}

if (Test-Path -LiteralPath $localGo) {
    $go = $localGo
}
else {
    $goCommand = Get-Command go -ErrorAction Stop
    $go = $goCommand.Source
}

New-Item -ItemType Directory -Force -Path $OutputDirectory | Out-Null
Push-Location $repositoryRoot
try {
    & $go test ./...
    if ($LASTEXITCODE -ne 0) {
        throw 'go test failed'
    }
    if (-not $SkipRace) {
        $originalPath = $env:Path
        $originalCGO = $env:CGO_ENABLED
        $originalCC = $env:CC
        try {
            $gcc = Get-Command gcc -ErrorAction SilentlyContinue
            if ($null -eq $gcc) {
                $mingwDirectory = 'C:\msys64\mingw64\bin'
                $mingwGcc = Join-Path $mingwDirectory 'gcc.exe'
                if (-not (Test-Path -LiteralPath $mingwGcc)) {
                    throw 'go test -race requires MinGW GCC. Install it or use -SkipRace only for a non-release local build.'
                }
                $env:Path = "$mingwDirectory;$env:Path"
            }
            $env:CGO_ENABLED = '1'
            $env:CC = 'gcc'
            & $go test -race ./...
            if ($LASTEXITCODE -ne 0) {
                throw 'go test -race failed'
            }
        }
        finally {
            $env:Path = $originalPath
            $env:CGO_ENABLED = $originalCGO
            $env:CC = $originalCC
        }
    }
    & $go vet ./...
    if ($LASTEXITCODE -ne 0) {
        throw 'go vet failed'
    }
    & $go build -trimpath -ldflags "-s -w -X main.version=$Version" -o (Join-Path $OutputDirectory 'cnd.exe') .\cmd\cnd
    if ($LASTEXITCODE -ne 0) {
        throw 'go build failed'
    }
    $repairExecutable = Join-Path $OutputDirectory 'cnd网络修复.exe'
    $rollbackExecutable = Join-Path $OutputDirectory 'cnd网络修复-撤销.exe'
    $speedExecutable = Join-Path $OutputDirectory 'cnd网速检测.exe'
    & $go build -trimpath -ldflags "-s -w -X main.version=$Version" -o $repairExecutable .\cmd\cnd-repair
    if ($LASTEXITCODE -ne 0) {
        throw 'go build cnd-repair failed'
    }
    Copy-Item -LiteralPath $repairExecutable -Destination $rollbackExecutable -Force
    Copy-Item -LiteralPath $repairExecutable -Destination $speedExecutable -Force
    $networkExecutable = Join-Path $OutputDirectory 'cnd-network.exe'
    & $go build -trimpath -ldflags "-s -w -X main.version=$Version" -o $networkExecutable .\cmd\cnd-network
    if ($LASTEXITCODE -ne 0) {
        throw 'go build cnd-network failed'
    }
}
finally {
    Pop-Location
}

Write-Output (Join-Path $OutputDirectory 'cnd.exe')
Write-Output (Join-Path $OutputDirectory 'cnd网络修复.exe')
Write-Output (Join-Path $OutputDirectory 'cnd网络修复-撤销.exe')
Write-Output (Join-Path $OutputDirectory 'cnd网速检测.exe')
Write-Output (Join-Path $OutputDirectory 'cnd-network.exe')
