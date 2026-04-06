Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
[System.Windows.Forms.Application]::EnableVisualStyles()

function Ensure-ConfigFile {
    param(
        [string]$ConfigPath,
        [string]$TemplatePath
    )

    if (-not (Test-Path -LiteralPath $ConfigPath)) {
        if (-not (Test-Path -LiteralPath $TemplatePath)) {
            throw "缺少模板文件：$TemplatePath"
        }
        Copy-Item -LiteralPath $TemplatePath -Destination $ConfigPath
    }
}

function Get-NeteaseMusicU {
    param([string]$ConfigPath)

    if (-not (Test-Path -LiteralPath $ConfigPath)) {
        return ""
    }

    $content = Get-Content -LiteralPath $ConfigPath -Raw
    $sectionMatch = [regex]::Match($content, '(?ms)(^\[plugins\.netease\]\s*$)(.*?)(?=^\[|\z)')
    if (-not $sectionMatch.Success) {
        return ""
    }
    $match = [regex]::Match($sectionMatch.Groups[2].Value, '(?m)^\s*music_u\s*=\s*(.+?)\s*$')
    if ($match.Success) {
        return $match.Groups[1].Value.Trim()
    }
    $cookieMatch = [regex]::Match($sectionMatch.Groups[2].Value, '(?m)^\s*cookie\s*=\s*(.+?)\s*$')
    if ($cookieMatch.Success) {
        return Extract-MusicU -Text $cookieMatch.Groups[1].Value
    }
    return ""
}

function Set-NeteaseMusicU {
    param(
        [string]$ConfigPath,
        [string]$MusicU
    )

    $MusicU = $MusicU.Trim()
    $content = Get-Content -LiteralPath $ConfigPath -Raw
    $sectionPattern = '(?ms)(^\[plugins\.netease\]\s*$)(.*?)(?=^\[|\z)'
    $match = [regex]::Match($content, $sectionPattern)
    if (-not $match.Success) {
        if (-not [string]::IsNullOrWhiteSpace($MusicU)) {
            $content = $content.TrimEnd() + "`r`n`r`n[plugins.netease]`r`nmusic_u = $MusicU`r`n"
        }
    }
    else {
        $sectionText = $match.Groups[2].Value
        if ($sectionText -match '(?m)^\s*music_u\s*=') {
            $updatedSection = [regex]::Replace($sectionText, '(?m)^\s*music_u\s*=.*$', "music_u = $MusicU")
        }
        elseif (-not [string]::IsNullOrWhiteSpace($MusicU)) {
            $updatedSection = $sectionText.TrimEnd() + "`r`nmusic_u = $MusicU`r`n"
        }
        else {
            $updatedSection = $sectionText
        }
        if ($updatedSection -match '(?m)^\s*cookie\s*=') {
            $updatedSection = [regex]::Replace($updatedSection, '(?m)^\s*cookie\s*=.*$', "cookie =")
        }
        $content = $content.Substring(0, $match.Groups[2].Index) + $updatedSection + $content.Substring($match.Groups[2].Index + $match.Groups[2].Length)
    }

    Set-Content -LiteralPath $ConfigPath -Value $content -Encoding UTF8
}

function Extract-MusicU {
    param([string]$Text)

    if ([string]::IsNullOrWhiteSpace($Text)) {
        return ""
    }

    $trimmed = $Text.Trim()
    $trimmed = $trimmed -replace '^[`"''\s]+', ''
    $trimmed = $trimmed -replace '[`"''\s]+$', ''

    if ($trimmed -match '(^|[;\s])MUSIC_U=([^;\r\n]+)') {
        return $Matches[2].Trim()
    }

    if ($trimmed -match '^\s*MUSIC_U\s*=\s*(.+?)\s*$') {
        return $Matches[1].Trim()
    }

    if ($trimmed -match '^\s*Cookie\s*:\s*(.+)$') {
        $cookieText = $Matches[1].Trim()
        if ($cookieText -match '(^|[;\s])MUSIC_U=([^;\r\n]+)') {
            return $Matches[2].Trim()
        }
    }

    foreach ($line in ($trimmed -split "`r?`n")) {
        $cookieLine = $line.Trim()
        if ([string]::IsNullOrWhiteSpace($cookieLine)) {
            continue
        }
        if ($cookieLine.StartsWith("#")) {
            continue
        }
        $parts = $cookieLine -split "`t"
        if ($parts.Length -ge 7 -and $parts[5] -eq "MUSIC_U") {
            return $parts[6].Trim()
        }
    }

    if ($trimmed -notmatch '[=;\s]' -and $trimmed.Length -ge 20) {
        return $trimmed
    }

    return ""
}

