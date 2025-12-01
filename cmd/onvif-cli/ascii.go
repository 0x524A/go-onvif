package main

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"strings"
)

// ASCIIConfig controls ASCII art generation parameters
type ASCIIConfig struct {
	Width   int    // Output width in characters
	Height  int    // Output height in characters
	Invert  bool   // Invert brightness
	Quality string // "high", "medium", "low"
}

// DefaultASCIIConfig returns a sensible default configuration
func DefaultASCIIConfig() ASCIIConfig {
	return ASCIIConfig{
		Width:   120,
		Height:  40,
		Invert:  false,
		Quality: "medium",
	}
}

// ASCIICharsets define different character options
var (
	// Full charset with many shades
	charsetFull = []rune{' ', '.', ':', '-', '=', '+', '*', '#', '%', '@'}

	// Medium charset - balanced
	charsetMedium = []rune{' ', '.', '-', '=', '+', '#', '@'}

	// Simple charset - just a few chars
	charsetSimple = []rune{' ', '-', '#', '@'}

	// Block charset - using block characters
	charsetBlock = []rune{' ', '░', '▒', '▓', '█'}

	// Detailed charset
	charsetDetailed = []rune{' ', '`', '.', ',', ':', ';', '!', 'i', 'l', 'I',
		'o', 'O', '0', 'e', 'E', 'p', 'P', 'x', 'X', '$', 'D', 'W', 'M', '@', '#'}
)

// ImageToASCII converts image bytes to ASCII art
// Supports JPEG and PNG formats
func ImageToASCII(imageData []byte, config ASCIIConfig) (string, error) {
	// Decode image from bytes
	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return "", fmt.Errorf("failed to decode image: %w", err)
	}

	return imageToASCIIFromImage(img, config, "unknown")
}

// imageToASCIIFromImage is the core conversion function
func imageToASCIIFromImage(img image.Image, config ASCIIConfig, format string) (string, error) {
	// Validate configuration
	if config.Width <= 0 {
		config.Width = 120
	}
	if config.Height <= 0 {
		config.Height = 40
	}
	if config.Quality == "" {
		config.Quality = "medium"
	}

	// Select character set based on quality
	charset := charsetMedium
	switch strings.ToLower(config.Quality) {
	case "high", "detailed":
		charset = charsetDetailed
	case "medium":
		charset = charsetMedium
	case "low", "simple":
		charset = charsetSimple
	case "block":
		charset = charsetBlock
	case "full":
		charset = charsetFull
	}

	// Get image bounds
	bounds := img.Bounds()
	width := bounds.Max.X - bounds.Min.X
	height := bounds.Max.Y - bounds.Min.Y

	// Calculate scaling factors
	scaleX := float64(width) / float64(config.Width)
	scaleY := float64(height) / float64(config.Height)

	// Build ASCII representation
	var result strings.Builder
	for y := 0; y < config.Height; y++ {
		for x := 0; x < config.Width; x++ {
			// Sample pixel from image
			srcX := int(float64(x) * scaleX)
			srcY := int(float64(y) * scaleY)

			// Bounds check
			if srcX >= width {
				srcX = width - 1
			}
			if srcY >= height {
				srcY = height - 1
			}

			// Get pixel color
			r, g, b, _ := img.At(bounds.Min.X+srcX, bounds.Min.Y+srcY).RGBA()

			// Convert to grayscale brightness (0-255)
			brightness := calculateBrightness(r, g, b)

			// Invert if requested
			if config.Invert {
				brightness = 255 - brightness
			}

			// Map brightness to character
			charIndex := int(float64(brightness) / 255.0 * float64(len(charset)-1))
			if charIndex >= len(charset) {
				charIndex = len(charset) - 1
			}
			if charIndex < 0 {
				charIndex = 0
			}

			result.WriteRune(charset[charIndex])
		}
		result.WriteRune('\n')
	}

	return result.String(), nil
}

// calculateBrightness converts RGB to brightness (0-255)
// Uses standard luminance formula
func calculateBrightness(r, g, b uint32) int {
	// Convert 16-bit color to 8-bit
	r8 := uint8(r >> 8)
	g8 := uint8(g >> 8)
	b8 := uint8(b >> 8)

	// Use standard brightness calculation
	// https://en.wikipedia.org/wiki/Relative_luminance
	brightness := int(0.299*float64(r8) + 0.587*float64(g8) + 0.114*float64(b8))

	if brightness > 255 {
		brightness = 255
	}
	if brightness < 0 {
		brightness = 0
	}

	return brightness
}

// FormatASCIIOutput formats ASCII art with header and footer info
func FormatASCIIOutput(ascii string, imageInfo ImageInfo) string {
	var result strings.Builder

	// Header
	result.WriteString("\n")
	result.WriteString("╔════════════════════════════════════════════════════════════════╗\n")
	result.WriteString("║                    📷 CAMERA SNAPSHOT (ASCII)                    ║\n")
	result.WriteString("╚════════════════════════════════════════════════════════════════╝\n")
	result.WriteString("\n")

	// Image info
	if imageInfo.Width > 0 && imageInfo.Height > 0 {
		result.WriteString(fmt.Sprintf("📊 Original: %dx%d pixels\n", imageInfo.Width, imageInfo.Height))
	}
	if imageInfo.SizeBytes > 0 {
		result.WriteString(fmt.Sprintf("💾 Size: %s\n", formatBytes(imageInfo.SizeBytes)))
	}
	if imageInfo.CaptureTime != "" {
		result.WriteString(fmt.Sprintf("⏱️  Captured: %s\n", imageInfo.CaptureTime))
	}
	if imageInfo.Format != "" {
		result.WriteString(fmt.Sprintf("📁 Format: %s\n", imageInfo.Format))
	}
	result.WriteString("\n")

	// ASCII art
	result.WriteString(ascii)

	// Footer
	result.WriteString("\n")
	result.WriteString("╔════════════════════════════════════════════════════════════════╗\n")
	result.WriteString("💡 Tip: Higher resolution snapshots show better detail\n")
	result.WriteString("╚════════════════════════════════════════════════════════════════╝\n")

	return result.String()
}

// ImageInfo holds metadata about the snapshot
type ImageInfo struct {
	Width       int    // Original width in pixels
	Height      int    // Original height in pixels
	SizeBytes   int64  // File size in bytes
	Format      string // Image format (JPEG, PNG, etc)
	CaptureTime string // Capture timestamp
}

// formatBytes converts bytes to human-readable format
func formatBytes(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
}

// CreateASCIIHighQuality creates a high-quality ASCII representation
func CreateASCIIHighQuality(imageData []byte) (string, error) {
	config := ASCIIConfig{
		Width:   160,
		Height:  50,
		Invert:  false,
		Quality: "high",
	}
	return ImageToASCII(imageData, config)
}
