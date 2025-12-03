package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/0x524a/onvif-go"
)

const (
	version           = "1.0.0"
	defaultTimeoutSec = 30
	maxRetryAttempts  = 10
	retryDelaySec     = 5
	maxIdleTimeoutSec = 90
	unknownStatus     = "Unknown"
)

type CameraReport struct {
	Timestamp       string                  `json:"timestamp"`
	UtilityVersion  string                  `json:"utility_version"`
	ConnectionInfo  ConnectionInfo          `json:"connection_info"`
	DeviceInfo      *DeviceInfoResult       `json:"device_info"`
	Capabilities    *CapabilitiesResult     `json:"capabilities"`
	Profiles        *ProfilesResult         `json:"profiles"`
	StreamURIs      []StreamURIResult       `json:"stream_uris"`
	SnapshotURIs    []SnapshotURIResult     `json:"snapshot_uris"`
	VideoEncoders   []VideoEncoderResult    `json:"video_encoders"`
	ImagingSettings []ImagingSettingsResult `json:"imaging_settings"`
	PTZStatus       []PTZStatusResult       `json:"ptz_status"`
	PTZPresets      []PTZPresetsResult      `json:"ptz_presets"`
	SystemDateTime  *SystemDateTimeResult   `json:"system_datetime"`
	RawResponses    map[string]interface{}  `json:"raw_responses,omitempty"`
	Errors          []ErrorLog              `json:"errors"`
}

type ConnectionInfo struct {
	Endpoint string `json:"endpoint"`
	Username string `json:"username"`
	TestDate string `json:"test_date"`
}

type DeviceInfoResult struct {
	Success      bool                     `json:"success"`
	Data         *onvif.DeviceInformation `json:"data,omitempty"`
	Error        string                   `json:"error,omitempty"`
	ResponseTime string                   `json:"response_time"`
}

type CapabilitiesResult struct {
	Success      bool                `json:"success"`
	Data         *onvif.Capabilities `json:"data,omitempty"`
	Error        string              `json:"error,omitempty"`
	ResponseTime string              `json:"response_time"`
}

type ProfilesResult struct {
	Success      bool             `json:"success"`
	Data         []*onvif.Profile `json:"data,omitempty"`
	Count        int              `json:"count"`
	Error        string           `json:"error,omitempty"`
	ResponseTime string           `json:"response_time"`
}

type StreamURIResult struct {
	ProfileToken string          `json:"profile_token"`
	ProfileName  string          `json:"profile_name"`
	Success      bool            `json:"success"`
	Data         *onvif.MediaURI `json:"data,omitempty"`
	Error        string          `json:"error,omitempty"`
	ResponseTime string          `json:"response_time"`
}

type SnapshotURIResult struct {
	ProfileToken string          `json:"profile_token"`
	ProfileName  string          `json:"profile_name"`
	Success      bool            `json:"success"`
	Data         *onvif.MediaURI `json:"data,omitempty"`
	Error        string          `json:"error,omitempty"`
	ResponseTime string          `json:"response_time"`
}

type VideoEncoderResult struct {
	ProfileToken string                           `json:"profile_token"`
	ProfileName  string                           `json:"profile_name"`
	Success      bool                             `json:"success"`
	Data         *onvif.VideoEncoderConfiguration `json:"data,omitempty"`
	Error        string                           `json:"error,omitempty"`
	ResponseTime string                           `json:"response_time"`
}

type ImagingSettingsResult struct {
	VideoSourceToken string                 `json:"video_source_token"`
	Success          bool                   `json:"success"`
	Data             *onvif.ImagingSettings `json:"data,omitempty"`
	Error            string                 `json:"error,omitempty"`
	ResponseTime     string                 `json:"response_time"`
}

type PTZStatusResult struct {
	ProfileToken string           `json:"profile_token"`
	ProfileName  string           `json:"profile_name"`
	Success      bool             `json:"success"`
	Data         *onvif.PTZStatus `json:"data,omitempty"`
	Error        string           `json:"error,omitempty"`
	ResponseTime string           `json:"response_time"`
}

type PTZPresetsResult struct {
	ProfileToken string             `json:"profile_token"`
	ProfileName  string             `json:"profile_name"`
	Success      bool               `json:"success"`
	Data         []*onvif.PTZPreset `json:"data,omitempty"`
	Count        int                `json:"count"`
	Error        string             `json:"error,omitempty"`
	ResponseTime string             `json:"response_time"`
}

