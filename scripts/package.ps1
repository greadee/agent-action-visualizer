[CmdletBinding()]
param(
    [string]$Version,
    [string]$OutputDirectory,
    [string]$Wails = "wails",
    [switch]$SkipInstaller,
    [switch]$AllowDirty,
    [switch]$DryRun
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repositoryRoot = Split-Path -Parent $PSScriptRoot
$desktopRoot = Join-Path $repositoryRoot "apps\desktop"
$versionFile = Join-Path $repositoryRoot "VERSION"

if ([string]::IsNullOrWhiteSpace($Version)) {
    $Version = (Get-Content -LiteralPath $versionFile -Raw).Trim()
}
if ($Version -notmatch '^\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?$') {
    throw "Version must be SemVer without a leading v."
}
if ([string]::IsNullOrWhiteSpace($OutputDirectory)) {
    $OutputDirectory = Join-Path $repositoryRoot "artifacts"
}
$wailsConfig = Get-Content -LiteralPath (Join-Path $desktopRoot "wails.json") -Raw | ConvertFrom-Json
if ($wailsConfig.info.productVersion -ne $Version) {
    throw "VERSION and wails.json info.productVersion must match."
}
if (-not $AllowDirty) {
    $changes = @(git -C $repositoryRoot status --porcelain)
    if ($changes.Count -gt 0) {
        throw "Release packaging requires a clean checkout. Use -AllowDirty only for non-release local validation."
    }
}

$commit = if ($env:GITHUB_SHA) { $env:GITHUB_SHA.Substring(0, [Math]::Min(12, $env:GITHUB_SHA.Length)) } else { (git -C $repositoryRoot rev-parse --short=12 HEAD).Trim() }
$sourceDateEpoch = if ($env:SOURCE_DATE_EPOCH) { $env:SOURCE_DATE_EPOCH } else { (git -C $repositoryRoot show -s --format=%ct HEAD).Trim() }
if ($sourceDateEpoch -notmatch '^\d+$') {
    throw "SOURCE_DATE_EPOCH must be a Unix timestamp."
}
$env:SOURCE_DATE_EPOCH = $sourceDateEpoch

$assetStem = "agent-action-visualizer-v$Version-windows-amd64"
$binaryName = "$assetStem.exe"
$ldflags = "-X github.com/greadee/agent-action-visualizer/internal/buildinfo.Version=$Version -X github.com/greadee/agent-action-visualizer/internal/buildinfo.Commit=$commit -X github.com/greadee/agent-action-visualizer/internal/buildinfo.BuildDate=$sourceDateEpoch"
$companionBuilds = @(
    @{ Name = "aav.exe"; Package = "./cmd/aav" },
    @{ Name = "aav-codex-hook.exe"; Package = "./cmd/aav-codex-hook" },
    @{ Name = "aav-wrapper.exe"; Package = "./cmd/aav-wrapper" }
)
$installerToolDirectory = Join-Path $desktopRoot "build\windows\installer\package-tools"
$arguments = @("build", "-clean", "-platform", "windows/amd64", "-trimpath", "-ldflags", $ldflags, "-o", $binaryName)
if (-not $SkipInstaller) {
    $arguments += "-nsis"
}

if ($DryRun) {
    Write-Output "version=$Version"
    Write-Output "output=$OutputDirectory"
    Write-Output "companions=$($companionBuilds.Name -join ',')"
    Write-Output "command=$Wails $($arguments -join ' ')"
    exit 0
}

New-Item -ItemType Directory -Path $OutputDirectory -Force | Out-Null
New-Item -ItemType Directory -Path $installerToolDirectory -Force | Out-Null
Push-Location $repositoryRoot
try {
    foreach ($companion in $companionBuilds) {
        $companionPath = Join-Path $installerToolDirectory $companion.Name
        & go build -trimpath -ldflags $ldflags -o $companionPath $companion.Package
        if ($LASTEXITCODE -ne 0) {
            throw "Companion build failed for $($companion.Name) with exit code $LASTEXITCODE."
        }
    }
} finally {
    Pop-Location
}

Push-Location $desktopRoot
try {
    & $Wails @arguments
    if ($LASTEXITCODE -ne 0) {
        throw "Wails packaging failed with exit code $LASTEXITCODE."
    }
} finally {
    Pop-Location
}

$binaryPath = Join-Path $desktopRoot "build\bin\$binaryName"
if (-not (Test-Path -LiteralPath $binaryPath)) {
    throw "Expected portable executable was not produced: $binaryName"
}
Copy-Item -LiteralPath $binaryPath -Destination (Join-Path $OutputDirectory $binaryName) -Force
foreach ($companion in $companionBuilds) {
    Copy-Item -LiteralPath (Join-Path $installerToolDirectory $companion.Name) -Destination (Join-Path $OutputDirectory $companion.Name) -Force
}

if (-not $SkipInstaller) {
    $installers = @(Get-ChildItem -LiteralPath (Join-Path $desktopRoot "build\bin") -Filter "*-installer.exe" -File)
    if ($installers.Count -ne 1) {
        throw "Expected exactly one NSIS installer, found $($installers.Count)."
    }
    Copy-Item -LiteralPath $installers[0].FullName -Destination (Join-Path $OutputDirectory "$assetStem-installer.exe") -Force
}

$checksumPath = Join-Path $OutputDirectory "SHA256SUMS.txt"
$checksums = Get-ChildItem -LiteralPath $OutputDirectory -File |
    Where-Object { $_.Name -ne "SHA256SUMS.txt" } |
    Sort-Object Name |
    ForEach-Object { "$(($_ | Get-FileHash -Algorithm SHA256).Hash.ToLowerInvariant())  $($_.Name)" }
Set-Content -LiteralPath $checksumPath -Value $checksums -Encoding ascii
Write-Output "packaged=$OutputDirectory"
