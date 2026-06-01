package config

import (
	"log"
	"os"
	"strings"
)

type Config struct {
	DBPath                     string
	JWTSecret                  string
	Port                       string
	UploadDir                  string
	WebAuthnID                 string
	WebAuthnOrigin             string
	CORSOrigins                string
	AgentServiceURL            string
	AgentInternalCallbackURL   string
}

func Load() *Config {
	jwtSecret := getEnv("JWT_SECRET", "")
	if jwtSecret == "" || jwtSecret == "dev-secret-key-change-in-production-min32chars" {
		if os.Getenv("GIN_MODE") == "release" {
			log.Fatal("FATAL: JWT_SECRET must be set to a strong, unique value in production. " +
				"Generate one with: openssl rand -base64 48")
		}
		if jwtSecret == "" {
			jwtSecret = "dev-secret-key-change-in-production-min32chars"
		}
		log.Println("WARNING: Using default JWT secret. Set JWT_SECRET for production use.")
	}
	if len(jwtSecret) < 32 {
		log.Fatal("FATAL: JWT_SECRET must be at least 32 characters long")
	}

	return &Config{
		DBPath:                   getEnv("DB_PATH", "./ancientcoins.db"),
		JWTSecret:                jwtSecret,
		Port:                     getEnv("PORT", "8080"),
		UploadDir:                getEnv("UPLOAD_DIR", "./uploads"),
		WebAuthnID:               getEnv("WEBAUTHN_RP_ID", "localhost"),
		WebAuthnOrigin:           getEnv("WEBAUTHN_ORIGIN", "http://localhost:8080"),
		CORSOrigins:              getEnv("CORS_ORIGINS", ""),
		AgentServiceURL:          getEnv("AGENT_SERVICE_URL", "http://localhost:8081"),
		AgentInternalCallbackURL: getEnv("AGENT_INTERNAL_CALLBACK_URL", "http://localhost:8080"),
	}
}

// AllowedOrigins returns the list of origins permitted for CORS.
// Uses CORS_ORIGINS env var (comma-separated), falling back to WebAuthn origins.
func (c *Config) AllowedOrigins() []string {
	if c.CORSOrigins != "" {
		origins := strings.Split(c.CORSOrigins, ",")
		for i := range origins {
			origins[i] = strings.TrimSpace(origins[i])
		}
		return origins
	}
	// Fall back to WebAuthn origins + common dev origins
	origins := strings.Split(c.WebAuthnOrigin, ",")
	for i := range origins {
		origins[i] = strings.TrimSpace(origins[i])
	}
	origins = append(origins, "http://localhost:5173", "http://localhost:8080")
	return origins
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