type SystemDateTimeResult struct {
	Success      bool        `json:"success"`
	Data         interface{} `json:"data,omitempty"`
	Error        string      `json:"error,omitempty"`
	ResponseTime string      `json:"response_time"`
}

type ErrorLog struct {
	Operation string `json:"operation"`
	Error     string `json:"error"`
	Timestamp string `json:"timestamp"`
}

var (
	endpoint   = flag.String("endpoint", "", "ONVIF device endpoint (e.g., http://192.168.1.201/onvif/device_service)")
	username   = flag.String("username", "", "ONVIF username")
	password   = flag.String("password", "", "ONVIF password")
	outputDir  = flag.String("output", "./camera-logs", "Output directory for logs")
	timeout    = flag.Int("timeout", 30, "Request timeout in seconds") //nolint:mnd // Default timeout value
	verbose    = flag.Bool("verbose", false, "Verbose output")
	captureXML = flag.Bool("capture-xml", false, "Capture raw SOAP XML traffic and create tar.gz archive")
)

//nolint:funlen,gocognit,gocyclo // Main function has high complexity due to multiple diagnostic operations
func main() {
	flag.Parse()

	fmt.Printf("ONVIF Camera Diagnostic Utility v%s\n", version)
	fmt.Println("========================================")
	fmt.Println()

	// Validate inputs
	if *endpoint == "" || *username == "" || *password == "" {
		fmt.Println("Error: Missing required parameters")
		fmt.Println()
		fmt.Println("Usage:")
		flag.PrintDefaults()
		fmt.Println()
		fmt.Println("Example:")
		fmt.Println("  ./onvif-diagnostics -endpoint " +
			"http://192.168.1.201/onvif/device_service " +
			"-username service -password Service.1234")
		os.Exit(1)
	}

	// Create output directory
	if err := os.MkdirAll(*outputDir, 0750); err != nil { //nolint:mnd // 0750 appropriate for diagnostic output
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Initialize report
	report := &CameraReport{
		Timestamp:      time.Now().Format(time.RFC3339),
		UtilityVersion: version,
		ConnectionInfo: ConnectionInfo{
			Endpoint: *endpoint,
			Username: *username,
			TestDate: time.Now().Format("2006-01-02"),
		},
		Errors:       make([]ErrorLog, 0),
		RawResponses: make(map[string]interface{}),
	}

	// Setup XML capture if requested
	var loggingTransport *LoggingTransport
	var xmlCaptureDir string

	if *captureXML {
		timestamp := time.Now().Format("20060102-150405")
		xmlCaptureDir = filepath.Join(*outputDir, "temp_"+timestamp)
		if err := os.MkdirAll(xmlCaptureDir, 0750); err != nil { //nolint:mnd // 0750 appropriate for diagnostic output
			log.Fatalf("Failed to create XML capture directory: %v", err)
		}

		loggingTransport = &LoggingTransport{
			Transport: &http.Transport{
				MaxIdleConns:        maxRetryAttempts,
				MaxIdleConnsPerHost: retryDelaySec,
				IdleConnTimeout:     maxIdleTimeoutSec * time.Second,
			},
			LogDir:  xmlCaptureDir,
			Counter: 0,
		}

		if *verbose {
			fmt.Printf("📦 XML capture enabled, saving to: %s\n", xmlCaptureDir)
		}
	}

	// Create ONVIF client
	var client *onvif.Client
	var err error

	if loggingTransport != nil {
		httpClient := &http.Client{
			Timeout:   time.Duration(*timeout) * time.Second,
			Transport: loggingTransport,
		}
		client, err = onvif.NewClient(
			*endpoint,
			onvif.WithCredentials(*username, *password),
			onvif.WithHTTPClient(httpClient),
		)
	} else {
		client, err = onvif.NewClient(
			*endpoint,
			onvif.WithCredentials(*username, *password),
			onvif.WithTimeout(time.Duration(*timeout)*time.Second),
		)
	}

	if err != nil {
		log.Fatalf("Failed to create ONVIF client: %v", err)
	}

	ctx := context.Background()

	fmt.Println("Starting diagnostic collection...")
	fmt.Println()

	// Test 1: Get Device Information
	logStepf("1. Getting device information...")
	report.DeviceInfo = testGetDeviceInformation(ctx, client, report)

	// Test 2: Get System Date and Time
	logStepf("2. Getting system date and time...")
	report.SystemDateTime = testGetSystemDateTime(ctx, client, report)

	// Test 3: Get Capabilities
	logStepf("3. Getting capabilities...")
	report.Capabilities = testGetCapabilities(ctx, client, report)

	// Test 4: Initialize (discover services)
	logStepf("4. Discovering service endpoints...")
	if err := client.Initialize(ctx); err != nil {
		logErrorf("Service discovery failed: %v", err)
		report.Errors = append(report.Errors, ErrorLog{
			Operation: "Initialize",
			Error:     err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		})
	} else {
		logSuccessf("Service endpoints discovered")
	}

	// Test 5: Get Profiles
	logStepf("5. Getting media profiles...")
	report.Profiles = testGetProfiles(ctx, client, report)

	// Test 6: Get Stream URIs (for each profile)
	if report.Profiles != nil && report.Profiles.Success {
		logStepf("6. Getting stream URIs for all profiles...")
		report.StreamURIs = testGetStreamURIs(ctx, client, report.Profiles.Data, report)
	}

	// Test 7: Get Snapshot URIs (for each profile)
	if report.Profiles != nil && report.Profiles.Success {
		logStepf("7. Getting snapshot URIs for all profiles...")
		report.SnapshotURIs = testGetSnapshotURIs(ctx, client, report.Profiles.Data, report)
	}

	// Test 8: Get Video Encoder Configurations
	if report.Profiles != nil && report.Profiles.Success {
		logStepf("8. Getting video encoder configurations...")
		report.VideoEncoders = testGetVideoEncoders(ctx, client, report.Profiles.Data, report)
	}

	// Test 9: Get Imaging Settings
	if report.Profiles != nil && report.Profiles.Success {
		logStepf("9. Getting imaging settings...")
		report.ImagingSettings = testGetImagingSettings(ctx, client, report.Profiles.Data, report)
	}

	// Test 10: Get PTZ Status (if PTZ is available)
	if report.Profiles != nil && report.Profiles.Success {
		logStepf("10. Getting PTZ status...")
		report.PTZStatus = testGetPTZStatus(ctx, client, report.Profiles.Data, report)
	}

	// Test 11: Get PTZ Presets (if PTZ is available)
	if report.Profiles != nil && report.Profiles.Success {
		logStepf("11. Getting PTZ presets...")
		report.PTZPresets = testGetPTZPresets(ctx, client, report.Profiles.Data, report)
	}

	// Generate output filename based on device info
	filename := generateFilename(report)
	outputPath := filepath.Join(*outputDir, filename)

	// Save report
	logStepf("Saving diagnostic report...")
	if err := saveReport(report, outputPath); err != nil {
		log.Fatalf("Failed to save report: %v", err)
	}

	// Create XML archive if capture was enabled
	if *captureXML && loggingTransport != nil {
		fmt.Println()
		logStepf("Creating XML capture archive...")

		// Generate archive name based on device info
		var archiveName string
		if report.DeviceInfo != nil && report.DeviceInfo.Success {
			manufacturer := sanitizeFilename(report.DeviceInfo.Data.Manufacturer)
			model := sanitizeFilename(report.DeviceInfo.Data.Model)
			firmware := sanitizeFilename(report.DeviceInfo.Data.FirmwareVersion)
			timestamp := time.Now().Format("20060102-150405")
			archiveName = fmt.Sprintf("%s_%s_%s_xmlcapture_%s.tar.gz", manufacturer, model, firmware, timestamp)
		} else {
			timestamp := time.Now().Format("20060102-150405")
			archiveName = fmt.Sprintf("unknown_device_xmlcapture_%s.tar.gz", timestamp)
		}

		archivePath := filepath.Join(*outputDir, archiveName)

		if err := createTarGz(xmlCaptureDir, archivePath); err != nil {
			logErrorf("Failed to create XML archive: %v", err)
		} else {
			logSuccessf("XML archive created: %s", archiveName)
			logSuccessf("Total SOAP calls captured: %d", loggingTransport.Counter)

			// Remove temporary directory
			if err := os.RemoveAll(xmlCaptureDir); err != nil {
				logErrorf("Warning: Failed to remove temp directory: %v", err)
			}
		}
	}

	fmt.Println()
	fmt.Println("========================================")
	fmt.Printf("✓ Diagnostic collection complete!\n")
	fmt.Printf("  Report saved to: %s\n", outputPath)
	fmt.Printf("  Total errors: %d\n", len(report.Errors))

	if report.DeviceInfo != nil && report.DeviceInfo.Success {
		fmt.Printf("\n  Device: %s %s\n", report.DeviceInfo.Data.Manufacturer, report.DeviceInfo.Data.Model)
		fmt.Printf("  Firmware: %s\n", report.DeviceInfo.Data.FirmwareVersion)
	}

	if report.Profiles != nil && report.Profiles.Success {
		fmt.Printf("  Profiles: %d\n", report.Profiles.Count)
	}

	fmt.Println()
	if *captureXML {
		fmt.Println("Both JSON report and XML capture archive saved to camera-logs/")
		fmt.Println("Share both files for comprehensive analysis.")
	} else {
		fmt.Println("Use -capture-xml flag to also capture raw SOAP XML traffic.")
		fmt.Println("Please share this file for analysis and test creation.")
	}
	fmt.Println("========================================")
}

func testGetDeviceInformation(ctx context.Context, client *onvif.Client, report *CameraReport) *DeviceInfoResult {
	start := time.Now()
	result := &DeviceInfoResult{}

	info, err := client.GetDeviceInformation(ctx)
	result.ResponseTime = time.Since(start).String()

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		logErrorf("Failed: %v", err)
		report.Errors = append(report.Errors, ErrorLog{
			Operation: "GetDeviceInformation",
			Error:     err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		})
	} else {
		result.Success = true
		result.Data = info
		logSuccessf("Manufacturer: %s, Model: %s", info.Manufacturer, info.Model)
	}

	return result
}

