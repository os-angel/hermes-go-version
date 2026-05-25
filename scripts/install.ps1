# install.ps1 — Instala hermes-go en Windows.
#
# Uso:
#   iex (irm https://raw.githubusercontent.com/TU_USUARIO/hermes-go/main/scripts/install.ps1)
#
# Lo que hace:
#   1. Verifica Node.js >= 18
#   2. Descarga el binario hermes-go_windows_amd64.zip desde GitHub Releases
#   3. Instala el bridge (bridge.js + npm install) en ~/.hermes-go/bridge/
#   4. Copia config de ejemplo a ~/.hermes-go/config.yaml si no existe
#   5. Agrega el directorio de instalacion al PATH del usuario

$ErrorActionPreference = 'Stop'

$Repo        = "os-angel/hermes-go-version"
$InstallDir  = "$env:LOCALAPPDATA\hermes-go\bin"
$HermesHome  = if ($env:HERMES_GO_HOME) { $env:HERMES_GO_HOME } else { "$env:USERPROFILE\.hermes-go" }
$BridgeDir   = "$HermesHome\bridge"

function Write-Info  { Write-Host "[hermes-go] $args" -ForegroundColor Green }
function Write-Warn  { Write-Host "[hermes-go] $args" -ForegroundColor Yellow }
function Write-Err   { Write-Host "[hermes-go] ERROR: $args" -ForegroundColor Red; exit 1 }

# --- Verificar Node.js ---
function Check-Node {
    if (-not (Get-Command node -ErrorAction SilentlyContinue)) {
        Write-Err "Node.js no encontrado. Instala Node.js >= 18 desde https://nodejs.org y vuelve a ejecutar este script."
    }
    $nodeVersion = (node -e "process.stdout.write(process.version)").TrimStart('v')
    $nodeMajor   = [int]($nodeVersion.Split('.')[0])
    if ($nodeMajor -lt 18) {
        Write-Err "Node.js >= 18 requerido. Version actual: v$nodeVersion"
    }
    Write-Info "Node.js v$nodeVersion detectado."
}

# --- Obtener ultima version ---
function Get-LatestVersion {
    $releaseUrl = "https://api.github.com/repos/$Repo/releases/latest"
    try {
        $release = Invoke-RestMethod -Uri $releaseUrl -Headers @{ 'User-Agent' = 'hermes-go-installer' }
        return $release.tag_name
    } catch {
        Write-Err "No se pudo obtener la ultima version desde GitHub: $_"
    }
}

# --- Descargar binario ---
function Install-Binary {
    param($Version)
    $url    = "https://github.com/$Repo/releases/download/$Version/hermes-go_windows_amd64.zip"
    $tmpZip = "$env:TEMP\hermes-go.zip"
    $tmpDir = "$env:TEMP\hermes-go-install"

    Write-Info "Descargando hermes-go $Version para windows_amd64..."
    Invoke-WebRequest -Uri $url -OutFile $tmpZip -UseBasicParsing

    if (Test-Path $tmpDir) { Remove-Item $tmpDir -Recurse -Force }
    Expand-Archive -Path $tmpZip -DestinationPath $tmpDir

    if (-not (Test-Path $InstallDir)) { New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null }
    Copy-Item "$tmpDir\hermes-go.exe" "$InstallDir\hermes-go.exe" -Force

    Remove-Item $tmpZip, $tmpDir -Recurse -Force
    Write-Info "Binario instalado en $InstallDir\hermes-go.exe"
}

# --- Instalar bridge ---
function Install-Bridge {
    param($Version)
    $url    = "https://github.com/$Repo/releases/download/$Version/bridge.zip"
    $tmpZip = "$env:TEMP\hermes-go-bridge.zip"
    $tmpDir = "$env:TEMP\hermes-go-bridge"

    Write-Info "Descargando bridge WhatsApp..."
    Invoke-WebRequest -Uri $url -OutFile $tmpZip -UseBasicParsing

    if (Test-Path $tmpDir) { Remove-Item $tmpDir -Recurse -Force }
    Expand-Archive -Path $tmpZip -DestinationPath $tmpDir

    if (-not (Test-Path $BridgeDir)) { New-Item -ItemType Directory -Path $BridgeDir -Force | Out-Null }

    # Copiar contenido (puede venir con un subdirectorio)
    $innerDir = Get-ChildItem $tmpDir | Where-Object { $_.PSIsContainer } | Select-Object -First 1
    $source   = if ($innerDir) { $innerDir.FullName } else { $tmpDir }
    Copy-Item "$source\*" $BridgeDir -Recurse -Force

    Remove-Item $tmpZip, $tmpDir -Recurse -Force

    Write-Info "Instalando dependencias Node.js del bridge..."
    Push-Location $BridgeDir
    npm install --omit=dev --silent
    Pop-Location
    Write-Info "Bridge instalado en $BridgeDir"
}

# --- Config de ejemplo ---
function Install-Config {
    $cfg = "$HermesHome\config.yaml"
    if (Test-Path $cfg) {
        Write-Warn "config.yaml ya existe en $cfg, no se sobreescribe."
        return
    }
    $url = "https://raw.githubusercontent.com/$Repo/main/config.example.yaml"
    if (-not (Test-Path $HermesHome)) { New-Item -ItemType Directory -Path $HermesHome -Force | Out-Null }
    try {
        Invoke-WebRequest -Uri $url -OutFile $cfg -UseBasicParsing
        Write-Info "Config de ejemplo copiado a $cfg"
        Write-Warn "Edita $cfg para agregar tu API key y configuracion."
    } catch {
        Write-Warn "No se pudo descargar config de ejemplo: $_"
    }
}

# --- Agregar al PATH del usuario ---
function Add-ToPath {
    $currentPath = [Environment]::GetEnvironmentVariable('PATH', 'User')
    if ($currentPath -notlike "*$InstallDir*") {
        [Environment]::SetEnvironmentVariable('PATH', "$InstallDir;$currentPath", 'User')
        $env:PATH = "$InstallDir;$env:PATH"
        Write-Info "PATH actualizado. Reinicia tu terminal para aplicar."
    }
}

# --- Main ---
Write-Info "Instalando hermes-go..."
Check-Node

$version = Get-LatestVersion
Write-Info "Version: $version"

Install-Binary $version
Install-Bridge $version
Install-Config
Add-ToPath

Write-Host ""
Write-Info "Instalacion completa."
Write-Host ""
Write-Host "  Siguiente paso: edita tu config:"
Write-Host "    $HermesHome\config.yaml"
Write-Host ""
Write-Host "  Luego arranca el agente:"
Write-Host "    hermes-go --config `"$HermesHome\config.yaml`""
Write-Host ""
