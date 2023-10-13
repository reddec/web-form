package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/reddec/web-form/internal/assets"
	"github.com/reddec/web-form/internal/engine"
	"github.com/reddec/web-form/internal/notifications/webhook"
	"github.com/reddec/web-form/internal/schema"
	"github.com/reddec/web-form/internal/storage"

	_ "modernc.org/sqlite"

	"github.com/alexedwards/scs/redisstore"
	"github.com/alexedwards/scs/v2"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-chi/chi/v5"
	"github.com/gomodule/redigo/redis"
	"github.com/hashicorp/go-multierror"
	"github.com/jessevdk/go-flags"
	oidclogin "github.com/reddec/oidc-login"
)

//nolint:gochecknoglobals
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	builtBy = "unknown"
)

const (
	description = "Self-hosted Web Forms"
	name        = "web-forms"
)

type Config struct {
	Configs        string `long:"configs" env:"CONFIGS" description:"File or directory with YAML configurations" default:"configs"`
	Storage        string `long:"storage" env:"STORAGE" description:"Storage type" default:"database" choice:"database" choice:"files"`
	DisableListing bool   `long:"disable-listing" env:"DISABLE_LISTING" description:"Disable listing in UI"`
	DB             struct {
		Dialect    string `long:"dialect" env:"DIALECT" description:"SQL dialect" default:"sqlite3" choice:"postgres" choice:"sqlite3"`
		URL        string `long:"url" env:"URL" description:"Database URL" default:"file://form.sqlite"`
		Migrations string `long:"migrations" env:"MIGRATIONS" description:"Migrations dir" default:"migrations"`
		Migrate    bool   `long:"migrate" env:"MIGRATE" description:"Apply migration on start"`
	} `group:"Database storage" namespace:"db" env-namespace:"DB"`
	Files struct {
		Path string `long:"path" env:"PATH" description:"Root dir for form results" default:"results"`
	} `group:"Files storage" namespace:"files" env-namespace:"FILES"`
	Webhooks struct {
		Buffer int `long:"buffer" env:"BUFFER" description:"Buffer size before processing" default:"100"`
	} `group:"Webhooks general configuration" namespace:"webhooks" env-namespace:"WEBHOOKS"`
	HTTP struct {
		Assets       string        `long:"assets" env:"ASSETS" description:"Directory for assets (static) files"`
		Bind         string        `long:"bind" env:"BIND" description:"Binding address" default:":8080"`
		DisableXSRF  bool          `long:"disable-xsrf" env:"DISABLE_XSRF" description:"Disable XSRF validation. Useful for API"`
		TLS          bool          `long:"tls" env:"TLS" description:"Enable TLS"`
		Key          string        `long:"key" env:"KEY" description:"Private TLS key" default:"server.key"`
		Cert         string        `long:"cert" env:"CERT" description:"Public TLS certificate" default:"server.crt"`
		ReadTimeout  time.Duration `long:"read-timeout" env:"READ_TIMEOUT" description:"Read timeout to prevent slow client attack" default:"5s"`
		WriteTimeout time.Duration `long:"write-timeout" env:"WRITE_TIMEOUT" description:"Write timeout to prevent slow consuming clients attack" default:"5s"`
	} `group:"HTTP server configuration" namespace:"http" env-namespace:"HTTP"`
	OIDC struct {
		Enable              bool   `long:"enable" env:"ENABLE" description:"Enable OIDC protection"`
		ClientID            string `long:"client-id" env:"CLIENT_ID" description:"OIDC client ID"`
		ClientSecret        string `long:"client-secret" env:"CLIENT_SECRET" description:"OIDC client secret"`
		Issuer              string `long:"issuer" env:"ISSUER" description:"Issuer URL (without .well-known)"`
		RedisURL            string `long:"redis-url" env:"REDIS_URL" description:"Optional Redis URL for sessions. If not set - in-memory will be used"`
		RedisIdle           int    `long:"redis-idle" env:"REDIS_IDLE" description:"Redis maximum number of idle connections" default:"1"`
		RedisMaxConnections int    `long:"redis-max-connections" env:"REDIS_MAX_CONNECTIONS" description:"Redis maximum number of active connections" default:"10"`
	} `group:"OIDC configuration" namespace:"oidc" env-namespace:"OIDC"`
	ServerURL string `long:"server-url" env:"SERVER_URL" description:"Server public URL. Used for OIDC redirects. If not set - it will try to deduct"`
}

