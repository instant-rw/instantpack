package node

import (
	"testing"

	"github.com/railwayapp/railpack/core/app"
	"github.com/stretchr/testify/require"
)

func TestBunInstallCommand(t *testing.T) {
	t.Run("uses frozen lockfile by default", func(t *testing.T) {
		require.Equal(t, "bun install --frozen-lockfile", bunInstallCommand(app.NewEnvironment(nil)))
	})

	t.Run("bypasses frozen lockfile with INSTANTPACK env", func(t *testing.T) {
		env := app.NewEnvironment(&map[string]string{
			"INSTANTPACK_BUN_NO_FROZEN_LOCKFILE": "true",
		})
		require.Equal(t, "bun install", bunInstallCommand(env))
	})

	t.Run("bypasses frozen lockfile with legacy RAILPACK env", func(t *testing.T) {
		env := app.NewEnvironment(&map[string]string{
			"RAILPACK_BUN_NO_FROZEN_LOCKFILE": "1",
		})
		require.Equal(t, "bun install", bunInstallCommand(env))
	})
}