function Quote-Argument {
    param([string]$Value)

    if ([string]::IsNullOrEmpty($Value)) {
        return '""'
    }
    if ($Value -notmatch '[\s"]') {
        return $Value
    }
    return '"' + ($Value -replace '"', '\"') + '"'
}

function Join-Arguments {
    param([string[]]$Items)
    return ($Items | ForEach-Object { Quote-Argument $_ }) -join ' '
}

$repoRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$configPath = Join-Path $repoRoot "config.ini"
$templatePath = Join-Path $repoRoot "config_example.ini"
$exePath = Join-Path $repoRoot "netease-batch.exe"
$builderPath = Join-Path $repoRoot "build_netease_batch_windows.bat"
$downloadDir = Join-Path $repoRoot "downloads"

Ensure-ConfigFile -ConfigPath $configPath -TemplatePath $templatePath
$initialMusicU = Get-NeteaseMusicU -ConfigPath $configPath

$form = New-Object System.Windows.Forms.Form
$form.Text = "网易云歌单批量下载器"
$form.StartPosition = "CenterScreen"
$form.ClientSize = New-Object System.Drawing.Size(860, 700)
$form.MinimumSize = New-Object System.Drawing.Size(860, 700)

$font = New-Object System.Drawing.Font("Microsoft YaHei UI", 9)
$form.Font = $font

$lblTitle = New-Object System.Windows.Forms.Label
$lblTitle.Location = New-Object System.Drawing.Point(20, 15)
$lblTitle.Size = New-Object System.Drawing.Size(780, 28)
$lblTitle.Font = New-Object System.Drawing.Font("Microsoft YaHei UI", 14, [System.Drawing.FontStyle]::Bold)
$lblTitle.Text = "双击打开，填好信息后即可开始下载"
$form.Controls.Add($lblTitle)

$lblHint = New-Object System.Windows.Forms.Label
$lblHint.Location = New-Object System.Drawing.Point(20, 48)
$lblHint.Size = New-Object System.Drawing.Size(670, 38)
$lblHint.Text = "可以手动粘贴 MUSIC_U，也可以留空，让程序在 Windows 下自动从 Edge/Chrome/Brave/Firefox 读取 Cookie。"
$form.Controls.Add($lblHint)

$btnCookieGuide = New-Object System.Windows.Forms.Button
$btnCookieGuide.Location = New-Object System.Drawing.Point(710, 48)
$btnCookieGuide.Size = New-Object System.Drawing.Size(130, 30)
$btnCookieGuide.Text = "如何获取 Cookie"
$form.Controls.Add($btnCookieGuide)

$lblMusicU = New-Object System.Windows.Forms.Label
$lblMusicU.Location = New-Object System.Drawing.Point(20, 96)
$lblMusicU.Size = New-Object System.Drawing.Size(120, 24)
$lblMusicU.Text = "MUSIC_U"
$form.Controls.Add($lblMusicU)

$txtMusicU = New-Object System.Windows.Forms.TextBox
$txtMusicU.Location = New-Object System.Drawing.Point(140, 94)
$txtMusicU.Size = New-Object System.Drawing.Size(560, 26)
$txtMusicU.UseSystemPasswordChar = $true
$txtMusicU.Text = $initialMusicU
$form.Controls.Add($txtMusicU)

$btnOpenSite = New-Object System.Windows.Forms.Button
$btnOpenSite.Location = New-Object System.Drawing.Point(710, 92)
$btnOpenSite.Size = New-Object System.Drawing.Size(130, 30)
$btnOpenSite.Text = "打开网易云"
$form.Controls.Add($btnOpenSite)

