package cmd

import (
	"fmt"
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
	jumpHost     string
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new SSH proxy configuration",
	Long: `Create a new SSH Host entry with proxy settings.

Without flags, launches the interactive wizard.
Provide --alias, --host, and --proxy-type (plus proxy-specific flags) to skip the wizard.`,
	Example: `  sshady create
  sshady create --alias web --host 1.2.3.4 --proxy-type tor
  sshady create --alias web --host 1.2.3.4 --proxy-type socks5 \
    --proxy-host proxy.example.com --proxy-port 1080 \
    --proxy-user alice --proxy-pass s3cr3t
  sshady create --alias internal --host 10.0.0.5 --proxy-type jump \
    --jump-host bastion@192.168.1.1`,
	RunE: func(cmd *cobra.Command, args []string) error {
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
	f.StringVar(&createFlags.proxyPass, "proxy-pass", "", "proxy password")
	f.StringVar(&createFlags.jumpHost, "jump-host", "", "SSH jump host (user@host)")
}

type entryDefaults struct {
	alias        string
	host         string
	user         string
	port         string
	identityFile string
	proxyType    proxy.Type
	proxyHost    string
	proxyPort    string
	proxyUser    string
	proxyPass    string
	jumpHost     string
	hasExisting  bool
}

func proxyTypeLabel(t proxy.Type) string {
	switch t {
	case proxy.TypeSOCKS5:
		return "SOCKS5"
	case proxy.TypeHTTP:
		return "HTTP CONNECT"
	case proxy.TypeTor:
		return "Tor (auto: 127.0.0.1:9050)"
	case proxy.TypeJump:
		return "SSH Jump Host"
	}
	return ""
}

func askEntry(d entryDefaults) (sshconf.HostConfig, error) {
	currentUser := d.user
	if currentUser == "" {
		currentUser = "root"
		if u, err := user.Current(); err == nil {
			currentUser = u.Username
		}
	}
	port := d.port
	if port == "" {
		port = "22"
	}

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
				Default: d.alias,
			},
			Validate: survey.Required,
		},
		{
			Name: "HostName",
			Prompt: &survey.Input{
				Message: "Target hostname or IP:",
				Default: d.host,
			},
			Validate: survey.Required,
		},
		{
			Name: "User",
			Prompt: &survey.Input{
				Message: "SSH user:",
				Default: currentUser,
			},
			Validate: survey.Required,
		},
		{
			Name: "Port",
			Prompt: &survey.Input{
				Message: "SSH port:",
				Default: port,
			},
		},
		{
			Name: "IdentityFile",
			Prompt: &survey.Input{
				Message: "SSH identity file (leave empty to skip):",
				Default: d.identityFile,
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
				Default: proxyTypeLabel(d.proxyType),
			},
			Validate: survey.Required,
		},
	}

	if err := survey.Ask(baseQuestions, &base); err != nil {
		return sshconf.HostConfig{}, fmt.Errorf("wizard cancelled")
	}

	proxyCfg := proxy.Config{}

	switch base.ProxyType {
	case "SOCKS5", "HTTP CONNECT":
		pType := proxy.TypeSOCKS5
		if base.ProxyType == "HTTP CONNECT" {
			pType = proxy.TypeHTTP
		}

		sameType := d.proxyType == pType
		proxyHost := ""
		proxyPort := "1080"
		proxyUser := ""
		hasAuth := false
		if sameType {
			proxyHost = d.proxyHost
			proxyPort = d.proxyPort
			proxyUser = d.proxyUser
			hasAuth = d.proxyUser != "" || d.proxyPass != ""
		}

		var proxyAnswers struct {
			ProxyHost string
			ProxyPort string
			ProxyAuth bool
			ProxyUser string
			ProxyPass string
		}

		proxyQ := []*survey.Question{
			{
				Name: "ProxyHost",
				Prompt: &survey.Input{
					Message: "Proxy host:",
					Default: proxyHost,
				},
				Validate: survey.Required,
			},
			{
				Name: "ProxyPort",
				Prompt: &survey.Input{
					Message: "Proxy port:",
					Default: proxyPort,
				},
			},
			{
				Name:   "ProxyAuth",
				Prompt: &survey.Confirm{Message: "Authentication required?", Default: hasAuth},
			},
		}
		if err := survey.Ask(proxyQ, &proxyAnswers); err != nil {
			return sshconf.HostConfig{}, fmt.Errorf("wizard cancelled")
		}

		if proxyAnswers.ProxyAuth {
			passPrompt := &survey.Password{Message: "Proxy password:"}
			var passValidate survey.Validator = survey.Required
			if sameType && d.proxyPass != "" {
				passPrompt = &survey.Password{Message: "Proxy password (leave empty to keep current):"}
				passValidate = nil
			}

			authQ := []*survey.Question{
				{
					Name: "ProxyUser",
					Prompt: &survey.Input{
						Message: "Proxy username:",
						Default: proxyUser,
					},
					Validate: survey.Required,
				},
				{
					Name:     "ProxyPass",
					Prompt:   passPrompt,
					Validate: passValidate,
				},
			}
			if err := survey.Ask(authQ, &proxyAnswers); err != nil {
				return sshconf.HostConfig{}, fmt.Errorf("wizard cancelled")
			}
			if proxyAnswers.ProxyPass == "" && sameType {
				proxyAnswers.ProxyPass = d.proxyPass
			}
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
		jumpDefault := ""
		if d.proxyType == proxy.TypeJump {
			jumpDefault = d.jumpHost
		}
		var jumpAnswer struct {
			JumpHost string
		}
		if err := survey.Ask([]*survey.Question{
			{
				Name: "JumpHost",
				Prompt: &survey.Input{
					Message: "Jump host (user@host or host):",
					Default: jumpDefault,
				},
				Validate: survey.Required,
			},
		}, &jumpAnswer); err != nil {
			return sshconf.HostConfig{}, fmt.Errorf("wizard cancelled")
		}
		proxyCfg = proxy.Config{Type: proxy.TypeJump, JumpHost: jumpAnswer.JumpHost}
	}

	return sshconf.HostConfig{
		Alias:        base.Alias,
		HostName:     base.HostName,
		User:         base.User,
		Port:         base.Port,
		IdentityFile: base.IdentityFile,
		Proxy:        proxyCfg,
	}, nil
}

