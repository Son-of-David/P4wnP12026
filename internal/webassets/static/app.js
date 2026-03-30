function wsURL(path) {
  const proto = location.protocol === "https:" ? "wss://" : "ws://";
  return proto + location.host + path;
}

function stripAnsiControls(text) {
  return text.replace(/\x1b\[[0-9;?]*[ -/]*[@-~]/g, "");
}

function scrollOutputIfNearBottom(el, thresholdPx = 96) {
  if (!el) return;
  const near =
    el.scrollHeight - el.scrollTop - el.clientHeight < thresholdPx;
  if (near) el.scrollTop = el.scrollHeight;
}

function createTerminalAdapter(el) {
  if (typeof window.Terminal === "function") {
    const term = new Terminal({
      cursorBlink: true,
      fontFamily: "monospace",
      fontSize: 14,
      convertEol: true,
      scrollback: 10000
    });
    term.open(el);
    return {
      focus: () => term.focus(),
      write: (data) => term.write(data),
      onData: (cb) => term.onData(cb),
      get cols() { return term.cols; },
      get rows() { return term.rows; }
    };
  }

  // Offline-safe fallback when CDN xterm.js is unavailable.
  const out = document.createElement("pre");
  out.className = "basic-term-output";
  const input = document.createElement("textarea");
  input.className = "basic-term-input";
  input.spellcheck = false;
  input.autocapitalize = "off";
  input.autocomplete = "off";
  input.autocorrect = "off";
  input.placeholder = "Terminal input";
  el.appendChild(out);
  el.appendChild(input);

  let dataHandler = null;
  input.addEventListener("keydown", (ev) => {
    if (!dataHandler || ev.key !== "Enter") return;
    const line = input.value || "";
    dataHandler(line + "\r");
    input.value = "";
    ev.preventDefault();
  });

  return {
    focus: () => input.focus(),
    write: (data) => {
      if (typeof data === "string") {
        out.textContent += stripAnsiControls(data);
      } else {
        out.textContent += stripAnsiControls(new TextDecoder().decode(data));
      }
      scrollOutputIfNearBottom(out);
    },
    onData: (cb) => { dataHandler = cb; },
    get cols() { return 120; },
    get rows() { return 40; }
  };
}

function setupTerminal(elementId, endpoint) {
  const el = document.getElementById(elementId);
  const term = createTerminalAdapter(el);
  term.focus();
  // Mobile browsers often need explicit user gestures to re-focus xterm's hidden textarea.
  const focusTerm = () => term.focus();
  el.addEventListener("click", focusTerm);
  el.addEventListener("touchstart", focusTerm, { passive: true });

  const ws = new WebSocket(wsURL(endpoint));
  ws.binaryType = "arraybuffer";

  ws.onopen = () => {
    sendResize();
  };

  ws.onmessage = (ev) => {
    if (typeof ev.data === "string") {
      term.write(ev.data);
      return;
    }
    term.write(new Uint8Array(ev.data));
  };

  ws.onclose = () => term.write("\r\n[session closed]\r\n");
  ws.onerror = () => term.write("\r\n[websocket error]\r\n");

  term.onData((data) => ws.readyState === 1 && ws.send(data));

  function sendResize() {
    if (ws.readyState !== 1) return;
    ws.send(JSON.stringify({
      type: "resize",
      cols: term.cols,
      rows: term.rows
    }));
  }

  window.addEventListener("resize", sendResize);
  return {
    term,
    send(data) {
      if (ws.readyState === 1) {
        ws.send(data);
        return true;
      }
      return false;
    },
    isOpen() {
      return ws.readyState === 1;
    }
  };
}

async function postJSON(url, body) {
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body || {})
  });
  return res.json();
}

const BOOT_FIRMWARE_CONFIRM_MSG =
  "This device will reboot in 5 seconds after a boot firmware switch. Press okay to confirm.";

const BOOT_REBOOT_ACK_MSG = "Device will reboot in 5 seconds.";

let bootCheckOutputTimer = null;

function modeMessage(msg, ok) {
  const out = document.getElementById("modeResult");
  out.textContent = msg;
  out.style.color = ok ? "#006400" : "#a00000";
}

