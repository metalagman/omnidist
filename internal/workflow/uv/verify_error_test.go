package uv

import (
	"archive/zip"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/metalagman/omnidist/internal/config"
	"github.com/metalagman/omnidist/internal/paths"
	"github.com/metalagman/omnidist/internal/workflow/shared"
)

func TestVerifyDetectsWheelMismatches(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv(shared.EnvVersionName, "1.0.0")

	cfg := testConfig()
	createDistArtifacts(cfg)
	Stage(cfg, StageOptions{})

	uvDist := cfg.Distributions["uv"]
	version := "1.0.0"
	target := cfg.Targets[0]
	wheelPath, _ := wheelPathForTarget(uvDist, target, version)

	tests := []struct {
		name    string
		mutate  func(entries map[string][]byte)
		wantErr string
	}{
		{
			name: "missing_launcher",
			mutate: func(entries map[string][]byte) {
				distName := shared.NormalizePythonDistributionName(uvDist.Package)
				delete(entries, distName+"/_launcher.py")
			},
			wantErr: "missing launcher",
		},
		{
			name: "missing_metadata",
			mutate: func(entries map[string][]byte) {
				distName := shared.NormalizePythonDistributionName(uvDist.Package)
				delete(entries, distName+"-1.0.0.dist-info/METADATA")
			},
			wantErr: "missing METADATA",
		},
		{
			name: "metadata_package_name_mismatch",
			mutate: func(entries map[string][]byte) {
				distName := shared.NormalizePythonDistributionName(uvDist.Package)
				path := distName + "-1.0.0.dist-info/METADATA"
				entries[path] = []byte("Metadata-Version: 2.1\nName: wrong-name\nVersion: 1.0.0\n")
			},
			wantErr: "package name mismatch in METADATA",
		},
		{
			name: "metadata_version_mismatch",
			mutate: func(entries map[string][]byte) {
				distName := shared.NormalizePythonDistributionName(uvDist.Package)
				path := distName + "-1.0.0.dist-info/METADATA"
				entries[path] = []byte("Metadata-Version: 2.1\nName: omnidist\nVersion: 9.9.9\n")
			},
			wantErr: "version mismatch in METADATA",
		},
		{
			name: "missing_wheel_meta",
			mutate: func(entries map[string][]byte) {
				distName := shared.NormalizePythonDistributionName(uvDist.Package)
				delete(entries, distName+"-1.0.0.dist-info/WHEEL")
			},
			wantErr: "missing WHEEL metadata",
		},
		{
			name: "wheel_meta_tag_mismatch",
			mutate: func(entries map[string][]byte) {
				distName := shared.NormalizePythonDistributionName(uvDist.Package)
				path := distName + "-1.0.0.dist-info/WHEEL"
				entries[path] = []byte("Wheel-Version: 1.0\nGenerator: omnidist\nRoot-Is-Purelib: false\nTag: py3-none-any\n")
			},
			wantErr: "platform tag mismatch in WHEEL metadata",
		},
		{
			name: "missing_entrypoints",
			mutate: func(entries map[string][]byte) {
				distName := shared.NormalizePythonDistributionName(uvDist.Package)
				delete(entries, distName+"-1.0.0.dist-info/entry_points.txt")
			},
			wantErr: "missing entry_points.txt",
		},
		{
			name: "entrypoint_mismatch",
			mutate: func(entries map[string][]byte) {
				distName := shared.NormalizePythonDistributionName(uvDist.Package)
				path := distName + "-1.0.0.dist-info/entry_points.txt"
				entries[path] = []byte("[console_scripts]\nwrong=wrong:main\n")
			},
			wantErr: "console entrypoint mismatch",
		},
		{
			name: "missing_record",
			mutate: func(entries map[string][]byte) {
				distName := shared.NormalizePythonDistributionName(uvDist.Package)
				delete(entries, distName+"-1.0.0.dist-info/RECORD")
			},
			wantErr: "missing RECORD",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Stage(cfg, StageOptions{})
			entries := readWheelEntries(t, wheelPath)
			tt.mutate(entries)
			writeModifiedWheel(t, wheelPath, entries)

			result := Verify(cfg)
			if result.Valid {
				t.Fatalf("Verify().Valid = true, want false")
			}
			assertContainsError(t, result.Errors, tt.wantErr)
		})
	}
}

