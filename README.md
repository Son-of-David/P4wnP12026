### P4wnP1 2026 by S.o.D ###

![P4wnP12026](docs/images/P4wnP12026.png)
![P4wnP1 A.L.O.A Armhf](docs/images/P4wnP1%20A.L.O.A%20Armhf.png)
![Architecture](docs/images/Architecture.png)
![P4wnP1 status](docs/images/P4wnP1_status.png)
![Interfaces](docs/images/interfaces.png)

**P4wnP1 2026** is a Go web control panel aimed at Raspberry Pi Zero 2 W workflows that sit in the **P4wnP1 / A.L.O.A** tradition. It packages a small HTTP UI, service helpers, and interactive terminals so you can run authorized wireless lab and field tests from a browser—without treating rogue AP tricks, deauthentication, or hash capture as product goals (contrast with tools like Pwnagotchi that lean that way). Use it only where you have **explicit permission** and comply with local law; this project is for **education and authorized security testing**.

The stack assumes you understand the original P4wnP1 model: **USB Ethernet** to the Pi for stable management access, **monitor mode on `wlan0mon`** for capture-oriented tools, and **boot profiles** (`usb_gadget` vs `pi_defaults`) that pair with your choice of onboard vs external Wi‑Fi adapter. If you rely on Wi‑Fi for admin access while switching profiles or adapters, you can lock yourself out—a reboot from a known-good configuration usually restores service.

**GPS:** GPS tooling is integrated with the host’s `gpsd` setup; if you feed position over the network, a common pattern is a **UDP listener on port 9999** (adjust to match your `gpsd` / client configuration).

**Interactive sessions:** Airgeddon runs inside **tmux** over a web PTY; keyboard shortcuts and tmux prefix keys apply as they would in a normal terminal. A second shell session is rooted at `/usr/share/S.O.D/` for local scripts and tooling.

**Default credentials:** Images and profiles in this family often ship with **default Wi‑Fi and P4wnP1 passkeys**. Treat those as **insecure**—change them before any real deployment.

Stock behavior, hardware expectations, and upstream concepts are best understood from the official **P4wnP1** lineage. See: [RoganDawes/P4wnP1](https://github.com/RoganDawes/P4wnP1).

## Operations

- **Web UI:** listens on port **8001** (bind address depends on your Pi network setup).
- **Boot profile:** switch between **`usb_gadget`** and **`pi_defaults`** from the UI when you need gadget-style USB networking vs standard Pi defaults; confirm prompts and expect a **reboot** where the flow requires it.
- **Kismet:** start/stop from the UI; the helper redirects the browser to Kismet on port **2501** when appropriate.
- **Monitor mode:** Kismet and Airgeddon scripts expect **`wlan0mon`**; **USB Ethernet** is the supported path for stable control while using monitor mode (USB Ethernet is **not** supported on Android for this workflow).
- **GPS:** use the UI start/stop actions, which drive the bundled `gpsd` scripts under `gpsd/scripts/` on the device (`/root/P4wnP12026/...` when deployed as documented in the tree).
- **Terminals:** **Airgeddon** (tmux) and a general shell under **`/usr/share/S.O.D/`** via WebSocket PTYs.

## Credits / License

- **P4wnP1 / A.L.O.A** ecosystem and ideas trace to **MaMe82** and the broader **P4wnP1** community; this repository is a derivative control layer and filesystem mirror, not a replacement for upstream firmware documentation.
- Respect the **licenses** of bundled upstream assets under `vendor/`, `usr/`, and other third-party trees when redistributing.

## Build

Cross-compile for Raspberry Pi (ARMv7) with vendored modules:

```bash
go mod tidy
go mod vendor
env \
  GOOS=linux \
  GOARCH=arm \
  GOARM=7 \
  GOFLAGS=-mod=vendor \
  go build -o ./build/P4wnP12026 ./cmd/P4wnP12026
```

The HTML/CSS/JS UI is embedded in the binary (`internal/webassets`), so you can run the binary from any working directory; you do not need a separate `web/` folder beside it.

```bash
./build/P4wnP12026
```

## GitHub setup

If this directory is not yet a Git repository:

```bash
cd /path/to/P4wnP12026
git init
git add -A
git status   # review; vendor/ is intentionally tracked for a full mirror
git commit -m "Initial commit: P4wnP1 2026 control panel"
```

Create an empty repository on GitHub (no README/license templates if you want a clean first push), then:

```bash
git branch -M main
git remote add origin https://github.com/<you>/<repo>.git
git push -u origin main
```

Add screenshots to **`docs/images/`** using the filenames referenced at the top of this README so the image links resolve on GitHub.
