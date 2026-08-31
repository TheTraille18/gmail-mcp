package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/TheTraille18/gmail-mcp/internal/config"
	"github.com/TheTraille18/gmail-mcp/internal/gmail"
	appserver "github.com/TheTraille18/gmail-mcp/internal/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func headersMiddleWare(token string, next http.Handler) http.Handler {

	expected := "Bearer " + token
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token == "" || r.Header.Get("Authorization") != expected {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	ctx := context.Background()

	TOKEN := os.Getenv("MCP_API_TOKEN")
	if TOKEN == "" {
		log.Fatal("MCP_API_TOKEN is required")
	}

	cfg := config.Load()
	g, err := gmail.New(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}

	server := appserver.New(g)
	mcpHandler := mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return server },
		&mcp.StreamableHTTPOptions{
			Stateless:    true,
			JSONResponse: true,
		},
	)

	mux := http.NewServeMux()
	mux.Handle("/mcp", mcpHandler)
	// ALB path rule for shared grocery ALB (e.g. /gmail/mcp)
	mux.Handle("/gmail/mcp", mcpHandler)

	PORT := os.Getenv("PORT")

	addr := ":" + envOr("PORT", PORT)
	log.Printf("gmail-mcp HTTP listening on %s", addr)
	if err := http.ListenAndServe(addr, headersMiddleWare(TOKEN, mux)); err != nil {
		log.Fatal(err)
	}

}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
