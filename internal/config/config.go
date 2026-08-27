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
