package cmd

import (
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

// Options contains values parsed from Mihomo's process-level CLI flags.
type Options struct {
	Version                       bool
	TestConfig                    bool
	GeodataMode                   bool
	HomeDir                       string
	ConfigFile                    string
	ConfigString                  string
	AgeSecretKey                  string
	ExternalUI                    string
	ExternalController            string
	ExternalControllerTLS         string
	ExternalControllerUnix        string
	ExternalControllerPipe        string
	ExternalControllerRoutingMark int
	Secret                        string
	PostUp                        string
	PostDown                      string
}

// Execute parses the CLI and invokes run for the default Mihomo command.
// Subcommands execute their own handlers and do not invoke run.
func Execute(run func(Options)) error {
	options := Options{}
	return newRootCommand(&options, run).Execute()
}

func newRootCommand(options *Options, run func(Options)) *cobra.Command {
	root := &cobra.Command{
		Use:   "mihomo",
		Short: "Mihomo Meta proxy kernel",
		Long: `Mihomo is a rule-based proxy kernel.

Use the following subcommands for auxiliary operations:
  convert-ruleset  convert rule sets between YAML/Text and MRS formats
  generate         generate UUIDs, key pairs, and protocol parameters
  age              generate age keys and encrypt/decrypt files
  version          print Mihomo version and build information

Run "mihomo <command> --help" for detailed usage of a subcommand.`,
		Example: `  mihomo --config ./config.yaml
  mihomo generate uuid
  mihomo age keygen
  mihomo convert-ruleset domain yaml rules.yaml rules.mrs`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			if options.Version {
				printVersion()
				return nil
			}
			run(*options)
			return nil
		},
	}

	flags := root.Flags()
	flags.StringVarP(&options.HomeDir, "home-dir", "d", os.Getenv("CLASH_HOME_DIR"), "set configuration directory")
	flags.StringVarP(&options.ConfigFile, "config-file", "f", os.Getenv("CLASH_CONFIG_FILE"), "specify configuration file")
	flags.StringVar(&options.ConfigString, "config", os.Getenv("CLASH_CONFIG_STRING"), "specify base64-encoded configuration string")
	flags.StringVar(&options.AgeSecretKey, "age-secret-key", os.Getenv("CLASH_AGE_SECRET_KEY"), "set age secret key to decrypt configuration")
	flags.StringVar(&options.ExternalUI, "ext-ui", os.Getenv("CLASH_OVERRIDE_EXTERNAL_UI_DIR"), "override external ui directory")
	flags.StringVar(&options.ExternalController, "ext-ctl", os.Getenv("CLASH_OVERRIDE_EXTERNAL_CONTROLLER"), "override external controller address")
	flags.StringVar(&options.ExternalControllerTLS, "ext-ctl-tls", os.Getenv("CLASH_OVERRIDE_EXTERNAL_CONTROLLER_TLS"), "override external controller TLS address")
	flags.StringVar(&options.ExternalControllerUnix, "ext-ctl-unix", os.Getenv("CLASH_OVERRIDE_EXTERNAL_CONTROLLER_UNIX"), "override external controller Unix address")
	flags.StringVar(&options.ExternalControllerPipe, "ext-ctl-pipe", os.Getenv("CLASH_OVERRIDE_EXTERNAL_CONTROLLER_PIPE"), "override external controller pipe address")
	flags.IntVar(&options.ExternalControllerRoutingMark, "ext-ctl-routing-mark", getIntEnv("CLASH_OVERRIDE_EXTERNAL_CONTROLLER_ROUTING_MARK"), "override external controller routing mark")
	flags.StringVar(&options.Secret, "secret", os.Getenv("CLASH_OVERRIDE_SECRET"), "override secret for RESTful API")
	flags.StringVar(&options.PostUp, "post-up", os.Getenv("CLASH_POST_UP"), "set post-up script")
	flags.StringVar(&options.PostDown, "post-down", os.Getenv("CLASH_POST_DOWN"), "set post-down script")
	flags.BoolVarP(&options.GeodataMode, "geodata-mode", "m", false, "set geodata mode")
	flags.BoolVarP(&options.Version, "version", "v", false, "show current version of mihomo")
	flags.BoolVarP(&options.TestConfig, "test-config", "t", false, "test configuration and exit")

	// Keep the old flag names accepted by the standard flag package.
	flags.StringVar(&options.HomeDir, "d", options.HomeDir, "deprecated alias for --home-dir")
	flags.StringVar(&options.ConfigFile, "f", options.ConfigFile, "deprecated alias for --config-file")
	flags.BoolVar(&options.GeodataMode, "m", options.GeodataMode, "deprecated alias for --geodata-mode")
	flags.BoolVar(&options.Version, "v", options.Version, "deprecated alias for --version")
	flags.BoolVar(&options.TestConfig, "t", options.TestConfig, "deprecated alias for --test-config")
	for _, name := range []string{"d", "f", "m", "v", "t"} {
		_ = flags.MarkHidden(name)
	}

	root.AddCommand(newConvertRuleSetCommand(), newGenerateCommand(), newAgeCommand(), newVersionCommand())
	return root
}

func getIntEnv(key string) int {
	value, _ := strconv.Atoi(os.Getenv(key))
	return value
}

// addSubcommandHelp adds the explicit "<parent> help [command]" form in
// addition to Cobra's standard "--help" and root-level "help" command.
func addSubcommandHelp(parent *cobra.Command) {
	parent.AddCommand(&cobra.Command{
		Use:   "help [command]",
		Short: "Help about a command",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			target := parent
			if len(args) == 1 {
				var err error
				target, _, err = parent.Find([]string{args[0]})
				if err != nil {
					return err
				}
			}
			return target.Help()
		},
	})
}
