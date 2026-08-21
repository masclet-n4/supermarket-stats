package pocketbase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type Client struct {
	baseURL  string
	identity string
	password string
	token    string
	http     *http.Client
	mu       sync.Mutex
	log      *log.Logger
}

func New(baseURL, identity, password string, httpClient *http.Client) *Client {
	if httpClient == nil { httpClient = &http.Client{} }
	return &Client{baseURL: baseURL, identity: identity, password: password, http: httpClient, log: log.Default()}
}

type authResponse struct { Token string `json:"token"` }

func (c *Client) Authenticate(ctx context.Context) error {
	c.mu.Lock()
	identity, password := c.identity, c.password
	c.mu.Unlock()
	if identity == "" || password == "" { return fmt.Errorf("pocketbase credentials are required") }
	var response authResponse
	if err := c.requestUnlocked(ctx, http.MethodPost, "/api/collections/_superusers/auth-with-password", nil, map[string]string{"identity": identity, "password": password}, &response, true); err != nil { return err }
	if response.Token == "" { return fmt.Errorf("pocketbase authentication returned an empty token") }
	c.token = response.Token
	return nil
}

func (c *Client) refresh(ctx context.Context) error {
	c.mu.Lock()
	identity, password := c.identity, c.password
	c.mu.Unlock()
	if identity == "" || password == "" { return fmt.Errorf("pocketbase credentials are required") }
	var response authResponse
	if err := c.requestUnlocked(ctx, http.MethodPost, "/api/collections/_superusers/auth-with-password", nil, map[string]string{"identity": identity, "password": password}, &response, true); err != nil { return err }
	if response.Token == "" { return fmt.Errorf("pocketbase authentication returned an empty token") }
	c.mu.Lock()
	c.token = response.Token
	c.mu.Unlock()
	return nil
}

func (c *Client) request(ctx context.Context, method, path string, query url.Values, body any, out any, noRefresh ...bool) error {
	return c.requestUnlocked(ctx, method, path, query, body, out, noRefresh...)
}

func (c *Client) requestUnlocked(ctx context.Context, method, path string, query url.Values, body any, out any, noRefresh ...bool) error {
	started := time.Now()
	defer func() {
		if elapsed := time.Since(started); elapsed >= time.Second { c.log.Printf("slow pocketbase request method=%s path=%s duration_ms=%d", method, path, elapsed.Milliseconds()) }
	}()
	requestURL := c.baseURL + path
	if len(query) > 0 { requestURL += "?" + query.Encode() }
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil { return err }
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, requestURL, reader)
	if err != nil { return err }
	req.Header.Set("Accept", "application/json")
	if body != nil { req.Header.Set("Content-Type", "application/json") }
	c.mu.Lock(); token := c.token; c.mu.Unlock()
	if token != "" { req.Header.Set("Authorization", token) }
	resp, err := c.http.Do(req)
	if err != nil { return err }
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil { return err }
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode == http.StatusUnauthorized && c.identity != "" && path != "/api/collections/_superusers/auth-with-password" && len(noRefresh) == 0 {
			if err := c.refresh(ctx); err != nil { return err }
			return c.requestUnlocked(ctx, method, path, query, body, out, true)
		}
		return fmt.Errorf("pocketbase %s %s: status %d: %s", method, path, resp.StatusCode, data)
	}
	if out != nil && len(data) > 0 { return json.Unmarshal(data, out) }
	return nil
}

func (c *Client) List(ctx context.Context, collection string, query url.Values, out any) error { return c.request(ctx, http.MethodGet, "/api/collections/"+url.PathEscape(collection)+"/records", query, nil, out) }
func (c *Client) Get(ctx context.Context, collection, id string, out any) error { return c.request(ctx, http.MethodGet, "/api/collections/"+url.PathEscape(collection)+"/records/"+url.PathEscape(id), nil, nil, out) }
func (c *Client) Create(ctx context.Context, collection string, body any, out any) error { return c.request(ctx, http.MethodPost, "/api/collections/"+url.PathEscape(collection)+"/records", nil, body, out) }
func (c *Client) Update(ctx context.Context, collection, id string, body any, out any) error { return c.request(ctx, http.MethodPatch, "/api/collections/"+url.PathEscape(collection)+"/records/"+url.PathEscape(id), nil, body, out) }
