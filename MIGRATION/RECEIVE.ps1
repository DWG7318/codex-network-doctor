[CmdletBinding()]
param(
    [string]$PackageRoot,
    [string]$DestinationRoot = 'C:\LCnd',
    [switch]$VerifyOnly,
    [switch]$SkipTests
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

if ([string]::IsNullOrWhiteSpace($PackageRoot)) {
    $PackageRoot = Split-Path -Parent $PSScriptRoot
}

function Resolve-FullPath {
    param([Parameter(Mandatory = $true)][string]$Path)
    return [System.IO.Path]::GetFullPath($Path)
}

function Assert-InsidePackage {
    param(
        [Parameter(Mandatory = $true)][string]$PackageFullPath,
        [Parameter(Mandatory = $true)][string]$CandidatePath
    )
    $prefix = $PackageFullPath.TrimEnd('\') + '\'
    $candidate = Resolve-FullPath -Path $CandidatePath
    if (-not $candidate.StartsWith($prefix, [System.StringComparison]::OrdinalIgnoreCase)) {
        throw "清单路径越出迁移包：$CandidatePath"
    }
    return $candidate
}

$package = Resolve-FullPath -Path $PackageRoot
$workspace = Join-Path $package 'workspace'
$manifest = Join-Path $package 'manifest.sha256'

if (-not (Test-Path -LiteralPath $workspace -PathType Container)) {
    throw "迁移包缺少 workspace：$workspace"
}
if (-not (Test-Path -LiteralPath $manifest -PathType Leaf)) {
    throw "迁移包缺少 manifest.sha256：$manifest"
}

$forbiddenTopLevel = @(
    (Join-Path $workspace 'LCSP'),
    (Join-Path $workspace 'LCSP-staged-apex-verify-20260803T123700Z')
)
foreach ($forbidden in $forbiddenTopLevel) {
    if (Test-Path -LiteralPath $forbidden) {
        throw "迁移包混入了明确排除的 LCSP 内容：$forbidden"
    }
}
$lcspArchives = @(Get-ChildItem -LiteralPath $workspace -File -Filter 'lcsp*.tar.gz' -ErrorAction SilentlyContinue)
if ($lcspArchives.Count -gt 0) {
    throw "迁移包混入了明确排除的 LCSP 归档：$($lcspArchives[0].FullName)"
}

$manifestLines = @(Get-Content -LiteralPath $manifest -Encoding UTF8 | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
if ($manifestLines.Count -eq 0) {
    throw 'manifest.sha256 为空'
}

$verified = 0
foreach ($line in $manifestLines) {
    if ($line -notmatch '^(?<hash>[0-9a-fA-F]{64})  (?<path>.+)$') {
        throw "无效的清单行：$line"
    }
    $relativePath = $Matches.path.Replace('/', '\')
    if ([System.IO.Path]::IsPathRooted($relativePath)) {
        throw "清单包含绝对路径：$relativePath"
    }
    $candidate = Assert-InsidePackage -PackageFullPath $package -CandidatePath (Join-Path $package $relativePath)
    if (-not (Test-Path -LiteralPath $candidate -PathType Leaf)) {
        throw "清单文件不存在：$relativePath"
    }
    $actual = (Get-FileHash -LiteralPath $candidate -Algorithm SHA256).Hash
    if (-not $actual.Equals($Matches.hash, [System.StringComparison]::OrdinalIgnoreCase)) {
        throw "SHA-256 不匹配：$relativePath"
    }
    $verified++
}

Write-Output ("迁移包校验通过：{0} 个文件。" -f $verified)
if ($VerifyOnly) {
    $global:LASTEXITCODE = 0
    return
}

$destination = Resolve-FullPath -Path $DestinationRoot
if (Test-Path -LiteralPath $destination) {
    $existing = @(Get-ChildItem -LiteralPath $destination -Force -ErrorAction Stop)
    if ($existing.Count -gt 0) {
        throw "目标目录非空，拒绝覆盖：$destination"
    }
}
else {
    New-Item -ItemType Directory -Path $destination -Force | Out-Null
}

& robocopy.exe $workspace $destination /E /COPY:DAT /DCOPY:DAT /R:2 /W:1 /XJ /NFL /NDL /NP /NJH /NJS
$copyExitCode = $LASTEXITCODE
if ($copyExitCode -gt 7) {
    throw "复制 workspace 失败，robocopy exit code：$copyExitCode"
}

if (-not $SkipTests) {
    $go = Join-Path $destination '.tools\go\bin\go.exe'
    if (-not (Test-Path -LiteralPath $go -PathType Leaf)) {
        throw "包内 Go 工具链缺失：$go"
    }
    Push-Location $destination
    try {
        & $go test ./...
        if ($LASTEXITCODE -ne 0) {
            throw 'go test ./... 失败'
        }
        & $go vet ./...
        if ($LASTEXITCODE -ne 0) {
            throw 'go vet ./... 失败'
        }
        & powershell.exe -NoProfile -ExecutionPolicy Bypass -File (Join-Path $destination 'scripts\build.ps1') -SkipRace
        if ($LASTEXITCODE -ne 0) {
            throw 'scripts\build.ps1 -SkipRace 失败'
        }
    }
    finally {
        Pop-Location
    }
}

Write-Output "LCnd 已接收到：$destination"
Write-Output "请在 Codex 中把该目录添加为本地项目，然后打开 MIGRATION\00-RECEIVER-START-HERE.md。"
$global:LASTEXITCODE = 0
