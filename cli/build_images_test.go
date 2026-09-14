package cli

import (
	"testing"

	"github.com/railwayapp/railpack/core/generate"
	"github.com/railwayapp/railpack/core/plan"
)

func TestApplyBaseImageOverrides(t *testing.T) {
	origBuilder, origRuntime := generate.RailpackBuilderImage, plan.RailpackRuntimeImage
	t.Cleanup(func() {
		generate.RailpackBuilderImage, plan.RailpackRuntimeImage = origBuilder, origRuntime
	})

	applyBaseImageOverrides("", "")
	if generate.RailpackBuilderImage != origBuilder || plan.RailpackRuntimeImage != origRuntime {
		t.Fatal("no override should keep defaults")
	}

	t.Setenv("INSTANTPACK_BUILDER_IMAGE", "ghcr.io/instant-rw/instantpack-builder:mise-2026.6.5")
	t.Setenv("RAILPACK_RUNTIME_IMAGE", "ghcr.io/instant-rw/instantpack-runtime:mise-2026.6.5")
	applyBaseImageOverrides("", "")
	if generate.RailpackBuilderImage != "ghcr.io/instant-rw/instantpack-builder:mise-2026.6.5" {
		t.Fatalf("builder env override ignored: %s", generate.RailpackBuilderImage)
	}
	if plan.RailpackRuntimeImage != "ghcr.io/instant-rw/instantpack-runtime:mise-2026.6.5" {
		t.Fatalf("legacy runtime env override ignored: %s", plan.RailpackRuntimeImage)
	}

	applyBaseImageOverrides("example.com/builder:x", "example.com/runtime:y")
	if generate.RailpackBuilderImage != "example.com/builder:x" || plan.RailpackRuntimeImage != "example.com/runtime:y" {
		t.Fatal("flags must win over environment")
	}
}