$btnImportClipboard = New-Object System.Windows.Forms.Button
$btnImportClipboard.Location = New-Object System.Drawing.Point(710, 126)
$btnImportClipboard.Size = New-Object System.Drawing.Size(130, 30)
$btnImportClipboard.Text = "导入剪贴板"
$form.Controls.Add($btnImportClipboard)

$chkShowCookie = New-Object System.Windows.Forms.CheckBox
$chkShowCookie.Location = New-Object System.Drawing.Point(140, 124)
$chkShowCookie.Size = New-Object System.Drawing.Size(100, 24)
$chkShowCookie.Text = "显示"
$form.Controls.Add($chkShowCookie)

$lblCookieHint = New-Object System.Windows.Forms.Label
$lblCookieHint.Location = New-Object System.Drawing.Point(250, 126)
$lblCookieHint.Size = New-Object System.Drawing.Size(590, 40)
$lblCookieHint.Text = "留空会在 Windows 下自动读取浏览器 Cookie。手动方式：Chrome/Edge 按 F12 -> Application -> Cookies -> https://music.163.com"
$form.Controls.Add($lblCookieHint)

$lblURL = New-Object System.Windows.Forms.Label
$lblURL.Location = New-Object System.Drawing.Point(20, 170)
$lblURL.Size = New-Object System.Drawing.Size(120, 24)
$lblURL.Text = "歌单链接"
$form.Controls.Add($lblURL)

$txtURL = New-Object System.Windows.Forms.TextBox
$txtURL.Location = New-Object System.Drawing.Point(140, 168)
$txtURL.Size = New-Object System.Drawing.Size(700, 26)
$txtURL.Text = "https://music.163.com/#/playlist?id="
$form.Controls.Add($txtURL)

$lblOutput = New-Object System.Windows.Forms.Label
$lblOutput.Location = New-Object System.Drawing.Point(20, 210)
$lblOutput.Size = New-Object System.Drawing.Size(120, 24)
$lblOutput.Text = "输出目录"
$form.Controls.Add($lblOutput)

$txtOutput = New-Object System.Windows.Forms.TextBox
$txtOutput.Location = New-Object System.Drawing.Point(140, 208)
$txtOutput.Size = New-Object System.Drawing.Size(560, 26)
$txtOutput.Text = $downloadDir
$form.Controls.Add($txtOutput)

$btnBrowse = New-Object System.Windows.Forms.Button
$btnBrowse.Location = New-Object System.Drawing.Point(710, 206)
$btnBrowse.Size = New-Object System.Drawing.Size(130, 30)
$btnBrowse.Text = "浏览"
$form.Controls.Add($btnBrowse)

$btnImportCookieFile = New-Object System.Windows.Forms.Button
$btnImportCookieFile.Location = New-Object System.Drawing.Point(570, 246)
$btnImportCookieFile.Size = New-Object System.Drawing.Size(140, 30)
$btnImportCookieFile.Text = "导入 Cookie 文件"
$form.Controls.Add($btnImportCookieFile)

$lblQuality = New-Object System.Windows.Forms.Label
$lblQuality.Location = New-Object System.Drawing.Point(20, 250)
$lblQuality.Size = New-Object System.Drawing.Size(120, 24)
$lblQuality.Text = "音质"
$form.Controls.Add($lblQuality)

$cmbQuality = New-Object System.Windows.Forms.ComboBox
$cmbQuality.Location = New-Object System.Drawing.Point(140, 248)
$cmbQuality.Size = New-Object System.Drawing.Size(160, 26)
$cmbQuality.DropDownStyle = "DropDownList"
[void]$cmbQuality.Items.AddRange(@("标准", "较高", "无损", "Hi-Res"))
$cmbQuality.SelectedItem = "无损"
$form.Controls.Add($cmbQuality)

$lblConcurrency = New-Object System.Windows.Forms.Label
$lblConcurrency.Location = New-Object System.Drawing.Point(320, 250)
$lblConcurrency.Size = New-Object System.Drawing.Size(120, 24)
$lblConcurrency.Text = "并发数"
$form.Controls.Add($lblConcurrency)

$cmbConcurrency = New-Object System.Windows.Forms.ComboBox
$cmbConcurrency.Location = New-Object System.Drawing.Point(430, 248)
$cmbConcurrency.Size = New-Object System.Drawing.Size(100, 26)
$cmbConcurrency.DropDownStyle = "DropDownList"
[void]$cmbConcurrency.Items.AddRange(@("1", "2", "3", "4", "5", "6", "8"))
$cmbConcurrency.SelectedItem = "4"
$form.Controls.Add($cmbConcurrency)

