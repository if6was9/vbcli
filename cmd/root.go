package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"vbcli/internal/vestaboard"
)

const (
	flagModel          = "model"
	envVestaboardModel = "VESTABOARD_MODEL"
	envVestaboardToken = "VESTABOARD_TOKEN"
)

type options struct {
	model   string
	align   string
	justify string
	verbose bool
}

func NewRootCmd(stdin io.Reader, stdout, stderr io.Writer) *cobra.Command {
	opts := &options{}

	cmd := &cobra.Command{
		Use:           "vbcli",
		Short:         "CLI for interacting with the Vestaboard API",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_ = cmd.Help()
			return errors.New("a subcommand is required: send-raw, send, format, clear, or get")
		},
	}

	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.PersistentFlags().BoolVarP(&opts.verbose, "verbose", "v", false, "Enable verbose HTTP logging")

	sendRawCmd := &cobra.Command{
		Use:   "send-raw [characters-json|-]",
		Short: "Send raw characters payload to the Vestaboard API",
		Args:  maxArgsWithHelp(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRaw(cmd, stdin, stdout, stderr, opts, args)
		},
	}

	sendCmd := &cobra.Command{
		Use:   "send [message|-]",
		Short: "Render template text via VBML then send characters to the Vestaboard API",
		Args:  maxArgsWithHelp(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			formatOnly, err := cmd.Flags().GetBool("format")
			if err != nil {
				return err
			}
			return runSend(cmd, stdin, stdout, stderr, opts, args, formatOnly)
		},
	}
	sendCmd.Flags().StringVarP(&opts.model, flagModel, "m", "", "VBML model for send: flagship or note")
	sendCmd.Flags().StringVarP(&opts.align, "align", "a", "center", "VBML align for send: top, center, or bottom")
	sendCmd.Flags().StringVarP(&opts.justify, "justify", "j", "center", "VBML justify for send: left, center, right, or justified")
	sendCmd.Flags().Bool("format", false, "Print VBML compose output and skip sending to Cloud API")

	formatCmd := &cobra.Command{
		Use:   "format <message|->",
		Short: "Format template text via VBML and print characters JSON",
		Args:  exactArgsWithHelp(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSend(cmd, stdin, stdout, stderr, opts, args, true)
		},
	}
	formatCmd.Flags().StringVarP(&opts.model, flagModel, "m", "", "VBML model for format: flagship or note")
	formatCmd.Flags().StringVarP(&opts.align, "align", "a", "center", "VBML align for format: top, center, or bottom")
	formatCmd.Flags().StringVarP(&opts.justify, "justify", "j", "center", "VBML justify for format: left, center, right, or justified")

	clearCmd := &cobra.Command{
		Use:   "clear",
		Short: "Clear the display (equivalent to `vbcli send ''`)",
		Args:  exactArgsWithHelp(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSend(cmd, stdin, stdout, stderr, opts, []string{""}, false)
		},
	}
	clearCmd.Flags().StringVarP(&opts.model, flagModel, "m", "", "VBML model for clear: flagship or note")
	clearCmd.Flags().StringVarP(&opts.align, "align", "a", "center", "VBML align for clear: top, center, or bottom")
	clearCmd.Flags().StringVarP(&opts.justify, "justify", "j", "center", "VBML justify for clear: left, center, right, or justified")

	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Fetch the current display state as JSON",
		Args:  exactArgsWithHelp(0),
		RunE: func(cmd *cobra.Command, _ []string) error {
			layoutOnly, err := cmd.Flags().GetBool("layout")
			if err != nil {
				return err
			}
			return runGet(cmd, stdout, stderr, opts, layoutOnly)
		},
	}
	getCmd.Flags().BoolP("layout", "l", false, "Print only currentMessage.layout")

	cmd.AddCommand(sendRawCmd, sendCmd, formatCmd, clearCmd, getCmd)

	return cmd
}

func runRaw(cmd *cobra.Command, stdin io.Reader, stdout, stderr io.Writer, opts *options, args []string) error {
	ctx := cmd.Context()
	client, err := buildClient(stderr, opts)
	if err != nil {
		return err
	}

	resolved, err := resolveCommandInput(cmd, stdin, args, "characters-json")
	if err != nil {
		return err
	}
	characters, err := parseCharacters(resolved)
	if err != nil {
		return usageError(cmd, fmt.Errorf("raw input must be a JSON array of arrays of integers: %w", err))
	}
	if err := client.SendCharacters(ctx, characters); err != nil {
		return err
	}
	return nil
}

