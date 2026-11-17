package node

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/railwayapp/railpack/core/branding"
	"github.com/railwayapp/railpack/core/generate"
)

const (
	DefaultSvelteKitStartCommand    = "node build"
	DefaultSvelteKitOutputDirectory = "build"

	svelteKitAdapterAuto   = "@sveltejs/adapter-auto"
	svelteKitAdapterNode   = "@sveltejs/adapter-node"
	svelteKitAdapterStatic = "@sveltejs/adapter-static"
)

var (
	// SvelteKit 3 takes its options, adapter included, from the sveltekit() Vite plugin (possible since 2.62) and ignores
	// svelte.config. Older Vite configs never import an adapter, so they fall through to svelte.config.
	// https://svelte.dev/docs/kit/@sveltejs-kit-vite
	svelteKitConfigFiles = []string{
		"vite.config.ts", "vite.config.js", "vite.config.mts", "vite.config.mjs",
		"svelte.config.js", "svelte.config.ts", "svelte.config.mjs", "svelte.config.cjs",
	}

	// An adapter the config imports or requires. Anchoring `import ... from` to the start of a line skips
	// commented-out imports, which are common after switching adapters.
	svelteKitAdapterImportRegex = regexp.MustCompile(`(?m)(?:^\s*import\s[^'"]*?from\s*|\b(?:require|import)\(\s*)['"]((?:@sveltejs/adapter-|svelte-adapter-)[\w.-]+)['"]`)

	// adapter-node's `out` and adapter-static's `pages` options, both `build` by default. `\b` keeps longer keys such as
	// `outDir` or `timeout` from matching.
	svelteKitNodeOutRegex     = regexp.MustCompile(`\bout\s*:\s*['"]([^'"]+)['"]`)
	svelteKitStaticPagesRegex = regexp.MustCompile(`\bpages\s*:\s*['"]([^'"]+)['"]`)

	// Adapters that build the app for another host's runtime instead of as a Node server.
	svelteKitHostAdapters = map[string]string{
		"@sveltejs/adapter-cloudflare":         "Cloudflare",
		"@sveltejs/adapter-cloudflare-workers": "Cloudflare Workers",
		"@sveltejs/adapter-netlify":            "Netlify",
		"@sveltejs/adapter-vercel":             "Vercel",
	}
)

// The adapter decides what `vite build` emits, and the config names the one in use; package.json often still lists the
// adapter-auto that `sv create` added after the app switched to another adapter.
func (p *NodeProvider) detectSvelteKitAdapter(ctx *generate.GenerateContext) {
	for _, name := range svelteKitConfigFiles {
		if !ctx.App.HasFile(name) {
			continue
		}
		contents, err := ctx.App.ReadFile(name)
		if err != nil {
			continue
		}
		if match := svelteKitAdapterImportRegex.FindStringSubmatch(contents); match != nil {
			p.svelteKitAdapter, p.svelteKitConfigFile, p.svelteKitConfig = match[1], name, contents
			return
		}
	}

	for _, adapter := range []string{svelteKitAdapterNode, svelteKitAdapterAuto} {
		if p.hasDependency(adapter) {
			p.svelteKitAdapter = adapter
			return
		}
	}

	installed := slices.Concat(slices.Collect(maps.Keys(p.packageJson.Dependencies)), slices.Collect(maps.Keys(p.packageJson.DevDependencies)))
	slices.Sort(installed)
	for _, name := range installed {
		if strings.HasPrefix(name, "@sveltejs/adapter-") {
			p.svelteKitAdapter = name
			return
		}
	}
}

func (p *NodeProvider) configureSvelteKit(ctx *generate.GenerateContext, build *generate.CommandStepBuilder) {
	if p.svelteKitAdapter != svelteKitAdapterAuto {
		return
	}

	// Causes @sveltejs/adapter-auto to select adapter-node, so the build emits a Node server.
	// Oddly, GCP_BUILDPACKS is the only env var adapter-auto recognizes for Node; there is no generic/Railway signal.
	ctx.Logger.LogInfo("SvelteKit with adapter-auto detected, forcing adapter-node")
	build.AddVariables(map[string]string{"GCP_BUILDPACKS": "true"})
}

func (p *NodeProvider) getSvelteKitStartCommand() string {
	switch p.svelteKitAdapter {
	case svelteKitAdapterAuto:
		return DefaultSvelteKitStartCommand
	case svelteKitAdapterNode:
		if match := svelteKitNodeOutRegex.FindStringSubmatch(p.svelteKitConfig); match != nil {
			return "node " + match[1]
		}
		return DefaultSvelteKitStartCommand
	default:
		return ""
	}
}

// Names the adapter that left a SvelteKit app without a server to start, and the change that gives it one.
func (p *NodeProvider) svelteKitStartCommandHelp() string {
	config := p.svelteKitConfigFile
	if config == "" {
		config = "the SvelteKit config"
	}
	useNode := fmt.Sprintf("replace it with %s in package.json and %s, and the app will start with `%s`.", svelteKitAdapterNode, config, DefaultSvelteKitStartCommand)

	if host, ok := svelteKitHostAdapters[p.svelteKitAdapter]; ok {
		return fmt.Sprintf("This SvelteKit app uses %s, which builds it for %s rather than as a Node server.\nTo deploy it here, %s", p.svelteKitAdapter, host, useNode)
	}

	switch p.svelteKitAdapter {
	case svelteKitAdapterStatic:
		pages := DefaultSvelteKitOutputDirectory
		if match := svelteKitStaticPagesRegex.FindStringSubmatch(p.svelteKitConfig); match != nil {
			pages = match[1]
		}
		return fmt.Sprintf("This SvelteKit app uses %s, which builds static files rather than a Node server.\nSet %s=%s to serve those files, or %s", p.svelteKitAdapter, branding.ConfigEnv(OUTPUT_DIR_VAR), pages, useNode)
	case "":
		return fmt.Sprintf("No SvelteKit adapter was found.\nAdd %s to package.json and %s, and the app will start with `%s`.", svelteKitAdapterNode, config, DefaultSvelteKitStartCommand)
	default:
		return fmt.Sprintf("This SvelteKit app uses %s, which %s cannot start by itself.\nAdd a \"start\" script to package.json that runs the server it builds, or %s", p.svelteKitAdapter, branding.DisplayName(), useNode)
	}
}

func (p *NodeProvider) isSvelteKit() bool {
	return p.isSvelteKitPackage(p.workspace.Root)
}

func (p *NodeProvider) isSvelteKitPackage(pkg *WorkspacePackage) bool {
	if pkg == nil || pkg.PackageJson == nil {
		return false
	}

	return pkg.PackageJson.hasDependency("svelte") && pkg.PackageJson.hasDependency("@sveltejs/kit")
}