$chkLyrics = New-Object System.Windows.Forms.CheckBox
$chkLyrics.Location = New-Object System.Drawing.Point(720, 248)
$chkLyrics.Size = New-Object System.Drawing.Size(90, 24)
$chkLyrics.Text = "歌词"
$chkLyrics.Checked = $true
$form.Controls.Add($chkLyrics)

$chkCovers = New-Object System.Windows.Forms.CheckBox
$chkCovers.Location = New-Object System.Drawing.Point(720, 270)
$chkCovers.Size = New-Object System.Drawing.Size(90, 24)
$chkCovers.Text = "封面"
$chkCovers.Checked = $true
$form.Controls.Add($chkCovers)

$chkOverwrite = New-Object System.Windows.Forms.CheckBox
$chkOverwrite.Location = New-Object System.Drawing.Point(720, 292)
$chkOverwrite.Size = New-Object System.Drawing.Size(100, 24)
$chkOverwrite.Text = "覆盖已有文件"
$chkOverwrite.Checked = $false
$form.Controls.Add($chkOverwrite)

$btnSave = New-Object System.Windows.Forms.Button
$btnSave.Location = New-Object System.Drawing.Point(20, 300)
$btnSave.Size = New-Object System.Drawing.Size(120, 34)
$btnSave.Text = "保存 Cookie"
$form.Controls.Add($btnSave)

$btnCheck = New-Object System.Windows.Forms.Button
$btnCheck.Location = New-Object System.Drawing.Point(150, 300)
$btnCheck.Size = New-Object System.Drawing.Size(120, 34)
$btnCheck.Text = "检查 Cookie"
$form.Controls.Add($btnCheck)

$btnStart = New-Object System.Windows.Forms.Button
$btnStart.Location = New-Object System.Drawing.Point(280, 300)
$btnStart.Size = New-Object System.Drawing.Size(140, 34)
$btnStart.Text = "开始下载"
$form.Controls.Add($btnStart)

$btnOpenOutput = New-Object System.Windows.Forms.Button
$btnOpenOutput.Location = New-Object System.Drawing.Point(430, 300)
$btnOpenOutput.Size = New-Object System.Drawing.Size(140, 34)
$btnOpenOutput.Text = "打开输出目录"
$form.Controls.Add($btnOpenOutput)

$btnOpenConfig = New-Object System.Windows.Forms.Button
$btnOpenConfig.Location = New-Object System.Drawing.Point(580, 300)
$btnOpenConfig.Size = New-Object System.Drawing.Size(120, 34)
$btnOpenConfig.Text = "打开配置文件"
$form.Controls.Add($btnOpenConfig)

$btnBuild = New-Object System.Windows.Forms.Button
$btnBuild.Location = New-Object System.Drawing.Point(710, 336)
$btnBuild.Size = New-Object System.Drawing.Size(130, 34)
$btnBuild.Text = "构建 EXE"
$form.Controls.Add($btnBuild)

$txtLog = New-Object System.Windows.Forms.TextBox
$txtLog.Location = New-Object System.Drawing.Point(20, 382)
$txtLog.Size = New-Object System.Drawing.Size(820, 258)
$txtLog.Multiline = $true
$txtLog.ReadOnly = $true
$txtLog.ScrollBars = "Vertical"
$txtLog.BackColor = [System.Drawing.Color]::White
$txtLog.Font = New-Object System.Drawing.Font("Consolas", 9)
$form.Controls.Add($txtLog)

$lblStatus = New-Object System.Windows.Forms.Label
$lblStatus.Location = New-Object System.Drawing.Point(20, 650)
$lblStatus.Size = New-Object System.Drawing.Size(820, 24)
$lblStatus.Text = "就绪"
$form.Controls.Add($lblStatus)

$folderDialog = New-Object System.Windows.Forms.FolderBrowserDialog
$folderDialog.ShowNewFolderButton = $true

$script:Busy = $false