func testGetSystemDateTime(ctx context.Context, client *onvif.Client, report *CameraReport) *SystemDateTimeResult {
	start := time.Now()
	result := &SystemDateTimeResult{}

	dateTime, err := client.GetSystemDateAndTime(ctx)
	result.ResponseTime = time.Since(start).String()

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		logErrorf("Failed: %v", err)
		report.Errors = append(report.Errors, ErrorLog{
			Operation: "GetSystemDateAndTime",
			Error:     err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		})
	} else {
		result.Success = true
		result.Data = dateTime
		logSuccessf("Retrieved")
	}

	return result
}

func testGetCapabilities(ctx context.Context, client *onvif.Client, report *CameraReport) *CapabilitiesResult {
	start := time.Now()
	result := &CapabilitiesResult{}

	capabilities, err := client.GetCapabilities(ctx)
	result.ResponseTime = time.Since(start).String()

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		logErrorf("Failed: %v", err)
		report.Errors = append(report.Errors, ErrorLog{
			Operation: "GetCapabilities",
			Error:     err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		})
	} else {
		result.Success = true
		result.Data = capabilities

		services := []string{}
		if capabilities.Device != nil {
			services = append(services, "Device")
		}
		if capabilities.Media != nil {
			services = append(services, "Media")
		}
		if capabilities.PTZ != nil {
			services = append(services, "PTZ")
		}
		if capabilities.Imaging != nil {
			services = append(services, "Imaging")
		}
		if capabilities.Events != nil {
			services = append(services, "Events")
		}
		if capabilities.Analytics != nil {
			services = append(services, "Analytics")
		}

		logSuccessf("Services: %s", strings.Join(services, ", "))
	}

	return result
}

