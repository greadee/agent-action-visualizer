[CmdletBinding()]
param(
    [Parameter(Mandatory)]
    [string]$Installer,
    [int]$LaunchWaitSeconds = 5
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$installerPath = (Resolve-Path -LiteralPath $Installer).Path
$installDirectory = Join-Path $env:LOCALAPPDATA "Programs\greadee\Agent Action Visualizer"
$uninstallKey = "HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall\greadeeAgent Action Visualizer"
$startMenuShortcut = Join-Path ([Environment]::GetFolderPath("Programs")) "Agent Action Visualizer.lnk"
$desktopShortcut = Join-Path ([Environment]::GetFolderPath("Desktop")) "Agent Action Visualizer.lnk"
$expectedFiles = @(
    "agent-action-visualizer.exe",
    "aav.exe",
    "aav-codex-hook.exe",
    "aav-wrapper.exe",
    "uninstall.exe"
)

function Invoke-CheckedProcess {
    param(
        [Parameter(Mandatory)]
        [string]$FilePath,
        [string[]]$ArgumentList = @()
    )

    $process = Start-Process -FilePath $FilePath -ArgumentList $ArgumentList -PassThru -Wait
    if ($process.ExitCode -ne 0) {
        throw "$FilePath exited with code $($process.ExitCode)."
    }
}

function Wait-UntilRemoved {
    param(
        [Parameter(Mandatory)]
        [string]$Path,
        [int]$TimeoutSeconds = 15
    )

    $deadline = [DateTime]::UtcNow.AddSeconds($TimeoutSeconds)
    while ((Test-Path -LiteralPath $Path) -and [DateTime]::UtcNow -lt $deadline) {
        Start-Sleep -Milliseconds 250
    }
    if (Test-Path -LiteralPath $Path) {
        throw "Timed out waiting for removal: $Path"
    }
}

function Assert-Installed {
    foreach ($name in $expectedFiles) {
        $path = Join-Path $installDirectory $name
        if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
            throw "Installed file is missing: $path"
        }
    }
    if (-not (Test-Path -LiteralPath $uninstallKey)) {
        throw "Per-user uninstall registration is missing: $uninstallKey"
    }
    $registration = Get-ItemProperty -LiteralPath $uninstallKey
    $application = Join-Path $installDirectory "agent-action-visualizer.exe"
    $productVersion = (Get-Item -LiteralPath $application).VersionInfo.ProductVersion
    if ($registration.DisplayName -ne "Agent Action Visualizer" -or $registration.DisplayVersion -ne $productVersion) {
        throw "Unexpected uninstall registration metadata."
    }
    if ($registration.InstallLocation -ne $installDirectory) {
        throw "Unexpected install location: $($registration.InstallLocation)"
    }
    foreach ($shortcut in @($startMenuShortcut, $desktopShortcut)) {
        if (-not (Test-Path -LiteralPath $shortcut -PathType Leaf)) {
            throw "Installed shortcut is missing: $shortcut"
        }
    }
}

function Assert-Uninstalled {
    Wait-UntilRemoved -Path $installDirectory
    if (Test-Path -LiteralPath $uninstallKey) {
        throw "Per-user uninstall registration still exists: $uninstallKey"
    }
    foreach ($shortcut in @($startMenuShortcut, $desktopShortcut)) {
        if (Test-Path -LiteralPath $shortcut) {
            throw "Uninstalled shortcut still exists: $shortcut"
        }
    }
}

function Test-InstalledLaunch {
    $application = Join-Path $installDirectory "agent-action-visualizer.exe"
    $process = Start-Process -FilePath $application -PassThru
    try {
        Start-Sleep -Seconds $LaunchWaitSeconds
        $process.Refresh()
        if ($process.HasExited) {
            throw "Installed application exited during launch validation with code $($process.ExitCode)."
        }
    } finally {
        $process.Refresh()
        if (-not $process.HasExited) {
            Stop-Process -Id $process.Id
            $process.WaitForExit()
        }
    }
}

if (Test-Path -LiteralPath $installDirectory) {
    throw "Refusing to overwrite an existing installation: $installDirectory"
}
if (Test-Path -LiteralPath $uninstallKey) {
    throw "Refusing to overwrite an existing uninstall registration: $uninstallKey"
}

for ($cycle = 1; $cycle -le 2; $cycle++) {
    Write-Output "cycle=$cycle action=install"
    Invoke-CheckedProcess -FilePath $installerPath -ArgumentList @("/S")
    Assert-Installed

    Write-Output "cycle=$cycle action=launch"
    Test-InstalledLaunch

    Write-Output "cycle=$cycle action=uninstall"
    $uninstaller = Join-Path $installDirectory "uninstall.exe"
    Invoke-CheckedProcess -FilePath $uninstaller -ArgumentList @("/S")
    Assert-Uninstalled
}

Write-Output "installer_lifecycle=passed"
