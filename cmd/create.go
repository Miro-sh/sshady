package cmd

import (
	"fmt"
	"os"
	"os/user"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"

	"sshady/internal/proxy"
	"sshady/internal/sshconf"
)

var createFlags struct {
	alias         string
	host          string
	sshUser       string
	port          string
	identityFile  string
	proxyType     string
	proxyHost     string
	proxyPort     string
	proxyUser     string
	proxyPass     string
	proxyPassFile string
	jumpHost      string
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new SSH proxy configuration",
	Long: `Create a new SSH Host entry with proxy settings in ~/.ssh/config.

Without flags, launches the interactive wizard.
Provide --alias, --host, and --proxy-type to skip the wizard.

For proxy passwords, use --proxy-pass-file (reads from file) or set the
SSHADY_PROXY_PASS environment variable. Avoid --proxy-pass on the command line
as it exposes the password in the process list (visible via ps aux).`,
	Example: `  # Interactive wizard
  sshady create

  # SOCKS4 proxy (no auth)
  sshady create --alias s4 --host 10.0.0.5 \n    --proxy-type socks4 --proxy-host proxy.example.com

  # SOCKS5 proxy (password via environment variable — most secure)
  SSHADY_PROXY_PASS=s3cr3t sshady create \
    --alias myserver --host 1.2.3.4 --user admin \
    --proxy-type socks5 --proxy-host proxy.example.com --proxy-user alice

  # HTTP CONNECT proxy
  sshady create --alias web --host 10.0.0.5 \
    --proxy-type http --proxy-host proxy.example.com --proxy-port 8080

  # Tor routing
  sshady create --alias hidden --host example.onion --proxy-type tor

  # SSH jump host (bastion)
  sshady create --alias internal --host 10.0.0.5 \
    --proxy-type jump --jump-host bastion@192.168.1.1

  # Dry-run to preview
  sshady create --alias test --host 1.2.3.4 --proxy-type tor --dry-run`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Resolve password: env var < file < flag
		if createFlags.proxyPass == "" {
			if envPass := os.Getenv("SSHADY_PROXY_PASS"); envPass != "" {
				createFlags.proxyPass = envPass
			}
		}
		if createFlags.proxyPassFile != "" {
			data, err := os.ReadFile(createFlags.proxyPassFile)
			if err != nil {
				return fmt.Errorf("cannot read proxy password file %q: %w", createFlags.proxyPassFile, err)
			}
			createFlags.proxyPass = strings.TrimRight(string(data), "

")
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
	f.StringVar(&createFlags.sshUser, "user", "", "SSH user (default: current user)")
	f.StringVar(&createFlags.port, "port", "22", "SSH port")
	f.StringVar(&createFlags.identityFile, "identity-file", "", "path to SSH private key")
	f.StringVar(&createFlags.proxyType, "proxy-type", "", "proxy type: socks4, socks4a, socks5, http, tor, jump")
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
	return os.Getenv("USER") // fallback for restricted environments
}

func runWizard() error {
	fmt.Println()
	fmt.Println("  sshady — SSH proxy config generator")
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
			Name: "Alias",
			Prompt: &survey.Input{
				Message: "Host alias (e.g. myserver):",
				Help:    "A short, memorable name for 'ssh <alias>'. Alphanumeric, dots, underscores, hyphens only.",
			},
			Validate: survey.ComposeValidators(survey.Required, validateAliasField),
		},
		{
			Name: "HostName",
			Prompt: &survey.Input{
				Message: "Target hostname or IP:",
				Help:    "The actual hostname or IP address of the server you want to reach.",
			},
			Validate: survey.ComposeValidators(survey.Required, validateHostField),
		},
		{
			Name: "User",
			Prompt: &survey.Input{
				Message: "SSH user:",
				Default: cu,
				Help:    "The username for SSH authentication on the target server.",
			},
			Validate: survey.ComposeValidators(survey.Required, validateUserPassField),
		},
		{
			Name: "Port",
			Prompt: &survey.Input{
				Message: "SSH port:",
				Default: "22",
				Help:    "The port the SSH server listens on (default: 22).",
			},
			Validate: validatePortField,
		},
		{
			Name: "IdentityFile",
			Prompt: &survey.Input{
				Message: "SSH identity file (leave empty to skip):",
				Help:    "Path to an SSH private key file, e.g. ~/.ssh/id_ed25519.",
			},
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
				Help: "How your SSH traffic should be routed to hide your real IP.",
			},
			Validate: survey.Required,
		},
	}

	if err := survey.Ask(baseQuestions, &base); err != nil {
		return fmt.Errorf("wizard cancelled: %w", err)
	}

	proxyCfg, err := buildProxyConfigFromWizard(base)
	if err != nil {
		return err
	}

	cfg := sshconf.HostConfig{
		Alias:        base.Alias,
		HostName:     base.HostName,
		User:         base.User,
		Port:         base.Port,
		IdentityFile: base.IdentityFile,
		Proxy:        proxyCfg,
	}

	if err := sshconf.ValidateHostConfig(cfg); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	fmt.Println()
	fmt.Println("  Summary:")
	fmt.Printf("    Host %-15s ->  %s (user: %s, port: %s)\n", cfg.Alias, cfg.HostName, cfg.User, cfg.Port)
	if cfg.IdentityFile != "" {
		fmt.Printf("    Identity:  %s\n", cfg.IdentityFile)
	}
	fmt.Printf("    Proxy:     %s\n", cfg.Proxy.Summary())
	fmt.Println()

	if dryRun {
		fmt.Println("[DRY RUN] Would write the following to the SSH config:")
		fmt.Println()
		fmt.Print(cfg.Block())
		fmt.Println("\nRun without --dry-run to apply.")
		return nil
	}

	if !quietMode {
		var confirm bool
		if err := survey.AskOne(&survey.Confirm{
			Message: "Write to SSH config?",
			Default: true,
		}, &confirm); err != nil || !confirm {
			fmt.Println("Aborted.")
			return nil
		}
	}

	if err := sshconf.WriteEntry(cfg, forceWrite); err != nil {
		return err
	}

	fmt.Printf("\n✓ Added 'Host %s' to SSH config\n", cfg.Alias)
	fmt.Printf("  Connect with:  ssh %s\n\n", cfg.Alias)
	return nil
}