func testGetProfiles(ctx context.Context, client *onvif.Client, report *CameraReport) *ProfilesResult {
	start := time.Now()
	result := &ProfilesResult{}

	profiles, err := client.GetProfiles(ctx)
	result.ResponseTime = time.Since(start).String()

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		logErrorf("Failed: %v", err)
		report.Errors = append(report.Errors, ErrorLog{
			Operation: "GetProfiles",
			Error:     err.Error(),
			Timestamp: time.Now().Format(time.RFC3339),
		})
	} else {
		result.Success = true
		result.Data = profiles
		result.Count = len(profiles)
		logSuccessf("Found %d profile(s)", len(profiles))

		for i, profile := range profiles {
			if *verbose {
				fmt.Printf("   Profile %d: %s (Token: %s)\n", i+1, profile.Name, profile.Token)
				if profile.VideoEncoderConfiguration != nil && profile.VideoEncoderConfiguration.Resolution != nil {
					fmt.Printf("     Resolution: %dx%d, Encoding: %s\n",
						profile.VideoEncoderConfiguration.Resolution.Width,
						profile.VideoEncoderConfiguration.Resolution.Height,
						profile.VideoEncoderConfiguration.Encoding)
				}
			}
		}
	}

	return result
}

func testGetStreamURIs(ctx context.Context, client *onvif.Client, profiles []*onvif.Profile, report *CameraReport) []StreamURIResult {
	results := make([]StreamURIResult, 0)

	for _, profile := range profiles {
		start := time.Now()
		result := StreamURIResult{
			ProfileToken: profile.Token,
			ProfileName:  profile.Name,
		}

		streamURI, err := client.GetStreamURI(ctx, profile.Token)
		result.ResponseTime = time.Since(start).String()

		if err != nil {
			result.Success = false
			result.Error = err.Error()
			if *verbose {
				logErrorf("  Profile %s: %v", profile.Name, err)
			}
			report.Errors = append(report.Errors, ErrorLog{
				Operation: fmt.Sprintf("GetStreamURI[%s]", profile.Token),
				Error:     err.Error(),
				Timestamp: time.Now().Format(time.RFC3339),
			})
		} else {
			result.Success = true
			result.Data = streamURI
			if *verbose {
				logSuccessf("  Profile %s: %s", profile.Name, streamURI.URI)
			}
		}

		results = append(results, result)
	}

	successCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		}
	}
	logSuccessf("Retrieved %d/%d stream URIs", successCount, len(results))

	return results
}

