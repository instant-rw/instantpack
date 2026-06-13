package branding

import "fmt"

const (
	ProductName      = "Instantpack"
	BinaryName       = "instantpack"
	ConfigFileName   = "railpack.json"
	PlanFileName     = "railpack-plan.json"
	EnvPrefix        = "INSTANTPACK"
	LegacyEnvPrefix  = "RAILPACK"
	GitHubOwner      = "instant-rw"
	GitHubRepository = "instantpack"
	ContainerRegistry = "ghcr.io"
	DocsURL          = "https://github.com/instant-rw/instantpack"
	ConfigDocsURL    = "https://github.com/instant-rw/instantpack#configuration"
)

func DisplayName() string {
	return ProductName
}

func CLIHeader(version string) string {
	return fmt.Sprintf("%s %s", ProductName, version)
}

func GitHubRepositoryPath() string {
	return GitHubOwner + "/" + GitHubRepository
}

func BuilderImage(miseVersion string) string {
	return fmt.Sprintf("%s/%s-builder:mise-%s", ContainerRegistry, GitHubRepositoryPath(), miseVersion)
}

func RuntimeImage(miseVersion string) string {
	return fmt.Sprintf("%s/%s-runtime:mise-%s", ContainerRegistry, GitHubRepositoryPath(), miseVersion)
}

func FrontendImage() string {
	return fmt.Sprintf("%s/%s-frontend", ContainerRegistry, GitHubRepositoryPath())
}

func BuildKitTag() string {
	return BinaryName
}

func BuildKitLabel(action string) string {
	return fmt.Sprintf("[%s] %s", BuildKitTag(), action)
}

func DefaultAppImageName() string {
	return BinaryName + "-app"
}

func ConfigEnv(name string) string {
	return EnvPrefix + "_" + name
}

func LegacyConfigEnv(name string) string {
	return LegacyEnvPrefix + "_" + name
}

func ConfigEnvPrefixes() []string {
	return []string{EnvPrefix, LegacyEnvPrefix}
}

func VerboseEnvVars() []string {
	return []string{ConfigEnv("VERBOSE"), LegacyConfigEnv("VERBOSE")}
}

func DefaultPackageSource() string {
	return BinaryName + " default"
}
