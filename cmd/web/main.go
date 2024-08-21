package main

import (
	"Portfolio/internal/data"
	"Portfolio/internal/mailer"
	"database/sql"
	"flag"
	"fmt"
	"github.com/alexedwards/scs/postgresstore"
	"github.com/alexedwards/scs/v2"
	"github.com/go-playground/form/v4"
	_ "github.com/lib/pq"
	"log/slog"
	"os"
	"sync"
	"time"
)

func main() {

	// setting the configuration variables
	var cfg config

	// generic variables
	flag.Int64Var(&cfg.port, "port", 4000, "HTTP service address")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")

	// PostgreSQL variables
	flag.StringVar(&cfg.db.dsn, "dsn", "", "PostgreSQL Database DSN")

	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
	flag.DurationVar(&cfg.db.maxIdleTime, "db-max-idle-time", 15*time.Minute, "PostgreSQL max connection idle time")

	// SMTP variables
	flag.StringVar(&cfg.smtp.host, "smtp-host", "", "SMTP host")
	flag.Int64Var(&cfg.smtp.port, "smtp-port", 2525, "SMTP port")
	flag.StringVar(&cfg.smtp.username, "smtp-username", "", "SMTP username")
	flag.StringVar(&cfg.smtp.password, "smtp-password", "", "SMTP password")
	flag.StringVar(&cfg.smtp.sender, "smtp-sender", "Antoine's Portfolio <no-reply@adebarbarin.com", "SMTP sender")

	// cleaning frequency
	frequency := flag.Duration("frequency", time.Hour*2, "expired tokens and unactivated users cleaning frequency")

	flag.Parse()

	// setting the logging level according to the environment
	var opts *slog.HandlerOptions

	if cfg.env == "development" {
		opts = &slog.HandlerOptions{Level: slog.LevelDebug}
	} else {
		opts = &slog.HandlerOptions{Level: slog.LevelInfo}
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, opts))

	// checking the SMTP info
	if cfg.smtp.username == "" || cfg.smtp.password == "" || cfg.smtp.host == "" {
		fmt.Println("SMTP credentials are required")
		os.Exit(1)
	}

	// checking the dsn info
	if cfg.db.dsn == "" {
		logger.Error("dsn is required")
		os.Exit(1)
	}

	// connecting to the database
	db, err := openDB(cfg.db.dsn)
	if err != nil {
		logger.Error(fmt.Errorf("openDB error: %w", err).Error())
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("DB pool connection opened successfully!")

	// caching the templates
	templateCache, err := newTemplateCache()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	// initializing the application components
	formDecoder := form.NewDecoder()

	sessionManager := scs.New()
	sessionManager.Store = postgresstore.New(db)
	sessionManager.Lifetime = 24 * time.Hour
	sessionManager.Cookie.Secure = true

	app := &application{
		logger:         logger,
		mailer:         mailer.New(cfg.smtp.host, int(cfg.smtp.port), cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender),
		sessionManager: sessionManager,
		templateCache:  templateCache,
		formDecoder:    formDecoder,
		config:         &cfg,
		models:         data.NewModels(db),
		wg:             new(sync.WaitGroup),
	}

	// Clean expired tokens every N duration with no timeout
	go app.cleanExpiredTokens(*frequency, time.Hour*0)

	// Clean expired unactivated users every N duration with 1 hour timeout
	go app.cleanExpiredUnactivatedUsers(*frequency, time.Hour)

	// Running the server
	err = app.serve()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	os.Exit(1)
}

func openDB(dsn string) (*sql.DB, error) {

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
