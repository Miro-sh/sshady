package cmd

import (
	"fmt"
	"os"
	"os/user"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"sshady/internal/proxy"
	"sshady/internal/sshconf"
)

var createFlags struct {
	alias        string
	host         string
	sshUser      string
	port         string
	identityFile string
	proxyType    string
	proxyHost    string
	proxyPort    string
	proxyUser    string
	proxyPass    string
	proxyPassFile string
	jumpHost     string
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new SSH proxy configuration",
	Long: `Create a new SSH Host entry with proxy settings.

Without flags, launches the interactive wizard.
Provide --alias, --host, and --proxy-type (plus proxy-specific flags) to skip the wizard.

Proxy password can be provided via --proxy-pass-file (recommended) or --proxy-pass (visible in process list).
Use SSHADY_PROXY_PASS environment variable for the most secure option.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Resolve password from env var if not provided via flags
		if createFlags.proxyPass == "" && createFlags.proxyPassFile == "" {
			if envPass := os.Getenv("SSHADY_PROXY_PASS"); envPass != "" {
				createFlags.proxyPass = envPass
			}
		}
		// Read password from file if specified
		if createFlags.proxyPassFile != "" {
			data, err := os.ReadFile(createFlags.proxyPassFile)
			if err != nil {
				return fmt.Errorf("cannot read proxy password file: %w", err)
			}
			// Trim trailing newline
			createFlags.proxyPass = string(data)
			if len(createFlags.proxyPass) > 0 && createFlags.proxyPass[len(createFlags.proxyPass)-1] == '
' {
				createFlags.proxyPass = createFlags.proxyPass[:len(createFlags.proxyPass)-1]
			}
		}

		if createFlags.alias != "" && createFlags.host != "" && createFlags.proxyType != "" {
			return runNonInteractive()
		}
		return runWizard()
	},
}

func init() {
	rootCmd.AddCommand(createCmd)

	f := createCmd.Flags()
	f.StringVar(&createFlags.alias, "alias", "", "SSH host alias")
	f.StringVar(&createFlags.host, "host", "", "target hostname or IP")
	f.StringVar(&createFlags.sshUser, "user", "", "SSH user")
	f.StringVar(&createFlags.port, "port", "22", "SSH port")
	f.StringVar(&createFlags.identityFile, "identity-file", "", "SSH identity file path")
	f.StringVar(&createFlags.proxyType, "proxy-type", "", "proxy type: socks5, http, tor, jump")
	f.StringVar(&createFlags.proxyHost, "proxy-host", "", "proxy hostname or IP")
	f.StringVar(&createFlags.proxyPort, "proxy-port", "1080", "proxy port")
	f.StringVar(&createFlags.proxyUser, "proxy-user", "", "proxy username")
	f.StringVar(&createFlags.proxyPass, "proxy-pass", "", "proxy password (WARNING: visible in process list)")
	f.StringVar(&createFlags.proxyPassFile, "proxy-pass-file", "", "read proxy password from file (recommended)")
	f.StringVar(&createFlags.jumpHost, "jump-host", "", "SSH jump host (user@host)")
}

// currentUser returns the current OS username, or empty string on error.
func currentUser() string {
	if u, err := user.Current(); err == nil {
		return u.Username
	}
	return ""
}

func runWizard() error {
	fmt.Println()
	fmt.Println("  sshady -- SSH proxy config generator")
	fmt.Println()

	cu := currentUser()

	var base struct {
		Alias        string
		HostName     string
		User         string
		Port         string
		IdentityFile string
		ProxyType    string
	}

	baseQuestions := []*survey.Question{
		{
			Name:     "Alias",
			Prompt:   &survey.Input{Message: "Host alias (e.g. myserver):"},
			Validate: survey.Required,
		},
		{
			Name:     "HostName",
			Prompt:   &survey.Input{Message: "Target hostname or IP:"},
			Validate: survey.Required,
		},
		{
			Name: "User",
			Prompt: &survey.Input{
				Message: "SSH user:",
				Default: cu,
			},
			Validate: survey.Required,
		},
		{
			Name: "Port",
			Prompt: &survey.Input{
				Message: "SSH port:",
				Default: "22",
			},
		},
		{
			Name:   "IdentityFile",
			Prompt: &survey.Input{Message: "SSH identity file (leave empty to skip):"},
		},
		{
			Name: "ProxyType",
			Prompt: &survey.Select{
				Message: "Proxy type:",
				Options: []string{
					"SOCKS5",
					"HTTP CONNECT",
					"Tor (auto: 127.0.0.1:9050)",
					"SSH Jump Host",
				},
			},
			Validate: survey.Required,
		},
	}

	if err := survey.Ask(baseQuestions, &base); err != nil {
		return fmt.Errorf("wizard cancelled: %w", err)
	}

	proxyCfg := proxy.Config{}

	switch base.ProxyType {
	case "SOCKS5", "HTTP CONNECT":
		var proxyAnswers struct {
			ProxyHost string
			ProxyPort string
			ProxyAuth bool
			ProxyUser string
			ProxyPass string
		}

		proxyQ := []*survey.Question{
			{
				Name:     "ProxyHost",
				Prompt:   &survey.Input{Message: "Proxy host:"},
				Validate: survey.Required,
			},
			{
				Name: "ProxyPort",
				Prompt: &survey.Input{
					Message: "Proxy port:",
					Default: "1080",
				},
			},
			{
				Name:   "ProxyAuth",
				Prompt: &survey.Confirm{Message: "Authentication required?", Default: false},
			},
		}
		if err := survey.Ask(proxyQ, &proxyAnswers); err != nil {
			return fmt.Errorf("wizard cancelled: %w", err)
		}

		if proxyAnswers.ProxyAuth {
			authQ := []*survey.Question{
				{
					Name:     "ProxyUser",
					Prompt:   &survey.Input{Message: "Proxy username:"},
					Validate: survey.Required,
				},
				{
					Name:     "ProxyPass",
					Prompt:   &survey.Password{Message: "Proxy password:"},
					Validate: survey.Required,
				},
			}
			if err := survey.Ask(authQ, &proxyAnswers); err != nil {
				return fmt.Errorf("wizard cancelled: %w", err)
			}
		}

		pType := proxy.TypeSOCKS5
		if base.ProxyType == "HTTP CONNECT" {
			pType = proxy.TypeHTTP
		}
		proxyCfg = proxy.Config{
			Type:     pType,
			Host:     proxyAnswers.ProxyHost,
			Port:     proxyAnswers.ProxyPort,
			Username: proxyAnswers.ProxyUser,
			Password: proxyAnswers.ProxyPass,
		}

	case "Tor (auto: 127.0.0.1:9050)":
		proxyCfg = proxy.Config{Type: proxy.TypeTor}

	case "SSH Jump Host":
		var jumpAnswer struct {
			JumpHost string
		}
		if err := survey.Ask([]*survey.Question{
			{
				Name:     "JumpHost",
				Prompt:   &survey.Input{Message: "Jump host (user@host or host):"},
				Validate: survey.Required,
			},
		}, &jumpAnswer); err != nil {
			return fmt.Errorf("wizard cancelled: %w", err)
		}
		proxyCfg = proxy.Config{Type: proxy.TypeJump, JumpHost: jumpAnswer.JumpHost}
	}

	cfg := sshconf.HostConfig{
		Alias:        base.Alias,
		HostName:     base.HostName,
		User:         base.User,
		Port:         base.Port,
		IdentityFile: base.IdentityFile,
		Proxy:        proxyCfg,
	}

	// Validate the config before showing summary
	if err := sshconf.ValidateHostConfig(cfg); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	fmt.Println()
	fmt.Println("  Summary:")
	fmt.Printf("    Host %-15s  ->  %s (user: %s, port: %s)
", cfg.Alias, cfg.HostName, cfg.User, cfg.Port)
	if cfg.IdentityFile != "" {
		fmt.Printf("    Identity:  %s
", cfg.IdentityFile)
	}
	fmt.Printf("    Proxy:     %s
", cfg.Proxy.Summary())
	fmt.Println()

	if dryRun {
		fmt.Println("[DRY RUN] Would write the following to ~/.ssh/config:")
		fmt.Println()
		fmt.Print(cfg.Block())
		fmt.Println("
Run without --dry-run to apply.")
		return nil
	}

	var confirm bool
	if err := survey.AskOne(&survey.Confirm{
		Message: "Write to ~/.ssh/config?",
		Default: true,
	}, &confirm); err != nil || !confirm {
		fmt.Println("Aborted.")
		return nil
	}

	if err := sshconf.WriteEntry(cfg); err != nil {
		return err
	}

	fmt.Printf("
Added 'Host %s' to ~/.ssh/config
", cfg.Alias)
	fmt.Printf("Connect with:  ssh %s

", cfg.Alias)
	return nil
}

func runNonInteractive() error {
	sshUser := createFlags.sshUser
	if sshUser == "" {
		sshUser = currentUser()
	}

	cfg := sshconf.HostConfig{
		Alias:        createFlags.alias,
		HostName:     createFlags.host,
		User:         sshUser,
		Port:         createFlags.port,
		IdentityFile: createFlags.identityFile,
	}

	switch createFlags.proxyType {
	case "socks5":
		if createFlags.proxyHost == "" {
			return fmt.Errorf("--proxy-host is required for proxy type 'socks5'")
		}
		cfg.Proxy = proxy.Config{
			Type:     proxy.TypeSOCKS5,
			Host:     createFlags.proxyHost,
			Port:     createFlags.proxyPort,
			Username: createFlags.proxyUser,
			Password: createFlags.proxyPass,
		}
	case "http":
		if createFlags.proxyHost == "" {
			return fmt.Errorf("--proxy-host is required for proxy type 'http'")
		}
		cfg.Proxy = proxy.Config{
			Type:     proxy.TypeHTTP,
			Host:     createFlags.proxyHost,
			Port:     createFlags.proxyPort,
			Username: createFlags.proxyUser,
			Password: createFlags.proxyPass,
		}
	case "tor":
		cfg.Proxy = proxy.Config{Type: proxy.TypeTor}
	case "jump":
		if createFlags.jumpHost == "" {
			return fmt.Errorf("--jump-host is required for proxy type 'jump'")
		}
		cfg.Proxy = proxy.Config{Type: proxy.TypeJump, JumpHost: createFlags.jumpHost}
	default:
		return fmt.Errorf("unknown proxy type %q; use: socks5, http, tor, jump", createFlags.proxyType)
	}

	// Validate before writing
	if err := sshconf.ValidateHostConfig(cfg); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if dryRun {
		fmt.Println("[DRY RUN] Would write the following to ~/.ssh/config:")
		fmt.Println()
		fmt.Print(cfg.Block())
		fmt.Println("
Run without --dry-run to apply.")
		return nil
	}

	if err := sshconf.WriteEntry(cfg); err != nil {
		return err
	}

	fmt.Printf("Added 'Host %s' to ~/.ssh/config
", cfg.Alias)
	fmt.Printf("Connect with:  ssh %s
", cfg.Alias)
	return nil
}