// buildProxyConfigFromWizard handles the proxy-specific wizard prompts.
func buildProxyConfigFromWizard(base struct {
	Alias        string
	HostName     string
	User         string
	Port         string
	IdentityFile string
	ProxyType    string
}) (proxy.Config, error) {
	switch base.ProxyType {
	case "SOCKS4", "SOCKS4a", "SOCKS5", "HTTP CONNECT":
		var pa struct {
			ProxyHost string
			ProxyPort string
			ProxyAuth bool
			ProxyUser string
			ProxyPass string
		}
		pq := []*survey.Question{
			{
				Name:     "ProxyHost",
				Prompt:   &survey.Input{Message: "Proxy host:", Help: "Hostname or IP of your proxy server."},
				Validate: survey.ComposeValidators(survey.Required, validateHostField),
			},
			{
				Name: "ProxyPort",
				Prompt: &survey.Input{
					Message: "Proxy port:",
					Default: "1080",
					Help:    "Port the proxy listens on.",
				},
				Validate: validatePortField,
			},
			{
				Name:   "ProxyAuth",
				Prompt: &survey.Confirm{Message: "Authentication required?", Default: false},
			},
		}
		if err := survey.Ask(pq, &pa); err != nil {
			return proxy.Config{}, fmt.Errorf("wizard cancelled: %w", err)
		}
		if pa.ProxyAuth {
			aq := []*survey.Question{
				{
					Name:     "ProxyUser",
					Prompt:   &survey.Input{Message: "Proxy username:"},
					Validate: survey.ComposeValidators(survey.Required, validateUserPassField),
				},
				{
					Name:     "ProxyPass",
					Prompt:   &survey.Password{Message: "Proxy password:"},
					Validate: survey.Required,
				},
			}
			if err := survey.Ask(aq, &pa); err != nil {
				return proxy.Config{}, fmt.Errorf("wizard cancelled: %w", err)
			}
		}
		pType := proxy.TypeSOCKS5
		switch base.ProxyType {
		case "SOCKS4":
			pType = proxy.TypeSOCKS4
		case "SOCKS4a":
			pType = proxy.TypeSOCKS4a
		case "HTTP CONNECT":
			pType = proxy.TypeHTTP
		}
		return proxy.Config{
			Type: pType, Host: pa.ProxyHost, Port: pa.ProxyPort,
			Username: pa.ProxyUser, Password: pa.ProxyPass,
		}, nil

	case "Tor (auto: 127.0.0.1:9050)":
		if verboseMode {
			fmt.Println("  Checking if Tor is running on 127.0.0.1:9050...")
		}
		return proxy.Config{Type: proxy.TypeTor}, nil

	case "SSH Jump Host":
		var ja struct{ JumpHost string }
		if err := survey.Ask([]*survey.Question{{
			Name:     "JumpHost",
			Prompt:   &survey.Input{Message: "Jump host (user@host or host):", Help: "Format: user@bastion-host"},
			Validate: survey.Required,
		}}, &ja); err != nil {
			return proxy.Config{}, fmt.Errorf("wizard cancelled: %w", err)
		}
		return proxy.Config{Type: proxy.TypeJump, JumpHost: ja.JumpHost}, nil
	}
	return proxy.Config{}, fmt.Errorf("unknown proxy type selected")
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
	case "socks4":
		if createFlags.proxyHost == "" {
			return fmt.Errorf("--proxy-host is required for proxy type 'socks4'")
		}
		cfg.Proxy = proxy.Config{
			Type: proxy.TypeSOCKS4, Host: createFlags.proxyHost, Port: createFlags.proxyPort,
			Username: createFlags.proxyUser, Password: createFlags.proxyPass,
		}
	case "socks4a":
		if createFlags.proxyHost == "" {
			return fmt.Errorf("--proxy-host is required for proxy type 'socks4a'")
		}
		cfg.Proxy = proxy.Config{
			Type: proxy.TypeSOCKS4a, Host: createFlags.proxyHost, Port: createFlags.proxyPort,
			Username: createFlags.proxyUser, Password: createFlags.proxyPass,
		}
	case "socks5":
		if createFlags.proxyHost == "" {
			return fmt.Errorf("--proxy-host is required for proxy type 'socks5'")
		}
		cfg.Proxy = proxy.Config{
			Type: proxy.TypeSOCKS5, Host: createFlags.proxyHost, Port: createFlags.proxyPort,
			Username: createFlags.proxyUser, Password: createFlags.proxyPass,
		}
	case "http":
		if createFlags.proxyHost == "" {
			return fmt.Errorf("--proxy-host is required for proxy type 'http'")
		}
		cfg.Proxy = proxy.Config{
			Type: proxy.TypeHTTP, Host: createFlags.proxyHost, Port: createFlags.proxyPort,
			Username: createFlags.proxyUser, Password: createFlags.proxyPass,
		}
	case "tor":
		cfg.Proxy = proxy.Config{Type: proxy.TypeTor}
	case "jump":
		if createFlags.jumpHost == "" {
			return fmt.Errorf("--jump-host is required for proxy type 'jump'")
		}
		cfg.Proxy = proxy.Config{Type: proxy.TypeJump, JumpHost: createFlags.jumpHost}
	default:
		return fmt.Errorf("unknown proxy type %q; supported: %s", createFlags.proxyType, proxy.AllowedTypeStrings())
	}

	if err := sshconf.ValidateHostConfig(cfg); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if dryRun {
		fmt.Println("[DRY RUN] Would write the following to the SSH config:")
		fmt.Println()
		fmt.Print(cfg.Block())
		fmt.Println("\nRun without --dry-run to apply.")
		return nil
	}

	if err := sshconf.WriteEntry(cfg, forceWrite); err != nil {
		return err
	}

	if !quietMode {
		fmt.Printf("✓ Added 'Host %s' to SSH config\n", cfg.Alias)
		fmt.Printf("  Connect with:  ssh %s\n", cfg.Alias)
	}
	return nil
}

// ── Wizard validators ─────────────────────────────────────

func validateAliasField(val interface{}) error {
	s, ok := val.(string)
	if !ok {
		return fmt.Errorf("expected string")
	}
	return proxy.ValidateAlias(s)
}

func validateHostField(val interface{}) error {
	s, ok := val.(string)
	if !ok {
		return fmt.Errorf("expected string")
	}
	return proxy.ValidateHost(s)
}

func validatePortField(val interface{}) error {
	s, ok := val.(string)
	if !ok {
		return fmt.Errorf("expected string")
	}
	if s == "" {
		return nil
	}
	return proxy.ValidatePort(s)
}

func validateUserPassField(val interface{}) error {
	s, ok := val.(string)
	if !ok {
		return fmt.Errorf("expected string")
	}
	return proxy.ValidateUserPass(s)
}
