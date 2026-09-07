package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/kobberholm/go-simply-cli/internal/config"
	"github.com/kobberholm/go-simply-cli/internal/input"
	"github.com/kobberholm/go-simply-cli/internal/output"
	"github.com/kobberholm/go-simply-cli/internal/simply"
	sdk "github.com/kobberholm/go-simply-sdk"
	"github.com/spf13/cobra"
)

type options struct {
	apiKey         string
	authMode       string
	account        string
	output         string
	interactive    bool
	nonInteractive bool
	yes            bool
}

type dependencies struct {
	newClient func(simply.Config) (simply.Client, error)
	secret    input.PromptFunc
	line      input.PromptFunc
}

func defaultDependencies() dependencies {
	return dependencies{
		newClient: simply.NewClient,
		secret:    input.Secret,
		line:      input.Line,
	}
}

// Run constructs and executes the CLI with caller-owned streams.
func Run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	root := newRootCommand(ctx, stdin, stdout, stderr, defaultDependencies())
	root.SetArgs(args)
	return root.Execute()
}

func newRootCommand(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer, deps dependencies) *cobra.Command {
	var opts options

	root := &cobra.Command{
		Use:           "simply-cli",
		Short:         "Manage Simply.com products and DNS",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return fmt.Errorf("a command is required; use %s --help", cmd.CommandPath())
		},
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if opts.interactive && opts.nonInteractive {
				return fmt.Errorf("--interactive and --non-interactive cannot be used together")
			}
			if opts.output != "table" && opts.output != "json" {
				return fmt.Errorf("--output must be table or json")
			}
			return nil
		},
	}
	root.SetContext(ctx)
	root.SetIn(stdin)
	root.SetOut(stdout)
	root.SetErr(stderr)

	flags := root.PersistentFlags()
	flags.StringVar(&opts.apiKey, "api-key", "", "Simply API key (or SIMPLY_API_KEY)")
	flags.StringVar(&opts.authMode, "auth-mode", "", "Authentication mode: bearer or basic (or SIMPLY_AUTH_MODE; defaults to bearer)")
	flags.StringVar(&opts.account, "account", "", "Basic-auth account (or SIMPLY_ACCOUNT)")
	flags.StringVar(&opts.output, "output", "table", "Output format: table or json")
	flags.BoolVar(&opts.interactive, "interactive", false, "Prompt for missing values, even when stdin is not a terminal")
	flags.BoolVar(&opts.nonInteractive, "non-interactive", false, "Disable prompts and fail on missing values")
	flags.BoolVar(&opts.yes, "yes", false, "Approve update, delete, and reload operations")

	auth := &cobra.Command{Use: "auth", Short: "Check and manage authentication"}
	auth.AddCommand(authCheckCommand(&opts, stdin, stdout, deps))

	products := &cobra.Command{Use: "products", Short: "Manage Simply.com products"}
	products.AddCommand(productsListCommand(&opts, stdin, stdout, deps))

	dns := &cobra.Command{Use: "dns", Short: "Manage DNS records and zones"}
	records := &cobra.Command{Use: "records", Short: "Manage DNS records"}
	records.AddCommand(recordsListCommand(&opts, stdin, stdout, deps))
	records.AddCommand(recordCommand("dns records add", "add", "Add a DNS record", true, false, false))
	records.AddCommand(recordCommand("dns records update", "update", "Replace a DNS record", true, true, false))
	records.AddCommand(recordCommand("dns records delete", "delete", "Delete a DNS record", false, true, true))
	dns.AddCommand(records)
	zone := &cobra.Command{Use: "zone", Short: "Manage DNS zones"}
	zone.AddCommand(zoneShowCommand(&opts, stdin, stdout, deps))
	dns.AddCommand(zone)
	dns.AddCommand(recordCommand("dns reload", "reload", "Reload the DNS zone", false, false, true))

	root.AddCommand(auth, products, dns)
	return root
}

