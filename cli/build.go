// entrypoint to the `build` subcommand
// Primarily bundles CLI options into the structure that `BuildWithBuildkitClient` expects

package cli

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/railwayapp/railpack/buildkit"
	"github.com/railwayapp/railpack/core"
	"github.com/railwayapp/railpack/core/app"
	"github.com/railwayapp/railpack/core/branding"
	"github.com/railwayapp/railpack/core/generate"
	"github.com/railwayapp/railpack/core/plan"
	"github.com/urfave/cli/v3"
)

var BuildCommand = &cli.Command{
	Name:                  "build",
	Aliases:               []string{"b"},
	Usage:                 "build an image with BuildKit",
	ArgsUsage:             "DIRECTORY",
	EnableShellCompletion: true,
	Flags: append([]cli.Flag{
		&cli.StringFlag{
			Name:  "name",
			Usage: "name of the image to build",
		},
		&cli.StringFlag{
			Name:  "output",
			Usage: "output the final filesystem to a local directory",
		},
		&cli.StringFlag{
			Name:  "platform",
			Usage: "platform to build for (e.g. linux/amd64, linux/arm64)",
		},
		&cli.StringFlag{
			Name:  "progress",
			Usage: "buildkit progress output mode. Values: auto, plain, tty",
			Value: "auto",
		},
		&cli.BoolFlag{
			Name:  "show-plan",
			Usage: "Show the build plan before building. This is useful for development and debugging.",
			Value: false,
		},
		&cli.StringFlag{
			Name:  "cache-key",
			Usage: "Unique id to prefix to cache keys",
		},
		&cli.StringSliceFlag{
			Name:  "cache-from",
			Usage: "External cache sources",
		},
		&cli.StringSliceFlag{
			Name:  "cache-to",
			Usage: "Cache export destinations",
		},
		&cli.BoolFlag{
			Name:  "no-cache",
			Usage: "Do not use cache when building",
			Value: false,
		},
		&cli.BoolFlag{
			Name:  "push",
			Usage: "push the image straight to its registry instead of loading it into the local Docker daemon",
			Value: false,
		},
		&cli.StringFlag{
			Name:  "builder-image",
			Usage: "override the build-time base image (default ghcr.io/instant-rw/instantpack-builder:mise-<version>)",
		},
		&cli.StringFlag{
			Name:  "runtime-image",
			Usage: "override the runtime base image (default ghcr.io/instant-rw/instantpack-runtime:mise-<version>)",
		},
		&cli.BoolFlag{
			Name:  "insecure-registry",
			Usage: "allow plain HTTP / self-signed registries when pushing or exporting cache",
			Value: false,
		},
		&cli.BoolFlag{
			Name:   "dump-llb",
			Hidden: true,
			Value:  false,
		},
	}, commonPlanFlags()...),
	Action: func(ctx context.Context, cmd *cli.Command) error {
		applyBaseImageOverrides(cmd.String("builder-image"), cmd.String("runtime-image"))
		buildResult, app, env, err := GenerateBuildResultForCommand(cmd)
		if err != nil {
			return cli.Exit(err, exitCodeForError(err))
		}

		if !cmd.Bool("dump-llb") {
			core.PrettyPrintBuildResult(buildResult, core.PrintOptions{Version: Version})
		}

		if !buildResult.Success {
			os.Exit(ExitCodeFailure)
			return nil
		}

		if cmd.Bool("show-plan") && !cmd.Bool("dump-llb") {
			planMap, err := addSchemaToPlanMap(buildResult.Plan)
			if err != nil {
				return cli.Exit(err, ExitCodeFailure)
			}

			serializedPlan, err := json.MarshalIndent(planMap, "", "  ")
			if err != nil {
				return cli.Exit(err, ExitCodeFailure)
			}

			core.PrettyPrintSectionHeader(os.Stdout, "Generated railpack-plan.json")
			core.PrettyPrintJSON(os.Stdout, serializedPlan)
		}

		err = validateSecrets(buildResult.Plan, env)
		if err != nil {
			return cli.Exit(err, ExitCodeFailure)
		}

		secretsHash := getSecretsHash(env)

		platformStr := cmd.String("platform")
		err = buildkit.BuildWithBuildkitClient(app.Source, buildResult.Plan, buildkit.BuildWithBuildkitClientOptions{
			ImageName:    cmd.String("name"),
			DumpLLB:      cmd.Bool("dump-llb"),
			OutputDir:    cmd.String("output"),
			ProgressMode: cmd.String("progress"),
			CacheKey:     cmd.String("cache-key"),
			// StringSlice to support multiple cache-from / cache-to entries, same shape as docker buildx
			ImportCache: cmd.StringSlice("cache-from"),
			ExportCache: cmd.StringSlice("cache-to"),
			SecretsHash: secretsHash,
			Secrets:     env.Variables,
			Platform:    platformStr,
			GitHubToken: os.Getenv("GITHUB_TOKEN"),
			NoCache:     cmd.Bool("no-cache"),
			Push:        cmd.Bool("push"),
			// Plain HTTP registries are only used by local Instant Cloud fleets
			InsecureRegistry: cmd.Bool("insecure-registry"),
		})
		if err != nil {
			return cli.Exit(err, ExitCodeFailure)
		}

		return nil
	},
}

// make sure all secrets referenced in the build plan are present in the environment
func validateSecrets(plan *plan.BuildPlan, env *app.Environment) error {
	for _, secret := range plan.Secrets {
		if _, ok := env.Variables[secret]; !ok {
			return fmt.Errorf("missing environment variable: %s. Please set using --env %s=%s", secret, secret, "...")
		}
	}
	return nil
}

// generate a hash all of build secrets to invalidate all caches when any secret changes
func getSecretsHash(env *app.Environment) string {
	var secretsValue strings.Builder
	for _, v := range env.Variables {
		secretsValue.WriteString(v)
	}
	hasher := sha256.New()
	hasher.Write([]byte(secretsValue.String()))
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

// applyBaseImageOverrides lets operators pin the builder/runtime base images
// independently of the mise version compiled into this binary. Flags win over
// INSTANTPACK_*_IMAGE / RAILPACK_*_IMAGE environment variables.
func applyBaseImageOverrides(builderFlag, runtimeFlag string) {
	if image := firstNonEmpty(builderFlag, os.Getenv(branding.ConfigEnv("BUILDER_IMAGE")), os.Getenv(branding.LegacyConfigEnv("BUILDER_IMAGE"))); image != "" {
		generate.RailpackBuilderImage = image
	}
	if image := firstNonEmpty(runtimeFlag, os.Getenv(branding.ConfigEnv("RUNTIME_IMAGE")), os.Getenv(branding.LegacyConfigEnv("RUNTIME_IMAGE"))); image != "" {
		plan.RailpackRuntimeImage = image
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
