Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Write-Step {
    param([string]$Message)
    Write-Host ""
    Write-Host "==> $Message" -ForegroundColor Cyan
}

function Ensure-ConfigFile {
    param(
        [string]$ConfigPath,
        [string]$TemplatePath
    )

    if (-not (Test-Path -LiteralPath $ConfigPath)) {
        if (-not (Test-Path -LiteralPath $TemplatePath)) {
            throw "Missing template file: $TemplatePath"
        }
        Copy-Item -LiteralPath $TemplatePath -Destination $ConfigPath
        Write-Host "Created config: $ConfigPath"
    }
}

function Set-NeteaseMusicU {
    param(
        [string]$ConfigPath,
        [string]$MusicU
    )

    $content = Get-Content -LiteralPath $ConfigPath -Raw
    $sectionPattern = '(?ms)(^\[plugins\.netease\]\s*$)(.*?)(?=^\[|\z)'
    $match = [regex]::Match($content, $sectionPattern)
    if (-not $match.Success) {
        $content = $content.TrimEnd() + "`r`n`r`n[plugins.netease]`r`nmusic_u = $MusicU`r`n"
    }
    else {
        $sectionText = $match.Groups[2].Value
        if ($sectionText -match '(?m)^\s*music_u\s*=') {
            $updatedSection = [regex]::Replace($sectionText, '(?m)^\s*music_u\s*=.*$', "music_u = $MusicU")
        }
        else {
            $updatedSection = $sectionText.TrimEnd() + "`r`nmusic_u = $MusicU`r`n"
        }
        $content = $content.Substring(0, $match.Groups[2].Index) + $updatedSection + $content.Substring($match.Groups[2].Index + $match.Groups[2].Length)
    }

    Set-Content -LiteralPath $ConfigPath -Value $content -Encoding UTF8
}

function Ask-NonEmpty {
    param([string]$Prompt)

    while ($true) {
        $value = Read-Host $Prompt
        if (-not [string]::IsNullOrWhiteSpace($value)) {
            return $value.Trim()
        }
        Write-Host "Input cannot be empty." -ForegroundColor Yellow
    }
}

$repoRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$configPath = Join-Path $repoRoot "config.ini"
$templatePath = Join-Path $repoRoot "config_example.ini"
$exePath = Join-Path $repoRoot "netease-batch.exe"
$builderPath = Join-Path $repoRoot "build_netease_batch_windows.bat"
$downloadScript = Join-Path $repoRoot "download_netease_playlist.bat"

Write-Step "Preparing files"
Ensure-ConfigFile -ConfigPath $configPath -TemplatePath $templatePath

if (-not (Test-Path -LiteralPath $exePath)) {
    Write-Step "Building netease-batch.exe"
    & $builderPath
    if ($LASTEXITCODE -ne 0) {
        throw "Build failed."
    }
}

Write-Step "Open NetEase Cloud Music in your browser"
Write-Host "A browser page will open. Log in to https://music.163.com first."
Write-Host "Then open DevTools (F12) -> Application/Storage -> Cookies -> https://music.163.com"
Write-Host "Copy the value of MUSIC_U and paste it back here."
Start-Process "https://music.163.com"

$musicU = Ask-NonEmpty -Prompt "Paste MUSIC_U"
Write-Step "Writing config.ini"
Set-NeteaseMusicU -ConfigPath $configPath -MusicU $musicU
Write-Host "Saved MUSIC_U to $configPath"

Write-Step "Checking cookie"
& $exePath -config $configPath -check
if ($LASTEXITCODE -ne 0) {
    Write-Host ""
    Write-Host "Cookie check failed. Edit config.ini and rerun this wizard." -ForegroundColor Red
    exit $LASTEXITCODE
}

$startDownload = Read-Host "Cookie looks good. Start first playlist download now? (Y/n)"
if ($startDownload -match '^(|y|yes)$') {
    $playlistURL = Ask-NonEmpty -Prompt "Playlist or album URL"
    $outputDir = Read-Host "Output directory (default: .\downloads)"
    if ([string]::IsNullOrWhiteSpace($outputDir)) {
        $outputDir = Join-Path $repoRoot "downloads"
    }

    Write-Step "Starting download"
    & $downloadScript $playlistURL $outputDir
    exit $LASTEXITCODE
}

Write-Host ""
Write-Host "Setup complete. Next time run:" -ForegroundColor Green
Write-Host "  download_netease_playlist.bat ""https://music.163.com/#/playlist?id=19723756"" ""D:\Music"""
