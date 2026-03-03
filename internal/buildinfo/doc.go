/*
Copyright © 2026 Nicholas Fedor <nick@nickfedor.com>
SPDX-License-Identifier: AGPL-3.0-or-later
*/

// Package buildinfo provides functionality for extracting module metadata
// from Go binaries using the debug/buildinfo package.
//
// This package enables go-remove to capture complete build information
// from Go binaries before deletion, supporting potential reinstallation
// through source reconstruction. It extracts module path, version,
// VCS revision, build settings, and computes SHA256 checksums.
//
// The package supports Linux and Windows platforms and handles various
// version formats including semantic versions (v1.2.3),
// pseudo-versions (v0.0.0-20260302120000-abc123), and development builds ((devel)).
//
// Usage:
//
//	extractor := buildinfo.NewExtractor()
//
//	// Check if file is a Go binary
//	if !extractor.IsGoBinary("/path/to/binary") {
//	    return fmt.Errorf("not a Go binary")
//	}
//
//	// Extract build information
//	ctx := context.Background()
//	info, err := extractor.Extract(ctx, "/path/to/binary")
//	if err != nil {
//	    return err
//	}
//
//	// Calculate checksum
//	checksum, err := extractor.CalculateChecksum("/path/to/binary")
//	if err != nil {
//	    return err
//	}
//
//	fmt.Printf("Module: %s@%s\n", info.ModulePath, info.Version)
//	fmt.Printf("VCS: %s@%s\n", info.VCSRevision, info.VCSTime)
//	fmt.Printf("Checksum: %s\n", checksum)
package buildinfo
