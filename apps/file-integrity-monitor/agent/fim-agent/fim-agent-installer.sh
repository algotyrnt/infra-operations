#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

LOG_FILE="/var/log/fim_install.log"
mkdir -p "$(dirname "$LOG_FILE")"
exec > >(tee -a "$LOG_FILE") 2>&1

echo "[INFO] Starting FIM Agent Installer (no git clone required)..."

# --------------------------------------------------
# Must run as root
# --------------------------------------------------
if [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
  echo "[ERROR] Please run as root."
  echo "[ERROR] Example: sudo bash fim-agent-installation.sh"
  exit 1
fi

# --------------------------------------------------
# Variables
# --------------------------------------------------
FIM_USER="${FIM_USER:-fimuser}"
FIM_GROUP="${FIM_GROUP:-fimuser}"
FIM_HOME="/home/${FIM_USER}"
FIM_DIR="${FIM_HOME}/FIM"
JSON_DIR="${FIM_DIR}/json_dir"
VENV_DIR="${FIM_DIR}/fimenv"

AGENT_DST="${FIM_DIR}/fim-agent.py"
UPLOADER_DST="${FIM_DIR}/data-uploader.py"
CONF_DST="${FIM_DIR}/fim-agent.conf"
ENV_SAMPLE_DST="${FIM_DIR}/.env.sample"
ENV_DST="${FIM_DIR}/.env"

AUDIT_RULES_FILE="/etc/audit/rules.d/audit.rules"
AUDIT_CONF_FILE="/etc/audit/auditd.conf"

SYSTEMD_FIM="/etc/systemd/system/fim.service"
SYSTEMD_UPLOADER="/etc/systemd/system/data-uploader.service"

# Raw GitHub base path 
# Please refer the documentation and enter the latest INSTALL_REF:- 

INSTALL_REF="${INSTALL_REF:-}"

RAW_BASE_URL="https://raw.githubusercontent.com/wso2-open-operations/infra-operations/${INSTALL_REF}/apps/file-integrity-monitor/agent/fim-agent"

AGENT_URL="${RAW_BASE_URL}/fim-agent.py"
UPLOADER_URL="${RAW_BASE_URL}/data-uploader.py"
CONF_URL="${RAW_BASE_URL}/fim-agent.conf"
ENV_SAMPLE_URL="${RAW_BASE_URL}/.env.sample"

# --------------------------------------------------
# Helpers
# --------------------------------------------------
backup_file() {
  local f="$1"
  if [[ -f "$f" ]]; then
    cp -av "$f" "${f}.bak.$(date +%F-%H%M%S)"
  fi
}

command_exists() {
  command -v "$1" >/dev/null 2>&1
}

download_file() {
  local url="$1"
  local dest="$2"

  echo "[INFO] Downloading: $url"
  if command_exists wget; then
    wget -q --show-progress -O "$dest" "$url"
  elif command_exists curl; then
    curl -fL "$url" -o "$dest"
  else
    echo "[ERROR] Neither wget nor curl is installed."
    exit 1
  fi

  if [[ ! -s "$dest" ]]; then
    echo "[ERROR] Download failed or file is empty: $dest"
    exit 1
  fi
}

reload_audit_rules() {
  echo "[INFO] Reloading audit rules..."
  if command_exists augenrules; then
    augenrules --load || true
  fi
  systemctl restart auditd || true
}

# --------------------------------------------------
# 1. Install required packages
# --------------------------------------------------
echo "[INFO] Installing required packages..."
export DEBIAN_FRONTEND=noninteractive
apt-get update -y
apt-get install -y \
  auditd \
  audispd-plugins \
  ca-certificates \
  python3 \
  python3-pip \
  wget \
  curl

systemctl enable auditd
systemctl restart auditd

# --------------------------------------------------
# 2. Create service user
# --------------------------------------------------
if ! id "$FIM_USER" >/dev/null 2>&1; then
  echo "[INFO] Creating user: $FIM_USER"
  useradd -m -s /usr/sbin/nologin "$FIM_USER"
else
  echo "[INFO] User already exists: $FIM_USER"
fi

# --------------------------------------------------
# 3. Create application directories
# --------------------------------------------------
echo "[INFO] Creating application directories..."
mkdir -p "$FIM_DIR"
mkdir -p "$JSON_DIR"

chown -R root:root "$FIM_DIR"
chmod 0755 "$FIM_DIR"
chmod 0755 "$JSON_DIR"

# --------------------------------------------------
# 4. Download project files directly from GitHub
# --------------------------------------------------
echo "[INFO] Downloading FIM agent files..."

download_file "$AGENT_URL" "$AGENT_DST"
download_file "$UPLOADER_URL" "$UPLOADER_DST"
download_file "$CONF_URL" "$CONF_DST"
download_file "$ENV_SAMPLE_URL" "$ENV_SAMPLE_DST"

chmod 0644 "$AGENT_DST"
chmod 0644 "$UPLOADER_DST"
chmod 0600 "$CONF_DST"
chmod 0600 "$ENV_SAMPLE_DST"

# Create .env from .env.sample only if .env does not exist
if [[ ! -f "$ENV_DST" ]]; then
  echo "[INFO] Creating .env from .env.sample"
  cp "$ENV_SAMPLE_DST" "$ENV_DST"
  chmod 0600 "$ENV_DST"
else
  echo "[INFO] Existing .env found, leaving it unchanged"
fi

# Ensure uploader user can read needed files
chown root:root "$AGENT_DST" "$UPLOADER_DST" "$CONF_DST" "$ENV_SAMPLE_DST" "$ENV_DST"
chmod 0644 "$AGENT_DST" "$UPLOADER_DST"
chmod 0640 "$CONF_DST" "$ENV_SAMPLE_DST" "$ENV_DST"
chgrp "$FIM_GROUP" "$CONF_DST" "$ENV_SAMPLE_DST" "$ENV_DST" || true

# --------------------------------------------------
# 5. Configure audit rules
# --------------------------------------------------
backup_file "$AUDIT_RULES_FILE"

echo "[INFO] Writing audit rules to $AUDIT_RULES_FILE..."
cat > "$AUDIT_RULES_FILE" <<'EOF'
-D
-b 65536
--backlog_wait_time 30000
-f 1

-w /etc/ -p wa -k etc_watch
-w /bin/ -p wa -k bin_watch
-w /sbin/ -p wa -k sbin_watch
-w /usr/bin/ -p wa -k usr_bin_watch
-w /usr/sbin/ -p wa -k usr_sbin_watch
-w /usr/local/bin/ -p wa -k usr_local_bin_watch
-w /lib/ -p wa -k lib_watch
-w /usr/lib/ -p wa -k usr_lib_watch
-w /lib64/ -p wa -k lib64_watch
-w /usr/lib64/ -p wa -k usr_lib64_watch
-w /usr/local/lib/ -p wa -k usr_local_lib_watch
-w /boot/ -p wa -k boot_watch
-w /usr/share/ -p wa -k usr_share_watch
EOF

chmod 0640 "$AUDIT_RULES_FILE"
chown root:root "$AUDIT_RULES_FILE"

reload_audit_rules

# --------------------------------------------------
# 6. Configure auditd.conf
# --------------------------------------------------
backup_file "$AUDIT_CONF_FILE"

echo "[INFO] Writing auditd configuration to $AUDIT_CONF_FILE..."
cat > "$AUDIT_CONF_FILE" <<'EOF'
local_events = yes
write_logs = yes
log_file = /var/log/audit/audit.log
log_group = adm
log_format = RAW
flush = INCREMENTAL_ASYNC
freq = 50
max_log_file = 100
num_logs = 5
priority_boost = 4
disp_qos = lossy
dispatcher = /sbin/audispd
name_format = NONE
max_log_file_action = ROTATE
space_left = 75
space_left_action = SYSLOG
action_mail_acct = root
admin_space_left = 50
admin_space_left_action = SUSPEND
disk_full_action = SUSPEND
disk_error_action = SUSPEND
EOF

systemctl restart auditd || true

# --------------------------------------------------
# 7. Setup Python virtual environment
# --------------------------------------------------
echo "[INFO] Setting up Python virtual environment..."

PYTHON_BIN="python3"
for ver in 3.12 3.11 3.10 3.9 3.8; do
  if command_exists "python${ver}"; then
    PYTHON_BIN="python${ver}"
    break
  fi
done

echo "[INFO] Using Python interpreter: $PYTHON_BIN"

PYTHON_MAJOR_MINOR="$($PYTHON_BIN -c 'import sys; print(f"{sys.version_info.major}.{sys.version_info.minor}")')"
VENV_PKG="python${PYTHON_MAJOR_MINOR}-venv"

echo "[INFO] Installing venv package: $VENV_PKG"
apt-get install -y "$VENV_PKG"

rm -rf "$VENV_DIR"
"$PYTHON_BIN" -m venv "$VENV_DIR"

# shellcheck disable=SC1091
source "$VENV_DIR/bin/activate"
pip install --upgrade pip
pip install boto3 python-dotenv
deactivate

chown -R root:root "$VENV_DIR"
chmod -R 0755 "$VENV_DIR"

echo "[INFO] Patching JSON_DIR in fim-agent.conf..."
sed -i "s|^JSON_DIR\s*=.*|JSON_DIR = ${JSON_DIR}|" "$CONF_DST"

# --------------------------------------------------
# 8. Create systemd services
# --------------------------------------------------
echo "[INFO] Creating systemd services..."

cat > "$SYSTEMD_FIM" <<EOF
[Unit]
Description=File Integrity Monitoring Service
After=network.target auditd.service
Wants=auditd.service

[Service]
Type=simple
Environment=FIM_DIR=${FIM_DIR}
Environment=BACKUP_DIR=${FIM_HOME}/BACKUP
ExecStart=${VENV_DIR}/bin/python ${FIM_DIR}/fim-agent.py
WorkingDirectory=${FIM_DIR}
Restart=always
RestartSec=5
User=root
Group=root
NoNewPrivileges=true
PrivateTmp=true

CPUAccounting=true
MemoryAccounting=true
CPUQuota=10%
MemoryMax=300M
MemorySwapMax=0
TasksMax=100

[Install]
WantedBy=multi-user.target
EOF

cat > "$SYSTEMD_UPLOADER" <<EOF
[Unit]
Description=Upload to S3 Service
After=network.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=${VENV_DIR}/bin/python ${FIM_DIR}/data-uploader.py
WorkingDirectory=${FIM_DIR}
Restart=always
RestartSec=5
User=root
Group=root
NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
EOF

chmod 0644 "$SYSTEMD_FIM" "$SYSTEMD_UPLOADER"
systemctl daemon-reload

# --------------------------------------------------
# 9. Do NOT start services automatically
# --------------------------------------------------
echo "[INFO] Services created, but not started yet."
echo "[INFO] This is intentional because credentials must be added first."

# --------------------------------------------------
# 10. Final instructions
# --------------------------------------------------
cat <<EOF

============================================================
FIM Agent installation completed successfully.
============================================================

Files installed to:
  ${FIM_DIR}

Downloaded files:
  ${AGENT_DST}
  ${UPLOADER_DST}
  ${CONF_DST}
  ${ENV_SAMPLE_DST}
  ${ENV_DST}

IMPORTANT:
Before starting the services, update your credentials in:
  1. ${CONF_DST}
  2. ${ENV_DST}

Suggested commands:
  nano ${CONF_DST}
  nano ${ENV_DST}

After updating credentials, run:
  systemctl enable fim.service
  systemctl enable data-uploader.service
  systemctl start fim.service
  systemctl start data-uploader.service

Check status:
  systemctl status fim.service
  systemctl status data-uploader.service

View logs:
  journalctl -u fim.service -f
  journalctl -u data-uploader.service -f

Installer log:
  ${LOG_FILE}

Notes:
- Existing audit config files were backed up before replacement.
- Services were not started automatically.
- If your Python code expects only one config source, keep credentials there consistently.

============================================================
EOF
