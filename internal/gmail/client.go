package gmail

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/TheTraille18/gmail-mcp/internal/auth"
	"github.com/TheTraille18/gmail-mcp/internal/config"
	gapi "google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// Client wraps the Gmail API service.
type Client struct {
	svc  *gapi.Service
	user string
}

// MessageSummary is a lightweight search result.
type MessageSummary struct {
	ID       string `json:"id"`
	ThreadID string `json:"thread_id"`
	Subject  string `json:"subject"`
	From     string `json:"from"`
	Date     string `json:"date"`
	Snippet  string `json:"snippet"`
}

// Message is a full email suitable for reading/summarizing.
type Message struct {
	ID       string   `json:"id"`
	ThreadID string   `json:"thread_id"`
	Subject  string   `json:"subject"`
	From     string   `json:"from"`
	To       string   `json:"to"`
	Date     string   `json:"date"`
	Snippet  string   `json:"snippet"`
	Body     string   `json:"body"`
	LabelIDs []string `json:"label_ids"`
}

// New creates a Gmail client using credentials/token from config.
func New(ctx context.Context, cfg config.Config) (*Client, error) {
	httpClient, err := auth.Client(ctx, cfg.CredentialsFile, cfg.TokenFile)
	if err != nil {
		return nil, fmt.Errorf("gmail auth: %w", err)
	}

	svc, err := gapi.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("create gmail service: %w", err)
	}

	return &Client{svc: svc, user: cfg.GmailUser}, nil
}

// ListLabelNames is a simple smoke-test helper.
func (c *Client) ListLabelNames(ctx context.Context) ([]string, error) {
	resp, err := c.svc.Users.Labels.List(c.user).Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(resp.Labels))
	for _, l := range resp.Labels {
		names = append(names, l.Name)
	}
	return names, nil
}

// Search finds messages matching a Gmail query (same syntax as the Gmail search box).
func (c *Client) Search(ctx context.Context, query string, max int64) ([]MessageSummary, error) {
	if max <= 0 {
		max = 10
	}
	if max > 50 {
		max = 50
	}

	list, err := c.svc.Users.Messages.List(c.user).
		Context(ctx).
		Q(query).
		MaxResults(max).
		Do()
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}

	out := make([]MessageSummary, 0, len(list.Messages))
	for _, m := range list.Messages {
		msg, err := c.svc.Users.Messages.Get(c.user, m.Id).
			Context(ctx).
			Format("metadata").
			MetadataHeaders("From", "Subject", "Date").
			Do()
		if err != nil {
			return nil, fmt.Errorf("get message metadata %s: %w", m.Id, err)
		}

		headers := headerMap(msg.Payload)
		out = append(out, MessageSummary{
			ID:       msg.Id,
			ThreadID: msg.ThreadId,
			Subject:  headers["Subject"],
			From:     headers["From"],
			Date:     headers["Date"],
			Snippet:  msg.Snippet,
		})
	}
	return out, nil
}

// Get returns one message by id, including a decoded text body.
func (c *Client) Get(ctx context.Context, id string) (*Message, error) {
	if id == "" {
		return nil, fmt.Errorf("message id is required")
	}

	msg, err := c.svc.Users.Messages.Get(c.user, id).
		Context(ctx).
		Format("full").
		Do()
	if err != nil {
		return nil, fmt.Errorf("get message %s: %w", id, err)
	}

	headers := headerMap(msg.Payload)
	body := extractBody(msg.Payload)

	return &Message{
		ID:       msg.Id,
		ThreadID: msg.ThreadId,
		Subject:  headers["Subject"],
		From:     headers["From"],
		To:       headers["To"],
		Date:     headers["Date"],
		Snippet:  msg.Snippet,
		Body:     body,
		LabelIDs: msg.LabelIds,
	}, nil
}

// Service exposes the underlying API for advanced use.
func (c *Client) Service() *gapi.Service {
	return c.svc
}

// User returns the Gmail user id (usually "me").
func (c *Client) User() string {
	return c.user
}

func headerMap(part *gapi.MessagePart) map[string]string {
	out := map[string]string{}
	if part == nil {
		return out
	}
	for _, h := range part.Headers {
		if h == nil {
			continue
		}
		out[h.Name] = h.Value
	}
	return out
}

func extractBody(part *gapi.MessagePart) string {
	if part == nil {
		return ""
	}

	// Prefer plain text over HTML.
	if plain := findPart(part, "text/plain"); plain != "" {
		return plain
	}
	if html := findPart(part, "text/html"); html != "" {
		return html
	}
	return ""
}

func findPart(part *gapi.MessagePart, mime string) string {
	if part == nil {
		return ""
	}
	if strings.EqualFold(part.MimeType, mime) && part.Body != nil && part.Body.Data != "" {
		if decoded, err := decodeBody(part.Body.Data); err == nil {
			return decoded
		}
	}
	for _, child := range part.Parts {
		if got := findPart(child, mime); got != "" {
			return got
		}
	}
	return ""
}

func decodeBody(data string) (string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(data)
	if err != nil {
		raw, err = base64.URLEncoding.DecodeString(data)
		if err != nil {
			return "", err
		}
	}
	return string(raw), nil
}
