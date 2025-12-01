package onvif

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestNormalizeEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "full URL with path",
			input:    "http://192.168.1.100/onvif/device_service",
			expected: "http://192.168.1.100/onvif/device_service",
			wantErr:  false,
		},
		{
			name:     "full URL with port and path",
			input:    "http://192.168.1.100:8080/onvif/device_service",
			expected: "http://192.168.1.100:8080/onvif/device_service",
			wantErr:  false,
		},
		{
			name:     "full URL without path",
			input:    "http://192.168.1.100",
			expected: "http://192.168.1.100/onvif/device_service",
			wantErr:  false,
		},
		{
			name:     "full URL with just slash",
			input:    "http://192.168.1.100/",
			expected: "http://192.168.1.100/onvif/device_service",
			wantErr:  false,
		},
		{
			name:     "IP address only",
			input:    "192.168.1.100",
			expected: "http://192.168.1.100/onvif/device_service",
			wantErr:  false,
		},
		{
			name:     "IP with port",
			input:    "192.168.1.100:8080",
			expected: "http://192.168.1.100:8080/onvif/device_service",
			wantErr:  false,
		},
		{
			name:     "IP with default HTTP port",
			input:    "192.168.1.100:80",
			expected: "http://192.168.1.100:80/onvif/device_service",
			wantErr:  false,
		},
		{
			name:     "hostname only",
			input:    "camera.local",
			expected: "http://camera.local/onvif/device_service",
			wantErr:  false,
		},
		{
			name:     "hostname with port",
			input:    "camera.local:8080",
			expected: "http://camera.local:8080/onvif/device_service",
			wantErr:  false,
		},
		{
			name:     "HTTPS URL",
			input:    "https://192.168.1.100/onvif/device_service",
			expected: "https://192.168.1.100/onvif/device_service",
			wantErr:  false,
		},
		{
			name:     "HTTPS with custom port",
			input:    "https://192.168.1.100:8443/onvif/device_service",
			expected: "https://192.168.1.100:8443/onvif/device_service",
			wantErr:  false,
		},
		{
			name:     "URL with custom path",
			input:    "http://192.168.1.100/custom/path",
			expected: "http://192.168.1.100/custom/path",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := normalizeEndpoint(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("normalizeEndpoint() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("normalizeEndpoint() unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("normalizeEndpoint() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNewClientWithVariousEndpoints(t *testing.T) {
	tests := []struct {
		name         string
		endpoint     string
		expectScheme string
		expectHost   string
		expectPath   string
	}{
		{
			name:         "IP only",
			endpoint:     "192.168.1.100",
			expectScheme: "http",
			expectHost:   "192.168.1.100",
			expectPath:   "/onvif/device_service",
		},
		{
			name:         "IP with port",
			endpoint:     "192.168.1.100:8080",
			expectScheme: "http",
			expectHost:   "192.168.1.100:8080",
			expectPath:   "/onvif/device_service",
		},
		{
			name:         "Full URL",
			endpoint:     "http://192.168.1.100/onvif/device_service",
			expectScheme: "http",
			expectHost:   "192.168.1.100",
			expectPath:   "/onvif/device_service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.endpoint)
			if err != nil {
				t.Fatalf("NewClient() error = %v", err)
			}

			if !strings.HasPrefix(client.endpoint, tt.expectScheme+"://") {
				t.Errorf("Expected scheme %s, got endpoint %s", tt.expectScheme, client.endpoint)
			}

			if !strings.Contains(client.endpoint, tt.expectHost) {
				t.Errorf("Expected host %s in endpoint %s", tt.expectHost, client.endpoint)
			}

			if !strings.HasSuffix(client.endpoint, tt.expectPath) {
				t.Errorf("Expected path %s in endpoint %s", tt.expectPath, client.endpoint)
			}
		})
	}
}

// Mock ONVIF server for comprehensive testing
type MockONVIFServer struct {
	server     *httptest.Server
	responses  map[string]string
	username   string
	password   string
	authFailed bool
}

func NewMockONVIFServer() *MockONVIFServer {
	mock := &MockONVIFServer{
		responses: make(map[string]string),
		username:  "admin",
		password:  "password",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", mock.handleRequest)
	mock.server = httptest.NewServer(mux)

	// Set up default responses
	mock.setupDefaultResponses()

	return mock
}

func (m *MockONVIFServer) URL() string {
	return m.server.URL
}

func (m *MockONVIFServer) Close() {
	m.server.Close()
}

func (m *MockONVIFServer) SetAuthFailure(fail bool) {
	m.authFailed = fail
}

func (m *MockONVIFServer) SetResponse(action string, response string) {
	m.responses[action] = response
}

func (m *MockONVIFServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	// Read request body
	body := make([]byte, 0)
	if r.Body != nil {
		defer func() { _ = r.Body.Close() }()
		buf := make([]byte, 1024)
		for {
			n, err := r.Body.Read(buf)
			if n > 0 {
				body = append(body, buf[:n]...)
			}
			if err != nil {
				break
			}
		}
	}
	requestBody := string(body)

	// Simple auth check
	if m.authFailed && strings.Contains(requestBody, "UsernameToken") {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Determine action
	var action string
	if strings.Contains(requestBody, "GetDeviceInformation") {
		action = "GetDeviceInformation"
	} else if strings.Contains(requestBody, "GetCapabilities") {
		action = "GetCapabilities"
	} else if strings.Contains(requestBody, "GetProfiles") {
		action = "GetProfiles"
	} else if strings.Contains(requestBody, "GetStreamURI") {
		action = "GetStreamURI"
	} else if strings.Contains(requestBody, "GetStatus") {
		action = "GetStatus"
	} else {
		action = "default"
	}

	response, exists := m.responses[action]
	if !exists {
		response = m.responses["default"]
	}

	w.Header().Set("Content-Type", "application/soap+xml")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(response)) // Writing to ResponseWriter; error is handled by http package
}

func (m *MockONVIFServer) setupDefaultResponses() {
	// GetDeviceInformation response
	m.responses["GetDeviceInformation"] = `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://www.w3.org/2003/05/soap-envelope">
    <soap:Body>
        <tds:GetDeviceInformationResponse xmlns:tds="http://www.onvif.org/ver10/device/wsdl">
            <tds:Manufacturer>Test Camera Inc</tds:Manufacturer>
            <tds:Model>TestCam 3000</tds:Model>
            <tds:FirmwareVersion>1.0.0</tds:FirmwareVersion>
            <tds:SerialNumber>12345</tds:SerialNumber>
            <tds:HardwareId>HW001</tds:HardwareId>
        </tds:GetDeviceInformationResponse>
    </soap:Body>
</soap:Envelope>`

	// GetCapabilities response
	m.responses["GetCapabilities"] = `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://www.w3.org/2003/05/soap-envelope">
    <soap:Body>
        <tds:GetCapabilitiesResponse xmlns:tds="http://www.onvif.org/ver10/device/wsdl">
            <tds:Capabilities>
                <tt:Device xmlns:tt="http://www.onvif.org/ver10/schema">
                    <tt:XAddr>` + m.server.URL + `/onvif/device_service</tt:XAddr>
                </tt:Device>
                <tt:Media xmlns:tt="http://www.onvif.org/ver10/schema">
                    <tt:XAddr>` + m.server.URL + `/onvif/media_service</tt:XAddr>
                </tt:Media>
                <tt:PTZ xmlns:tt="http://www.onvif.org/ver10/schema">
                    <tt:XAddr>` + m.server.URL + `/onvif/ptz_service</tt:XAddr>
                </tt:PTZ>
            </tds:Capabilities>
        </tds:GetCapabilitiesResponse>
    </soap:Body>
</soap:Envelope>`

	// GetProfiles response
	m.responses["GetProfiles"] = `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://www.w3.org/2003/05/soap-envelope">
    <soap:Body>
        <trt:GetProfilesResponse xmlns:trt="http://www.onvif.org/ver10/media/wsdl">
            <trt:Profiles token="Profile1" fixed="true">
                <tt:Name xmlns:tt="http://www.onvif.org/ver10/schema">Main Profile</tt:Name>
                <tt:VideoEncoderConfiguration xmlns:tt="http://www.onvif.org/ver10/schema">
                    <tt:Encoding>H264</tt:Encoding>
                    <tt:Resolution>
                        <tt:Width>1920</tt:Width>
                        <tt:Height>1080</tt:Height>
                    </tt:Resolution>
                </tt:VideoEncoderConfiguration>
            </trt:Profiles>
        </trt:GetProfilesResponse>
    </soap:Body>
</soap:Envelope>`

	// Default fault response
	m.responses["default"] = `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://www.w3.org/2003/05/soap-envelope">
    <soap:Body>
        <soap:Fault>
            <soap:Code>
                <soap:Value>soap:Receiver</soap:Value>
            </soap:Code>
            <soap:Reason>
                <soap:Text>Action not supported in mock</soap:Text>
            </soap:Reason>
        </soap:Fault>
    </soap:Body>
</soap:Envelope>`
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name      string
		endpoint  string
		wantError bool
	}{
		{
			name:      "valid http endpoint",
			endpoint:  "http://192.168.1.100/onvif/device_service",
			wantError: false,
		},
		{
			name:      "valid https endpoint",
			endpoint:  "https://camera.example.com/onvif",
			wantError: false,
		},
		{
			name:      "invalid endpoint",
			endpoint:  "not a url",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.endpoint)
			if (err != nil) != tt.wantError {
				t.Errorf("NewClient() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError && client == nil {
				t.Error("NewClient() returned nil client")
			}
		})
	}
}

func TestClientOptions(t *testing.T) {
	endpoint := "http://192.168.1.100/onvif"

	t.Run("WithCredentials", func(t *testing.T) {
		username := "admin"
		password := "test123"

		client, err := NewClient(endpoint, WithCredentials(username, password))
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}

		gotUser, gotPass := client.GetCredentials()
		if gotUser != username || gotPass != password {
			t.Errorf("GetCredentials() = (%v, %v), want (%v, %v)",
				gotUser, gotPass, username, password)
		}
	})

	t.Run("WithTimeout", func(t *testing.T) {
		timeout := 10 * time.Second
		client, err := NewClient(endpoint, WithTimeout(timeout))
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}

		if client.httpClient.Timeout != timeout {
			t.Errorf("HTTP client timeout = %v, want %v",
				client.httpClient.Timeout, timeout)
		}
	})

	t.Run("WithHTTPClient", func(t *testing.T) {
		customClient := &http.Client{
			Timeout: 5 * time.Second,
		}

		client, err := NewClient(endpoint, WithHTTPClient(customClient))
		if err != nil {
			t.Fatalf("NewClient() error = %v", err)
		}

		if client.httpClient != customClient {
			t.Error("Custom HTTP client not set")
		}
	})
}

