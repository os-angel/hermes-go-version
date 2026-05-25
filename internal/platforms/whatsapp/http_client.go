package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client es el cliente HTTP al bridge local (bridge.js).
// Fase 9.
type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

// bridgeMessage es el formato JSON que el bridge retorna en GET /messages.
type bridgeMessage struct {
	MessageID         string   `json:"messageId"`
	ChatID            string   `json:"chatId"`
	SenderID          string   `json:"senderId"`
	SenderName        string   `json:"senderName"`
	IsGroup           bool     `json:"isGroup"`
	Body              string   `json:"body"`
	HasMedia          bool     `json:"hasMedia"`
	MediaType         string   `json:"mediaType"`
	MediaURLs         []string `json:"mediaUrls"`
	QuotedMessageID   string   `json:"quotedMessageId"`
	Timestamp         int64    `json:"timestamp"`
}

// Send envia un mensaje de texto.
func (c *Client) Send(ctx context.Context, chatID, text, replyTo string) error {
	payload := map[string]any{"chatId": chatID, "message": text}
	if replyTo != "" {
		payload["replyTo"] = replyTo
	}
	return c.post(ctx, "/send", payload, nil)
}

// SendMedia envia un archivo.
func (c *Client) SendMedia(ctx context.Context, chatID, filePath, mediaType, caption, filename string) error {
	return c.post(ctx, "/send-media", map[string]any{
		"chatId":    chatID,
		"filePath":  filePath,
		"mediaType": mediaType,
		"caption":   caption,
		"fileName":  filename,
	}, nil)
}

// Typing envia el indicador de escritura.
func (c *Client) Typing(ctx context.Context, chatID string) error {
	return c.post(ctx, "/typing", map[string]any{"chatId": chatID}, nil)
}

// Poll retorna los mensajes pendientes de la queue del bridge (y la vacia).
func (c *Client) Poll(ctx context.Context) ([]bridgeMessage, error) {
	resp, err := c.http.Get(c.baseURL + "/messages")
	if err != nil {
		return nil, fmt.Errorf("poll: %w", err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	var msgs []bridgeMessage
	if err := json.Unmarshal(data, &msgs); err != nil {
		return nil, fmt.Errorf("parse messages: %w", err)
	}
	return msgs, nil
}

// Health retorna nil si el bridge responde.
func (c *Client) Health(ctx context.Context) error {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bridge health: status %d", resp.StatusCode)
	}
	return nil
}

// Status retorna el estado de conexion del bridge.
type Status struct {
	Connected bool   `json:"connected"`
	Phone     string `json:"phone"`
}

func (c *Client) GetStatus(ctx context.Context) (Status, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/status", nil)
	resp, err := c.http.Do(req)
	if err != nil {
		return Status{}, err
	}
	defer resp.Body.Close()
	var s Status
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return Status{}, err
	}
	return s, nil
}

func (c *Client) post(ctx context.Context, path string, payload, result any) error {
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("POST %s: %w", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("POST %s: status %d: %s", path, resp.StatusCode, data)
	}
	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}
