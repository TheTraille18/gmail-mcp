package main

import (
	"context"
	"log"

	"github.com/TheTraille18/gmail-mcp/internal/config"
	"github.com/TheTraille18/gmail-mcp/internal/gmail"
	appserver "github.com/TheTraille18/gmail-mcp/internal/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	ctx := context.Background()

	cfg := config.Load()
	g, err := gmail.New(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}

	s := appserver.New(g)
	if err := s.Run(ctx, &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}
