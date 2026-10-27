package cmd

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/metacubex/mihomo/component/age"
	"github.com/spf13/cobra"
)

func newAgeCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "age",
		Short: "encrypt or decrypt configuration files",
		Long: `Manage age encryption keys and encrypted configuration files.

Use "mihomo age <command> --help" or "mihomo age help <command>" for the
exact arguments of keygen, convert, encrypt, or decrypt. A source or target
path of "-" means stdin or stdout.`,
		Example: `  mihomo age keygen
  mihomo age convert <secret-key>
  mihomo age encrypt <public-key> config.yaml config.age
  mihomo age decrypt <secret-key> config.age config.yaml`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	command.AddCommand(
		&cobra.Command{Use: "keygen", Short: "generate an age X25519 key pair", Example: "  mihomo age keygen", Args: cobra.NoArgs, RunE: runAgeKeygen(false)},
		&cobra.Command{Use: "keygen-pq", Short: "generate a post-quantum age key pair", Example: "  mihomo age keygen-pq", Args: cobra.NoArgs, RunE: runAgeKeygen(true)},
		&cobra.Command{Use: "convert <secret-key>", Short: "convert age secret keys to public keys", Example: "  mihomo age convert AGE-SECRET-KEY-...", Args: cobra.ExactArgs(1), RunE: runAgeConvert},
		newAgeFileCommand("encrypt", "encrypt a file with an age public key", func(data []byte, key string) ([]byte, error) { return age.EncryptBytes(data, key) }),
		newAgeFileCommand("decrypt", "decrypt a file with an age secret key", func(data []byte, key string) ([]byte, error) { return age.DecryptBytes(data, key) }),
	)
	addSubcommandHelp(command)
	return command
}

func runAgeKeygen(postQuantum bool) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		var secret, public string
		var err error
		if postQuantum {
			secret, public, err = age.GenHybridKeyPair()
		} else {
			secret, public, err = age.GenX25519KeyPair()
		}
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "# created: %s\n# public key: %s\n%s\n", time.Now().Format(time.RFC3339), public, secret)
		return err
	}
}

func runAgeConvert(cmd *cobra.Command, args []string) error {
	publicKeys, err := age.ToPublicKeys(args[0])
	if err != nil {
		return err
	}
	for _, publicKey := range publicKeys {
		if _, err = fmt.Fprintln(cmd.OutOrStdout(), publicKey); err != nil {
			return err
		}
	}
	return nil
}

type ageTransform func([]byte, string) ([]byte, error)

func newAgeFileCommand(name, short string, transform ageTransform) *cobra.Command {
	keyName := "public-key"
	exampleFiles := "input.yaml output.age"
	if name == "decrypt" {
		keyName = "secret-key"
		exampleFiles = "input.age output.yaml"
	}
	return &cobra.Command{
		Use:     fmt.Sprintf("%s <%s> <source-file> <target-file>", name, keyName),
		Short:   short,
		Example: fmt.Sprintf("  mihomo age %s <key> %s", name, exampleFiles),
		Args:    cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := readCLIFile(cmd, args[1])
			if err != nil {
				return err
			}
			result, err := transform(data, args[0])
			if err != nil {
				return err
			}
			return writeCLIFile(cmd, args[2], result)
		},
	}
}

func readCLIFile(cmd *cobra.Command, path string) ([]byte, error) {
	if path == "-" {
		return io.ReadAll(cmd.InOrStdin())
	}
	return os.ReadFile(path)
}

func writeCLIFile(cmd *cobra.Command, path string, data []byte) error {
	if path == "-" {
		_, err := cmd.OutOrStdout().Write(data)
		return err
	}
	return os.WriteFile(path, data, 0644)
}
