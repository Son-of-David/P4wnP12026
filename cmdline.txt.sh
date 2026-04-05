#!/usr/bin/env bash
set -euo pipefail

SRC="/boot/firmware/cmdline.txt"
DEST_DIR="/root/P4wnP12026/bootdata/usb_gadget/cmdline.txt
BACKUP="/root/P4wnP12026/bootdata/defaults/cmdline.txt"

if [[ $EUID -ne 0 ]]; then
  echo "Run as root: sudo $0"
  exit 1
fi

if [[ ! -f "$SRC" ]]; then
  echo "ERROR: $SRC not found."
  exit 1
fi

mkdir -p "$DEST_DIR"
cp -a "$SRC" "$BACKUP"
echo "Backup written to: $BACKUP"

# Read cmdline as a single line (strip newlines just in case)
CMDLINE="$(tr -d '\n' < "$SRC" | sed -E 's/[[:space:]]+/ /g; s/^ //; s/ $//')"

# Sanity checks: must contain root=PARTUUID and rootwait
if ! grep -qE '(^| )root=PARTUUID=[0-9a-fA-F-]+-[0-9]+( |$)' <<<"$CMDLINE"; then
  echo "ERROR: cmdline does not contain root=PARTUUID=...-N"
  echo "Current cmdline:"
  echo "$CMDLINE"
  exit 1
fi

if ! grep -qE '(^| )rootwait( |$)' <<<"$CMDLINE"; then
  echo "ERROR: cmdline does not contain rootwait; refusing to edit."
  echo "Current cmdline:"
  echo "$CMDLINE"
  exit 1
fi

# If modules-load already exists, keep as-is (but ensure dwc2 is included)
if grep -qE '(^| )modules-load=' <<<"$CMDLINE"; then
  if grep -qE '(^| )modules-load=[^ ]*\bdwc2\b' <<<"$CMDLINE"; then
    echo "modules-load already includes dwc2; no change needed."
    exit 0
  else
    # append dwc2 to existing modules-load= list (comma-separated)
    NEW="$(sed -E 's/(^| )modules-load=([^ ]*)/\1modules-load=\2,dwc2/' <<<"$CMDLINE")"
  fi
else
  # Insert "modules-load=dwc2" right after rootwait
  NEW="$(sed -E 's/(^| )rootwait( |$)/\1rootwait modules-load=dwc2\2/' <<<"$CMDLINE")"
fi

# Final normalize spacing
NEW="$(sed -E 's/[[:space:]]+/ /g; s/^ //; s/ $//' <<<"$NEW")"

echo "Writing updated cmdline to $SRC:"
echo "$NEW"
printf "%s\n" "$NEW" > "$SRC"

echo "Done."
echo "Reboot to apply: reboot"