function Append-Log {
    param([string]$Text)

    if ([string]::IsNullOrEmpty($Text)) {
        return
    }
    $txtLog.AppendText($Text)
    if (-not $Text.EndsWith("`n")) {
        $txtLog.AppendText("`r`n")
    }
    $txtLog.SelectionStart = $txtLog.TextLength
    $txtLog.ScrollToCaret()
    [System.Windows.Forms.Application]::DoEvents()
}

function Set-Status {
    param([string]$Text)
    $lblStatus.Text = $Text
    [System.Windows.Forms.Application]::DoEvents()
}

function Set-Busy {
    param([bool]$Value)
    $script:Busy = $Value
    $btnSave.Enabled = -not $Value
    $btnCheck.Enabled = -not $Value
    $btnStart.Enabled = -not $Value
    $btnBrowse.Enabled = -not $Value
    $btnOpenSite.Enabled = -not $Value
    $btnBuild.Enabled = -not $Value
    $btnImportClipboard.Enabled = -not $Value
    $btnImportCookieFile.Enabled = -not $Value
    $btnCookieGuide.Enabled = -not $Value
}

function Save-ConfigFromUI {
    $musicU = $txtMusicU.Text.Trim()
    Ensure-ConfigFile -ConfigPath $configPath -TemplatePath $templatePath
    Set-NeteaseMusicU -ConfigPath $configPath -MusicU $musicU
    if ([string]::IsNullOrWhiteSpace($musicU)) {
        Append-Log "已清除手动填写的 MUSIC_U，运行时会在 Windows 下自动尝试读取浏览器 Cookie。"
    }
    else {
        Append-Log "已保存 MUSIC_U 到 config.ini"
    }
    return $true
}

function Resolve-QualityValue {
    switch ([string]$cmbQuality.SelectedItem) {
        "标准" { return "standard" }
        "较高" { return "high" }
        "无损" { return "lossless" }
        "Hi-Res" { return "hires" }
        default { return "lossless" }
    }
}

function Ensure-Exe {
    if (Test-Path -LiteralPath $exePath) {
        return $true
    }

    if (-not (Test-Path -LiteralPath $builderPath)) {
        [System.Windows.Forms.MessageBox]::Show("缺少 netease-batch.exe，且没有找到构建脚本。", "构建失败", [System.Windows.Forms.MessageBoxButtons]::OK, [System.Windows.Forms.MessageBoxIcon]::Error) | Out-Null
        return $false
    }

    Append-Log "未找到 netease-batch.exe，开始自动构建..."
    $code = Invoke-LoggedProcess -FilePath "cmd.exe" -Arguments @("/c", $builderPath) -SuccessText "构建完成。" -FailureText "构建失败。"
    return ($code -eq 0)
}

function Invoke-LoggedProcess {
    param(
        [string]$FilePath,
        [string[]]$Arguments,
        [string]$SuccessText,
        [string]$FailureText
    )

    $stdoutFile = [System.IO.Path]::GetTempFileName()
    $stderrFile = [System.IO.Path]::GetTempFileName()
    $argText = Join-Arguments -Items $Arguments

    try {
        Set-Busy $true
        Set-Status "运行中..."
        Append-Log "> $FilePath $argText"

        $process = Start-Process -FilePath $FilePath -ArgumentList $argText -WorkingDirectory $repoRoot -RedirectStandardOutput $stdoutFile -RedirectStandardError $stderrFile -PassThru -WindowStyle Hidden

        $lastStdoutLength = 0
        $lastStderrLength = 0

        while (-not $process.HasExited) {
            if (Test-Path -LiteralPath $stdoutFile) {
                $stdoutText = [System.IO.File]::ReadAllText($stdoutFile)
                if ($stdoutText.Length -gt $lastStdoutLength) {
                    Append-Log $stdoutText.Substring($lastStdoutLength)
                    $lastStdoutLength = $stdoutText.Length
                }
            }
            if (Test-Path -LiteralPath $stderrFile) {
                $stderrText = [System.IO.File]::ReadAllText($stderrFile)
                if ($stderrText.Length -gt $lastStderrLength) {
                    Append-Log $stderrText.Substring($lastStderrLength)
                    $lastStderrLength = $stderrText.Length
                }
            }
            [System.Windows.Forms.Application]::DoEvents()
            Start-Sleep -Milliseconds 150
        }

        if (Test-Path -LiteralPath $stdoutFile) {
            $stdoutText = [System.IO.File]::ReadAllText($stdoutFile)
            if ($stdoutText.Length -gt $lastStdoutLength) {
                Append-Log $stdoutText.Substring($lastStdoutLength)
            }
        }
        if (Test-Path -LiteralPath $stderrFile) {
            $stderrText = [System.IO.File]::ReadAllText($stderrFile)
            if ($stderrText.Length -gt $lastStderrLength) {
                Append-Log $stderrText.Substring($lastStderrLength)
            }
        }

        if ($process.ExitCode -eq 0) {
            Set-Status $SuccessText
            if ($SuccessText) {
                Append-Log $SuccessText
            }
        }
        else {
            Set-Status $FailureText
            if ($FailureText) {
                Append-Log $FailureText
            }
        }
        return $process.ExitCode
    }
    finally {
        Set-Busy $false
        Remove-Item -LiteralPath $stdoutFile, $stderrFile -ErrorAction SilentlyContinue
    }
}

