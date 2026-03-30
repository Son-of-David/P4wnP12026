#!/usr/bin/env bash
set -euo pipefail

BOOT_CFG="/boot/firmware/config.txt"
DEFAULT_CFG="/root/P4wnP12026/bootdata/defaults/config.txt"
USB_GADGET_CFG="/root/P4wnP12026/bootdata/usb_gadget/config.txt"

# Basic sanity checks
for f in "$BOOT_CFG" "$DEFAULT_CFG" "$USB_GADGET_CFG"; do
  if [[ ! -f "$f" ]]; then
    echo "Missing file: $f" >&2
    exit 2
  fi
done

# cmp returns 0 if files are identical, 1 if different, >1 on error
if cmp -s "$BOOT_CFG" "$DEFAULT_CFG"; then
  echo "Default Settings"
elif cmp -s "$BOOT_CFG" "$USB_GADGET_CFG"; then
  echo "Usb_Gadget Settings"
else
  echo "Custom/Unknown Settings"
  exit 1
fi
