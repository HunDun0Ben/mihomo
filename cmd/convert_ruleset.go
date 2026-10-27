package cmd

import (
	"github.com/metacubex/mihomo/rules/provider"
	"github.com/spf13/cobra"
)

func newConvertRuleSetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "convert-ruleset <behavior> <format> <source-file> <target-file>",
		Short: "convert a rule set to MRS format",
		Long: `Convert a rule set between YAML/Text and MRS formats.

behavior must be one of: domain, ipcidr, classical.
format must be one of: yaml, text, mrs.
For yaml or text input, the output is MRS; for mrs input, the output is text.`,
		Example: `  mihomo convert-ruleset domain yaml rules.yaml rules.mrs
  mihomo convert-ruleset domain mrs rules.mrs rules.txt`,
		Args: cobra.MinimumNArgs(4),
		RunE: func(_ *cobra.Command, args []string) error {
			provider.ConvertMain(args)
			return nil
		},
	}
}
