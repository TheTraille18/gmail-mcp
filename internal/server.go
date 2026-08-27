package mcp

import (
	"context"

	"github.com/TheTraille18/gmail-mcp/internal/gmail"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type searchEmailsInput struct {
	Query string `json:"query" jsonschema:"Gmail search query, e.g. is:unread newer_than:7d"`
	Max   int64  `json:"max,omitempty" jsonschema:"max results (default 10, max 50)"`
}
type searchEmailsOutput struct {
	Messages []gmail.MessageSummary `json:"messages"`
}

type getEmailInput struct {
	Id string `json:"id" jsonschema:"id of the email message"`
}
type getEmailOutput struct {
	Message gmail.Message `json:"message"`
}

type listLabelsOutput struct {
	Labels []gmail.Label `json:"labels"`
}

type applyLabelInput struct {
	Label      string   `json:"label" jsonschema:"Gmail label name to apply, e.g. Jobs or Jobs/Inbox"`
	MessageIDs []string `json:"message_ids" jsonschema:"Gmail message ids from search_gmail"`
}
type applyLabelOutput struct {
	Results []gmail.ApplyLabelResult `json:"results"`
}

// New builds the MCP server and registers tools that use the Gmail client.
func New(g *gmail.Client) *sdk.Server {
	s := sdk.NewServer(&sdk.Implementation{
		Name:    "gmail-mcp",
		Version: "0.1.0",
	}, nil)

	registerTools(s, g)
	return s
}

func registerTools(s *sdk.Server, g *gmail.Client) {

	sdk.AddTool(s, &sdk.Tool{
		Name:        "ping",
		Description: "Echo a message to verify the MCP server",
	}, ping)

	sdk.AddTool(s, &sdk.Tool{
		Name:        "search_gmail",
		Description: "Search gmail messages",
	}, func(
		ctx context.Context,
		_ *sdk.CallToolRequest,
		input searchEmailsInput,
	) (*sdk.CallToolResult, searchEmailsOutput, error) {
		msgs, err := g.Search(ctx, input.Query, input.Max)
		if err != nil {
			return nil, searchEmailsOutput{}, err
		}
		return nil, searchEmailsOutput{Messages: msgs}, nil
	})

	sdk.AddTool(s, &sdk.Tool{
		Name:        "get_gmail",
		Description: "Get gmail using ID",
	}, func(
		ctx context.Context,
		_ *sdk.CallToolRequest,
		input getEmailInput,
	) (*sdk.CallToolResult, getEmailOutput, error) {
		msg, err := g.Get(ctx, input.Id)
		if err != nil {
			return nil, getEmailOutput{}, err
		}
		return nil, getEmailOutput{Message: *msg}, nil
	})

	sdk.AddTool(s, &sdk.Tool{
		Name:        "list_labels",
		Description: "List Gmail labels (id + name). Use before apply_label to reuse existing Jobs/* labels.",
	}, func(
		ctx context.Context,
		_ *sdk.CallToolRequest,
		_ struct{},
	) (*sdk.CallToolResult, listLabelsOutput, error) {
		labels, err := g.ListLabels(ctx)
		if err != nil {
			return nil, listLabelsOutput{}, err
		}
		return nil, listLabelsOutput{Labels: labels}, nil
	})

	sdk.AddTool(s, &sdk.Tool{
		Name: "apply_label",
		Description: "Apply a Gmail label to one or more messages (creates the label if missing). " +
			"Use for job-related mail, e.g. label=Jobs after search_gmail finds recruiter/job emails.",
	}, func(
		ctx context.Context,
		_ *sdk.CallToolRequest,
		input applyLabelInput,
	) (*sdk.CallToolResult, applyLabelOutput, error) {
		results, err := g.ApplyLabel(ctx, input.Label, input.MessageIDs)
		if err != nil {
			return nil, applyLabelOutput{}, err
		}
		return nil, applyLabelOutput{Results: results}, nil
	})
}

type pingInput struct {
	Message string `json:"message" jsonschema:"text to echo back"`
}

type pingOutput struct {
	Reply string `json:"reply" jsonschema:"echoed reply"`
}

func ping(
	ctx context.Context,
	req *sdk.CallToolRequest,
	input pingInput,
) (*sdk.CallToolResult, pingOutput, error) {
	return nil, pingOutput{Reply: "pong: " + input.Message}, nil
}