func main() {
	var config Config
	parser := flags.NewParser(&config, flags.Default)
	parser.ShortDescription = name
	parser.LongDescription = fmt.Sprintf("%s \n%s %s, commit %s, built at %s by %s\nAuthor: reddec <owner@reddec.net>", description, name, version, commit, date, builtBy)
	_, err := parser.Parse()
	if err != nil {
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	if err := run(ctx, config); err != nil {
		slog.Error("run failed", "error", err)
		os.Exit(2) //nolint:gocritic
	}
}

func run(ctx context.Context, config Config) error {
	router := chi.NewRouter()

	// mock auth by default
	var authMiddleware = func(next http.Handler) http.Handler {
		return next
	}

	if config.OIDC.Enable {
		// setup auth provider from OIDC
		slog.Info("oidc enabled", "issuer", config.OIDC.Issuer)
		sessionManager := scs.New() // by default in-memory session store
		router.Use(sessionManager.LoadAndSave)

		if config.OIDC.RedisURL != "" {
			// setup redis pool for sessions
			slog.Info("oidc redis session storage enabled")
			redisPool := &redis.Pool{
				Dial: func() (redis.Conn, error) {
					return redis.DialURLContext(ctx, config.OIDC.RedisURL)
				},
				MaxIdle:     config.OIDC.RedisIdle,
				MaxActive:   config.OIDC.RedisMaxConnections,
				IdleTimeout: time.Hour,
			}
			defer redisPool.Close()
			sessionManager.Store = redisstore.New(redisPool)
		} else {
			slog.Info("oidc session storage in-memory")
		}

		auth, err := config.createAuth(ctx, sessionManager)
		if err != nil {
			return fmt.Errorf("create auth: %w", err)
		}
		authMiddleware = auth.Secure
		router.Mount(oidclogin.Prefix, auth)
	} else {
		slog.Info("no authorization used")
	}

	// static dir and user-defined asset dir are unprotected
	router.Mount("/static/", http.FileServer(http.FS(assets.Static)))
	if config.HTTP.Assets != "" {
		slog.Info("user-defined assets enabled", "assets-dir", config.HTTP.Assets)
		router.Mount("/assets/", http.StripPrefix("/assets", http.FileServer(http.Dir(config.HTTP.Assets))))
	}

	// scan forms from file system
	forms, err := schema.FormsFromFS(os.DirFS(config.Configs))
	if err != nil {
		return fmt.Errorf("read configs in %q: %w", config.Configs, err)
	}

	// create results storage
	store, err := config.createStorage(ctx)
	if err != nil {
		return fmt.Errorf("create storage: %w", err)
	}
	defer store.Close()
	slog.Info("storage prepared")

	// webhooks dispatcher
	webhooks := webhook.New(config.Webhooks.Buffer)

	srv, err := engine.New(engine.Config{
		Forms:           forms,
		Storage:         store,
		WebhooksFactory: webhooks,
		Listing:         !config.DisableListing,
	},
		engine.WithXSRF(!config.HTTP.DisableXSRF),
	)
	if err != nil {
		return fmt.Errorf("create engine: %w", err)
	}

	router.Group(func(r chi.Router) {
		r.Use(owasp)
		r.Use(authMiddleware)
		r.Use(func(handler http.Handler) http.Handler {
			return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				creds := credentialsFromRequest(request)
				reqCtx := schema.WithCredentials(request.Context(), creds)
				handler.ServeHTTP(writer, request.WithContext(reqCtx))
			})
		})
		r.Mount("/", srv)
	})

	server := &http.Server{
		Addr:         config.HTTP.Bind,
		Handler:      router,
		ReadTimeout:  config.HTTP.ReadTimeout,
		WriteTimeout: config.HTTP.WriteTimeout,
		BaseContext: func(listener net.Listener) context.Context {
			return ctx
		},
	}

	var wg multierror.Group

	wg.Go(func() error {
		webhooks.Run(ctx)
		return nil
	})

	wg.Go(func() error {
		var err error
		if config.HTTP.TLS {
			err = server.ListenAndServeTLS(config.HTTP.Cert, config.HTTP.Key)
		} else {
			err = server.ListenAndServe()
		}
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		return err
	})

	wg.Go(func() error {
		<-ctx.Done()
		return server.Close()
	})

	slog.Info("ready", "bind", config.HTTP.Bind, "storage", config.Storage)

	return wg.Wait().ErrorOrNil()
}