function showBootCheckOutput(text, ok) {
  const pre = document.getElementById("bootCheckOutput");
  if (!pre) return;
  if (bootCheckOutputTimer) {
    clearTimeout(bootCheckOutputTimer);
    bootCheckOutputTimer = null;
  }
  const body = text && String(text).trim() !== "" ? String(text) : "(no output)";
  pre.textContent = body;
  pre.hidden = false;
  pre.style.borderLeft = `6px solid ${ok ? "#006400" : "#a00000"}`;
  bootCheckOutputTimer = setTimeout(() => {
    pre.textContent = "";
    pre.hidden = true;
    bootCheckOutputTimer = null;
  }, 10000);
}

function kismetMessage(msg, ok) {
  const out = document.getElementById("kismetStatus");
  out.textContent = msg;
  out.style.color = ok ? "#006400" : "#a00000";
}

function airgeddonMessage(msg, ok) {
  const out = document.getElementById("airgeddonStatus");
  out.textContent = msg;
  out.style.color = ok ? "#006400" : "#a00000";
}

function gpsMessage(msg, ok) {
  const out = document.getElementById("gpsStatus");
  out.textContent = msg;
  out.style.color = ok ? "#006400" : "#a00000";
}

function openAirgeddonTerminalWindow({ runLaunch = false } = {}) {
  const win = window.open("", "_blank");
  if (!win) {
    airgeddonMessage("Popup blocked. Allow popups for this site.", false);
    return false;
  }
  const launchFlag = runLaunch ? "true" : "false";
  win.document.write(`<!doctype html>
<html><head><title>Airgeddon Terminal</title>
<link rel="stylesheet" href="/static/vendor/xterm.css">
<style>html,body{height:100%;margin:0;background:#000}#term{display:flex;flex-direction:column;height:100%;min-height:0}#term .xterm-viewport{overflow-y:auto !important;-webkit-overflow-scrolling:touch;touch-action:pan-y;overscroll-behavior:contain}</style>
</head><body><div id="term"></div>
<script src="/static/vendor/xterm.js"><\/script>
<script>
  function createTerm(el) {
    if (typeof window.Terminal === "function") {
      const term = new Terminal({cursorBlink:true,fontFamily:"monospace",fontSize:14,scrollback:10000,convertEol:true});
      term.open(el);
      return { term, focus: () => term.focus(), write: (d) => term.write(d), onData: (cb) => term.onData(cb), cols: () => term.cols, rows: () => term.rows };
    }
    const out = document.createElement("pre");
    out.style.cssText = "flex:1;min-height:0;margin:0;overflow:auto;-webkit-overflow-scrolling:touch;touch-action:pan-y;overscroll-behavior:contain;color:#fff;background:#000;padding:8px;font:14px monospace;white-space:pre-wrap;";
    const input = document.createElement("textarea");
    input.style.cssText = "flex:0 0 15%;min-height:48px;width:100%;box-sizing:border-box;background:#111;color:#fff;border:0;outline:none;padding:8px;font:14px monospace;";
    input.placeholder = "Terminal input";
    el.appendChild(out);
    el.appendChild(input);
    let cb = null;
    function stripAnsi(s) {
      return s.replace(/\\x1b\\[[0-9;?]*[ -/]*[@-~]/g, "");
    }
    input.addEventListener("keydown", (ev) => {
      if (!cb || ev.key !== "Enter") return;
      cb((input.value || "") + "\\r");
      input.value = "";
      ev.preventDefault();
    });
    return {
      focus: () => input.focus(),
      write: (d) => {
        out.textContent += stripAnsi(typeof d === "string" ? d : new TextDecoder().decode(d));
        const th = 96;
        if (out.scrollHeight - out.scrollTop - out.clientHeight < th) out.scrollTop = out.scrollHeight;
      },
      onData: (next) => { cb = next; },
      cols: () => 120,
      rows: () => 40
    };
  }
  const t = createTerm(document.getElementById("term"));
  const focusTerm = () => t.focus();
  document.getElementById("term").addEventListener("click", focusTerm);
  document.getElementById("term").addEventListener("touchstart", focusTerm, { passive: true });
  focusTerm();
  const proto = location.protocol==="https:" ? "wss://" : "ws://";
  const ws = new WebSocket(proto + location.host + "/ws/airgeddon");
  ws.binaryType = "arraybuffer";
  ws.onmessage = (ev) => {
    if (typeof ev.data === "string") return t.write(ev.data);
    t.write(new Uint8Array(ev.data));
  };
  ws.onclose = () => t.write("\\r\\n[session closed]\\r\\n");
  ws.onerror = () => t.write("\\r\\n[websocket error]\\r\\n");
  t.onData((d)=> ws.readyState===1 && ws.send(d));
  function sendResize() {
    if (ws.readyState!==1) return;
    ws.send(JSON.stringify({type:"resize",cols:t.cols(),rows:t.rows()}));
  }
  ws.onopen = () => {
    sendResize();
    if (${launchFlag}) {
      // Run launch script only after this terminal is connected and ready.
      ws.send("bash /root/P4wnP12026/airgeddon/scripts/launch.sh\\r");
    }
  };
  window.addEventListener("resize", sendResize);
<\/script></body></html>`);
  win.document.close();
  return true;
}

