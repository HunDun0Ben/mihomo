package cmd

import (
	"fmt"
	"runtime"
	"strings"

	C "github.com/metacubex/mihomo/constant"
	"github.com/metacubex/mihomo/constant/features"
	"github.com/spf13/cobra"
)

// newVersionCommand exposes version information as an explicit command while
// reusing the existing version output in the main runtime path.
func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "version",
		Aliases: []string{"--version"},
		Short:   "print Mihomo version and build information",
		Long:    "Print Mihomo version, platform, Go runtime, build time, and enabled feature tags.",
		Args:    cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			printVersion()
			return nil
		},
	}
}

func printVersion() {
	fmt.Printf("Mihomo Meta %s %s %s with %s %s\n",
		C.Version, runtime.GOOS, runtime.GOARCH, runtime.Version(), C.BuildTime)
	if tags := features.Tags(); len(tags) != 0 {
		fmt.Printf("Use tags: %s\n", strings.Join(tags, ", "))
	}
}
