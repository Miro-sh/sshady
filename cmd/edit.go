package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"sshady/internal/proxy"
	"sshady/internal/sshconf"
)

var editFlags struct {
	host         string
	sshUser      string
	port         string
	identityFile string
	proxyType    string
	proxyHost    string
	proxyPort    string
	proxyUser    string
	proxyPass    string
	jumpHost     string
}

var editCmd = &cobra.Command{
	Use:   "edit <alias>",
	Short: "Edit an existing SSH proxy configuration",
	Long: `Edit an entry previously created by sshady.

Without flags, launches the interactive wizard pre-filled with the
current values (leave the proxy password empty to keep it).

With flags, only the given fields are updated.`,
	Example: `  sshady edit myserver
  sshady edit myserver --port 443
  sshady edit myserver --proxy-type tor
  sshady edit myserver --proxy-user alice --proxy-pass s3cr3t`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		alias := args[0]

		existing, err := sshconf.ReadEntry(alias)
		if err != nil {
			return err
		}

		if flagsChanged(cmd) {
			return runEditNonInteractive(cmd, existing)
		}
		return runEditWizard(existing)
	},
}

func init() {
	rootCmd.AddCommand(editCmd)

	f := editCmd.Flags()
	f.StringVar(&editFlags.host, "host", "", "target hostname or IP")
	f.StringVar(&editFlags.sshUser, "user", "", "SSH user")
	f.StringVar(&editFlags.port, "port", "", "SSH port")
	f.StringVar(&editFlags.identityFile, "identity-file", "", "SSH identity file path (empty string to clear)")
	f.StringVar(&editFlags.proxyType, "proxy-type", "", "proxy type: socks5, http, tor, jump")
	f.StringVar(&editFlags.proxyHost, "proxy-host", "", "proxy hostname or IP")
	f.StringVar(&editFlags.proxyPort, "proxy-port", "", "proxy port")
	f.StringVar(&editFlags.proxyUser, "proxy-user", "", "proxy username")
	f.StringVar(&editFlags.proxyPass, "proxy-pass", "", "proxy password")
	f.StringVar(&editFlags.jumpHost, "jump-host", "", "SSH jump host (user@host)")
}

func flagsChanged(cmd *cobra.Command) bool {
	changed := false
	cmd.Flags().Visit(func(f *pflag.Flag) {
		changed = true
	})
	return changed
}

func runEditWizard(existing sshconf.HostConfig) error {
	fmt.Println()
	fmt.Printf("  sshady -- editing '%s' (press enter to keep current values)\n", existing.Alias)
	fmt.Println()

	cfg, err := askEntry(entryDefaults{
		alias:        existing.Alias,
		host:         existing.HostName,
		user:         existing.User,
		port:         existing.Port,
		identityFile: existing.IdentityFile,
		proxyType:    existing.Proxy.Type,
		proxyHost:    existing.Proxy.Host,
		proxyPort:    existing.Proxy.Port,
		proxyUser:    existing.Proxy.Username,
		proxyPass:    existing.Proxy.Password,
		jumpHost:     existing.Proxy.JumpHost,
		hasExisting:  true,
	})
	if err != nil {
		return err
	}

	printSummary(cfg)

	if !confirmWrite() {
		return nil
	}

	if err := sshconf.UpdateEntry(existing.Alias, cfg); err != nil {
		return err
	}

	fmt.Printf("\nUpdated 'Host %s' in ~/.ssh/config\n", cfg.Alias)
	fmt.Printf("Connect with:  ssh %s\n\n", cfg.Alias)
	return nil
}

func runEditNonInteractive(cmd *cobra.Command, existing sshconf.HostConfig) error {
	cfg := existing
	f := cmd.Flags()

	if f.Changed("host") {
		cfg.HostName = editFlags.host
	}
	if f.Changed("user") {
		cfg.User = editFlags.sshUser
	}
	if f.Changed("port") {
		cfg.Port = editFlags.port
	}
	if f.Changed("identity-file") {
		cfg.IdentityFile = editFlags.identityFile
	}

	if f.Changed("proxy-type") {
		switch editFlags.proxyType {
		case "socks5", "http":
			if editFlags.proxyHost == "" {
				return fmt.Errorf("--proxy-host is required when switching to proxy type %q", editFlags.proxyType)
			}
			pType := proxy.TypeSOCKS5
			if editFlags.proxyType == "http" {
				pType = proxy.TypeHTTP
			}
			port := editFlags.proxyPort
			if port == "" {
				port = "1080"
			}
			cfg.Proxy = proxy.Config{
				Type:     pType,
				Host:     editFlags.proxyHost,
				Port:     port,
				Username: editFlags.proxyUser,
				Password: editFlags.proxyPass,
			}
		case "tor":
			cfg.Proxy = proxy.Config{Type: proxy.TypeTor}
		case "jump":
			if editFlags.jumpHost == "" {
				return fmt.Errorf("--jump-host is required when switching to proxy type 'jump'")
			}
			cfg.Proxy = proxy.Config{Type: proxy.TypeJump, JumpHost: editFlags.jumpHost}
		default:
			return fmt.Errorf("unknown proxy type %q; use: socks5, http, tor, jump", editFlags.proxyType)
		}
	} else {
		if f.Changed("proxy-host") {
			cfg.Proxy.Host = editFlags.proxyHost
		}
		if f.Changed("proxy-port") {
			cfg.Proxy.Port = editFlags.proxyPort
		}
		if f.Changed("proxy-user") {
			cfg.Proxy.Username = editFlags.proxyUser
		}
		if f.Changed("proxy-pass") {
			cfg.Proxy.Password = editFlags.proxyPass
		}
		if f.Changed("jump-host") {
			cfg.Proxy.JumpHost = editFlags.jumpHost
		}
	}

	if err := sshconf.UpdateEntry(existing.Alias, cfg); err != nil {
		return err
	}

	fmt.Printf("Updated 'Host %s' in ~/.ssh/config\n", cfg.Alias)
	fmt.Printf("Connect with:  ssh %s\n", cfg.Alias)
	return nil
}
