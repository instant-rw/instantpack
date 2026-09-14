package buildkit

import (
	"maps"
	"path/filepath"
	"testing"

	"github.com/moby/buildkit/client"
	"github.com/stretchr/testify/require"
)

func TestGetImageName(t *testing.T) {
	tests := []struct {
		name     string
		appDir   string
		expected string
	}{
		{
			name:     "lowercase directory name",
			appDir:   "/path/to/myapp",
			expected: "myapp",
		},
		{
			name:     "uppercase directory name is lowercased",
			appDir:   "/path/to/SteelMC",
			expected: "steelmc",
		},
		{
			name:     "mixed case directory name is lowercased",
			appDir:   "/path/to/MyApp",
			expected: "myapp",
		},
		{
			name:     "directory name with hyphens",
			appDir:   "/path/to/my-app",
			expected: "my-app",
		},
		{
			name:     "empty path returns fallback",
			appDir:   "",
			expected: "instantpack-app",
		},
		{
			name:     "path ending with separator",
			appDir:   "/path/to/myapp" + string(filepath.Separator),
			expected: "instantpack-app",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getImageName(tt.appDir)
			if got != tt.expected {
				t.Errorf("getImageName(%q) = %q, want %q", tt.appDir, got, tt.expected)
			}
		})
	}
}

func TestExtractCacheType(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedType  string
		expectedAttrs map[string]string
	}{
		{
			name:         "registry cache type is extracted",
			input:        "type=registry,ref=localhost:5555/cache",
			expectedType: "registry",
			expectedAttrs: map[string]string{
				"ref": "localhost:5555/cache",
			},
		},
		{
			name:         "mode max kept in attrs",
			input:        "type=registry,ref=host.docker.internal:7890/node-bun:cache,mode=max",
			expectedType: "registry",
			expectedAttrs: map[string]string{
				"ref":  "host.docker.internal:7890/node-bun:cache",
				"mode": "max",
			},
		},
		{
			name:         "gha shape",
			input:        "type=gha,scope=my-cache",
			expectedType: "gha",
			expectedAttrs: map[string]string{
				"scope": "my-cache",
			},
		},
		{
			name:          "type only",
			input:         "type=registry",
			expectedType:  "registry",
			expectedAttrs: map[string]string{},
		},
		{
			name:         "spaces around keys and values",
			input:        " type = registry , ref = myapp:cache ",
			expectedType: "registry",
			expectedAttrs: map[string]string{
				"ref": "myapp:cache",
			},
		},
		{
			name:         "missing type is not defaulted",
			input:        "scope=my-cache,mode=max",
			expectedType: "",
			expectedAttrs: map[string]string{
				"scope": "my-cache",
				"mode":  "max",
			},
		},
		{
			name:          "empty attrs",
			input:         "",
			expectedType:  "",
			expectedAttrs: map[string]string{},
		},
		{
			name:         "skips segments without equals",
			input:        "type=registry,notavalue,ref=x",
			expectedType: "registry",
			expectedAttrs: map[string]string{
				"ref": "x",
			},
		},
		{
			name:         "value may contain equals",
			input:        "type=registry,ref=image:tag=latest",
			expectedType: "registry",
			expectedAttrs: map[string]string{
				"ref": "image:tag=latest",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, gotAttrs := extractCacheType(parseKeyValue(tt.input))
			if gotType != tt.expectedType {
				t.Errorf("extractCacheType(%q) type = %q, want %q", tt.input, gotType, tt.expectedType)
			}
			if !maps.Equal(gotAttrs, tt.expectedAttrs) {
				t.Errorf("extractCacheType(%q) attrs = %v, want %v", tt.input, gotAttrs, tt.expectedAttrs)
			}
		})
	}
}

func TestCacheEntriesFromFlags(t *testing.T) {
	t.Run("nil slice", func(t *testing.T) {
		require.Nil(t, cacheEntriesFromFlags(nil))
	})

	t.Run("empty slice", func(t *testing.T) {
		require.Nil(t, cacheEntriesFromFlags([]string{}))
	})

	t.Run("skips empty strings", func(t *testing.T) {
		got := cacheEntriesFromFlags([]string{"", "type=registry,ref=x", ""})
		require.Equal(t, []client.CacheOptionsEntry{
			{Type: "registry", Attrs: map[string]string{"ref": "x"}},
		}, got)
	})

	t.Run("multiple registry entries", func(t *testing.T) {
		got := cacheEntriesFromFlags([]string{
			"type=registry,ref=myapp:cache-main",
			"type=registry,ref=myapp:cache-branch",
		})
		require.Equal(t, []client.CacheOptionsEntry{
			{Type: "registry", Attrs: map[string]string{"ref": "myapp:cache-main"}},
			{Type: "registry", Attrs: map[string]string{"ref": "myapp:cache-branch"}},
		}, got)
	})

	t.Run("mixed registry and gha with mode", func(t *testing.T) {
		got := cacheEntriesFromFlags([]string{
			"type=registry,ref=host.docker.internal:7890/node-bun:cache,mode=max",
			"type=gha,scope=ci",
		})
		require.Equal(t, []client.CacheOptionsEntry{
			{
				Type: "registry",
				Attrs: map[string]string{
					"ref":  "host.docker.internal:7890/node-bun:cache",
					"mode": "max",
				},
			},
			{Type: "gha", Attrs: map[string]string{"scope": "ci"}},
		}, got)
	})
}

func TestExportEntriesPushAndInsecure(t *testing.T) {
	entries := exportEntries(BuildWithBuildkitClientOptions{Push: true, InsecureRegistry: true}, "10.0.0.5:7500/app:abc", "{}", nil)
	require.Len(t, entries, 1)
	require.Equal(t, client.ExporterImage, entries[0].Type)
	require.Equal(t, "true", entries[0].Attrs["push"])
	require.Equal(t, "true", entries[0].Attrs["registry.insecure"])
	require.Equal(t, "10.0.0.5:7500/app:abc", entries[0].Attrs["name"])
	require.Nil(t, entries[0].Output)

	secure := exportEntries(BuildWithBuildkitClientOptions{Push: true}, "ghcr.io/x/app:1", "{}", nil)
	_, hasInsecure := secure[0].Attrs["registry.insecure"]
	require.False(t, hasInsecure)

	local := exportEntries(BuildWithBuildkitClientOptions{OutputDir: "/tmp/out"}, "x", "{}", nil)
	require.Equal(t, client.ExporterLocal, local[0].Type)

	docker := exportEntries(BuildWithBuildkitClientOptions{}, "x", "{}", nil)
	require.Equal(t, client.ExporterDocker, docker[0].Type)
	require.NotNil(t, docker[0].Output)
}

func TestWithRegistryInsecure(t *testing.T) {
	entries := cacheEntriesFromFlags([]string{"type=registry,ref=10.0.0.5:7500/app:cache,mode=max", "type=local,dest=/tmp/c"})
	entries = withRegistryInsecure(entries, true)
	require.Equal(t, "true", entries[0].Attrs["registry.insecure"])
	_, local := entries[1].Attrs["registry.insecure"]
	require.False(t, local)
	require.Equal(t, "max", entries[0].Attrs["mode"])
}
