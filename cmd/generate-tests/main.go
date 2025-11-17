package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	onviftesting "github.com/0x524a/onvif-go/testing"
)

var (
	captureArchive = flag.String("capture", "", "Path to XML capture archive (.tar.gz)")
	outputDir      = flag.String("output", "./", "Output directory for generated test file")
	packageName    = flag.String("package", "onvif_test", "Package name for generated test")
)

const testTemplate = `package {{.PackageName}}

import (
	"context"
	"testing"
	"time"

	"github.com/0x524a/onvif-go"
	onviftesting "github.com/0x524a/onvif-go/testing"
)

// Test{{.CameraName}} tests ONVIF client against {{.CameraDescription}} captured responses
func Test{{.CameraName}}(t *testing.T) {
	// Load capture archive (relative to project root)
	captureArchive := "{{.CaptureArchiveRelPath}}"
	
	mockServer, err := onviftesting.NewMockSOAPServer(captureArchive)
	if err != nil {
		t.Fatalf("Failed to create mock server: %v", err)
	}
	defer mockServer.Close()

	// Create ONVIF client pointing to mock server
	client, err := onvif.NewClient(
		mockServer.URL()+"/onvif/device_service",
		onvif.WithCredentials("testuser", "testpass"),
	)
	if err != nil {
		t.Fatalf("Failed to create ONVIF client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("GetDeviceInformation", func(t *testing.T) {
		info, err := client.GetDeviceInformation(ctx)
		if err != nil {
			t.Errorf("GetDeviceInformation failed: %v", err)
			return
		}

		// Validate expected values
		if info.Manufacturer == "" {
			t.Error("Manufacturer is empty")
		}
		if info.Model == "" {
			t.Error("Model is empty")
		}
		if info.FirmwareVersion == "" {
			t.Error("FirmwareVersion is empty")
		}

		t.Logf("Device: %s %s (Firmware: %s)", info.Manufacturer, info.Model, info.FirmwareVersion)
	})

	t.Run("GetSystemDateAndTime", func(t *testing.T) {
		_, err := client.GetSystemDateAndTime(ctx)
		if err != nil {
			t.Errorf("GetSystemDateAndTime failed: %v", err)
		}
	})

	t.Run("GetCapabilities", func(t *testing.T) {
		caps, err := client.GetCapabilities(ctx)
		if err != nil {
			t.Errorf("GetCapabilities failed: %v", err)
			return
		}

		if caps.Device == nil {
			t.Error("Device capabilities is nil")
		}
		if caps.Media == nil {
			t.Error("Media capabilities is nil")
		}

		t.Logf("Capabilities: Device=%v, Media=%v, Imaging=%v, PTZ=%v",
			caps.Device != nil, caps.Media != nil, caps.Imaging != nil, caps.PTZ != nil)
	})

	t.Run("GetProfiles", func(t *testing.T) {
		profiles, err := client.GetProfiles(ctx)
		if err != nil {
			t.Errorf("GetProfiles failed: %v", err)
			return
		}

		if len(profiles) == 0 {
			t.Error("No profiles returned")
		}

		t.Logf("Found %d profile(s)", len(profiles))
		for i, profile := range profiles {
			t.Logf("  Profile %d: %s (Token: %s)", i+1, profile.Name, profile.Token)
		}
	})
{{range .AdditionalTests}}
	t.Run("{{.Name}}", func(t *testing.T) {
		{{.Code}}
	})
{{end}}
}
`

type TestData struct {
	PackageName           string
	CameraName            string
	CameraDescription     string
	CaptureArchiveRelPath string
	AdditionalTests       []AdditionalTest
}

type AdditionalTest struct {
	Name string
	Code string
}

