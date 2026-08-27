package gmail

import (
	"context"
	"fmt"
	"strings"

	gapi "google.golang.org/api/gmail/v1"
)

// Label is a Gmail label id + name.
type Label struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type,omitempty"` // system | user
}

// ApplyLabelResult is the outcome of labeling one message.
type ApplyLabelResult struct {
	MessageID string   `json:"message_id"`
	LabelID   string   `json:"label_id"`
	LabelName string   `json:"label_name"`
	LabelIDs  []string `json:"label_ids"`
}

// ListLabels returns user and system labels.
func (c *Client) ListLabels(ctx context.Context) ([]Label, error) {
	resp, err := c.svc.Users.Labels.List(c.user).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("list labels: %w", err)
	}
	out := make([]Label, 0, len(resp.Labels))
	for _, l := range resp.Labels {
		if l == nil {
			continue
		}
		out = append(out, Label{ID: l.Id, Name: l.Name, Type: l.Type})
	}
	return out, nil
}

// EnsureLabel returns an existing label by name (case-sensitive Gmail match)
// or creates a user label with that name.
func (c *Client) EnsureLabel(ctx context.Context, name string) (*Label, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("label name is required")
	}

	labels, err := c.ListLabels(ctx)
	if err != nil {
		return nil, err
	}
	for _, l := range labels {
		if l.Name == name {
			return &l, nil
		}
	}

	created, err := c.svc.Users.Labels.Create(c.user, &gapi.Label{
		Name:                  name,
		LabelListVisibility:   "labelShow",
		MessageListVisibility: "show",
	}).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("create label %q: %w", name, err)
	}
	return &Label{ID: created.Id, Name: created.Name, Type: created.Type}, nil
}

// ApplyLabel adds a label (by name) to one or more messages.
// Creates the label if it does not exist.
func (c *Client) ApplyLabel(ctx context.Context, labelName string, messageIDs []string) ([]ApplyLabelResult, error) {
	if len(messageIDs) == 0 {
		return nil, fmt.Errorf("at least one message id is required")
	}

	label, err := c.EnsureLabel(ctx, labelName)
	if err != nil {
		return nil, err
	}

	out := make([]ApplyLabelResult, 0, len(messageIDs))
	for _, id := range messageIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		msg, err := c.svc.Users.Messages.Modify(c.user, id, &gapi.ModifyMessageRequest{
			AddLabelIds: []string{label.ID},
		}).Context(ctx).Do()
		if err != nil {
			return out, fmt.Errorf("apply label %q to %s: %w", labelName, id, err)
		}
		out = append(out, ApplyLabelResult{
			MessageID: msg.Id,
			LabelID:   label.ID,
			LabelName: label.Name,
			LabelIDs:  msg.LabelIds,
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no valid message ids")
	}
	return out, nil
}