func TestClientEndpoint(t *testing.T) {
	endpoint := "http://192.168.1.100/onvif"
	client, err := NewClient(endpoint)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if got := client.Endpoint(); got != endpoint {
		t.Errorf("Endpoint() = %v, want %v", got, endpoint)
	}
}

func TestClientSetCredentials(t *testing.T) {
	client, err := NewClient("http://192.168.1.100/onvif")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	username := "newuser"
	password := "newpass"

	client.SetCredentials(username, password)

	gotUser, gotPass := client.GetCredentials()
	if gotUser != username || gotPass != password {
		t.Errorf("After SetCredentials(), GetCredentials() = (%v, %v), want (%v, %v)",
			gotUser, gotPass, username, password)
	}
}

func TestGetDeviceInformationWithMockServer(t *testing.T) {
	// Simple test server that returns HTTP 200
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/soap+xml")
		w.WriteHeader(http.StatusOK)
		// Return empty response - will cause EOF error which is expected for now
	}))
	defer server.Close()

	client, err := NewClient(
		server.URL,
		WithCredentials("admin", "password"),
	)
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	ctx := context.Background()
	_, err = client.GetDeviceInformation(ctx)
	// We expect an error since we're not returning valid SOAP
	if err == nil {
		t.Errorf("Expected error with empty response, but got none")
	}

	// This test just verifies the client can be created and make requests
	t.Logf("Expected error occurred: %v", err)
}

