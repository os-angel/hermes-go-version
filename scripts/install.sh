#!/usr/bin/env bash
# install.sh — Instala hermes-go en Linux, macOS y WSL.
#
# Uso:
#   curl -fsSL https://raw.githubusercontent.com/TU_USUARIO/hermes-go/main/scripts/install.sh | bash
#
# Lo que hace:
#   1. Verifica dependencias (Node.js >= 18, npm)
#   2. Descarga el binario pre-compilado para tu OS/arch desde GitHub Releases
#   3. Instala el bridge (bridge.js + npm install) en ~/.hermes-go/bridge/
#   4. Copia config de ejemplo a ~/.hermes-go/config.yaml si no existe
#   5. Agrega hermes-go al PATH en ~/.bashrc y ~/.zshrc

set -euo pipefail

REPO="os-angel/hermes-go-version"
INSTALL_DIR="${HOME}/.local/bin"
HERMES_HOME="${HERMES_GO_HOME:-${HOME}/.hermes-go}"
BRIDGE_DIR="${HERMES_HOME}/bridge"

RED='\033[0;31m'
GRN='\033[0;32m'
YLW='\033[1;33m'
NC='\033[0m'

info()  { echo -e "${GRN}[hermes-go]${NC} $*"; }
warn()  { echo -e "${YLW}[hermes-go]${NC} $*"; }
error() { echo -e "${RED}[hermes-go] ERROR:${NC} $*" >&2; exit 1; }

# --- Detectar OS y arch ---
detect_platform() {
    local os arch
    case "$(uname -s)" in
        Linux*)  os="linux" ;;
        Darwin*) os="darwin" ;;
        *)       error "OS no soportado: $(uname -s). Usa Windows con install.ps1." ;;
    esac
    case "$(uname -m)" in
        x86_64|amd64) arch="amd64" ;;
        arm64|aarch64) arch="arm64" ;;
        *) error "Arquitectura no soportada: $(uname -m)" ;;
    esac
    echo "${os}_${arch}"
}

# --- Verificar dependencias ---
check_deps() {
    if ! command -v node &>/dev/null; then
        error "Node.js no encontrado. Instala Node.js >= 18 desde https://nodejs.org y vuelve a ejecutar este script."
    fi
    local node_major
    node_major=$(node -e "process.stdout.write(process.version.split('.')[0].replace('v',''))")
    if [[ "${node_major}" -lt 18 ]]; then
        error "Node.js >= 18 requerido. Version actual: $(node --version)"
    fi
    if ! command -v npm &>/dev/null; then
        error "npm no encontrado. Instala Node.js >= 18 desde https://nodejs.org"
    fi
}

# --- Obtener ultima version ---
get_latest_version() {
    local version
    version=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
        | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
    if [[ -z "${version}" ]]; then
        error "No se pudo obtener la ultima version desde GitHub. Verifica tu conexion o el nombre del repo."
    fi
    echo "${version}"
}

# --- Descargar binario ---
download_binary() {
    local version="$1"
    local platform="$2"
    local url="https://github.com/${REPO}/releases/download/${version}/hermes-go_${platform}.tar.gz"
    local tmp_dir
    tmp_dir=$(mktemp -d)

    info "Descargando hermes-go ${version} para ${platform}..."
    if ! curl -fsSL "${url}" -o "${tmp_dir}/hermes-go.tar.gz"; then
        error "No se pudo descargar desde: ${url}"
    fi

    tar -xzf "${tmp_dir}/hermes-go.tar.gz" -C "${tmp_dir}"
    mkdir -p "${INSTALL_DIR}"
    install -m 755 "${tmp_dir}/hermes-go" "${INSTALL_DIR}/hermes-go"
    rm -rf "${tmp_dir}"
    info "Binario instalado en ${INSTALL_DIR}/hermes-go"
}

# --- Instalar bridge ---
install_bridge() {
    local version="$1"
    local url="https://github.com/${REPO}/releases/download/${version}/bridge.tar.gz"
    local tmp_dir
    tmp_dir=$(mktemp -d)

    info "Descargando bridge WhatsApp..."
    if ! curl -fsSL "${url}" -o "${tmp_dir}/bridge.tar.gz"; then
        error "No se pudo descargar el bridge desde: ${url}"
    fi

    mkdir -p "${BRIDGE_DIR}"
    tar -xzf "${tmp_dir}/bridge.tar.gz" -C "${BRIDGE_DIR}" --strip-components=1
    rm -rf "${tmp_dir}"

    info "Instalando dependencias Node.js del bridge..."
    (cd "${BRIDGE_DIR}" && npm install --omit=dev --silent)
    info "Bridge instalado en ${BRIDGE_DIR}"
}

# --- Instalar config de ejemplo ---
install_config() {
    local cfg="${HERMES_HOME}/config.yaml"
    if [[ -f "${cfg}" ]]; then
        warn "config.yaml ya existe en ${cfg}, no se sobreescribe."
        return
    fi
    local url="https://raw.githubusercontent.com/${REPO}/main/config.example.yaml"
    mkdir -p "${HERMES_HOME}"
    if curl -fsSL "${url}" -o "${cfg}"; then
        info "Config de ejemplo copiado a ${cfg}"
        warn "Edita ${cfg} para agregar tu API key y configuracion."
    fi
}

# --- Agregar al PATH ---
add_to_path() {
    local shell_rc=""
    local export_line="export PATH=\"\${HOME}/.local/bin:\${PATH}\""

    for rc in "${HOME}/.bashrc" "${HOME}/.zshrc"; do
        if [[ -f "${rc}" ]]; then
            if ! grep -q ".local/bin" "${rc}"; then
                echo "" >> "${rc}"
                echo "# hermes-go" >> "${rc}"
                echo "${export_line}" >> "${rc}"
                info "PATH actualizado en ${rc}"
            fi
        fi
    done

    if ! echo "${PATH}" | grep -q ".local/bin"; then
        warn "Reinicia tu terminal o ejecuta: export PATH=\"\${HOME}/.local/bin:\${PATH}\""
    fi
}

# --- Main ---
main() {
    info "Instalando hermes-go..."
    check_deps

    local platform version
    platform=$(detect_platform)
    version=$(get_latest_version)

    info "Version: ${version} | Plataforma: ${platform}"
    download_binary "${version}" "${platform}"
    install_bridge "${version}"
    install_config
    add_to_path

    echo ""
    info "Instalacion completa."
    echo ""
    echo "  Siguiente paso: edita tu config:"
    echo "    ${HERMES_HOME}/config.yaml"
    echo ""
    echo "  Luego arranca el agente:"
    echo "    hermes-go --config ${HERMES_HOME}/config.yaml"
    echo ""
}

main "$@"
