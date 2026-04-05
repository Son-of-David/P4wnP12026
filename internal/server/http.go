package server

import (
	"context"
	"encoding/json"
	"html/template"
	"io/fs"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"P4wnP12026/internal/bootdata"
	"P4wnP12026/internal/gps"
	"P4wnP12026/internal/kismet"
	"P4wnP12026/internal/ptyshell"
	"P4wnP12026/internal/webassets"
)

type Server struct {
	httpServer *http.Server
	templates  *template.Template
	kismetSvc  *kismet.Service
	gpsSvc     *gps.Service
	ptySvc     *ptyshell.Service
}

func New() *Server {
	t := template.Must(template.ParseFS(webassets.Templates, "templates/index.html"))
	staticFS, err := fs.Sub(webassets.Static, "static")
	if err != nil {
		log.Fatalf("web static assets: %v", err)
	}

	s := &Server{
		templates: t,
		kismetSvc: kismet.NewService(),
		gpsSvc:    gps.NewService(),
		ptySvc:    ptyshell.NewService(),
	}

	mux := http.NewServeMux()

	// Public UI + assets
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/kismet/status", s.handleKismetStatus)
	mux.HandleFunc("/kismet", s.handleKismetRedirect)

	// Restricted routes (only from directly connected Pi interface subnets)
	// Passing no iface names enables dynamic interface discovery.
	acl := s.ifaceOnly()
	mux.HandleFunc("/api/boot/usb_gadget", acl(s.handleBootUSBGadget))
	mux.HandleFunc("/api/boot/defaults", acl(s.handleBootDefaults))
	mux.HandleFunc("/api/boot/check", acl(s.handleBootCheck))
	mux.HandleFunc("/api/kismet/start", acl(s.handleKismetStart))
	mux.HandleFunc("/api/kismet/stop", acl(s.handleKismetStop))
	mux.HandleFunc("/api/gps/start", acl(s.handleGPSStart))
	mux.HandleFunc("/api/gps/stop", acl(s.handleGPSStop))
	mux.HandleFunc("/ws/shell", acl(s.ptySvc.HandleShell))
	mux.HandleFunc("/ws/airgeddon", acl(s.ptySvc.HandleAirgeddon))

	s.httpServer = &http.Server{
		Addr:         ":8001",
		Handler:      loggingMiddleware(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 0,
	}

	return s
}

func (s *Server) Addr() string {
	return s.httpServer.Addr
}

func (s *Server) Start() error {
	if err := ptyshell.EnsureSODDirExists(); err != nil {
		log.Printf("warning: failed to ensure /usr/share/S.o.D/ exists: %v", err)
	}
	return s.httpServer.ListenAndServe()
}

func (s *Server) handleIndex(w http.ResponseWriter, _ *http.Request) {
	_ = s.templates.ExecuteTemplate(w, "index.html", nil)
}

func (s *Server) handleKismetStart(w http.ResponseWriter, _ *http.Request) {
	if err := s.kismetSvc.Start(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"message": "startup.sh launched; check /usr/share/S.o.D/kismet/ for Kismet output",
	})
}

func (s *Server) handleKismetStop(w http.ResponseWriter, _ *http.Request) {
	if err := s.kismetSvc.Stop(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleKismetStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"running": s.kismetSvc.Status()})
}

func (s *Server) handleGPSStart(w http.ResponseWriter, _ *http.Request) {
	if err := s.gpsSvc.Start(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"message": "GPS start script launched",
	})
}

func (s *Server) handleGPSStop(w http.ResponseWriter, _ *http.Request) {
	if err := s.gpsSvc.Stop(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"message": "GPS stop script completed",
	})
}

func (s *Server) handleKismetRedirect(w http.ResponseWriter, r *http.Request) {
	host := r.Host
	if strings.Contains(host, ":") {
		var err error
		host, _, err = net.SplitHostPort(host)
		if err != nil {
			host = r.Host
		}
	}
	target := "http://" + host + ":2501"
	http.Redirect(w, r, target, http.StatusFound)
}

func (s *Server) handleBootUSBGadget(w http.ResponseWriter, _ *http.Request) {
	if err := bootdata.RunUSBGadget(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"message": "Device will reboot in 5 seconds.",
	})
}

func (s *Server) handleBootDefaults(w http.ResponseWriter, _ *http.Request) {
	if err := bootdata.RunDefaults(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"message": "Device will reboot in 5 seconds.",
	})
}

func (s *Server) handleBootCheck(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	out, err := bootdata.RunCheckOutput(ctx)
	payload := map[string]any{
		"ok":     err == nil,
		"output": out,
	}
	if err != nil {
		payload["error"] = err.Error()
	}
	writeJSON(w, http.StatusOK, payload)
}

func writeJSON(w http.ResponseWriter, code int, payload map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}

func (s *Server) ifaceOnly(allowedIfaces ...string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			host, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				writeJSON(w, http.StatusForbidden, map[string]any{"ok": false, "error": "invalid remote address"})
				return
			}
			ip := net.ParseIP(host)
			if ip == nil {
				writeJSON(w, http.StatusForbidden, map[string]any{"ok": false, "error": "invalid client ip"})
				return
			}
			if ip.IsLoopback() {
				next(w, r)
				return
			}
			if !ipAllowedOnIfaces(ip, allowedIfaces) {
				writeJSON(w, http.StatusForbidden, map[string]any{"ok": false, "error": "not on allowed subnet"})
				return
			}
			next(w, r)
		}
	}
}

func ipAllowedOnIfaces(clientIP net.IP, ifaces []string) bool {
	if len(ifaces) == 0 {
		detected, err := net.Interfaces()
		if err != nil {
			return false
		}
		for _, ifi := range detected {
			// Only consider active, non-loopback interfaces.
			if ifi.Flags&net.FlagUp == 0 || ifi.Flags&net.FlagLoopback != 0 {
				continue
			}
			if interfaceContainsClient(ifi, clientIP) {
				return true
			}
		}
		return false
	}

	for _, ifname := range ifaces {
		ifi, err := net.InterfaceByName(ifname)
		if err != nil {
			continue // interface missing -> ignore
		}
		if interfaceContainsClient(*ifi, clientIP) {
			return true
		}
	}

	return false
}

func interfaceContainsClient(ifi net.Interface, clientIP net.IP) bool {
	addrs, err := ifi.Addrs()
	if err != nil {
		return false
	}
	for _, a := range addrs {
		n, ok := a.(*net.IPNet)
		if !ok || n == nil || n.IP == nil || n.Mask == nil {
			continue
		}
		if n.IP.IsLoopback() {
			continue
		}
		// Match against interface subnet for either IPv4 or IPv6.
		if n.Contains(clientIP) {
			return true
		}
	}
	return false
}
