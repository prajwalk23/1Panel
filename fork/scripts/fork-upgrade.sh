#!/usr/bin/env bash
# fork: installs a fork release on a server that runs official 1Panel, and rolls it back.
#
#   bash fork-upgrade.sh <tag>                 install a fork release, for example v2.5.1.1
#   bash fork-upgrade.sh --rollback [dir]      restore the binaries from the last (or given) backup
#   bash fork-upgrade.sh --rollback-db [dir]   same, and also restore the panel database from the backup
#   bash fork-upgrade.sh --list                list backups
#
# Only /usr/local/bin/1panel-core and /usr/local/bin/1panel-agent are replaced. The release is built
# on the same upstream version as the running panel, so 1pctl, init scripts and language files stay
# as they are. Data under <InstallDir>/1panel and Docker are not touched.
set -euo pipefail

REPO="${FORK_REPO:-prajwalk23/1Panel}"
BIN_DIR="/usr/local/bin"
CTL="${BIN_DIR}/1pctl"
SERVICES="1panel-core 1panel-agent"

log() { printf '[fork] %s\n' "$*"; }
die() { printf '[fork] ERROR: %s\n' "$*" >&2; exit 1; }

[ "$(id -u)" -eq 0 ] || die "run this script as root"
[ -f "${CTL}" ] || die "${CTL} not found. Is 1Panel installed?"
for tool in curl tar sha256sum; do
    command -v "${tool}" >/dev/null 2>&1 || die "${tool} is required"
done

ctl_value() {
    grep -E "^$1=" "${CTL}" | head -n1 | cut -d= -f2- | tr -d '"' | tr -d "'"
}

INSTALL_DIR="$(ctl_value BASE_DIR)"
INSTALL_DIR="${INSTALL_DIR:-/opt}"
DATA_DIR="${INSTALL_DIR}/1panel"
BACKUP_ROOT="${DATA_DIR}/fork-backup"
PANEL_VERSION="$(ctl_value ORIGINAL_VERSION)"

use_systemd() {
    # No pipes here: with "set -o pipefail" a "systemctl ... | grep -q" can fail with SIGPIPE.
    [ -d /run/systemd/system ] && command -v systemctl >/dev/null 2>&1 && systemctl cat 1panel-core >/dev/null 2>&1
}

service_ctl() {
    local action="$1"
    if use_systemd; then
        # shellcheck disable=SC2086
        systemctl "${action}" ${SERVICES}
    elif command -v rc-service >/dev/null 2>&1; then
        for svc in ${SERVICES}; do rc-service "${svc}" "${action}"; done
    else
        for svc in ${SERVICES}; do service "${svc}" "${action}"; done
    fi
}

# Prints one PID per running service, empty if one of them is not running.
service_pids() {
    local svc pid out=""
    for svc in ${SERVICES}; do
        if use_systemd; then
            systemctl is-active --quiet "${svc}" || return 0
            pid="$(systemctl show -p MainPID --value "${svc}" 2>/dev/null || true)"
        else
            pid="$(pgrep -x "${svc}" | head -n1 || true)"
        fi
        { [ -n "${pid}" ] && [ "${pid}" != "0" ]; } || return 0
        out="${out}${svc}=${pid} "
    done
    printf '%s' "${out}"
}

# The panel must be running with the same processes after 10 seconds, so a crash loop
# (the services restart themselves) is not mistaken for a healthy start.
panel_stable() {
    local first second
    sleep 5
    first="$(service_pids)"
    [ -n "${first}" ] || return 1
    sleep 5
    second="$(service_pids)"
    [ -n "${second}" ] && [ "${first}" = "${second}" ]
}

detect_arch() {
    case "$(uname -m)" in
        x86_64 | amd64) echo amd64 ;;
        aarch64 | arm64) echo arm64 ;;
        *) die "unsupported architecture $(uname -m), only amd64 and arm64 releases are built" ;;
    esac
}

latest_backup() {
    [ -d "${BACKUP_ROOT}" ] || return 0
    find "${BACKUP_ROOT}" -mindepth 1 -maxdepth 1 -type d | sort | tail -n1
}

swap_binary() { # source target
    install -m 755 "$1" "$2.fork-new"
    mv -f "$2.fork-new" "$2"
}

