package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fabrixe/fabrixe/internal/api"
	"github.com/fabrixe/fabrixe/internal/config"
	"github.com/fabrixe/fabrixe/internal/db"
	"github.com/fabrixe/fabrixe/internal/mdns"
	tlsutil "github.com/fabrixe/fabrixe/internal/tls"
	"github.com/fabrixe/fabrixe/pkg/middleware"
)

const (
	banner = `
███████╗ █████╗ ██████╗ ██████╗ ██╗██╗  ██╗███████╗
██╔════╝██╔══██╗██╔══██╗██╔══██╗██║╚██╗██╔╝██╔════╝
█████╗  ███████║██████╔╝██████╔╝██║ ╚███╔╝ █████╗
██╔══╝  ██╔══██║██╔══██╗██╔══██╗██║ ██╔██╗ ██╔══╝
██║     ██║  ██║██████╔╝██║  ██║██║██╔╝ ██╗███████╗
╚═╝     ╚═╝  ╚═╝╚═════╝ ╚═╝  ╚═╝╚═╝╚═╝  ╚═╝╚══════╝

  Enterprise Infrastructure Management Platform
  Version 1.0.0 — https://fabrixe.local
`
)

func main() {
	configPath := flag.String("config", "/etc/fabrixe/config.yaml", "Path to configuration file")
	initMode := flag.Bool("init", false, "Initialize Fabrixe with default configuration")
	flag.Parse()

	fmt.Print(banner)

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize database
	database, err := db.New(cfg.Database.Path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: Failed to initialize database: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	if err := database.Migrate(); err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: Database migration failed: %v\n", err)
		os.Exit(1)
	}

	// Bootstrap first-run admin account
	if *initMode {
		if err := database.Bootstrap(); err != nil {
			fmt.Fprintf(os.Stderr, "FATAL: Bootstrap failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("[INFO] Fabrixe initialized. Default credentials: admin / FabrixeAdmin@2024")
		fmt.Println("[INFO] Change the admin password immediately after first login.")
	}

	// Ensure TLS certificates exist
	certDir := cfg.TLS.CertDir
	certFile := certDir + "/fabrixe.crt"
	keyFile := certDir + "/fabrixe.key"

	if err := os.MkdirAll(certDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: Cannot create cert directory: %v\n", err)
		os.Exit(1)
	}

	if _, err := os.Stat(certFile); os.IsNotExist(err) {
		fmt.Println("[INFO] Generating self-signed TLS certificate for fabrixe.local...")
		if err := tlsutil.GenerateSelfSigned(certFile, keyFile, cfg.TLS.Hostnames); err != nil {
			fmt.Fprintf(os.Stderr, "FATAL: TLS certificate generation failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("[INFO] TLS certificate generated.")
	}

	// Build router
	router := api.NewRouter(cfg, database)
	handler := middleware.Chain(
		router,
		middleware.RequestID,
		middleware.Logger,
		middleware.CORS(cfg.Security.AllowedOrigins),
		middleware.SecurityHeaders,
		middleware.RateLimit(cfg.Security.RateLimitPerMin),
	)

	// Configure HTTPS server
	tlsCfg := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		},
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      handler,
		TLSConfig:    tlsCfg,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start mDNS responder (fabrixe.local)
	mdnsServer, err := mdns.Advertise(cfg.MDNS.Hostname, cfg.Server.Port)
	if err != nil {
		fmt.Printf("[WARN] mDNS advertisement failed (fabrixe.local may not resolve): %v\n", err)
	} else {
		defer mdnsServer.Shutdown()
		fmt.Printf("[INFO] mDNS: advertised as %s.local\n", cfg.MDNS.Hostname)
	}

	// Start HTTPS server
	fmt.Printf("[INFO] Fabrixe starting on https://%s:%d\n", cfg.Server.Host, cfg.Server.Port)

	go func() {
		if err := srv.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "FATAL: Server error: %v\n", err)
			os.Exit(1)
		}
	}()

	// HTTP → HTTPS redirect
	if cfg.Server.HTTPRedirectPort > 0 {
		go func() {
			redirect := &http.Server{
				Addr:    fmt.Sprintf(":%d", cfg.Server.HTTPRedirectPort),
				Handler: http.HandlerFunc(httpsRedirect(cfg.Server.Port)),
			}
			_ = redirect.ListenAndServe()
		}()
	}

	fmt.Printf("[INFO] Dashboard: https://fabrixe.local\n")
	fmt.Printf("[INFO] Local IP:   https://%s:%d\n", cfg.Server.Host, cfg.Server.Port)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\n[INFO] Shutting down Fabrixe...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Server forced to shutdown: %v\n", err)
	}
	fmt.Println("[INFO] Fabrixe stopped.")
}

func httpsRedirect(port int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		target := fmt.Sprintf("https://%s:%d%s", r.Host, port, r.URL.RequestURI())
		http.Redirect(w, r, target, http.StatusMovedPermanently)
	}
}
