package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sony/gobreaker"
)

type Client struct {
	baseURL string
	http    *http.Client
	cb      *gobreaker.CircuitBreaker
}

type InvitePayload struct {
	TeamID   uint64 `json:"team_id"`
	UserID   uint64 `json:"user_id"`
	InviteBy uint64 `json:"invite_by"`
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	settings := gobreaker.Settings{
		Name:        "email",
		MaxRequests: 5,
		Interval:    30 * time.Second,
		Timeout:     15 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.Requests >= 10 && counts.TotalFailures*100/counts.Requests >= 50
		},
	}
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: timeout},
		cb:      gobreaker.NewCircuitBreaker(settings),
	}
}

func (c *Client) SendInvite(ctx context.Context, p InvitePayload) error {
	_, err := c.cb.Execute(func() (any, error) {
		body, err := json.Marshal(p)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/send", bytes.NewBuffer(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := c.http.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("email service status code: %d", resp.StatusCode)
		}
		return nil, nil
	})
	return err
}