function runShellCommand(command, { background = false } = {}) {
  return new Promise((resolve, reject) => {
    const ws = new WebSocket(wsURL("/ws/shell"));
    let opened = false;
    ws.onopen = () => {
      opened = true;
      const cmd = background
        ? `setsid nohup bash ${command} </dev/null >/tmp/P4wnP12026-shell.log 2>&1 &\n`
        : `bash ${command}\n`;
      ws.send(cmd);
      setTimeout(() => {
        ws.close();
        resolve();
      }, 1200);
    };
    ws.onerror = () => reject(new Error("failed to connect to shell websocket"));
    ws.onclose = () => {
      if (!opened) reject(new Error("shell websocket closed before command dispatch"));
    };
  });
}

document.getElementById("bootUsbGadget").addEventListener("click", async () => {
  if (!window.confirm(BOOT_FIRMWARE_CONFIRM_MSG)) {
    modeMessage("Cancelled.", false);
    return;
  }
  try {
    const result = await postJSON("/api/boot/usb_gadget");
    if (result.ok) {
      modeMessage(result.message || BOOT_REBOOT_ACK_MSG, true);
    } else {
      modeMessage(result.error || "Request failed.", false);
    }
  } catch (err) {
    modeMessage(String(err), false);
  }
});

document.getElementById("bootDefaults").addEventListener("click", async () => {
  if (!window.confirm(BOOT_FIRMWARE_CONFIRM_MSG)) {
    modeMessage("Cancelled.", false);
    return;
  }
  try {
    const result = await postJSON("/api/boot/defaults");
    if (result.ok) {
      modeMessage(result.message || BOOT_REBOOT_ACK_MSG, true);
    } else {
      modeMessage(result.error || "Request failed.", false);
    }
  } catch (err) {
    modeMessage(String(err), false);
  }
});

document.getElementById("bootCheck").addEventListener("click", async () => {
  modeMessage("Running check (up to 10 seconds)…", true);
  try {
    const result = await postJSON("/api/boot/check");
    const ok = !!result.ok;
    if (result.error && !ok) {
      modeMessage(result.error, false);
    } else {
      modeMessage(ok ? "Check completed." : (result.error || "Check finished with errors."), ok);
    }
    showBootCheckOutput(result.output != null ? result.output : "", ok);
  } catch (err) {
    modeMessage(String(err), false);
    if (bootCheckOutputTimer) {
      clearTimeout(bootCheckOutputTimer);
      bootCheckOutputTimer = null;
    }
    const pre = document.getElementById("bootCheckOutput");
    if (pre) {
      pre.textContent = "";
      pre.hidden = true;
    }
  }
});

document.getElementById("kismetStart").addEventListener("click", async () => {
  const result = await postJSON("/api/kismet/start");
  kismetMessage(
    result.ok ? (result.message || "Kismet startup script launched.") : result.error,
    !!result.ok
  );
});

document.getElementById("kismetStop").addEventListener("click", async () => {
  const result = await postJSON("/api/kismet/stop");
  kismetMessage(result.ok ? "Kismet stopped." : result.error, !!result.ok);
});

