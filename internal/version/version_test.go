package version

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
		wantErr bool
	}{
		{
			name: "double quote",
			content: `<?php
$VERSION = "1.0.0";
$VERSION_DATE = "2024-01-01 00:00:00";
?>`,
			want: "1.0.0",
		},
		{
			name: "single quote",
			content: `<?php
$VERSION = '2.3.4';
$VERSION_DATE = '2024-01-01 00:00:00';`,
			want: "2.3.4",
		},
		{
			name:    "spaces and tabs",
			content: "\t $VERSION  =  '3.0.0' ; ",
			want:    "3.0.0",
		},
		{
			name: "array style",
			content: `<?
$arModuleVersion = array(
	"VERSION" => "2026.2.0",
	"VERSION_DATE" => "2026-02-25 00:00:00",
);
?>`,
			want: "2026.2.0",
		},
		{
			name: "array style single quote",
			content: `<?
$arModuleVersion = array(
	'VERSION' => '1.2.3',
	'VERSION_DATE' => '2026-02-25 00:00:00',
);
?>`,
			want: "1.2.3",
		},
		{
			name: "no version",
			content: `<?php
$VERSION_DATE = '2024-01-01';
?>`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "version.php")
			err := os.WriteFile(path, []byte(tt.content), 0o644)
			if err != nil {
				t.Fatal(err)
			}
			got, err := ParseVersion(path)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseVersion() error = nil, wantErr true")
				}
				return
			}
			if err != nil {
				t.Errorf("ParseVersion() error = %v, wantErr false", err)
				return
			}
			if got != tt.want {
				t.Errorf("ParseVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBumpVersion(t *testing.T) {
	tests := []struct {
		name      string
		bumpLevel string
		content   string
		wantVer   string
	}{
		{
			name:      "patch",
			bumpLevel: "patch",
			content: `<?php
$VERSION = "1.0.0";
$VERSION_DATE = "2024-01-01 00:00:00";
?>`,
			wantVer: "1.0.1",
		},
		{
			name:      "minor",
			bumpLevel: "minor",
			content: `<?php
$VERSION = '1.0.0';
$VERSION_DATE = "2024-01-01 00:00:00";
?>`,
			wantVer: "1.1.0",
		},
		{
			name:      "major",
			bumpLevel: "major",
			content: `<?php
$VERSION = "0.9.9";
$VERSION_DATE = '2024-01-01 00:00:00';
?>`,
			wantVer: "1.0.0",
		},
		{
			name:      "array style bump",
			bumpLevel: "minor",
			content: `<?
$arModuleVersion = array(
	"VERSION" => "1.2.3",
	"VERSION_DATE" => "2024-01-01 00:00:00",
);
?>`,
			wantVer: "1.3.0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "version.php")
			err := os.WriteFile(path, []byte(tt.content), 0o644)
			if err != nil {
				t.Fatal(err)
			}
			err = BumpVersion(path, tt.bumpLevel)
			if err != nil {
				t.Errorf("BumpVersion() error = %v", err)
				return
			}
			ver, err := ParseVersion(path)
			if err != nil {
				t.Errorf("ParseVersion after bump error = %v", err)
				return
			}
			if ver != tt.wantVer {
				t.Errorf("after bump ver = %q, want %q", ver, tt.wantVer)
			}
			// Check date updated recently
			data, err := os.ReadFile(path)
			if err != nil {
				t.Errorf("read after bump: %v", err)
				return
			}
			if !strings.Contains(string(data), time.Now().Format("2006-01-02")) {
				t.Errorf("date not updated")
			}
		})
	}
}