func productsListCommand(opts *options, stdin io.Reader, stdout io.Writer, deps dependencies) *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "List products visible to the authenticated account",
		Example: "Interactive: simply-cli products list\nNon-interactive: SIMPLY_API_KEY=... simply-cli --non-interactive products list --output json",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientForCommand(cmd, *opts, stdin, stdout, deps)
			if err != nil {
				return err
			}
			products, response, err := client.Products().List(cmd.Context())
			if err != nil {
				return commandError("list products", err, response)
			}
			return output.Products(stdout, opts.output, products, response)
		},
	}
}

func recordsListCommand(opts *options, stdin io.Reader, stdout io.Writer, deps dependencies) *cobra.Command {
	command := &cobra.Command{
		Use:     "list",
		Short:   "List DNS records",
		Example: "Interactive: simply-cli dns records list\nNon-interactive: simply-cli --non-interactive dns records list --product example.com",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientForCommand(cmd, *opts, stdin, stdout, deps)
			if err != nil {
				return err
			}
			product, err := cmd.Flags().GetString("product")
			if err != nil {
				return err
			}
			records, response, err := client.DNS().ListRecords(cmd.Context(), product)
			if err != nil {
				return commandError("list DNS records", err, response)
			}
			return output.Records(stdout, opts.output, records, response)
		},
	}
	command.Flags().String("product", "", "Product/domain to manage")
	_ = command.MarkFlagRequired("product")
	return command
}

func zoneShowCommand(opts *options, stdin io.Reader, stdout io.Writer, deps dependencies) *cobra.Command {
	command := &cobra.Command{
		Use:     "show",
		Short:   "Show the DNS zone",
		Example: "Interactive: simply-cli dns zone show\nNon-interactive: simply-cli --non-interactive dns zone show --product example.com",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := clientForCommand(cmd, *opts, stdin, stdout, deps)
			if err != nil {
				return err
			}
			product, err := cmd.Flags().GetString("product")
			if err != nil {
				return err
			}
			zone, response, err := client.DNS().Zone(cmd.Context(), product)
			if err != nil {
				return commandError("show DNS zone", err, response)
			}
			return output.Zone(stdout, opts.output, zone, response)
		},
	}
	command.Flags().String("product", "", "Product/domain to manage")
	_ = command.MarkFlagRequired("product")
	return command
}

func clientForCommand(cmd *cobra.Command, opts options, stdin io.Reader, stdout io.Writer, deps dependencies) (simply.Client, error) {
	credentials, err := resolveCredentials(cmd, opts, stdin, stdout, deps)
	if err != nil {
		return nil, err
	}
	client, err := deps.newClient(simply.Config{APIKey: credentials.APIKey, Account: credentials.Account, AuthMode: credentials.AuthMode})
	if err != nil {
		return nil, fmt.Errorf("create Simply client: %w", err)
	}
	return client, nil
}

func commandError(operation string, err error, response sdk.Response) error {
	if response.StatusCode == 429 && response.RetryAfter != "" {
		return fmt.Errorf("%s failed: %w (retry after %s)", operation, err, response.RetryAfter)
	}
	return fmt.Errorf("%s failed: %w", operation, err)
}

func authCheckCommand(opts *options, stdin io.Reader, stdout io.Writer, deps dependencies) *cobra.Command {
	command := &cobra.Command{
		Use:     "check",
		Short:   "Validate credentials with a read-only product request",
		Example: "Interactive: simply-cli auth check\nNon-interactive: SIMPLY_API_KEY=... simply-cli --non-interactive auth check",
		RunE: func(cmd *cobra.Command, _ []string) error {
			credentials, err := resolveCredentials(cmd, *opts, stdin, stdout, deps)
			if err != nil {
				return err
			}
			client, err := deps.newClient(simply.Config{
				APIKey: credentials.APIKey, Account: credentials.Account, AuthMode: credentials.AuthMode,
			})
			if err != nil {
				return fmt.Errorf("create Simply client: %w", err)
			}
			_, response, err := client.Products().List(cmd.Context())
			if err != nil {
				if unauthorized(err) {
					return fmt.Errorf("authentication failed: %w", err)
				}
				if response.StatusCode == 429 && response.RetryAfter != "" {
					return fmt.Errorf("authentication check failed: %w (retry after %s)", err, response.RetryAfter)
				}
				return fmt.Errorf("authentication check failed: %w", err)
			}
			return writeAuthResult(stdout, opts.output, response.RateLimitLimit, response.RateLimitRemaining)
		},
	}
	return command
}