document.getElementById("openKismet").addEventListener("click", () => {
  window.location.href = "/kismet";
});

document.getElementById("airgeddonStart").addEventListener("click", async () => {
  try {
    const opened = openAirgeddonTerminalWindow({ runLaunch: true });
    if (!opened) {
      throw new Error("Unable to open Airgeddon terminal window.");
    }
    airgeddonMessage("Opening Airgeddon terminal window and launching script there...", true);
  } catch (err) {
    airgeddonMessage(String(err), false);
  }
});

document.getElementById("airgeddonStop").addEventListener("click", async () => {
  try {
    // Run teardown in a separate detached process, not in the tmux/airgeddon PTY.
    await runShellCommand("/root/P4wnP12026/airgeddon/scripts/teardown.sh", { background: true });
    airgeddonMessage("Airgeddon teardown launched in separate process.", true);
  } catch (err) {
    airgeddonMessage(String(err), false);
  }
});

document.getElementById("openAirgeddon478").addEventListener("click", () => {
  openAirgeddonTerminalWindow({ runLaunch: false });
});

document.getElementById("gpsStart").addEventListener("click", async () => {
  try {
    const result = await postJSON("/api/gps/start");
    gpsMessage(result.ok ? (result.message || "GPS start script launched.") : result.error, !!result.ok);
  } catch (err) {
    gpsMessage(String(err), false);
  }
});

document.getElementById("gpsStop").addEventListener("click", async () => {
  try {
    const result = await postJSON("/api/gps/stop");
    gpsMessage(result.ok ? (result.message || "GPS stop script completed.") : result.error, !!result.ok);
  } catch (err) {
    gpsMessage(String(err), false);
  }
});

document.getElementById("gpsView").addEventListener("click", () => {
  const win = window.open("", "_blank");
  if (!win) {
    gpsMessage("Popup blocked. Allow popups for this site.", false);
    return;
  }
  win.document.write(`<!doctype html>
<html><head><title>GPS View Terminal</title>
<link rel="stylesheet" href="/static/vendor/xterm.css">
<style>html,body{height:100%;margin:0;background:#000}#term{display:flex;flex-direction:column;height:100%;min-height:0}#term .xterm-viewport{overflow-y:auto !important;-webkit-overflow-scrolling:touch;touch-action:pan-y;overscroll-behavior:contain}</style>
</head><body><div id="term"></div>
<script src="/static/vendor/xterm.js"><\/script>
<script>
  const term = new Terminal({cursorBlink:true,fontFamily:"monospace",fontSize:14,scrollback:10000,convertEol:true});
  term.open(document.getElementById("term"));
  term.focus();
  const proto = location.protocol==="https:" ? "wss://" : "ws://";
  const ws = new WebSocket(proto + location.host + "/ws/shell");
  ws.binaryType = "arraybuffer";
  ws.onmessage = (ev) => {
    if (typeof ev.data === "string") return term.write(ev.data);
    term.write(new Uint8Array(ev.data));
  };
  ws.onclose = () => term.write("\\r\\n[session closed]\\r\\n");
  ws.onerror = () => term.write("\\r\\n[websocket error]\\r\\n");
  term.onData((d)=> ws.readyState===1 && ws.send(d));
  function sendResize() {
    if (ws.readyState!==1) return;
    ws.send(JSON.stringify({type:"resize",cols:term.cols,rows:term.rows}));
  }
  ws.onopen = () => {
    sendResize();
    ws.send("bash /root/P4wnP12026/gpsd/scripts/view.sh\\r");
  };
  window.addEventListener("resize", sendResize);
<\/script></body></html>`);
  win.document.close();
  gpsMessage("GPS view terminal opened. Script output should show port 12478 details there.", true);
});

async function refreshKismet() {
  try {
    const res = await fetch("/api/kismet/status");
    const data = await res.json();
    kismetMessage(data.running ? "Kismet is running." : "Kismet is stopped.", true);
  } catch (_) {
    kismetMessage("Unable to query Kismet status.", false);
  }
}

const shellCtl = setupTerminal("shellTerm", "/ws/shell");
void shellCtl;
refreshKismet();
