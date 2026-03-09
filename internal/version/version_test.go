package version

import (
	"os"
	"path/filepath"
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
		scheme    string
		bumpLevel string
		content   string
		wantVer   string
		wantErr   bool
	}{
		{
			name:      "semver patch",
			scheme:    "semver",
			bumpLevel: "patch",
			content:   `<?php $VERSION = "1.0.0"; ?>`,
			wantVer:   "1.0.1",
		},
		{
			name:      "semver minor",
			scheme:    "semver",
			bumpLevel: "minor",
			content:   `<?php $VERSION = "1.0.5"; ?>`,
			wantVer:   "1.1.0",
		},
		{
			name:      "semver major",
			scheme:    "semver",
			bumpLevel: "major",
			content:   `<?php $VERSION = "1.2.3"; ?>`,
			wantVer:   "2.0.0",
		},
		{
			name:      "calver patch same month",
			scheme:    "calver",
			bumpLevel: "patch",
			content:   `<?php $VERSION = "` + time.Now().Format("2006.1") + `.0"; ?>`,
			wantVer:   time.Now().Format("2006.1") + ".1",
		},
		{
			name:      "calver patch different month",
			scheme:    "calver",
			bumpLevel: "patch",
			content:   `<?php $VERSION = "2020.1.5"; ?>`,
			wantVer:   time.Now().Format("2006.1") + ".0",
		},
		{
			name:      "calver minor error",
			scheme:    "calver",
			bumpLevel: "minor",
			content:   `<?php $VERSION = "2020.1.5"; ?>`,
			wantErr:   true,
		},
		{
			name:      "year-semver patch same year",
			scheme:    "year-semver",
			bumpLevel: "patch",
			content:   `<?php $VERSION = "` + time.Now().Format("2006") + `.2.3"; ?>`,
			wantVer:   time.Now().Format("2006") + ".2.4",
		},
		{
			name:      "year-semver minor same year",
			scheme:    "year-semver",
			bumpLevel: "minor",
			content:   `<?php $VERSION = "` + time.Now().Format("2006") + `.2.3"; ?>`,
			wantVer:   time.Now().Format("2006") + ".3.0",
		},
		{
			name:      "year-semver new year",
			scheme:    "year-semver",
			bumpLevel: "patch",
			content:   `<?php $VERSION = "2020.2.3"; ?>`,
			wantVer:   time.Now().Format("2006") + ".1.0",
		},
		{
			name:      "year-semver major error",
			scheme:    "year-semver",
			bumpLevel: "major",
			content:   `<?php $VERSION = "2020.2.3"; ?>`,
			wantErr:   true,
		},
		{
			name:      "custom bump error",
			scheme:    "custom",
			bumpLevel: "patch",
			content:   `<?php $VERSION = "1.2.3"; ?>`,
			wantErr:   true,
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
			_, new, err := BumpVersion(path, tt.scheme, tt.bumpLevel)
			if tt.wantErr {
				if err == nil {
					t.Errorf("BumpVersion() error = nil, wantErr true")
				}
				return
			}
			if err != nil {
				t.Errorf("BumpVersion() error = %v, wantErr false", err)
				return
			}
			if new != tt.wantVer {
				t.Errorf("BumpVersion() new = %q, want %q", new, tt.wantVer)
			}
		})
	}
}
