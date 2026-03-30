#!/usr/bin/env bash
set -euo pipefail

echo "[+] Checking management link..."
if ip link show usbeth >/dev/null 2>&1 && ip -br link show usbeth | grep -q "UP"; then
  echo "[+] usbeth detected; stopping AP services so wlan0 is free..."

  # Stop things that keep wlan0 busy (AP mode / DHCP / NM)
  systemctl stop P4wnP1 2>/dev/null || true
  #systemctl stop hostapd 2>/dev/null || true
  systemctl stop dnsmasq 2>/dev/null || true
  systemctl stop NetworkManager 2>/dev/null || true
  systemctl stop wpa_supplicant 2>/dev/null || true
  pkill dnsmasq hostapd wpa_supplicant 2>/dev/null || true

  echo "[+] Resetting wlan0..."
  rfkill unblock wifi || true
  ip addr flush dev wlan0 || true
  ip link set wlan0 down || true
  sleep 1
  ip link set wlan0 up || true
  sleep 1

  echo "[+] Removing any stale monitor interface..."
  iw dev wlan0mon del 2>/dev/null || true

  echo "[+] Creating wlan0mon..."
  # If this fails with busy, something is still holding wlan0
  iw dev wlan0 interface add wlan0mon type monitor
  ip link set wlan0mon up

  echo "[+] Interfaces:"
  iw dev || true
  ip -br link || true

  echo "[+] Launching: kismet -c wlan0mon"
  systemctl restart kismet

else
  echo "[!] usbeth not UP; leaving wlan0 alone. Falling back to wlan1."
  echo "[+] Launching: kismet -c wlan1"
  systemctl restart kismet1
fi