func main() {
	flag.Parse()

	if *captureArchive == "" {
		fmt.Println("Error: -capture flag is required")
		fmt.Println()
		fmt.Println("Usage:")
		flag.PrintDefaults()
		fmt.Println()
		fmt.Println("Example:")
		fmt.Println("  ./generate-tests -capture camera-logs/Bosch_FLEXIDOME_indoor_5100i_IR_8.71.0066_xmlcapture_*.tar.gz")
		os.Exit(1)
	}

	// Load capture to get camera info
	capture, err := onviftesting.LoadCaptureFromArchive(*captureArchive)
	if err != nil {
		log.Fatalf("Failed to load capture: %v", err)
	}

	// Extract camera name from archive filename
	baseName := filepath.Base(*captureArchive)
	// Remove _xmlcapture_timestamp.tar.gz suffix
	parts := strings.Split(baseName, "_xmlcapture_")
	cameraID := parts[0]

	// Convert to valid Go identifier
	cameraName := strings.ReplaceAll(cameraID, "-", "")
	cameraName = strings.ReplaceAll(cameraName, ".", "")
	cameraName = strings.ReplaceAll(cameraName, " ", "")

	// Get device info from first exchange (GetDeviceInformation)
	cameraDesc := cameraID
	if len(capture.Exchanges) > 0 {
		// Try to parse device info from response
		for _, ex := range capture.Exchanges {
			if strings.Contains(ex.RequestBody, "GetDeviceInformation") {
				// Extract manufacturer and model from response
				manufacturer := extractXMLValue(ex.ResponseBody, "Manufacturer")
				model := extractXMLValue(ex.ResponseBody, "Model")
				firmware := extractXMLValue(ex.ResponseBody, "FirmwareVersion")
				if manufacturer != "" && model != "" {
					cameraDesc = fmt.Sprintf("%s %s (Firmware: %s)", manufacturer, model, firmware)
				}
				break
			}
		}
	}

	// Prepare test data
	// Make archive path relative if inside output directory
	relArchivePath := *captureArchive

	// If archive is in a sibling directory to output, make it relative
	if absOutput, err := filepath.Abs(*outputDir); err == nil {
		if absArchive, err := filepath.Abs(*captureArchive); err == nil {
			if rel, err := filepath.Rel(filepath.Dir(absOutput), absArchive); err == nil {
				relArchivePath = rel
			}
		}
	}

	testData := TestData{
		PackageName:           *packageName,
		CameraName:            cameraName,
		CameraDescription:     cameraDesc,
		CaptureArchiveRelPath: relArchivePath,
		AdditionalTests:       []AdditionalTest{},
	}

	// Generate test file
	tmpl, err := template.New("test").Parse(testTemplate)
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}

	// Create output file
	outputFile := filepath.Join(*outputDir, fmt.Sprintf("%s_test.go", strings.ToLower(cameraID)))
	f, err := os.Create(outputFile)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, testData); err != nil {
		log.Fatalf("Failed to execute template: %v", err)
	}

	fmt.Printf("✓ Generated test file: %s\n", outputFile)
	fmt.Printf("  Camera: %s\n", cameraDesc)
	fmt.Printf("  Captured operations: %d\n", len(capture.Exchanges))
	fmt.Println()
	fmt.Println("Run tests with:")
	fmt.Printf("  go test -v %s\n", outputFile)
}

func extractXMLValue(xmlStr, tagName string) string {
	// Simple extraction for basic tags
	start := fmt.Sprintf("<%s>", tagName)
	end := fmt.Sprintf("</%s>", tagName)

	startIdx := strings.Index(xmlStr, start)
	if startIdx == -1 {
		// Try with namespace prefix
		start = fmt.Sprintf(":%s>", tagName)
		startIdx = strings.Index(xmlStr, start)
		if startIdx == -1 {
			return ""
		}
		startIdx += len(start)
	} else {
		startIdx += len(start)
	}

	endIdx := strings.Index(xmlStr[startIdx:], end)
	if endIdx == -1 {
		// Try with namespace prefix
		end = fmt.Sprintf(":/%s>", tagName)
		endIdx = strings.Index(xmlStr[startIdx:], end)
		if endIdx == -1 {
			return ""
		}
	}

	return strings.TrimSpace(xmlStr[startIdx : startIdx+endIdx])
}
