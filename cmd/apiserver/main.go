package main

import (
	"crypto/tls"
	"fmt"
	"github.com/caarlos0/env"
	"github.com/justinas/nosurf"
	"github.com/rs/cors"
	"github.com/vardius/go-api-boilerplate/internal/userclient"
	"github.com/vardius/go-api-boilerplate/pkg/http/response"
	"github.com/vardius/go-api-boilerplate/pkg/jwt"
	"github.com/vardius/go-api-boilerplate/pkg/log"
	"github.com/vardius/go-api-boilerplate/pkg/recover"
	"github.com/vardius/go-api-boilerplate/pkg/security/authenticator"
	"github.com/vardius/go-api-boilerplate/pkg/socialmedia"
	"github.com/vardius/gorouter"
	"golang.org/x/crypto/acme/autocert"
	"net/http"
	"strconv"
	"time"
)

type config struct {
	Env            string   `env:"ENV"              envDefault:"development"`
	Host           string   `env:"HOST"             envDefault:"localhost"`
	Port           int      `env:"PORT"             envDefault:"443"`
	UserServerHost string   `env:"USER_SERVER_HOST"`
	UserServerPort int      `env:"USER_SERVER_PORT"`
	CertDirCache   string   `env:"CERT_DIR_CACHE"`
	Secret         string   `env:"SECRET"           envDefault:"secret"`
	Origins        []string `env:"ORIGINS"          envSeparator:"|"` // Origins should follow format: scheme "://" host [ ":" port ]
}

func setupServer(cfg *config, router gorouter.Router) *http.Server {
	certManager := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(cfg.Host),
		Cache:      autocert.DirCache(cfg.CertDirCache),
	}

	tlsConfig := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
		GetCertificate: certManager.GetCertificate,
	}

	srv := &http.Server{
		Addr:         ":" + strconv.Itoa(cfg.Port),
		Handler:      router,
		TLSConfig:    tlsConfig,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
	}

	return srv
}

func main() {
	cfg := config{}
	env.Parse(&cfg)

	logger := log.New(cfg.Env)
	rec := recover.WithLogger(recover.New(), logger)
	jwtService := jwt.New([]byte(cfg.Secret), time.Hour*24)
	auth := authenticator.WithToken(jwtService.Decode)
	userClient := userclient.New(fmt.Sprintf("%s:%d", cfg.UserServerHost, cfg.UserServerPort))

	// Global middleware
	router := gorouter.New(
		logger.LogRequest,
		cors.Default().Handler,
		nosurf.NewPure,
		response.WithXSS,
		response.WithHSTS,
		response.AsJSON,
		auth.FromHeader("API"),
		auth.FromQuery("authToken"),
		rec.RecoverHandler,
	)

	// Routes
	// Social media auth routes
	router.POST("/auth/google/callback", socialmedia.NewGoogle(userClient, jwtService))
	router.POST("/auth/facebook/callback", socialmedia.NewFacebook(userClient, jwtService))
	// User domain
	router.Mount("/users", userClient.AsRouter())

	srv := setupServer(&cfg, router)

	logger.Critical(nil, "%v\n", srv.ListenAndServeTLS("", ""))
}