func testGetSnapshotURIs(ctx context.Context, client *onvif.Client, profiles []*onvif.Profile, report *CameraReport) []SnapshotURIResult {
	results := make([]SnapshotURIResult, 0)

	for _, profile := range profiles {
		start := time.Now()
		result := SnapshotURIResult{
			ProfileToken: profile.Token,
			ProfileName:  profile.Name,
		}

		snapshotURI, err := client.GetSnapshotURI(ctx, profile.Token)
		result.ResponseTime = time.Since(start).String()

		if err != nil {
			result.Success = false
			result.Error = err.Error()
			if *verbose {
				logErrorf("  Profile %s: %v", profile.Name, err)
			}
			report.Errors = append(report.Errors, ErrorLog{
				Operation: fmt.Sprintf("GetSnapshotURI[%s]", profile.Token),
				Error:     err.Error(),
				Timestamp: time.Now().Format(time.RFC3339),
			})
		} else {
			result.Success = true
			result.Data = snapshotURI
			if *verbose {
				logSuccessf("  Profile %s: %s", profile.Name, snapshotURI.URI)
			}
		}

		results = append(results, result)
	}

	successCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		}
	}
	logSuccessf("Retrieved %d/%d snapshot URIs", successCount, len(results))

	return results
}

func testGetVideoEncoders(
	ctx context.Context,
	client *onvif.Client,
	profiles []*onvif.Profile,
	report *CameraReport,
) []VideoEncoderResult {
	results := make([]VideoEncoderResult, 0)

	for _, profile := range profiles {
		if profile.VideoEncoderConfiguration == nil {
			continue
		}

		start := time.Now()
		result := VideoEncoderResult{
			ProfileToken: profile.Token,
			ProfileName:  profile.Name,
		}

		config, err := client.GetVideoEncoderConfiguration(ctx, profile.VideoEncoderConfiguration.Token)
		result.ResponseTime = time.Since(start).String()

		if err != nil {
			result.Success = false
			result.Error = err.Error()
			if *verbose {
				logErrorf("  Profile %s: %v", profile.Name, err)
			}
			report.Errors = append(report.Errors, ErrorLog{
				Operation: fmt.Sprintf("GetVideoEncoderConfiguration[%s]", profile.Token),
				Error:     err.Error(),
				Timestamp: time.Now().Format(time.RFC3339),
			})
		} else {
			result.Success = true
			result.Data = config
			if *verbose && config.Resolution != nil && config.RateControl != nil {
				logSuccessf("  Profile %s: %s %dx%d @ %dfps",
					profile.Name, config.Encoding,
					config.Resolution.Width, config.Resolution.Height,
					config.RateControl.FrameRateLimit)
			}
		}

		results = append(results, result)
	}

	successCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		}
	}
	logSuccessf("Retrieved %d/%d video encoder configs", successCount, len(results))

	return results
}