$btnOpenSite.Add_Click({
    Start-Process "https://music.163.com"
    Append-Log "已打开 https://music.163.com"
})

$btnCookieGuide.Add_Click({
    $guide = @"
如何获取 MUSIC_U

方式 0：自动读取
1. 在 Edge、Chrome、Brave 或 Firefox 里登录网易云。
2. 这里把 MUSIC_U 留空。
3. 点击“检查 Cookie”或“开始下载”。

方式 1：浏览器开发者工具
1. 点击“打开网易云”并完成登录。
2. 按 F12。
3. 打开 Application。
4. 进入 Storage -> Cookies -> https://music.163.com。
5. 找到 MUSIC_U。
6. 双击 Value 并复制。
7. 粘贴回这个窗口，或者复制整段 Cookie 后点“导入剪贴板”。

方式 2：导出 cookie.txt
1. 使用浏览器 Cookie 导出扩展。
2. 导出 music.163.com 的 Cookie。
3. 点击这里的“导入 Cookie 文件”。

说明
- 只能使用你自己的已登录 Cookie。
- 自动读取仅支持 Windows，且浏览器本地要有可用的登录数据。
- 导入后建议先点“检查 Cookie”。
"@
    [System.Windows.Forms.MessageBox]::Show($guide, "如何获取 MUSIC_U", [System.Windows.Forms.MessageBoxButtons]::OK, [System.Windows.Forms.MessageBoxIcon]::Information) | Out-Null
    Append-Log "已打开 MUSIC_U 获取说明。"
})

$btnImportClipboard.Add_Click({
    try {
        $text = [System.Windows.Forms.Clipboard]::GetText()
    }
    catch {
        [System.Windows.Forms.MessageBox]::Show("无法读取剪贴板。", "剪贴板错误", [System.Windows.Forms.MessageBoxButtons]::OK, [System.Windows.Forms.MessageBoxIcon]::Error) | Out-Null
        return
    }

    $musicU = Extract-MusicU -Text $text
    if ([string]::IsNullOrWhiteSpace($musicU)) {
        [System.Windows.Forms.MessageBox]::Show("剪贴板里没有找到 MUSIC_U 或 Cookie 文本。", "导入失败", [System.Windows.Forms.MessageBoxButtons]::OK, [System.Windows.Forms.MessageBoxIcon]::Warning) | Out-Null
        return
    }

    $txtMusicU.Text = $musicU
    Append-Log "已从剪贴板导入 MUSIC_U。"
})

$chkShowCookie.Add_CheckedChanged({
    $txtMusicU.UseSystemPasswordChar = -not $chkShowCookie.Checked
})

$btnBrowse.Add_Click({
    if (Test-Path -LiteralPath $txtOutput.Text) {
        $folderDialog.SelectedPath = $txtOutput.Text
    }
    if ($folderDialog.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) {
        $txtOutput.Text = $folderDialog.SelectedPath
    }
})

