package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// Build information - set by ldflags during build
var (
	Version   = "0.1.0"
	Commit    = "HEAD"
	BuildDate = "unknown"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long: `Display version information for the distributed filesystem CLI.

Examples:
  dfs version           # Show version
  dfs version --json    # Output as JSON`,
	Run: runVersion,
}

var versionJSON bool

func init() {
	versionCmd.Flags().BoolVar(&versionJSON, "json", false, "output as JSON")
}

func runVersion(cmd *cobra.Command, args []string) {
	if versionJSON {
		fmt.Printf(`{
  "version": "%s",
  "commit": "%s",
  "buildDate": "%s",
  "goVersion": "%s",
  "platform": "%s/%s"
}`,
			Version,
			Commit,
			BuildDate,
			runtime.Version(),
			runtime.GOOS,
			runtime.GOARCH,
		)
		fmt.Println()
	} else {
		fmt.Printf("dfs version %s\n", Version)
		fmt.Printf("Commit: %s\n", Commit)
		fmt.Printf("Built: %s\n", BuildDate)
		fmt.Printf("Go: %s\n", runtime.Version())
		fmt.Printf("Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	}
}