func testGetImagingSettings(
	ctx context.Context,
	client *onvif.Client,
	profiles []*onvif.Profile,
	report *CameraReport,
) []ImagingSettingsResult {
	results := make([]ImagingSettingsResult, 0)
	processed := make(map[string]bool)

	for _, profile := range profiles {
		if profile.VideoSourceConfiguration == nil {
			continue
		}

		token := profile.VideoSourceConfiguration.SourceToken
		if processed[token] {
			continue
		}
		processed[token] = true

		start := time.Now()
		result := ImagingSettingsResult{
			VideoSourceToken: token,
		}

		settings, err := client.GetImagingSettings(ctx, token)
		result.ResponseTime = time.Since(start).String()

		if err != nil {
			result.Success = false
			result.Error = err.Error()
			if *verbose {
				logErrorf("  Video source %s: %v", token, err)
			}
			report.Errors = append(report.Errors, ErrorLog{
				Operation: fmt.Sprintf("GetImagingSettings[%s]", token),
				Error:     err.Error(),
				Timestamp: time.Now().Format(time.RFC3339),
			})
		} else {
			result.Success = true
			result.Data = settings
			if *verbose {
				fmt.Printf("   ✓ Video source %s: Retrieved\n", token)
			}
		}

		results = append(results, result)
	}

	successCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		}
	}
	logSuccessf("Retrieved %d/%d imaging settings", successCount, len(results))

	return results
}

func testGetPTZStatus(
	ctx context.Context,
	client *onvif.Client,
	profiles []*onvif.Profile,
	report *CameraReport,
) []PTZStatusResult {
	results := make([]PTZStatusResult, 0)

	for _, profile := range profiles {
		if profile.PTZConfiguration == nil {
			continue
		}

		start := time.Now()
		result := PTZStatusResult{
			ProfileToken: profile.Token,
			ProfileName:  profile.Name,
		}

		status, err := client.GetStatus(ctx, profile.Token)
		result.ResponseTime = time.Since(start).String()

		if err != nil {
			result.Success = false
			result.Error = err.Error()
			if *verbose {
				logErrorf("  Profile %s: %v", profile.Name, err)
			}
			report.Errors = append(report.Errors, ErrorLog{
				Operation: fmt.Sprintf("GetPTZStatus[%s]", profile.Token),
				Error:     err.Error(),
				Timestamp: time.Now().Format(time.RFC3339),
			})
		} else {
			result.Success = true
			result.Data = status
			if *verbose {
				logSuccessf("  Profile %s: Retrieved", profile.Name)
			}
		}

		results = append(results, result)
	}

	if len(results) == 0 {
		logInfof("No PTZ configurations found")
	} else {
		successCount := 0
		for _, r := range results {
			if r.Success {
				successCount++
			}
		}
		logSuccessf("Retrieved %d/%d PTZ status", successCount, len(results))
	}

	return results
}

func testGetPTZPresets(
	ctx context.Context,
	client *onvif.Client,
	profiles []*onvif.Profile,
	report *CameraReport,
) []PTZPresetsResult {
	results := make([]PTZPresetsResult, 0)

	for _, profile := range profiles {
		if profile.PTZConfiguration == nil {
			continue
		}

		start := time.Now()
		result := PTZPresetsResult{
			ProfileToken: profile.Token,
			ProfileName:  profile.Name,
		}

		presets, err := client.GetPresets(ctx, profile.Token)
		result.ResponseTime = time.Since(start).String()

		if err != nil {
			result.Success = false
			result.Error = err.Error()
			if *verbose {
				logErrorf("  Profile %s: %v", profile.Name, err)
			}
			report.Errors = append(report.Errors, ErrorLog{
				Operation: fmt.Sprintf("GetPTZPresets[%s]", profile.Token),
				Error:     err.Error(),
				Timestamp: time.Now().Format(time.RFC3339),
			})
		} else {
			result.Success = true
			result.Data = presets
			result.Count = len(presets)
			if *verbose {
				logSuccessf("  Profile %s: %d preset(s)", profile.Name, len(presets))
			}
		}

		results = append(results, result)
	}

	if len(results) == 0 {
		logInfof("No PTZ configurations found")
	} else {
		successCount := 0
		totalPresets := 0
		for _, r := range results {
			if r.Success {
				successCount++
				totalPresets += r.Count
			}
		}
		logSuccessf("Retrieved presets from %d/%d PTZ profiles (%d total presets)", successCount, len(results), totalPresets)
	}

	return results
}

