#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

LOG_FILE="/var/log/fim_install.log"
mkdir -p "$(dirname "$LOG_FILE")"
exec > >(tee -a "$LOG_FILE") 2>&1

echo "[INFO] Starting FIM Agent One-Click Installer..."

# ---------------------------
# Must run as root
# ---------------------------
if [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
  echo "[ERROR] Please run as root: sudo bash oneclickinstallation.sh"
  exit 1
fi

# ---------------------------
# Variables
# ---------------------------
FIM_USER="${FIM_USER:-fimuser}"
FIM_DIR="/home/${FIM_USER}/FIM"
JSON_DIR="${FIM_DIR}/json_dir"
VENV_DIR="${FIM_DIR}/fimenv"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

AGENT_SRC="${SCRIPT_DIR}/fim-agent.py"
UPLOADER_SRC="${SCRIPT_DIR}/data-uploader.py"
CONF_SRC="${SCRIPT_DIR}/fim-agent.conf"

AGENT_DST="${FIM_DIR}/fim-agent.py"
UPLOADER_DST="${FIM_DIR}/data-uploader.py"
CONF_DST="${FIM_DIR}/fim-agent.conf"

# Use audit.rules (overwrite after backup)
AUDIT_RULES_FILE="/etc/audit/rules.d/audit.rules"
AUDIT_CONF_FILE="/etc/audit/auditd.conf"

# ---------------------------
# Helpers
# ---------------------------
backup_file() {
  local f="$1"
  if [[ -f "$f" ]]; then
    cp -av "$f" "${f}.bak.$(date +%F-%H%M%S)"
  fi
}

need_file() {
  local f="$1"
  if [[ ! -f "$f" ]]; then
    echo "[ERROR] Missing required file: $f"
    echo "       Run this installer from the repo root or ensure the file exists."
    exit 1
  fi
}

reload_audit_rules() {
  echo "[INFO] Loading audit rules..."
  if command -v augenrules &>/dev/null; then
    augenrules --load
  else
    # Fallback
    service auditd restart || true
  fi
  systemctl restart auditd
}

# ---------------------------
# 1. Create user (nologin) if not exists
# ---------------------------
if ! id "$FIM_USER" &>/dev/null; then
  echo "[INFO] Creating user $FIM_USER..."
  useradd -m -s /usr/sbin/nologin "$FIM_USER"
else
  echo "[INFO] User $FIM_USER already exists"
fi

# ---------------------------
# 2. Install packages
# ---------------------------
echo "[INFO] Installing packages..."
export DEBIAN_FRONTEND=noninteractive
apt-get update -y
apt-get install -y auditd audispd-plugins ca-certificates python3 python3-pip

systemctl enable auditd
systemctl restart auditd

# ---------------------------
# 3. Configure audit.rules (overwrite /etc/audit/rules.d/audit.rules)
# ---------------------------
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

chmod 640 "$AUDIT_RULES_FILE"
chown root:root "$AUDIT_RULES_FILE"

reload_audit_rules

# ---------------------------
# 4. Configure auditd.conf (OPTIONAL)
# NOTE: Overwriting this file can break on some systems if keys differ.
# Keep it only if you know your target OS versions support these keys.
# ---------------------------
backup_file "$AUDIT_CONF_FILE"

echo "[INFO] Updating auditd.conf..."
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

systemctl restart auditd

# ---------------------------
# 5. Create directories
# ---------------------------
echo "[INFO] Creating FIM directories..."
mkdir -p "$JSON_DIR"
chown -R "$FIM_USER:$FIM_USER" "$FIM_DIR" || true
chmod 750 "$FIM_DIR" || true
chmod 750 "$JSON_DIR" || true

# ---------------------------
# 6. Copy agent files to target location (repo remains intact)
# ---------------------------
need_file "$AGENT_SRC"
need_file "$UPLOADER_SRC"
need_file "$CONF_SRC"

echo "[INFO] Copying FIM agent files to $FIM_DIR..."
install -m 0644 -o root -g root "$AGENT_SRC" "$AGENT_DST"
install -m 0644 -o root -g root "$UPLOADER_SRC" "$UPLOADER_DST"
install -m 0600 -o root -g root "$CONF_SRC" "$CONF_DST"

# If your services run as root, root owning the dir is fine:
chown -R root:root "$FIM_DIR"
chmod 0755 "$FIM_DIR"
chmod 0755 "$JSON_DIR"

# ---------------------------
# 7. Setup Python virtual environment
# ---------------------------
echo "[INFO] Setting up Python virtual environment..."

# Find best python
PYTHON_BIN="python3"
for ver in 3.12 3.11 3.10 3.9 3.8; do
  if command -v "python${ver}" &>/dev/null; then
    PYTHON_BIN="python${ver}"
    break
  fi
done
echo "[INFO] Using Python interpreter: $PYTHON_BIN"

PYTHON_MAJOR_MINOR=$($PYTHON_BIN -c 'import sys; print(f"{sys.version_info.major}.{sys.version_info.minor}")')
VENV_PKG="python${PYTHON_MAJOR_MINOR}-venv"
echo "[INFO] Installing venv package: $VENV_PKG"
apt-get install -y "$VENV_PKG"

rm -rf "$VENV_DIR"
$PYTHON_BIN -m venv "$VENV_DIR"

# shellcheck disable=SC1091
source "$VENV_DIR/bin/activate"
pip install --upgrade pip
pip install boto3
pip install python-dotenv
deactivate

chown -R root:root "$VENV_DIR"

# ---------------------------
# 8. Create systemd services
# ---------------------------
echo "[INFO] Creating systemd services..."

cat > /etc/systemd/system/fim.service <<EOF
[Unit]
Description=File Integrity Monitoring Service
After=network.target auditd.service
Wants=auditd.service

[Service]
Type=simple
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

cat > /etc/systemd/system/data-uploader.service <<EOF
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
User=fimuser
Group=fimuser
NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload

# ---------------------------
# 9. Enable & start services
# ---------------------------
echo "[INFO] Enabling and starting services..."
systemctl enable --now fim.service
systemctl enable --now data-uploader.service

echo "[INFO] Showing service status:"
systemctl --no-pager --full status fim.service || true
systemctl --no-pager --full status data-uploader.service || true

echo "[INFO] Installation complete."
echo "[INFO] Logs: $LOG_FILE"
