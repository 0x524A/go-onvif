package onvif

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Client represents an ONVIF client for communicating with IP cameras
type Client struct {
	endpoint   string
	username   string
	password   string
	httpClient *http.Client
	mu         sync.RWMutex
	
	// Service endpoints
	mediaEndpoint   string
	ptzEndpoint     string
	imagingEndpoint string
	eventEndpoint   string
}

// ClientOption is a functional option for configuring the Client
type ClientOption func(*Client)

// WithTimeout sets the HTTP client timeout
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// WithCredentials sets the authentication credentials
func WithCredentials(username, password string) ClientOption {
	return func(c *Client) {
		c.username = username
		c.password = password
	}
}

// NewClient creates a new ONVIF client
// The endpoint can be provided in multiple formats:
//   - Full URL: "http://192.168.1.100/onvif/device_service"
//   - IP with port: "192.168.1.100:80" (http assumed, /onvif/device_service added)
//   - IP only: "192.168.1.100" (http://IP:80/onvif/device_service used)
func NewClient(endpoint string, opts ...ClientOption) (*Client, error) {
	// Normalize endpoint to full URL
	normalizedEndpoint, err := normalizeEndpoint(endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint: %w", err)
	}

	client := &Client{
		endpoint: normalizedEndpoint,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 5,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}

	// Apply options
	for _, opt := range opts {
		opt(client)
	}

	return client, nil
}

// normalizeEndpoint converts various endpoint formats to a full ONVIF URL
func normalizeEndpoint(endpoint string) (string, error) {
	// Check if endpoint starts with a scheme
	if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
		// Parse as full URL
		parsedURL, err := url.Parse(endpoint)
		if err != nil {
			return "", err
		}
		if parsedURL.Host == "" {
			return "", fmt.Errorf("URL missing host")
		}
		// If path is empty or just "/", add default ONVIF path
		if parsedURL.Path == "" || parsedURL.Path == "/" {
			parsedURL.Path = "/onvif/device_service"
		}
		return parsedURL.String(), nil
	}

	// No scheme - treat as IP, IP:port, hostname, or hostname:port
	// Add http:// scheme and validate
	fullURL := "http://" + endpoint + "/onvif/device_service"
	parsedURL, err := url.Parse(fullURL)
	if err != nil {
		return "", fmt.Errorf("invalid IP address or hostname: %w", err)
	}
	
	if parsedURL.Host == "" {
		return "", fmt.Errorf("invalid endpoint format")
	}

	return fullURL, nil
}

// fixLocalhostURL replaces localhost/loopback addresses in service URLs with the actual camera host
// Some cameras incorrectly report localhost (127.0.0.1, 0.0.0.0, localhost) in their capability URLs
func (c *Client) fixLocalhostURL(serviceURL string) string {
	if serviceURL == "" {
		return serviceURL
	}

	// Parse the service URL
	parsedService, err := url.Parse(serviceURL)
	if err != nil {
		return serviceURL // Return original if parsing fails
	}

	// Check if the service URL has a localhost/loopback address
	host := parsedService.Hostname()
	if host == "localhost" || host == "127.0.0.1" || host == "0.0.0.0" || host == "::1" {
		// Parse the client's endpoint to get the actual camera address
		parsedClient, err := url.Parse(c.endpoint)
		if err != nil {
			return serviceURL // Return original if parsing fails
		}

		// Replace the host but keep the port from service URL if specified
		servicePort := parsedService.Port()
		if servicePort != "" {
			parsedService.Host = parsedClient.Hostname() + ":" + servicePort
		} else {
			parsedService.Host = parsedClient.Hostname()
			// Use client's port if service doesn't specify one
			if clientPort := parsedClient.Port(); clientPort != "" {
				parsedService.Host = parsedClient.Hostname() + ":" + clientPort
			}
		}

		return parsedService.String()
	}

	return serviceURL
}

// Initialize discovers and initializes service endpoints
func (c *Client) Initialize(ctx context.Context) error {
	// Get device information and capabilities
	capabilities, err := c.GetCapabilities(ctx)
	if err != nil {
		return fmt.Errorf("failed to get capabilities: %w", err)
	}

	// Extract service endpoints and fix any localhost addresses
	// Some cameras incorrectly report localhost instead of their actual IP
	if capabilities.Media != nil && capabilities.Media.XAddr != "" {
		c.mediaEndpoint = c.fixLocalhostURL(capabilities.Media.XAddr)
	}
	if capabilities.PTZ != nil && capabilities.PTZ.XAddr != "" {
		c.ptzEndpoint = c.fixLocalhostURL(capabilities.PTZ.XAddr)
	}
	if capabilities.Imaging != nil && capabilities.Imaging.XAddr != "" {
		c.imagingEndpoint = c.fixLocalhostURL(capabilities.Imaging.XAddr)
	}
	if capabilities.Events != nil && capabilities.Events.XAddr != "" {
		c.eventEndpoint = c.fixLocalhostURL(capabilities.Events.XAddr)
	}

	return nil
}

// Endpoint returns the device endpoint
func (c *Client) Endpoint() string {
	return c.endpoint
}

// SetCredentials updates the authentication credentials
func (c *Client) SetCredentials(username, password string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.username = username
	c.password = password
}

// GetCredentials returns the current credentials
func (c *Client) GetCredentials() (string, string) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.username, c.password
}

// DownloadFile downloads a file from the given URL with authentication
// Returns the raw file bytes
func (c *Client) DownloadFile(ctx context.Context, url string) ([]byte, error) {
	// Create a new HTTP request with context
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication if credentials are provided
	if c.username != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	// Set User-Agent header
	req.Header.Set("User-Agent", "onvif-go-client")

	// Execute the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed with status code %d", resp.StatusCode)
	}

	// Read all data from response body
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return data, nil
}
