package cmd

import (
	"encoding/base64"
	"fmt"

	"github.com/gofrs/uuid/v5"
	"github.com/metacubex/mihomo/component/ech"
	"github.com/metacubex/mihomo/component/generator"
	"github.com/metacubex/mihomo/transport/sudoku"
	"github.com/metacubex/mihomo/transport/vless/encryption"
	"github.com/spf13/cobra"
)

func newGenerateCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "generate",
		Aliases: []string{"gen"},
		Short:   "generate keys or identifiers",
		Long: `Generate identifiers, key pairs, and protocol-specific parameters.

Use "mihomo generate <name> --help" or "mihomo generate help <name>" to see
the arguments for one generator.
Available generators include uuid, reality-keypair, wg-keypair, ech-keypair,
vless-mlkem768, vless-x25519, and sudoku-keypair.`,
		Example: `  mihomo generate uuid
  mihomo generate reality-keypair
  mihomo generate ech-keypair example.com
  mihomo generate vless-mlkem768`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	command.AddCommand(
		generateNoArgs("uuid", "generate a UUID", func(cmd *cobra.Command) error {
			value, err := uuid.NewV4()
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), value)
			return err
		}),
		generateNoArgs("reality-keypair", "generate a Reality X25519 key pair", func(cmd *cobra.Command) error {
			privateKey, err := generator.GenX25519PrivateKey()
			if err != nil {
				return err
			}
			return printKeyPair(cmd, base64.RawURLEncoding.EncodeToString(privateKey.Bytes()), base64.RawURLEncoding.EncodeToString(privateKey.PublicKey().Bytes()))
		}),
		generateNoArgs("wg-keypair", "generate a WireGuard X25519 key pair", func(cmd *cobra.Command) error {
			privateKey, err := generator.GenX25519PrivateKey()
			if err != nil {
				return err
			}
			return printKeyPair(cmd, base64.StdEncoding.EncodeToString(privateKey.Bytes()), base64.StdEncoding.EncodeToString(privateKey.PublicKey().Bytes()))
		}),
		&cobra.Command{
			Use:     "ech-keypair <plain-server-name>",
			Short:   "generate an ECH config and private key",
			Example: "  mihomo generate ech-keypair example.com",
			Args:    cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				config, key, err := ech.GenECHConfig(args[0])
				if err != nil {
					return err
				}
				if _, err = fmt.Fprintln(cmd.OutOrStdout(), "Config:", config); err != nil {
					return err
				}
				_, err = fmt.Fprintln(cmd.OutOrStdout(), "Key:", key)
				return err
			},
		},
		newGenerateVLESSCommand("vless-mlkem768", "generate VLESS ML-KEM768/X25519 parameters", encryption.GenMLKEM768),
		newGenerateVLESSCommand("vless-x25519", "generate VLESS X25519 parameters", encryption.GenX25519),
		generateNoArgs("sudoku-keypair", "generate a Sudoku key pair", func(cmd *cobra.Command) error {
			privateKey, publicKey, err := sudoku.GenKeyPair()
			if err != nil {
				return err
			}
			if _, err = fmt.Fprintln(cmd.OutOrStdout(), "PrivateKey: "+privateKey); err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), "PublicKey: "+publicKey)
			return err
		}),
	)
	addSubcommandHelp(command)
	return command
}

func generateNoArgs(use, short string, run func(*cobra.Command) error) *cobra.Command {
	return &cobra.Command{Use: use, Short: short, Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error { return run(cmd) }}
}

type vlessGenerator func(string) (string, string, string, error)

func newGenerateVLESSCommand(use, short string, generate vlessGenerator) *cobra.Command {
	return &cobra.Command{
		Use:     use + " [value]",
		Short:   short,
		Example: "  mihomo generate " + use,
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			value := ""
			if len(args) == 1 {
				value = args[0]
			}
			first, second, hash, err := generate(value)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Seed: %s\nClient: %s\nHash32: %s\n-----------------------\n      Lazy-Config      \n-----------------------\n[Server] decryption: \"mlkem768x25519plus.native.600s.%s\"\n[Client] encryption: \"mlkem768x25519plus.native.0rtt.%s\"\n", first, second, hash, first, second)
			return err
		},
	}
}

func printKeyPair(cmd *cobra.Command, privateKey, publicKey string) error {
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), "PrivateKey: "+privateKey); err != nil {
		return err
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), "PublicKey: "+publicKey)
	return err
}