func generateFilename(report *CameraReport) string {
	timestamp := time.Now().Format("20060102-150405")

	if report.DeviceInfo != nil && report.DeviceInfo.Success {
		manufacturer := sanitizeFilename(report.DeviceInfo.Data.Manufacturer)
		model := sanitizeFilename(report.DeviceInfo.Data.Model)
		firmware := sanitizeFilename(report.DeviceInfo.Data.FirmwareVersion)

		return fmt.Sprintf("%s_%s_%s_%s.json", manufacturer, model, firmware, timestamp)
	}

	return fmt.Sprintf("unknown_camera_%s.json", timestamp)
}

func sanitizeFilename(s string) string {
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "\\", "-")
	s = strings.ReplaceAll(s, ":", "-")
	s = strings.ReplaceAll(s, "*", "-")
	s = strings.ReplaceAll(s, "?", "-")
	s = strings.ReplaceAll(s, "\"", "-")
	s = strings.ReplaceAll(s, "<", "-")
	s = strings.ReplaceAll(s, ">", "-")
	s = strings.ReplaceAll(s, "|", "-")

	return s
}

func saveReport(report *CameraReport, filename string) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	if err := os.WriteFile(filename, data, 0600); err != nil { //nolint:mnd // 0600 appropriate for diagnostic files
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

//nolint:unparam // args parameter is kept for printf-style consistency, even though currently unused
func logStepf(format string, args ...interface{}) {
	if len(args) > 0 {
		fmt.Printf("→ %s\n", fmt.Sprintf(format, args...))
	} else {
		fmt.Printf("→ %s\n", format)
	}
}

func logSuccessf(format string, args ...interface{}) {
	fmt.Printf("  ✓ %s\n", fmt.Sprintf(format, args...))
}

func logErrorf(format string, args ...interface{}) {
	fmt.Printf("  ✗ %s\n", fmt.Sprintf(format, args...))
}

func logInfof(format string, args ...interface{}) {
	fmt.Printf("  ℹ %s\n", fmt.Sprintf(format, args...))
}

// XML Capture functionality

// XMLCapture stores a request/response pair.
type XMLCapture struct {
	Timestamp     string `json:"timestamp"`
	Operation     int    `json:"operation"`
	OperationName string `json:"operation_name"`
	Endpoint      string `json:"endpoint"`
	RequestBody   string `json:"request_body"`
	ResponseBody  string `json:"response_body"`
	StatusCode    int    `json:"status_code"`
	Error         string `json:"error,omitempty"`
}

// LoggingTransport wraps http.RoundTripper to log requests and responses.
type LoggingTransport struct {
	Transport http.RoundTripper
	LogDir    string
	Counter   int
}

func (t *LoggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.Counter++
	capture := XMLCapture{
		Timestamp: time.Now().Format(time.RFC3339),
		Operation: t.Counter,
		Endpoint:  req.URL.String(),
	}

	// Capture request body
	if req.Body != nil {
		bodyBytes, err := io.ReadAll(req.Body)
		if err == nil {
			capture.RequestBody = string(bodyBytes)
			// Extract operation name from SOAP body
			capture.OperationName = extractSOAPOperation(capture.RequestBody)
			// Restore the body for the actual request
			req.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
		}
	}

	// Make the actual request
	resp, err := t.Transport.RoundTrip(req)
	if err != nil {
		capture.Error = err.Error()
		t.saveCapture(&capture)

		return nil, fmt.Errorf("round trip failed: %w", err)
	}

	// Capture response
	capture.StatusCode = resp.StatusCode
	if resp.Body != nil {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err == nil {
			capture.ResponseBody = string(bodyBytes)
			// Restore the body for the caller
			resp.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
		}
	}

	t.saveCapture(&capture)

	return resp, nil
}

// prettyPrintXML formats XML with proper indentation using a simple algorithm.
func prettyPrintXML(xmlStr string) string {
	if xmlStr == "" {
		return ""
	}

	var formatted bytes.Buffer
	decoder := xml.NewDecoder(strings.NewReader(xmlStr))
	encoder := xml.NewEncoder(&formatted)
	encoder.Indent("", "  ")

	for {
		token, err := decoder.Token()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			// If formatting fails, return original
			return xmlStr
		}

		if err := encoder.EncodeToken(token); err != nil {
			return xmlStr
		}
	}

	if err := encoder.Flush(); err != nil {
		return xmlStr
	}

	return formatted.String()
}