func runSend(cmd *cobra.Command, stdin io.Reader, stdout, stderr io.Writer, opts *options, args []string, formatOnly bool) error {
	ctx := cmd.Context()
	client, err := buildClient(stderr, opts)
	if err != nil {
		return err
	}

	resolved, err := resolveCommandInput(cmd, stdin, args, "message")
	if err != nil {
		return err
	}
	model, err := resolveModel(opts.model)
	if err != nil {
		return usageError(cmd, err)
	}
	align, err := resolveAlign(opts.align)
	if err != nil {
		return usageError(cmd, err)
	}
	justify, err := resolveJustify(opts.justify)
	if err != nil {
		return usageError(cmd, err)
	}

	resolved = decodeTemplateEscapes(resolved)
	resolved = substituteTemplateCharacterAliases(resolved)
	characters, err := client.FormatMessage(ctx, resolved, model, align, justify)
	if err != nil {
		return err
	}
	if formatOnly {
		out, err := json.Marshal(characters)
		if err != nil {
			return fmt.Errorf("encode formatted output: %w", err)
		}
		if _, err := fmt.Fprintln(stdout, string(out)); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
		return nil
	}
	if err := client.SendCharacters(ctx, characters); err != nil {
		return err
	}
	return nil
}

func runGet(cmd *cobra.Command, stdout, stderr io.Writer, opts *options, layoutOnly bool) error {
	ctx := cmd.Context()
	client, err := buildClient(stderr, opts)
	if err != nil {
		return err
	}

	stateJSON, err := client.GetCurrent(ctx)
	if err != nil {
		return err
	}

	if layoutOnly {
		layout, err := extractLayout(stateJSON)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintln(stdout, layout); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
		return nil
	}

	if _, err := fmt.Fprintln(stdout, string(stateJSON)); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}

func extractLayout(stateJSON []byte) (string, error) {
	var payload struct {
		CurrentMessage struct {
			Layout string `json:"layout"`
		} `json:"currentMessage"`
	}
	if err := json.Unmarshal(stateJSON, &payload); err != nil {
		return "", fmt.Errorf("decode API response: %w", err)
	}
	if payload.CurrentMessage.Layout == "" {
		return "", errors.New("currentMessage.layout not found")
	}
	return payload.CurrentMessage.Layout, nil
}

func buildClient(stderr io.Writer, opts *options) (*vestaboard.Client, error) {
	token := strings.TrimSpace(os.Getenv(envVestaboardToken))
	return vestaboard.NewClient(token, vestaboard.WithVerboseLogging(opts.verbose, stderr))
}

func decodeTemplateEscapes(input string) string {
	unquoted, err := strconv.Unquote(`"` + input + `"`)
	if err != nil {
		return input
	}
	return unquoted
}

func resolveModel(value string) (string, error) {
	model := strings.ToLower(strings.TrimSpace(value))
	if model == "" {
		model = strings.ToLower(strings.TrimSpace(os.Getenv(envVestaboardModel)))
	}
	if model == "" {
		model = "flagship"
	}
	switch model {
	case "flagship", "note":
		return model, nil
	default:
		return "", fmt.Errorf("invalid --model %q (expected \"flagship\" or \"note\")", value)
	}
}

func resolveAlign(value string) (string, error) {
	align := strings.ToLower(strings.TrimSpace(value))
	if align == "" {
		align = "center"
	}
	switch align {
	case "top", "center", "bottom":
		return align, nil
	default:
		return "", fmt.Errorf("invalid --align %q (expected \"top\", \"center\", or \"bottom\")", value)
	}
}

func resolveJustify(value string) (string, error) {
	justify := strings.ToLower(strings.TrimSpace(value))
	if justify == "" {
		justify = "center"
	}
	switch justify {
	case "left", "center", "right", "justified":
		return justify, nil
	default:
		return "", fmt.Errorf("invalid --justify %q (expected \"left\", \"center\", \"right\", or \"justified\")", value)
	}
}

func usageError(cmd *cobra.Command, err error) error {
	_ = cmd.Help()
	return err
}

func exactArgsWithHelp(n int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) == n {
			return nil
		}
		_ = cmd.Help()
		return fmt.Errorf("accepts %d arg(s), received %d", n, len(args))
	}
}

func maxArgsWithHelp(n int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) <= n {
			return nil
		}
		_ = cmd.Help()
		return fmt.Errorf("accepts at most %d arg(s), received %d", n, len(args))
	}
}

func resolveCommandInput(cmd *cobra.Command, stdin io.Reader, args []string, argName string) (string, error) {
	if len(args) == 1 {
		value, err := resolveValue(stdin, args[0])
		if err != nil {
			return "", fmt.Errorf("read input: %w", err)
		}
		return value, nil
	}

	if stdinIsTerminal(stdin) {
		return "", usageError(cmd, fmt.Errorf("missing %s argument (or pipe stdin)", argName))
	}

	data, err := io.ReadAll(stdin)
	if err != nil {
		return "", fmt.Errorf("read input: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}

func stdinIsTerminal(stdin io.Reader) bool {
	file, ok := stdin.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

func resolveValue(stdin io.Reader, value string) (string, error) {
	if value != "-" {
		return value, nil
	}

	data, err := io.ReadAll(stdin)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func parseCharacters(value string) ([][]int, error) {
	var characters [][]int
	if err := json.Unmarshal([]byte(value), &characters); err != nil {
		return nil, err
	}
	if len(characters) == 0 {
		return nil, errors.New("empty array")
	}
	return characters, nil
}