func resolveCredentials(cmd *cobra.Command, opts options, stdin io.Reader, stdout io.Writer, deps dependencies) (config.Config, error) {
	values := config.Values{APIKeyFlag: opts.apiKey, AuthModeFlag: opts.authMode, AccountFlag: opts.account, Lookup: os.Getenv}
	resolved, err := config.Resolve(values)
	if err == nil {
		return resolved, nil
	}
	if opts.nonInteractive || (!opts.interactive && !input.IsTerminal(stdin)) {
		return config.Config{}, err
	}
	if resolved, promptErr := promptMissingCredentials(values, stdin, stdout, deps); promptErr != nil {
		return config.Config{}, promptErr
	} else {
		return resolved, nil
	}
}

func promptMissingCredentials(values config.Values, stdin io.Reader, stdout io.Writer, deps dependencies) (config.Config, error) {
	if values.APIKeyFlag == "" && values.Lookup("SIMPLY_API_KEY") == "" {
		value, err := deps.secret(stdin, stdout, "API key: ")
		if err != nil {
			return config.Config{}, fmt.Errorf("read API key: %w", err)
		}
		values.APIKeyFlag = value
	}
	mode := values.AuthModeFlag
	if mode == "" {
		mode = values.Lookup("SIMPLY_AUTH_MODE")
	}
	if mode == "basic" && values.AccountFlag == "" && values.Lookup("SIMPLY_ACCOUNT") == "" {
		value, err := deps.line(stdin, stdout, "Account: ")
		if err != nil {
			return config.Config{}, fmt.Errorf("read account: %w", err)
		}
		values.AccountFlag = value
	}
	return config.Resolve(values)
}

func unauthorized(err error) bool {
	var value interface{ IsUnauthorized() bool }
	return errors.As(err, &value) && value.IsUnauthorized()
}

func writeAuthResult(stdout io.Writer, format, limit, remaining string) error {
	if format == "json" {
		return json.NewEncoder(stdout).Encode(map[string]any{
			"valid": true, "rate_limit": limit, "rate_limit_remaining": remaining,
		})
	}
	_, err := fmt.Fprintf(stdout, "Credentials are valid\nRate limit remaining: %s\n", remaining)
	return err
}

func placeholderCommand(use, short, example string) *cobra.Command {
	return &cobra.Command{
		Use:     use,
		Short:   short,
		Example: example,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return fmt.Errorf("%s is not implemented yet", cmd.CommandPath())
		},
	}
}

func recordCommand(commandPath, use, short string, recordFields, id, confirmation bool) *cobra.Command {
	command := placeholderCommand(use, short, "Interactive: simply-cli "+commandPath+"\nNon-interactive: simply-cli --non-interactive "+commandPath)
	flags := command.Flags()
	flags.String("product", "", "Product/domain to manage")
	if recordFields {
		flags.String("name", "", "DNS record name")
		flags.String("type", "", "DNS record type")
		flags.String("value", "", "DNS record value")
		flags.Int("ttl", 0, "DNS record TTL in seconds")
	}
	if id {
		flags.String("id", "", "DNS record ID")
	}
	if confirmation {
		command.Annotations = map[string]string{"confirmation": "interactive confirmation required; use --yes in non-interactive mode"}
		command.Long = short + ". Interactive mode asks for confirmation immediately before the request. Non-interactive mode requires --yes."
	}
	if recordFields || id || commandPath == "dns records list" || commandPath == "dns zone show" || commandPath == "dns reload" {
		if err := command.MarkFlagRequired("product"); err != nil {
			panic(err)
		}
	}
	if id {
		if err := command.MarkFlagRequired("id"); err != nil {
			panic(err)
		}
	}
	if recordFields {
		for _, name := range []string{"name", "type", "value"} {
			if err := command.MarkFlagRequired(name); err != nil {
				panic(err)
			}
		}
	}
	return command
}