func TestVerifyDetectsRecordErrors(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv(shared.EnvVersionName, "1.0.0")

	cfg := testConfig()
	createDistArtifacts(cfg)
	Stage(cfg, StageOptions{})

	uvDist := cfg.Distributions["uv"]
	version := "1.0.0"
	target := cfg.Targets[0]
	wheelPath, _ := wheelPathForTarget(uvDist, target, version)
	distName := shared.NormalizePythonDistributionName(uvDist.Package)
	recordPath := distName + "-1.0.0.dist-info/RECORD"

	tests := []struct {
		name    string
		mutate  func(record string) string
		wantErr string
	}{
		{
			name: "record_empty",
			mutate: func(record string) string {
				return ""
			},
			wantErr: "missing RECORD in wheel",
		},
		{
			name: "record_line_invalid",
			mutate: func(record string) string {
				return "invalid-line\n"
			},
			wantErr: "invalid RECORD line",
		},
		{
			name: "record_self_entry_with_hash",
			mutate: func(record string) string {
				lines := strings.Split(record, "\n")
				for i, line := range lines {
					if strings.HasPrefix(line, recordPath) {
						lines[i] = recordPath + ",sha256=abc,123"
					}
				}
				return strings.Join(lines, "\n")
			},
			wantErr: "RECORD self-entry must have empty hash and size",
		},
		{
			name: "record_references_missing_file",
			mutate: func(record string) string {
				return record + "missing_file.py,sha256=abc,123\n"
			},
			wantErr: "references missing file",
		},
		{
			name: "record_hash_mismatch",
			mutate: func(record string) string {
				lines := strings.Split(record, "\n")
				for i, line := range lines {
					if strings.Contains(line, ".py") && !strings.HasPrefix(line, recordPath) {
						parts := strings.Split(line, ",")
						parts[1] = "sha256=wrong"
						lines[i] = strings.Join(parts, ",")
						break
					}
				}
				return strings.Join(lines, "\n")
			},
			wantErr: "RECORD mismatch for",
		},
		{
			name: "record_missing_self_entry",
			mutate: func(record string) string {
				lines := strings.Split(record, "\n")
				newLines := []string{}
				for _, line := range lines {
					if !strings.HasPrefix(line, recordPath) {
						newLines = append(newLines, line)
					}
				}
				return strings.Join(newLines, "\n")
			},
			wantErr: "RECORD missing self-entry",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Stage(cfg, StageOptions{})
			entries := readWheelEntries(t, wheelPath)
			record := string(entries[recordPath])
			entries[recordPath] = []byte(tt.mutate(record))
			writeModifiedWheel(t, wheelPath, entries)

			result := Verify(cfg)
			if result.Valid {
				t.Fatalf("Verify().Valid = true, want false")
			}
			assertContainsError(t, result.Errors, tt.wantErr)
		})
	}
}

func readWheelEntries(t *testing.T, path string) map[string][]byte {
	t.Helper()
	reader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("zip.OpenReader(%q) error = %v", path, err)
	}
	defer reader.Close()

	entries := make(map[string][]byte)
	for _, f := range reader.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("f.Open(%q) error = %v", f.Name, err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("io.ReadAll(%q) error = %v", f.Name, err)
		}
		entries[f.Name] = data
	}
	return entries
}

