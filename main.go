package main

import (
	"catalog-agent/audit"
	"catalog-agent/handlers"
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	APIPort              int           `yaml:"apiPort"`
	ScanInterval         time.Duration `yaml:"scanInterval"`
	HealthInterval       time.Duration `yaml:"healthInterval"`
	ConfigReloadInterval time.Duration `yaml:"configReloadInterval"`
	LogLevel             string        `yaml:"logLevel"`
	ServerURL            string        `yaml:"serverUrl"`
	LDAP                 LDAPConfig    `yaml:"ldap"`
	Storage              StorageConfig `yaml:"storage"`
}

type LDAPConfig struct {
	Server   string        `yaml:"server"`
	Port     int           `yaml:"port"`
	BaseDN   string        `yaml:"baseDN"`
	BindUser string        `yaml:"bindUser"`
	BindPass string        `yaml:"bindPass"`
	Timeout  time.Duration `yaml:"timeout"`
	Enabled  bool          `yaml:"enabled"`
}

type StorageConfig struct {
	CorporateResources string `yaml:"corporateResources"`
	PersonalResources  string `yaml:"personalResources"`
}

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "configs/config.yaml", "Path to config file")
	flag.Parse()

	// Load configuration
	var config Config
	if err := loadConfig(configPath, &config); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	logger := audit.NewLogger(config.LogLevel)
	defer logger.Close()

	logger.Info("Starting Catalog Agent...")

	// Initialize AD client
	adConfig := handlers.ADConfig{
		Server:   config.LDAP.Server,
		Port:     config.LDAP.Port,
		BaseDN:   config.LDAP.BaseDN,
		BindUser: config.LDAP.BindUser,
		BindPass: config.LDAP.BindPass,
		Timeout:  config.LDAP.Timeout,
		Enabled:  config.LDAP.Enabled,
	}
	adClient := handlers.NewADClient(adConfig, logger)

	// Initialize resource loader
	resourceLoader := handlers.NewResourceLoader(
		config.Storage.CorporateResources,
		config.Storage.PersonalResources,
	)

	// Initialize catalog handler
	catalogHandler := handlers.NewCatalogHandler(logger, adClient, resourceLoader)

	// Setup routes
	mux := http.NewServeMux()
	mux.HandleFunc("/catalog", catalogHandler.GetUserCatalog)
	mux.HandleFunc("/execute", catalogHandler.ExecuteResource)
	mux.HandleFunc("/personal/add", catalogHandler.AddPersonalResource)
	mux.HandleFunc("/personal/delete", catalogHandler.DeletePersonalResource)

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Start HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", config.APIPort),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Info("Server starting on port %d", config.APIPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server shutdown failed: %v", err)
	}

	// Cleanup
	adClient.Close()
	logger.Info("Server stopped")
}

func loadConfig(path string, config *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, config)
}
