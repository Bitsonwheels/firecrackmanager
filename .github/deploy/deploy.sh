#!/usr/bin/env bash
set -Eeuo pipefail

require_env() {
  local name="$1"
  if [[ -z "${!name:-}" ]]; then
    echo "Missing required environment variable: ${name}" >&2
    exit 1
  fi
}

require_env APP_BINARY
require_env APP_DATA_DIR
require_env APP_PORT
require_env DEPLOY_HOST
require_env DEPLOY_USER
require_env DEPLOY_SSH_PORT
require_env SSH_KEY_PATH

artifact="dist/${APP_BINARY}"
checksum="${artifact}.sha256"

if [[ ! -f "${artifact}" ]]; then
  echo "Missing artifact: ${artifact}" >&2
  exit 1
fi

if [[ ! -f "${checksum}" ]]; then
  echo "Missing artifact checksum: ${checksum}" >&2
  exit 1
fi

remote="${DEPLOY_USER}@${DEPLOY_HOST}"
tmp_suffix="${GITHUB_RUN_ID:-manual}-${GITHUB_RUN_ATTEMPT:-0}-${GITHUB_JOB:-deploy}-$$"
tmp_suffix="${tmp_suffix//[^A-Za-z0-9._-]/-}"
remote_tmp="/tmp/firecrackmanager-${tmp_suffix}"
ssh_opts=(
  -i "${SSH_KEY_PATH}"
  -p "${DEPLOY_SSH_PORT}"
  -o BatchMode=yes
  -o StrictHostKeyChecking=yes
)

ssh "${ssh_opts[@]}" "${remote}" "rm -rf '${remote_tmp}' && install -d -m 700 '${remote_tmp}'"
scp -i "${SSH_KEY_PATH}" -P "${DEPLOY_SSH_PORT}" -o BatchMode=yes -o StrictHostKeyChecking=yes \
  "${artifact}" "${checksum}" "${remote}:${remote_tmp}/"

ssh "${ssh_opts[@]}" "${remote}" \
  "APP_BINARY='${APP_BINARY}' APP_DATA_DIR='${APP_DATA_DIR}' APP_PORT='${APP_PORT}' REMOTE_TMP='${remote_tmp}' bash -s" <<'REMOTE_SCRIPT'
set -Eeuo pipefail

artifact="${REMOTE_TMP}/${APP_BINARY}"
checksum="${REMOTE_TMP}/${APP_BINARY}.sha256"
service_file="/etc/systemd/system/firecrackmanager.service"
config_dir="/etc/firecrackmanager"
config_file="${config_dir}/settings.json"
builder_dir="/home/Builder"

cd "${REMOTE_TMP}"
sha256sum -c "${APP_BINARY}.sha256"

# 部署前先准备持久化目录，避免服务启动时写到默认 /var/lib。
install -d -m 755 \
  "${APP_DATA_DIR}" \
  "${APP_DATA_DIR}/kernels" \
  "${APP_DATA_DIR}/rootfs" \
  "${APP_DATA_DIR}/sockets" \
  "${APP_DATA_DIR}/snapshots" \
  "${APP_DATA_DIR}/disks" \
  "${APP_DATA_DIR}/logs" \
  "${builder_dir}" \
  "${config_dir}"

install -m 755 "${artifact}" /usr/local/bin/firecrackmanager

cat > "${config_file}" <<EOF_CONFIG
{
    "listen_port": ${APP_PORT},
    "listen_address": "0.0.0.0",
    "data_dir": "${APP_DATA_DIR}",
    "database_path": "${APP_DATA_DIR}/firecrackmanager.db",
    "log_file": "${APP_DATA_DIR}/logs/firecrackmanager.log",
    "pid_file": "/run/firecrackmanager.pid",
    "enable_host_network_management": false,
    "builder_dir": "${builder_dir}"
}
EOF_CONFIG
chmod 644 "${config_file}"

cat > "${service_file}" <<EOF_SERVICE
[Unit]
Description=FireCrackManager - MicroVM Management Daemon
Wants=network-online.target
After=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/firecrackmanager -config ${config_file}
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal
NoNewPrivileges=false
ProtectSystem=strict
ProtectHome=read-only
ReadWritePaths=${APP_DATA_DIR} ${config_dir} /run ${builder_dir}
PrivateTmp=true
AmbientCapabilities=CAP_NET_ADMIN CAP_SYS_ADMIN CAP_KILL CAP_NET_RAW CAP_CHOWN CAP_DAC_OVERRIDE CAP_SETUID CAP_SETGID CAP_SYS_CHROOT CAP_MKNOD CAP_FOWNER
CapabilityBoundingSet=CAP_NET_ADMIN CAP_SYS_ADMIN CAP_KILL CAP_NET_RAW CAP_CHOWN CAP_DAC_OVERRIDE CAP_SETUID CAP_SETGID CAP_SYS_CHROOT CAP_MKNOD CAP_FOWNER

[Install]
WantedBy=multi-user.target
EOF_SERVICE
chmod 644 "${service_file}"

systemctl daemon-reload
systemctl enable firecrackmanager
systemctl restart firecrackmanager
systemctl is-active --quiet firecrackmanager

# systemd active 只代表进程已启动，HTTP 端口可能还在初始化。
for _ in {1..30}; do
  if curl --fail --silent --max-time 2 "http://0.0.0.0:${APP_PORT}/" >/dev/null; then
    break
  fi
  sleep 2
done
curl --fail --silent --show-error --max-time 10 "http://0.0.0.0:${APP_PORT}/" >/dev/null
systemctl --no-pager --full status firecrackmanager

rm -rf "${REMOTE_TMP}"
REMOTE_SCRIPT
