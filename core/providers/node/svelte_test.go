package node

import (
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"testing"

	"github.com/railwayapp/railpack/core/generate"
	testingUtils "github.com/railwayapp/railpack/core/testing"
	"github.com/stretchr/testify/require"
)

func TestSvelteKitAdapterAuto(t *testing.T) {
	ctx := testingUtils.CreateGenerateContext(t, "../../../examples/node-svelte-kit")
	provider := NodeProvider{}
	require.NoError(t, provider.Initialize(ctx))
	require.NoError(t, provider.Plan(ctx))

	require.Equal(t, DefaultSvelteKitStartCommand, ctx.Deploy.StartCmd)
	require.Equal(t, "true", buildVariables(t, ctx)["GCP_BUILDPACKS"])
}

func TestSvelteKitAdapterNodeStartsBuiltServer(t *testing.T) {
	tests := []struct {
		name   string
		file   string
		config string
		want   string
	}{
		{
			name:   "default output directory",
			file:   "svelte.config.js",
			config: "import adapter from '@sveltejs/adapter-node';\nexport default { kit: { adapter: adapter() } };\n",
			want:   "node build",
		},
		{
			name:   "custom output directory",
			file:   "svelte.config.js",
			config: "import adapter from \"@sveltejs/adapter-node\";\nexport default { kit: { outDir: '.svelte-kit', adapter: adapter({ out: 'server' }) } };\n",
			want:   "node server",
		},
		{
			name:   "SvelteKit 3 options in the Vite config",
			file:   "vite.config.ts",
			config: "import adapter from '@sveltejs/adapter-node';\nimport { sveltekit } from '@sveltejs/kit/vite';\nimport { defineConfig } from 'vite';\n\nexport default defineConfig({ build: { outDir: 'dist' }, plugins: [sveltekit({ adapter: adapter({ out: 'server' }) })] });\n",
			want:   "node server",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := writeSvelteKitApp(t, map[string]string{"@sveltejs/adapter-node": "^5.0.0"}, map[string]string{tt.file: tt.config})
			_, ctx := planSvelteKitApp(t, dir)

			require.Equal(t, tt.want, ctx.Deploy.StartCmd)
			require.NotContains(t, buildVariables(t, ctx), "GCP_BUILDPACKS")
		})
	}
}

// `sv create` installs adapter-auto, and apps that later switch adapters often keep it in package.json.
func TestSvelteKitAdapterImportedByConfigWinsOverInstalledAdapterAuto(t *testing.T) {
	dir := writeSvelteKitApp(t,
		map[string]string{"@sveltejs/adapter-auto": "^6.0.0", "@sveltejs/adapter-cloudflare": "^7.0.0"},
		map[string]string{"svelte.config.js": "// import adapter from '@sveltejs/adapter-auto';\nimport adapter from '@sveltejs/adapter-cloudflare';\nexport default { kit: { adapter: adapter() } };\n"},
	)
	provider, ctx := planSvelteKitApp(t, dir)

	require.Equal(t, "@sveltejs/adapter-cloudflare", provider.svelteKitAdapter)
	require.Empty(t, ctx.Deploy.StartCmd)
	require.NotContains(t, buildVariables(t, ctx), "GCP_BUILDPACKS")
}

func TestSvelteKitAdapterFromPackageJsonWithoutConfig(t *testing.T) {
	dir := writeSvelteKitApp(t, map[string]string{"@sveltejs/adapter-node": "^5.0.0"}, nil)
	_, ctx := planSvelteKitApp(t, dir)

	require.Equal(t, DefaultSvelteKitStartCommand, ctx.Deploy.StartCmd)
}

func TestSvelteKitStartCommandHelpNamesAdapterAndFix(t *testing.T) {
	tests := []struct {
		name     string
		adapter  string
		file     string
		config   string
		contains []string
	}{
		{
			name:     "adapter for another host",
			adapter:  "@sveltejs/adapter-cloudflare",
			file:     "svelte.config.js",
			config:   "import adapter from '@sveltejs/adapter-cloudflare';\nexport default { kit: { adapter: adapter() } };\n",
			contains: []string{"@sveltejs/adapter-cloudflare", "Cloudflare", "@sveltejs/adapter-node in package.json and svelte.config.js", "`node build`"},
		},
		{
			name:     "adapter for another host in a SvelteKit 3 Vite config",
			adapter:  "@sveltejs/adapter-vercel",
			file:     "vite.config.js",
			config:   "import adapter from '@sveltejs/adapter-vercel';\nimport { sveltekit } from '@sveltejs/kit/vite';\nexport default { plugins: [sveltekit({ adapter: adapter() })] };\n",
			contains: []string{"@sveltejs/adapter-vercel", "Vercel", "@sveltejs/adapter-node in package.json and vite.config.js"},
		},
		{
			name:     "static adapter",
			adapter:  "@sveltejs/adapter-static",
			file:     "svelte.config.js",
			config:   "import adapter from '@sveltejs/adapter-static';\nexport default { kit: { adapter: adapter({ pages: 'public' }) } };\n",
			contains: []string{"@sveltejs/adapter-static", "INSTANTPACK_SPA_OUTPUT_DIR=public", "@sveltejs/adapter-node"},
		},
		{
			name:     "community adapter",
			adapter:  "svelte-adapter-bun",
			file:     "svelte.config.js",
			config:   "import adapter from 'svelte-adapter-bun';\nexport default { kit: { adapter: adapter() } };\n",
			contains: []string{"svelte-adapter-bun", "\"start\" script", "@sveltejs/adapter-node"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := writeSvelteKitApp(t, map[string]string{tt.adapter: "*"}, map[string]string{tt.file: tt.config})
			provider, ctx := planSvelteKitApp(t, dir)

			require.Empty(t, ctx.Deploy.StartCmd)
			help := provider.StartCommandHelp()
			for _, part := range tt.contains {
				require.Contains(t, help, part)
			}
		})
	}
}

func writeSvelteKitApp(t *testing.T, adapters map[string]string, configFiles map[string]string) string {
	t.Helper()
	dir := t.TempDir()

	devDependencies := map[string]string{"@sveltejs/kit": "^2.0.0", "svelte": "^5.0.0", "vite": "^7.0.0"}
	maps.Copy(devDependencies, adapters)
	packageJson, err := json.Marshal(map[string]any{
		"name":            "app",
		"type":            "module",
		"scripts":         map[string]string{"build": "vite build"},
		"devDependencies": devDependencies,
	})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), packageJson, 0o644))

	for name, contents := range configFiles {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(contents), 0o644))
	}
	return dir
}

func planSvelteKitApp(t *testing.T, dir string) (*NodeProvider, *generate.GenerateContext) {
	t.Helper()
	ctx := testingUtils.CreateGenerateContext(t, dir)
	provider := &NodeProvider{}
	require.NoError(t, provider.Initialize(ctx))
	require.NoError(t, provider.Plan(ctx))
	return provider, ctx
}

func buildVariables(t *testing.T, ctx *generate.GenerateContext) map[string]string {
	t.Helper()
	step := ctx.GetStepByName("build")
	require.NotNil(t, step)
	build, ok := (*step).(*generate.CommandStepBuilder)
	require.True(t, ok)
	return build.Variables
}
