package notification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type Notification struct {
	ID          int64           `json:"id"`
	EventID     string          `json:"event_id"`
	BookingID   string          `json:"booking_id"`
	EventType   string          `json:"event_type"`
	RecipientID string          `json:"recipient_id"`
	Channel     string          `json:"channel"`
	Message     string          `json:"message"`
	Status      string          `json:"status"`
	Payload     json.RawMessage `json:"payload"`
	CreatedAt   string          `json:"created_at"`
	ReadAt      string          `json:"read_at,omitempty"`
}

type ListResponse struct {
	Notifications []Notification `json:"notifications"`
}

type UnreadCountResponse struct {
	UnreadCount int64 `json:"unread_count"`
}

type MarkReadResponse struct {
	ID     int64  `json:"id"`
	ReadAt string `json:"read_at"`
}

type DevStatusResponse struct {
	Service             string         `json:"service"`
	Store               string         `json:"store"`
	KafkaBrokers        []string       `json:"kafka_brokers"`
	BookingTopic        string         `json:"booking_topic"`
	DLQTopic            string         `json:"dlq_topic"`
	ConsumerGroup       string         `json:"consumer_group"`
	TotalNotifications  int64          `json:"total_notifications"`
	RecipientID         string         `json:"recipient_id"`
	RecipientTotal      int64          `json:"recipient_total"`
	RecipientUnread     int64          `json:"recipient_unread"`
	LatestNotifications []Notification `json:"latest_notifications"`
}

type HTTPError struct {
	StatusCode int
	Message    string
}

func (e HTTPError) Error() string {
	return e.Message
}

func NewClient(baseURL string) (*Client, error) {
	if strings.TrimSpace(baseURL) == "" {
		return nil, errors.New("NOTIFICATION_SERVICE_URL no esta configurado")
	}
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: http.DefaultClient,
	}, nil
}

func (c *Client) ListNotifications(ctx context.Context, recipientID string, limit int) (*ListResponse, error) {
	var response ListResponse
	err := c.get(ctx, "/notifications", query(recipientID, limit), &response)
	return &response, err
}

func (c *Client) GetUnreadCount(ctx context.Context, recipientID string) (*UnreadCountResponse, error) {
	var response UnreadCountResponse
	err := c.get(ctx, "/notifications/unread-count", query(recipientID, 0), &response)
	return &response, err
}

func (c *Client) MarkNotificationRead(ctx context.Context, notificationID int64, recipientID string) (*MarkReadResponse, error) {
	var response MarkReadResponse
	err := c.patch(ctx, "/notifications/"+strconv.FormatInt(notificationID, 10)+"/read", query(recipientID, 0), &response)
	return &response, err
}

func (c *Client) GetDevStatus(ctx context.Context, recipientID string, limit int) (*DevStatusResponse, error) {
	var response DevStatusResponse
	err := c.get(ctx, "/dev/status", query(recipientID, limit), &response)
	return &response, err
}

func (c *Client) get(ctx context.Context, path string, params url.Values, target any) error {
	return c.request(ctx, http.MethodGet, path, params, target)
}

func (c *Client) patch(ctx context.Context, path string, params url.Values, target any) error {
	return c.request(ctx, http.MethodPatch, path, params, target)
}

func (c *Client) request(ctx context.Context, method string, path string, params url.Values, target any) error {
	endpoint := c.baseURL + path
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("notification-service no disponible: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var payload struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&payload)
		if payload.Error == "" {
			payload.Error = resp.Status
		}
		return HTTPError{StatusCode: resp.StatusCode, Message: payload.Error}
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("respuesta invalida de notification-service: %w", err)
	}
	return nil
}

func query(recipientID string, limit int) url.Values {
	params := url.Values{}
	if recipientID != "" {
		params.Set("recipient_id", recipientID)
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	return params
}