func (t *LoggingTransport) saveCapture(capture *XMLCapture) {
	// Create filename base using operation name
	baseFilename := fmt.Sprintf("capture_%03d_%s", capture.Operation, capture.OperationName)

	// Save as individual JSON file
	filename := filepath.Join(t.LogDir, baseFilename+".json")
	data, err := json.MarshalIndent(capture, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal capture: %v", err)

		return
	}

	if err := os.WriteFile(filename, data, 0600); err != nil { //nolint:mnd // 0600 appropriate for diagnostic files
		log.Printf("Failed to write capture: %v", err)
	}

	// Pretty-print and save XML files for easier viewing
	reqFile := filepath.Join(t.LogDir, baseFilename+"_request.xml")
	prettyRequest := prettyPrintXML(capture.RequestBody)
	if err := os.WriteFile(
		reqFile, []byte(prettyRequest), 0600, //nolint:mnd // 0600 appropriate for diagnostic files
	); err != nil {
		log.Printf("Failed to write request XML: %v", err)
	}

	respFile := filepath.Join(t.LogDir, baseFilename+"_response.xml")
	prettyResponse := prettyPrintXML(capture.ResponseBody)
	if err := os.WriteFile(
		respFile, []byte(prettyResponse), 0600, //nolint:mnd // 0600 appropriate for diagnostic files
	); err != nil {
		log.Printf("Failed to write response XML: %v", err)
	}
}

// extractSOAPOperation extracts the operation name from a SOAP request body.
func extractSOAPOperation(soapBody string) string {
	// Look for the operation element in the SOAP Body
	// Typical format: <Body><GetDeviceInformation xmlns="...">...</GetDeviceInformation></Body>

	// Find the Body element
	bodyStart := strings.Index(soapBody, "<Body")
	if bodyStart == -1 {
		return unknownStatus
	}

	// Find the closing > of the Body opening tag
	bodyOpenEnd := strings.Index(soapBody[bodyStart:], ">")
	if bodyOpenEnd == -1 {
		return unknownStatus
	}
	bodyContentStart := bodyStart + bodyOpenEnd + 1

	// Find the first element after <Body>
	// Skip whitespace and find next <
	for bodyContentStart < len(soapBody) && soapBody[bodyContentStart] <= ' ' {
		bodyContentStart++
	}

	if bodyContentStart >= len(soapBody) || soapBody[bodyContentStart] != '<' {
		return unknownStatus
	}

	// Extract the tag name
	tagStart := bodyContentStart + 1
	tagEnd := tagStart
	for tagEnd < len(soapBody) && soapBody[tagEnd] != ' ' && soapBody[tagEnd] != '>' && soapBody[tagEnd] != '/' {
		tagEnd++
	}

	if tagEnd > tagStart {
		tagName := soapBody[tagStart:tagEnd]
		// Remove namespace prefix if present (e.g., "tds:GetDeviceInformation" -> "GetDeviceInformation")
		if colonIdx := strings.Index(tagName, ":"); colonIdx != -1 {
			return tagName[colonIdx+1:]
		}

		return tagName
	}

	return "Unknown"
}

// createTarGz creates a tar.gz archive from a directory.
func createTarGz(sourceDir, archivePath string) error {
	// Create archive file
	archiveFile, err := os.Create(archivePath) //nolint:gosec // Archive path is validated before use
	if err != nil {
		return fmt.Errorf("failed to create archive file: %w", err)
	}
	defer func() {
		_ = archiveFile.Close()
	}()

	// Create gzip writer
	gzWriter := gzip.NewWriter(archiveFile)
	defer func() {
		_ = gzWriter.Close()
	}()

	// Create tar writer
	tarWriter := tar.NewWriter(gzWriter)
	defer func() {
		_ = tarWriter.Close()
	}()

	// Walk through source directory
	if err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if path == sourceDir {
			return nil
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return fmt.Errorf("failed to create tar header: %w", err)
		}

		// Set name to relative path
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}
		header.Name = relPath

		// Write header
		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write tar header: %w", err)
		}

		// If it's a file, write its content
		if !info.IsDir() {
			file, err := os.Open(path) //nolint:gosec // File path is from filepath.Walk, safe
			if err != nil {
				return fmt.Errorf("failed to open file: %w", err)
			}
			defer func() {
				_ = file.Close()
			}()

			if _, err := io.Copy(tarWriter, file); err != nil {
				return fmt.Errorf("failed to write file to tar: %w", err)
			}
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to walk source directory: %w", err)
	}

	return nil
}
