package main

import (
	"fmt"
	"log"

	"github.com/edge/storage-cabinet/internal/license"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewDevelopment()

	// Test 1: License disabled
	fmt.Println("=== Test 1: License Disabled ===")
	svc1, err := license.NewService(false, "", "", 0, logger)
	if err != nil {
		log.Fatalf("Failed: %v", err)
	}
	if svc1.IsEnabled() {
		log.Fatal("Expected disabled, but got enabled")
	}
	fmt.Println("✓ License disabled mode works")

	// Test 2: Valid license
	fmt.Println("\n=== Test 2: Valid License ===")
	svc2, err := license.NewService(
		true,
		"/tmp/test_license.lic",
		"/tmp/test_vendor_pubkey.pem",
		72*3600000000000, // 72 hours in nanoseconds
		logger,
	)
	if err != nil {
		log.Fatalf("Failed to load license: %v", err)
	}

	if err := svc2.Check(); err != nil {
		log.Fatalf("License check failed: %v", err)
	}
	fmt.Printf("✓ License valid, max devices: %d\n", svc2.GetMaxDevices())

	// Test 3: License info
	fmt.Println("\n=== Test 3: License Info ===")
	info := svc2.GetLicenseInfo()
	for k, v := range info {
		fmt.Printf("  %s: %v\n", k, v)
	}

	fmt.Println("\n✓ All license tests passed!")
}