$btnImportCookieFile.Add_Click({
    $dialog = New-Object System.Windows.Forms.OpenFileDialog
    $dialog.Title = "选择 cookie.txt 或导出的 Cookie 文件"
    $dialog.Filter = "Cookie 文件|*.txt;*.cookie;*.cookies|所有文件|*.*"
    if ($dialog.ShowDialog() -ne [System.Windows.Forms.DialogResult]::OK) {
        return
    }

    try {
        $text = Get-Content -LiteralPath $dialog.FileName -Raw
    }
    catch {
        [System.Windows.Forms.MessageBox]::Show("读取所选文件失败。", "读取失败", [System.Windows.Forms.MessageBoxButtons]::OK, [System.Windows.Forms.MessageBoxIcon]::Error) | Out-Null
        return
    }

    $musicU = Extract-MusicU -Text $text
    if ([string]::IsNullOrWhiteSpace($musicU)) {
        [System.Windows.Forms.MessageBox]::Show("所选文件中没有找到 MUSIC_U。", "导入失败", [System.Windows.Forms.MessageBoxButtons]::OK, [System.Windows.Forms.MessageBoxIcon]::Warning) | Out-Null
        return
    }

    $txtMusicU.Text = $musicU
    Append-Log "已从文件导入 MUSIC_U: $($dialog.FileName)"
})

$btnSave.Add_Click({
    [void](Save-ConfigFromUI)
})

$btnCheck.Add_Click({
    if ($script:Busy) {
        return
    }
    if (-not (Save-ConfigFromUI)) {
        return
    }
    if (-not (Ensure-Exe)) {
        return
    }
    [void](Invoke-LoggedProcess -FilePath $exePath -Arguments @("-config", $configPath, "-check") -SuccessText "Cookie 校验通过。" -FailureText "Cookie 校验失败。")
})

$btnStart.Add_Click({
    if ($script:Busy) {
        return
    }
    if (-not (Save-ConfigFromUI)) {
        return
    }
    if (-not (Ensure-Exe)) {
        return
    }

    $playlistURL = $txtURL.Text.Trim()
    if ([string]::IsNullOrWhiteSpace($playlistURL)) {
        [System.Windows.Forms.MessageBox]::Show("歌单链接不能为空。", "缺少链接", [System.Windows.Forms.MessageBoxButtons]::OK, [System.Windows.Forms.MessageBoxIcon]::Warning) | Out-Null
        return
    }

    $outputPath = $txtOutput.Text.Trim()
    if ([string]::IsNullOrWhiteSpace($outputPath)) {
        $outputPath = $downloadDir
        $txtOutput.Text = $outputPath
    }
    if (-not (Test-Path -LiteralPath $outputPath)) {
        [System.IO.Directory]::CreateDirectory($outputPath) | Out-Null
    }

    $args = @(
        "-config", $configPath,
        "-url", $playlistURL,
        "-out", $outputPath,
        "-quality", (Resolve-QualityValue),
        "-concurrency", [string]$cmbConcurrency.SelectedItem
    )
    if (-not $chkLyrics.Checked) {
        $args += "-lyrics=false"
    }
    if (-not $chkCovers.Checked) {
        $args += "-covers=false"
    }
    if ($chkOverwrite.Checked) {
        $args += "-overwrite=true"
    }

    [void](Invoke-LoggedProcess -FilePath $exePath -Arguments $args -SuccessText "下载完成。" -FailureText "下载结束，但有错误。")
})

$btnOpenOutput.Add_Click({
    $outputPath = $txtOutput.Text.Trim()
    if ([string]::IsNullOrWhiteSpace($outputPath)) {
        $outputPath = $downloadDir
    }
    if (-not (Test-Path -LiteralPath $outputPath)) {
        [System.IO.Directory]::CreateDirectory($outputPath) | Out-Null
    }
    Start-Process $outputPath
})

$btnOpenConfig.Add_Click({
    Ensure-ConfigFile -ConfigPath $configPath -TemplatePath $templatePath
    Start-Process $configPath
})

$btnBuild.Add_Click({
    if ($script:Busy) {
        return
    }
    [void](Ensure-Exe)
})

Append-Log "界面已就绪。"
Append-Log "提示：可以手动粘贴 MUSIC_U，也可以留空让程序自动读取浏览器 Cookie。"
Append-Log "然后点击“检查 Cookie”或“开始下载”。"
Set-Status "就绪"

[void]$form.ShowDialog()
