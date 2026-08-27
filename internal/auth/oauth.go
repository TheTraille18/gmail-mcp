package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
)

// Client returns an HTTP client authorized for Gmail read + label modify.
// credentialsFile is the Google OAuth desktop client JSON.
// tokenFile stores the user access/refresh token (Career Pilot Python format supported).
// If the token is missing or revoked, a browser login flow runs and saves a new token.
// Scope changes require re-consent (delete token.json or run with force re-login).
func Client(ctx context.Context, credentialsFile, tokenFile string) (*http.Client, error) {
	cfg, err := oauthConfig(credentialsFile)
	if err != nil {
		return nil, err
	}

	tok, err := tokenFromFile(tokenFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "no usable token; starting browser login...")
		tok, err = Login(ctx, cfg, tokenFile)
		if err != nil {
			return nil, err
		}
	}

	src := &savingTokenSource{
		src:       oauth2.ReuseTokenSource(tok, cfg.TokenSource(ctx, tok)),
		tokenFile: tokenFile,
		current:   tok,
	}

	// Force a token refresh now so we detect revoked tokens before API calls.
	if _, err := src.Token(); err != nil {
		fmt.Fprintln(os.Stderr, "token refresh failed; starting browser login...")
		tok, err = Login(ctx, cfg, tokenFile)
		if err != nil {
			return nil, err
		}
		src = &savingTokenSource{
			src:       oauth2.ReuseTokenSource(tok, cfg.TokenSource(ctx, tok)),
			tokenFile: tokenFile,
			current:   tok,
		}
	}

	return oauth2.NewClient(ctx, src), nil
}

// Login runs the desktop OAuth consent flow and writes tokenFile.
func Login(ctx context.Context, cfg *oauth2.Config, tokenFile string) (*oauth2.Token, error) {
	// Match installed-app redirect "http://localhost" (any port).
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen for oauth redirect: %w", err)
	}
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port
	cfgCopy := *cfg
	cfgCopy.RedirectURL = fmt.Sprintf("http://localhost:%d/", port)

	state, err := randomState()
	if err != nil {
		return nil, err
	}

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("state") != state {
			errCh <- fmt.Errorf("oauth state mismatch")
			http.Error(w, "state mismatch", http.StatusBadRequest)
			return
		}
		if errMsg := r.URL.Query().Get("error"); errMsg != "" {
			errCh <- fmt.Errorf("oauth error: %s", errMsg)
			http.Error(w, errMsg, http.StatusBadRequest)
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("missing oauth code")
			http.Error(w, "missing code", http.StatusBadRequest)
			return
		}
		fmt.Fprint(w, "Authentication successful. You can close this tab.")
		codeCh <- code
	})

	srv := &http.Server{Handler: mux}
	go func() {
		_ = srv.Serve(ln)
	}()
	defer srv.Shutdown(context.Background())

	authURL := cfgCopy.AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("prompt", "consent"),
	)
	fmt.Fprintf(os.Stderr, "Open this URL to authorize Gmail access:\n%s\n", authURL)
	_ = openBrowser(authURL)

	var code string
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-errCh:
		return nil, err
	case code = <-codeCh:
	case <-time.After(5 * time.Minute):
		return nil, fmt.Errorf("oauth timed out waiting for browser consent")
	}

	tok, err := cfgCopy.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("exchange auth code: %w", err)
	}
	if err := saveToken(tokenFile, tok); err != nil {
		return nil, fmt.Errorf("save token: %w", err)
	}
	fmt.Fprintln(os.Stderr, "saved new token to", tokenFile)
	return tok, nil
}

func oauthConfig(credentialsFile string) (*oauth2.Config, error) {
	data, err := os.ReadFile(credentialsFile)
	if err != nil {
		return nil, fmt.Errorf("read credentials file: %w", err)
	}
	cfg, err := google.ConfigFromJSON(data, gmail.GmailModifyScope)
	if err != nil {
		return nil, fmt.Errorf("parse credentials: %w", err)
	}
	return cfg, nil
}

func tokenFromFile(path string) (*oauth2.Token, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var raw map[string]any
	if err := json.NewDecoder(f).Decode(&raw); err != nil {
		return nil, err
	}

	tok := &oauth2.Token{}

	// Go oauth2 shape
	if v, ok := raw["access_token"].(string); ok {
		tok.AccessToken = v
	}
	// Python google-auth shape (Career Pilot)
	if tok.AccessToken == "" {
		if v, ok := raw["token"].(string); ok {
			tok.AccessToken = v
		}
	}

	if v, ok := raw["refresh_token"].(string); ok {
		tok.RefreshToken = v
	}
	if v, ok := raw["token_type"].(string); ok {
		tok.TokenType = v
	}
	if tok.TokenType == "" {
		tok.TokenType = "Bearer"
	}

	if v, ok := raw["expiry"].(string); ok && v != "" {
		for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
			if t, err := time.Parse(layout, v); err == nil {
				tok.Expiry = t
				break
			}
		}
	}

	if tok.AccessToken == "" && tok.RefreshToken == "" {
		return nil, fmt.Errorf("token file missing access/refresh token")
	}
	return tok, nil
}

func saveToken(path string, token *oauth2.Token) error {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(token)
}

type savingTokenSource struct {
	src       oauth2.TokenSource
	tokenFile string
	mu        sync.Mutex
	current   *oauth2.Token
}

func (s *savingTokenSource) Token() (*oauth2.Token, error) {
	tok, err := s.src.Token()
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.current == nil ||
		tok.AccessToken != s.current.AccessToken ||
		tok.RefreshToken != s.current.RefreshToken {
		if err := saveToken(s.tokenFile, tok); err != nil {
			return nil, err
		}
		s.current = tok
	}
	return tok, nil
}

func randomState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported OS for browser open")
	}
	return cmd.Start()
}