func writeModifiedWheel(t *testing.T, path string, entries map[string][]byte) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("os.Create(%q) error = %v", path, err)
	}
	defer f.Close()

	w := zip.NewWriter(f)
	for name, data := range entries {
		header := &zip.FileHeader{
			Name:   name,
			Method: zip.Store,
		}
		header.SetMode(0644)
		if strings.Contains(name, "/bin/") {
			header.SetMode(0755)
		}
		writer, err := w.CreateHeader(header)
		if err != nil {
			t.Fatalf("w.CreateHeader(%q) error = %v", name, err)
		}
		if _, err := writer.Write(data); err != nil {
			t.Fatalf("writer.Write(%q) error = %v", name, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("w.Close() error = %v", err)
	}
}

func assertContainsError(t *testing.T, errors []string, want string) {
	t.Helper()
	for _, err := range errors {
		if strings.Contains(err, want) {
			return
		}
	}
	t.Fatalf("Errors %v do not contain %q", errors, want)
}

func TestWriteStagingPyprojectErrors(t *testing.T) {
	err := writeStagingPyproject(" ", "1.0.0")
	if err == nil || !strings.Contains(err.Error(), "package name is empty") {
		t.Fatalf("writeStagingPyproject() with empty name error = %v", err)
	}

	err = writeStagingPyproject("pkg", " ")
	if err == nil || !strings.Contains(err.Error(), "version is empty") {
		t.Fatalf("writeStagingPyproject() with empty version error = %v", err)
	}
}

func TestReadStagingPyprojectVersionErrors(t *testing.T) {
	t.Chdir(t.TempDir())

	// Missing file
	_, err := readStagingPyprojectVersion()
	if !os.IsNotExist(err) {
		t.Fatalf("readStagingPyprojectVersion() missing file error = %v", err)
	}

	// Missing version in file
	os.MkdirAll(paths.UVDir, 0755)
	os.WriteFile(paths.UVPyprojectPath, []byte("[project]\nname = \"pkg\"\n"), 0644)
	_, err = readStagingPyprojectVersion()
	if err == nil || !strings.Contains(err.Error(), "missing project.version") {
		t.Fatalf("readStagingPyprojectVersion() missing version error = %v", err)
	}

	// Empty version in file
	os.WriteFile(paths.UVPyprojectPath, []byte("[project]\nname = \"pkg\"\nversion = \"\"\n"), 0644)
	_, err = readStagingPyprojectVersion()
	if err == nil || !strings.Contains(err.Error(), "missing project.version") {
		t.Fatalf("readStagingPyprojectVersion() empty version error = %v, want missing project.version", err)
	}
}

func TestResolveUVStagingVersionErrors(t *testing.T) {
	// Test 1: invalid pyproject version (not semver/pep440)
	t.Run("invalid_pyproject", func(t *testing.T) {
		t.Chdir(t.TempDir())
		os.MkdirAll(paths.UVDir, 0755)
		os.WriteFile(paths.UVPyprojectPath, []byte("[project]\nname = \"pkg\"\nversion = \"not-pep440\"\n"), 0644)
		got, err := resolveUVStagingVersion(nil, false)
		if err != nil {
			t.Fatalf("resolveUVStagingVersion() error = %v", err)
		}
		if got != "not-pep440" {
			t.Fatalf("resolveUVStagingVersion() = %q, want not-pep440", got)
		}
	})

	// Test 2: fallback when pyproject missing but shared.ResolveStageVersion fails
	t.Run("fallback_fails", func(t *testing.T) {
		t.Chdir(t.TempDir())
		cfg := &config.Config{Version: config.VersionConfig{Source: "file"}}
		_, err := resolveUVStagingVersion(cfg, false)
		if err == nil {
			t.Fatalf("resolveUVStagingVersion() fallback error = nil, want error")
		}
	})
}

func TestStageWheelErrors(t *testing.T) {
	t.Chdir(t.TempDir())
	cfg := testConfig()
	uvDist, _ := uvDistribution(cfg)
	target := cfg.Targets[0]

	// source binary missing
	err := stageWheel(cfg, uvDist, target, "1.0.0")
	if err == nil || !strings.Contains(err.Error(), "read built binary") {
		t.Fatalf("stageWheel() missing binary error = %v", err)
	}
}
