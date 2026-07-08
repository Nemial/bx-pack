package version_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Nemial/bx-pack/internal/version"
)

func FuzzParseVersion(f *testing.F) {
	// Seed corpus with valid cases
	f.Add(`<?php
$VERSION = "1.0.0";
$VERSION_DATE = "2024-01-01 00:00:00";
?>`)
	f.Add(`<?php
$VERSION = '2.3.4';
$VERSION_DATE = '2024-01-01 00:00:00';
?>`)
	f.Add(`<?
$arModuleVersion = array(
	"VERSION" => "2026.2.0",
	"VERSION_DATE" => "2026-02-25 00:00:00",
);
?>`)
	f.Add(`<?php $VERSION = "1.0.0"; ?>`)
	f.Add(`<?php $VERSION = ""; ?>`)
	f.Add(`<?php // no version here ?>`)

	f.Fuzz(func(t *testing.T, content string) {
		dir := t.TempDir()
		path := filepath.Join(dir, "version.php")

		err := os.WriteFile(path, []byte(content), 0o600)
		if err != nil {
			t.Skip("failed to write test file")
		}

		// ParseVersion should not panic
		_, _ = version.ParseVersion(path)
	})
}

func FuzzBumpVersion(f *testing.F) {
	// Seed corpus with valid version files
	f.Add(`<?php $VERSION = "1.0.0"; ?>`, "semver", "patch")
	f.Add(`<?php $VERSION = "1.2.3"; ?>`, "semver", "minor")
	f.Add(`<?php $VERSION = "2.0.0"; ?>`, "semver", "major")
	f.Add(`<?php $VERSION = "2026.1.0"; ?>`, "calver", "patch")
	f.Add(`<?php $VERSION = "2026.2.3"; ?>`, "year-semver", "patch")

	f.Fuzz(func(t *testing.T, content, scheme, bumpLevel string) {
		dir := t.TempDir()
		path := filepath.Join(dir, "version.php")

		err := os.WriteFile(path, []byte(content), 0o600)
		if err != nil {
			t.Skip("failed to write test file")
		}

		// BumpVersion should not panic
		_, _, _ = version.BumpVersion(path, scheme, bumpLevel)
	})
}
