package config

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Config struct {
	Server        ServerConfig
	Gitea         GiteaConfig
	MySQL         MySQLConfig
	Webhook       WebhookConfig
	APIKey        string
	JWTSecret     string
	AllowedOrigins []string
}

type ServerConfig struct {
	Port string
}

type GiteaConfig struct {
	URL   string
	Token string
	Repo  string
}

type MySQLConfig struct {
	DSN string
}

type WebhookConfig struct {
	Secret string
}

func Load() *Config {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Load .env file if present
	loadEnvFile()

	originsStr := getEnv("ALLOWED_ORIGINS", "http://localhost:3456,http://localhost:3001")
	origins := strings.Split(originsStr, ",")
	for i := range origins {
		origins[i] = strings.TrimSpace(origins[i])
	}

	cfg := &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", "8000"),
		},
		Gitea: GiteaConfig{
			URL:   getEnv("GITEA_URL", "http://localhost:3000"),
			Token: getEnv("GITEA_TOKEN", ""),
			Repo:  getEnv("GITEA_REPO", "xuzong/knowledge-vault"),
		},
		MySQL: MySQLConfig{
			DSN: getEnv("MYSQL_DSN", ""),
		},
		Webhook: WebhookConfig{
			Secret: getEnv("WEBHOOK_SECRET", ""),
		},
		APIKey:         getEnv("API_KEY", ""),
		AllowedOrigins: origins,
		JWTSecret:      requireEnvOrGenerate("JWT_SECRET"),
	}
	return cfg
}

// requireEnvOrGenerate returns the value of the env var or generates a random
// 32-byte hex secret. A fixed default is never used to avoid shared-secret
// deployments leaking auth across instances.
func requireEnvOrGenerate(key string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		log.Fatal().Err(err).Msgf("failed to generate random %s", key)
	}
	secret := hex.EncodeToString(b)
	fmt.Fprintf(os.Stderr, "[config] WARNING: %s not set — generated ephemeral secret (tokens will not survive restart)\n", key)
	os.Setenv(key, secret)
	return secret
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func loadEnvFile() {
	f, err := os.Open(".env")
	if err != nil {
		return // no .env file, that's fine
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		// Strip surrounding single or double quotes
		if len(val) >= 2 {
			if (val[0] == '\'' && val[len(val)-1] == '\'') || (val[0] == '"' && val[len(val)-1] == '"') {
				val = val[1 : len(val)-1]
			}
		}
		// Only set if not already in environment
		if os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
}
