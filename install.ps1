# TarsWingGuard One-Line Installer (PowerShell)
# Usage: iwr -useb https://raw.githubusercontent.com/HP-jh/TarsWingGuard/main/install.ps1 | iex

$ErrorActionPreference = "Stop"
$repo = "HP-jh/TarsWingGuard"
$version = if ($env:VERSION) { $env:VERSION } else { "0.1.0" }
$installDir = if ($env:INSTALL_DIR) { $env:INSTALL_DIR } else { "$env:LOCALAPPDATA\TarsWingGuard" }

# Detect platform
$arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }
if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { $arch = "arm64" }
$os = "windows"
$binary = "twg-v${version}-${os}-${arch}.exe"
$url = "https://github.com/${repo}/releases/download/v${version}/${binary}"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  TarsWingGuard Installer v${version}" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Step 1: welcome
Write-Host "[1/7] 欢迎使用 TarsWingGuard 安装向导" -ForegroundColor Green
Write-Host "      平台: ${os}/${arch}"
Write-Host ""

# Step 2: check dependencies
Write-Host "[2/7] 检查依赖..." -ForegroundColor Green
if (-not (Get-Command curl -ErrorAction SilentlyContinue)) {
    Write-Host "      缺少 curl，请安装后重试" -ForegroundColor Red
    exit 1
}
Write-Host "      依赖检查通过"
Write-Host ""

# Step 3: knowledge config prompt
Write-Host "[3/7] 知识库配置（可选）..." -ForegroundColor Green
$enableKb = Read-Host "      是否启用 Obsidian 知识库? [y/N]"
$obsidianVault = ""
$enableVoice = ""
if ($enableKb -match "^[Yy]$") {
    $obsidianVault = Read-Host "      请输入 Obsidian vault 路径 (如 C:\Users\you\Obsidian)"
    $voice = Read-Host "      是否启用语音输入纠错? [y/N]"
    if ($voice -match "^[Yy]$") {
        $enableVoice = "true"
    }
    Write-Host "      知识库配置完成"
} else {
    Write-Host "      跳过知识库配置"
}
Write-Host ""

# Step 4: download
Write-Host "[4/7] 下载 ${binary}..." -ForegroundColor Green
$tmpDir = [System.IO.Path]::GetTempPath() + [System.Guid]::NewGuid().ToString()
New-Item -ItemType Directory -Path $tmpDir | Out-Null
try {
    curl -fsSL -o "$tmpDir\twg.exe" $url
    Write-Host "      下载完成"
} catch {
    Write-Host "      下载失败: $_" -ForegroundColor Red
    exit 1
}
Write-Host ""

# Step 5: verify checksum
Write-Host "[5/7] 校验文件..." -ForegroundColor Green
$checksumUrl = "https://github.com/${repo}/releases/download/v${version}/SHA256SUMS.txt"
try {
    curl -fsSL -o "$tmpDir\SHA256SUMS.txt" $checksumUrl | Out-Null
    $expected = (Get-Content "$tmpDir\SHA256SUMS.txt" | Select-String $binary).Line.Split(" ")[0]
    $actual = (Get-FileHash "$tmpDir\twg.exe" -Algorithm SHA256).Hash.ToLower()
    if ($expected -eq $actual) {
        Write-Host "      校验通过"
    } else {
        Write-Host "      警告: 校验和验证失败，继续安装（风险自负）" -ForegroundColor Yellow
    }
} catch {
    Write-Host "      无校验和文件，跳过"
}
Write-Host ""

# Step 6: install
Write-Host "[6/7] 安装到 ${installDir}..." -ForegroundColor Green
New-Item -ItemType Directory -Path $installDir -Force | Out-Null
Copy-Item "$tmpDir\twg.exe" "$installDir\twg.exe" -Force
Copy-Item "$tmpDir\twg.exe" "$installDir\wing-guard.exe" -Force 2>$null
Copy-Item "$tmpDir\twg.exe" "$installDir\knowledge.exe" -Force 2>$null
Write-Host "      安装完成"
Write-Host ""

# Step 7: setup config
Write-Host "[7/7] 初始化配置..." -ForegroundColor Green
$configDir = "$env:USERPROFILE\.config\tarswingguard"
New-Item -ItemType Directory -Path $configDir -Force | Out-Null
if (-not (Test-Path "$configDir\wing.yaml")) {
    $vaultVal = if ($obsidianVault) { $obsidianVault } else { "" }
    @"
version: "0.1.0"
schema: wing/v1
core:
  name: TarsWingCore
  description: 我的网关
  version: "0.1.0"
modules:
  - id: wing-guard
    name: "羽翼防护"
    version: "0.1.0"
    tier: pro
    transport: stdio
    command: wing-guard
    deps: [core-waf]
  - id: knowledge
    name: "知识大脑"
    version: "0.1.0"
    tier: pro
    transport: stdio
    command: knowledge
    env:
      - name: OBSIDIAN_VAULT
        value: "$vaultVal"
"@ | Out-File -FilePath "$configDir\wing.yaml" -Encoding utf8
    Write-Host "      已创建默认配置: $configDir\wing.yaml"
} else {
    Write-Host "      配置已存在，跳过"
}

if ($obsidianVault) {
    @"
OBSIDIAN_VAULT=$obsidianVault
ENABLE_VOICE_CORRECTION=$enableVoice
"@ | Out-File -FilePath "$configDir\knowledge.env" -Encoding utf8
    Write-Host "      已创建知识库环境: $configDir\knowledge.env"
}
Write-Host ""

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  安装完成!" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "  快速开始:"
Write-Host "    twg -list              列出所有模块"
Write-Host "    twg -resolve pro       查看 pro 套餐模块"
Write-Host "    twg -tiers             查看套餐对比"
Write-Host "    wing-guard -version    查看羽翼防护版本"
Write-Host "    knowledge -version     查看知识大脑版本"
Write-Host ""
Write-Host "  启用羽翼防护:"
Write-Host "    在 TSG config.json 中添加:"
Write-Host '    {"wing": {"mode": "tier", "tier": "pro"}}'
Write-Host ""
Write-Host "  知识库 + Obsidian:"
Write-Host "    设置环境变量 OBSIDIAN_VAULT=C:\path\to\vault"
Write-Host "    knowledge -vault C:\path\to\vault"
Write-Host ""
Write-Host "  文档: https://github.com/${repo}"

# Add to PATH if not already
$pathEntry = $installDir
$currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($currentPath -notlike "*$pathEntry*") {
    Write-Host ""
    Write-Host "  提示: 已将 ${installDir} 添加到用户 PATH" -ForegroundColor Yellow
    [Environment]::SetEnvironmentVariable("Path", "$currentPath;$pathEntry", "User")
}