func TestGetDeviceInformationWithAuth(t *testing.T) {
	// Test unauthorized response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	ctx := context.Background()
	_, err = client.GetDeviceInformation(ctx)
	if err == nil {
		t.Errorf("Expected authentication error, but got none")
	}

	t.Logf("Authentication error (expected): %v", err)
}

func TestInitializeEndpointDiscovery(t *testing.T) {
	// Test that Initialize can handle network errors gracefully
	client, err := NewClient(
		"http://192.168.999.999/onvif/device_service", // non-existent IP
		WithCredentials("admin", "password"),
		WithTimeout(1*time.Second),
	)
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = client.Initialize(ctx)
	// We expect this to fail due to network timeout
	if err == nil {
		t.Errorf("Expected network error, but got none")
	}

	t.Logf("Network error (expected): %v", err)
}

func TestGetProfilesRequiresInitialization(t *testing.T) {
	client, err := NewClient(
		"http://192.168.1.100/onvif/device_service",
		WithCredentials("admin", "password"),
	)
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	ctx := context.Background()
	_, err = client.GetProfiles(ctx)
	// Should fail because Initialize was not called
	if err == nil {
		t.Errorf("Expected error when GetProfiles called without Initialize")
	}

	t.Logf("Expected error: %v", err)
}

func TestContextTimeout(t *testing.T) {
	mock := NewMockONVIFServer()
	defer mock.Close()

	client, err := NewClient(
		mock.URL(),
		WithCredentials("admin", "password"),
	)
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	// Create context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// This should timeout
	_, err = client.GetDeviceInformation(ctx)
	if err == nil {
		t.Errorf("Expected timeout error, but got none")
	}

	if !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("Expected context deadline exceeded error, got: %v", err)
	}
}

