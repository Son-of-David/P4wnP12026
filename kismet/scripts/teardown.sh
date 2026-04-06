#!/usr/bin/env bash
set -euo pipefail
echo "[+] Checking management link..."
if ip link show usbeth >/dev/null 2>&1 && ip -br link show usbeth | grep -q "UP"; then
  echo "[+] usbeth detected; stopping wlan0mon..."
# Kill any journalctl follow that might be attached to this terminal
  pkill -f 'journalctl.*-f.*P4wnP1' 2>/dev/null || true
  echo "[+] Stopping Kismet Services..."
  systemctl stop kismet
  systemctl stop kismet1
  echo "[+] Stopping Kismet..."
  pkill -f '^kismet$' 2>/dev/null || true
  pkill -f kismet_cap_linux_wifi 2>/dev/null || true

  echo "[+] Removing monitor interface (wlan0mon) if present..."
  ip link set wlan0mon down 2>/dev/null || true
  iw dev wlan0mon del 2>/dev/null || true

  echo "[+] Releasing wlan0.."
  systemctl stop wpa_supplicant 2>/dev/null || true
  systemctl stop NetworkManager 2>/dev/null || true

  echo "[+] Resetting wlan0..."
  rfkill unblock wifi || true
  ip addr flush dev wlan0 2>/dev/null || true
  ip link set wlan0 down 2>/dev/null || true
  iw dev wlan0 set type managed 2>/dev/null || true
  rfkill unblock wifi || true
  sleep 2

  echo "[+] Strong reset..."
  modprobe -r brcmfmac 2>/dev/null || true
  sleep 1
  modprobe brcmfmac 2>/dev/null || true
  sleep 2
  ip link set wlan0 up 2>/dev/null || true
  sleep 1

  echo "[+] Restarting networking + P4wnP1..."
  systemctl start dbus 2>/dev/null || true
  systemctl restart dnsmasq 2>/dev/null || true
  systemctl restart P4wnP1 2>/dev/null || true

  echo "[+] Quick health:"
  systemctl is-active P4wnP1 NetworkManager wpa_supplicant dnsmasq 2>/dev/null || true
  iw dev || true
  ip -br addr show usbeth wlan0 2>/dev/null || true

  echo "[+] Recent P4wnP1 logs (last 30 lines):"
  journalctl -u P4wnP1 -n 30 --no-pager || true

  echo "[+] Done."
else
  echo "[+] usbeth not detected; stopping kismet..."
  systemctl stop kismet1