func printSummary(cfg sshconf.HostConfig) {
	fmt.Println()
	fmt.Println("  Summary:")
	fmt.Printf("    Host %-15s  ->  %s (user: %s, port: %s)\n", cfg.Alias, cfg.HostName, cfg.User, cfg.Port)
	if cfg.IdentityFile != "" {
		fmt.Printf("    Identity:  %s\n", cfg.IdentityFile)
	}
	fmt.Printf("    Proxy:     %s\n", cfg.Proxy.Summary())
	fmt.Println()
}

func confirmWrite() bool {
	var confirm bool
	if err := survey.AskOne(&survey.Confirm{
		Message: "Write to ~/.ssh/config?",
		Default: true,
	}, &confirm); err != nil || !confirm {
		fmt.Println("Aborted.")
		return false
	}
	return true
}

func runWizard() error {
	fmt.Println()
	fmt.Println("  sshady -- SSH proxy config generator")
	fmt.Println()

	cfg, err := askEntry(entryDefaults{})
	if err != nil {
		return err
	}

	printSummary(cfg)

	if !confirmWrite() {
		return nil
	}

	if err := sshconf.WriteEntry(cfg); err != nil {
		return err
	}

	fmt.Printf("\nAdded 'Host %s' to ~/.ssh/config\n", cfg.Alias)
	fmt.Printf("Connect with:  ssh %s\n\n", cfg.Alias)
	return nil
}

func runNonInteractive() error {
	sshUser := createFlags.sshUser
	if sshUser == "" {
		if u, err := user.Current(); err == nil {
			sshUser = u.Username
		}
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

	if err := sshconf.WriteEntry(cfg); err != nil {
		return err
	}

	fmt.Printf("Added 'Host %s' to ~/.ssh/config\n", cfg.Alias)
	fmt.Printf("Connect with:  ssh %s\n", cfg.Alias)
	return nil
}
