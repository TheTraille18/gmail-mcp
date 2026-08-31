package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	CredentialsFile string
	TokenFile       string
	GmailUser       string
}

func Load() Config {
	creds := os.Getenv("GOOGLE_CREDENTIALS_FILE")
	if creds == "" {
		creds = filepath.Join("credentials", "credentials.json")
	}

	token := os.Getenv("GOOGLE_TOKEN_FILE")
	if token == "" {
		token = filepath.Join("credentials", "token.json")
	}

	// ECS/Secrets Manager: inject JSON via env and materialize under /tmp.
	if raw := os.Getenv("GOOGLE_CREDENTIALS_JSON"); raw != "" {
		path := filepath.Join(os.TempDir(), "gmail-credentials.json")
		if err := os.WriteFile(path, []byte(raw), 0o600); err == nil {
			creds = path
		}
	}
	if raw := os.Getenv("GOOGLE_TOKEN_JSON"); raw != "" {
		path := filepath.Join(os.TempDir(), "gmail-token.json")
		if err := os.WriteFile(path, []byte(raw), 0o600); err == nil {
			token = path
		}
	}

	user := os.Getenv("GMAIL_USER")
	if user == "" {
		user = "me"
	}

	return Config{
		CredentialsFile: creds,
		TokenFile:       token,
		GmailUser:       user,
	}
}