func (cfg *Config) createStorage(ctx context.Context) (storage.ClosableStorage, error) {
	switch cfg.Storage {
	case "files":
		return storage.NopCloser(storage.NewFileStore(cfg.Files.Path)), nil
	case "database":
		db, err := storage.NewDB(ctx, cfg.DB.Dialect, cfg.DB.URL)
		if err != nil {
			return nil, fmt.Errorf("create storag: %w", err)
		}
		if cfg.shouldMigrate() {
			slog.Info("migrating database")
			return db, db.Migrate(ctx, cfg.DB.Migrations)
		}
		slog.Info("migration skipped")
		return db, nil
	default:
		return nil, fmt.Errorf("unknown storage type %q", cfg.Storage)
	}
}

func (cfg *Config) createAuth(ctx context.Context, sessionManager *scs.SessionManager) (service *oidclogin.OIDC, err error) {
	return oidclogin.New(ctx, oidclogin.Config{
		IssuerURL:      cfg.OIDC.Issuer,
		ClientID:       cfg.OIDC.ClientID,
		ClientSecret:   cfg.OIDC.ClientSecret,
		ServerURL:      cfg.ServerURL,
		Scopes:         []string{oidc.ScopeOpenID, "profile"},
		SessionManager: sessionManager,
		BeforeAuth: func(writer http.ResponseWriter, req *http.Request) error {
			sessionManager.Put(req.Context(), "redirect-to", req.URL.String())
			return nil
		},
		PostAuth: func(writer http.ResponseWriter, req *http.Request, idToken *oidc.IDToken) error {
			to := sessionManager.PopString(req.Context(), "redirect-to")
			if to != "" {
				writer.Header().Set("Location", to)
			}
			return nil
		},
		Logger: &LoggerFunc{},
	})
}

func (cfg *Config) shouldMigrate() bool {
	if !cfg.DB.Migrate {
		return false
	}
	var migrate = false
	err := filepath.WalkDir(cfg.DB.Migrations, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() || err != nil {
			return err
		}
		if filepath.Ext(path) == ".sql" {
			migrate = true
			return filepath.SkipAll
		}
		return nil
	})
	return migrate && err == nil
}

type LoggerFunc struct{}

func (lf LoggerFunc) Log(level oidclogin.Level, message string) {
	switch level {
	case oidclogin.LogInfo:
		slog.Info(message, "source", "oidc")
	case oidclogin.LogWarn:
		slog.Warn(message, "source", "oidc")
	case oidclogin.LogError:
		slog.Error(message, "source", "oidc")
	default:
		slog.Debug(message, "source", "oidc")
	}
}

func owasp(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("X-Frame-Options", "DENY")
		writer.Header().Set("X-XSS-Protection", "1")
		writer.Header().Set("X-Content-Type-Options", "nosniff")
		writer.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(writer, request)
	})
}

func credentialsFromRequest(req *http.Request) *schema.Credentials {
	token := oidclogin.Token(req)
	if token == nil {
		return nil
	}
	// optimized version to avoid multiple unmarshalling
	var claims struct {
		Username string   `json:"preferred_username"` //nolint:tagliatelle
		Email    string   `json:"email"`
		Groups   []string `json:"groups"`
	}
	_ = token.Claims(&claims)
	// workaround for username
	claims.Username = firstOf(claims.Username, claims.Email, token.Subject)
	return &schema.Credentials{
		User:   claims.Username,
		Groups: claims.Groups,
		Email:  claims.Email,
	}
}

func firstOf[T comparable](values ...T) T {
	var def T
	for _, v := range values {
		if v != def {
			return v
		}
	}
	return def
}