restore() { # backup-dir restore-db
    local dir="$1" with_db="$2"
    [ -n "${dir}" ] && [ -d "${dir}" ] || die "no backup found in ${BACKUP_ROOT}"
    [ -f "${dir}/1panel-core" ] && [ -f "${dir}/1panel-agent" ] || die "${dir} does not contain both binaries"
    log "restoring binaries from ${dir}"
    service_ctl stop || true
    swap_binary "${dir}/1panel-core" "${BIN_DIR}/1panel-core"
    swap_binary "${dir}/1panel-agent" "${BIN_DIR}/1panel-agent"
    if [ "${with_db}" = "yes" ]; then
        [ -d "${dir}/db" ] || die "${dir} has no database backup"
        log "restoring database from ${dir}/db"
        cp -a "${dir}/db/." "${DATA_DIR}/db/"
    fi
    service_ctl start
    log "rollback done. Panel version: $(ctl_value ORIGINAL_VERSION)"
}

install_release() {
    local tag="$1"
    [[ "${tag}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]] || die "tag must look like v2.5.1.1"
    local base="${tag%.*}"
    if [ "${PANEL_VERSION}" != "${base}" ] && [ "${FORK_FORCE:-}" != "1" ]; then
        die "this panel runs ${PANEL_VERSION:-unknown}, but ${tag} is built on ${base}. Update the panel to ${base} first (or set FORK_FORCE=1 if you know what you are doing)."
    fi
    local arch file url tmp pkg backup
    arch="$(detect_arch)"
    file="1panel-${tag}-linux-${arch}.tar.gz"
    url="https://github.com/${REPO}/releases/download/${tag}"
    tmp="$(mktemp -d)"
    trap 'rm -rf "${tmp}"' EXIT

    log "downloading ${file} from ${REPO}"
    curl -fsSL --retry 3 -o "${tmp}/checksums.txt" "${url}/checksums.txt" || die "cannot download checksums.txt for ${tag}"
    curl -fL --retry 3 -o "${tmp}/${file}" "${url}/${file}" || die "cannot download ${file}"
    (cd "${tmp}" && grep -E "  \*?${file}\$" checksums.txt | sed 's/\*//' | sha256sum -c -) || die "checksum check failed for ${file}"

    tar -xzf "${tmp}/${file}" -C "${tmp}"
    pkg="${tmp}/1panel-${tag}-linux-${arch}"
    [ -x "${pkg}/1panel-core" ] && [ -x "${pkg}/1panel-agent" ] || die "package does not contain both binaries"

    backup="${BACKUP_ROOT}/$(date +%Y%m%d-%H%M%S)"
    mkdir -p "${backup}"
    chmod 700 "${BACKUP_ROOT}" "${backup}"

    log "stopping the panel"
    service_ctl stop

    log "backing up binaries and database to ${backup}"
    cp -a "${BIN_DIR}/1panel-core" "${BIN_DIR}/1panel-agent" "${CTL}" "${backup}/"
    if [ -d "${DATA_DIR}/db" ]; then
        cp -a "${DATA_DIR}/db" "${backup}/db"
    fi
    printf 'previous=%s\ninstalled=%s\n' "${PANEL_VERSION}" "${tag}" > "${backup}/FORK_INFO"

    log "installing ${tag}"
    swap_binary "${pkg}/1panel-core" "${BIN_DIR}/1panel-core"
    swap_binary "${pkg}/1panel-agent" "${BIN_DIR}/1panel-agent"

    log "starting the panel"
    if ! service_ctl start; then
        log "start failed, rolling back"
        restore "${backup}" no
        die "installation of ${tag} failed and was rolled back"
    fi
    if ! panel_stable; then
        log "the panel did not stay up, rolling back. Logs: journalctl -u 1panel-core -u 1panel-agent -n 80"
        restore "${backup}" no
        die "installation of ${tag} failed and was rolled back"
    fi
    log "installed ${tag}. Backup: ${backup}"
    log "roll back with: bash $0 --rollback ${backup}"
}

case "${1:-}" in
    --rollback) restore "${2:-$(latest_backup)}" no ;;
    --rollback-db) restore "${2:-$(latest_backup)}" yes ;;
    --list)
        if [ -d "${BACKUP_ROOT}" ]; then
            for dir in $(find "${BACKUP_ROOT}" -mindepth 1 -maxdepth 1 -type d | sort); do
                printf '%s  %s\n' "${dir}" "$(tr '\n' ' ' < "${dir}/FORK_INFO" 2>/dev/null || true)"
            done
        fi
        ;;
    v*) install_release "$1" ;;
    *)
        sed -n '2,10p' "$0" | sed 's/^# \{0,1\}//'
        exit 1
        ;;
esac