func TestONVIFError(t *testing.T) {
	err := NewONVIFError("Sender", "InvalidArgs", "Invalid parameter value")

	if err.Code != "Sender" {
		t.Errorf("Code = %v, want %v", err.Code, "Sender")
	}

	if err.Reason != "InvalidArgs" {
		t.Errorf("Reason = %v, want %v", err.Reason, "InvalidArgs")
	}

	expectedError := "ONVIF error [Sender]: InvalidArgs - Invalid parameter value"
	if err.Error() != expectedError {
		t.Errorf("Error() = %v, want %v", err.Error(), expectedError)
	}

	if !IsONVIFError(err) {
		t.Error("IsONVIFError() returned false for ONVIF error")
	}
}

func BenchmarkNewClient(b *testing.B) {
	endpoint := "http://192.168.1.100/onvif"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := NewClient(endpoint)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetDeviceInformation(b *testing.B) {
	mock := NewMockONVIFServer()
	defer mock.Close()

	client, err := NewClient(
		mock.URL(),
		WithCredentials("admin", "password"),
	)
	if err != nil {
		b.Fatalf("NewClient() failed: %v", err)
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.GetDeviceInformation(ctx)
		if err != nil {
			b.Fatalf("GetDeviceInformation() failed: %v", err)
		}
	}
}

// Example test
func ExampleClient_GetDeviceInformation() {
	// Create client
	client, err := NewClient(
		"http://192.168.1.100/onvif/device_service",
		WithCredentials("admin", "password"),
		WithTimeout(30*time.Second),
	)
	if err != nil {
		panic(err)
	}

	// Get device information
	ctx := context.Background()
	info, err := client.GetDeviceInformation(ctx)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Camera: %s %s\n", info.Manufacturer, info.Model)
	fmt.Printf("Firmware: %s\n", info.FirmwareVersion)
}

func TestFixLocalhostURL(t *testing.T) {
	tests := []struct {
		name        string
		clientURL   string
		serviceURL  string
		expectedURL string
	}{
		{
			name:        "localhost hostname",
			clientURL:   "http://192.168.1.100/onvif/device_service",
			serviceURL:  "http://localhost/onvif/media_service",
			expectedURL: "http://192.168.1.100/onvif/media_service",
		},
		{
			name:        "127.0.0.1 loopback",
			clientURL:   "http://192.168.1.100:8080/onvif/device_service",
			serviceURL:  "http://127.0.0.1/onvif/ptz_service",
			expectedURL: "http://192.168.1.100:8080/onvif/ptz_service",
		},
		{
			name:        "0.0.0.0 address",
			clientURL:   "http://192.168.1.100/onvif/device_service",
			serviceURL:  "http://0.0.0.0/onvif/imaging_service",
			expectedURL: "http://192.168.1.100/onvif/imaging_service",
		},
		{
			name:        "IPv6 loopback",
			clientURL:   "http://192.168.1.100/onvif/device_service",
			serviceURL:  "http://[::1]/onvif/events_service",
			expectedURL: "http://192.168.1.100/onvif/events_service",
		},
		{
			name:        "localhost with different port",
			clientURL:   "http://192.168.1.100/onvif/device_service",
			serviceURL:  "http://localhost:8080/onvif/media_service",
			expectedURL: "http://192.168.1.100:8080/onvif/media_service",
		},
		{
			name:        "valid IP address unchanged",
			clientURL:   "http://192.168.1.100/onvif/device_service",
			serviceURL:  "http://192.168.1.100/onvif/media_service",
			expectedURL: "http://192.168.1.100/onvif/media_service",
		},
		{
			name:        "different valid IP unchanged",
			clientURL:   "http://192.168.1.100/onvif/device_service",
			serviceURL:  "http://192.168.1.50/onvif/media_service",
			expectedURL: "http://192.168.1.50/onvif/media_service",
		},
		{
			name:        "HTTPS localhost",
			clientURL:   "https://192.168.1.100/onvif/device_service",
			serviceURL:  "https://localhost/onvif/media_service",
			expectedURL: "https://192.168.1.100/onvif/media_service",
		},
		{
			name:        "client with port, service localhost no port",
			clientURL:   "http://192.168.1.100:80/onvif/device_service",
			serviceURL:  "http://localhost/onvif/media_service",
			expectedURL: "http://192.168.1.100:80/onvif/media_service",
		},
		{
			name:        "empty service URL",
			clientURL:   "http://192.168.1.100/onvif/device_service",
			serviceURL:  "",
			expectedURL: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				endpoint: tt.clientURL,
			}

			result := client.fixLocalhostURL(tt.serviceURL)
			if result != tt.expectedURL {
				t.Errorf("fixLocalhostURL() = %v, want %v", result, tt.expectedURL)
			}
		})
	}
}

