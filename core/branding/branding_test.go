package branding

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuilderImage(t *testing.T) {
	require.Equal(t, "ghcr.io/instant-rw/instantpack-builder:mise-2026.6.5", BuilderImage("2026.6.5"))
}

func TestRuntimeImage(t *testing.T) {
	require.Equal(t, "ghcr.io/instant-rw/instantpack-runtime:mise-2026.6.5", RuntimeImage("2026.6.5"))
}

func TestBuildKitLabel(t *testing.T) {
	require.Equal(t, "[instantpack] merge layers", BuildKitLabel("merge layers"))
}

func TestConfigEnvPrefixes(t *testing.T) {
	require.Equal(t, "INSTANTPACK_BUN_NO_FROZEN_LOCKFILE", ConfigEnv("BUN_NO_FROZEN_LOCKFILE"))
	require.Equal(t, "RAILPACK_BUN_NO_FROZEN_LOCKFILE", LegacyConfigEnv("BUN_NO_FROZEN_LOCKFILE"))
}