func TestInitializeWithLocalhostURLs(t *testing.T) {
	// Create a mock server
	mock := NewMockONVIFServer()
	defer mock.Close()

	// Set a GetCapabilities response with localhost URLs
	capabilitiesResponse := `<?xml version="1.0" encoding="UTF-8"?>
<SOAP-ENV:Envelope xmlns:SOAP-ENV="http://www.w3.org/2003/05/soap-envelope">
	<SOAP-ENV:Body>
		<tds:GetCapabilitiesResponse xmlns:tds="http://www.onvif.org/ver10/device/wsdl">
			<tds:Capabilities>
				<tt:Media xmlns:tt="http://www.onvif.org/ver10/schema">
					<tt:XAddr>http://localhost:8080/onvif/media_service</tt:XAddr>
				</tt:Media>
				<tt:PTZ xmlns:tt="http://www.onvif.org/ver10/schema">
					<tt:XAddr>http://127.0.0.1/onvif/ptz_service</tt:XAddr>
				</tt:PTZ>
				<tt:Imaging xmlns:tt="http://www.onvif.org/ver10/schema">
					<tt:XAddr>http://0.0.0.0/onvif/imaging_service</tt:XAddr>
				</tt:Imaging>
			</tds:Capabilities>
		</tds:GetCapabilitiesResponse>
	</SOAP-ENV:Body>
</SOAP-ENV:Envelope>`

	mock.SetResponse("GetCapabilities", capabilitiesResponse)

	// Create client pointing to mock server
	client, err := NewClient(
		mock.URL()+"/onvif/device_service",
		WithCredentials("admin", "admin"),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Initialize should fix localhost URLs
	ctx := context.Background()
	err = client.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// Parse the mock server URL to get host
	mockURL, _ := url.Parse(mock.URL())
	expectedHost := mockURL.Host

	// Verify media endpoint was fixed (localhost:8080 should be replaced with mock host)
	if strings.Contains(client.mediaEndpoint, "localhost") {
		t.Errorf("Media endpoint still contains localhost: %v", client.mediaEndpoint)
	}
	if !strings.Contains(client.mediaEndpoint, expectedHost) {
		t.Logf("Media endpoint: %v, Expected to contain: %v", client.mediaEndpoint, expectedHost)
		// The port 8080 from service URL should be preserved
		expectedMediaURL := "http://" + mockURL.Hostname() + ":8080/onvif/media_service"
		if client.mediaEndpoint != expectedMediaURL {
			t.Errorf("Media endpoint = %v, want %v", client.mediaEndpoint, expectedMediaURL)
		}
	}

	// Verify PTZ endpoint was fixed (127.0.0.1 should be replaced with mock host)
	if strings.Contains(client.ptzEndpoint, "127.0.0.1") && !strings.Contains(expectedHost, "127.0.0.1") {
		t.Errorf("PTZ endpoint still contains 127.0.0.1: %v", client.ptzEndpoint)
	}

	// Verify Imaging endpoint was fixed (0.0.0.0 should be replaced with mock host)
	if strings.Contains(client.imagingEndpoint, "0.0.0.0") {
		t.Errorf("Imaging endpoint still contains 0.0.0.0: %v", client.imagingEndpoint)
	}
}
