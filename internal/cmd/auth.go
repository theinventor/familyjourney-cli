package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/spf13/cobra"
	"github.com/theinventor/familyjourney-cli/internal/client"
	"github.com/theinventor/familyjourney-cli/internal/config"
	"github.com/theinventor/familyjourney-cli/internal/credstore"
	"github.com/theinventor/familyjourney-cli/internal/exitcode"
)

const storageFlagDescription = "where to persist the API token: auto (default; keychain if available, else file), keychain, or file"

func newAuthCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "auth",
		Short: "Manage saved parent API credentials",
		Long: `Manage FamilyJourney parent API credentials.

Resolution order:
  1. --profile <name>
  2. FAMILYJOURNEY_API_TOKEN and FAMILYJOURNEY_API_URL
  3. default saved profile

auth logout only removes the local profile. It does not call the server
logout endpoint because that endpoint rotates the parent API token.`,
	}
	c.AddCommand(newAuthSaveCmd())
	c.AddCommand(newAuthLoginCmd())
	c.AddCommand(newAuthStatusCmd())
	c.AddCommand(newAuthListCmd())
	c.AddCommand(newAuthUseCmd())
	c.AddCommand(newAuthLogoutCmd())
	return c
}

func resolveStorage(storage string) (string, error) {
	backend, err := credstore.ResolveBackend(storage)
	if err != nil {
		return "", exitcode.Wrap(exitcode.Usage, err)
	}
	return backend, nil
}

func persistProfile(name string, p config.Profile, secret, backend string) (string, error) {
	canon, err := credstore.Put(name, backend, secret)
	if err != nil {
		return "", err
	}
	switch canon {
	case credstore.BackendKeychain:
		p.APIToken = ""
	case credstore.BackendFile:
		p.APIToken = secret
	}
	p.Backend = canon

	f, err := config.Load()
	if err != nil {
		return "", err
	}
	f.Put(name, p)
	if err := f.Save(); err != nil {
		return "", err
	}
	return canon, nil
}

func newAuthSaveCmd() *cobra.Command {
	var profileName, apiToken, apiURL, userEmail, userName, storage string
	c := &cobra.Command{
		Use:   "save",
		Short: "Save an existing parent API token as a profile",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if profileName == "" || apiToken == "" {
				return exitcode.Wrap(exitcode.Usage, fmt.Errorf("--profile and --api-token are both required"))
			}
			if apiURL == "" {
				apiURL = client.DefaultAPIURL
			}
			backend, err := resolveStorage(storage)
			if err != nil {
				return err
			}
			canon, err := persistProfile(profileName, config.Profile{
				APIURL:    strings.TrimRight(apiURL, "/"),
				UserEmail: userEmail,
				UserName:  userName,
			}, apiToken, backend)
			if err != nil {
				return err
			}
			f, _ := config.Load()
			return printJSON(cmd.OutOrStdout(), map[string]any{
				"profile":         profileName,
				"api_url":         strings.TrimRight(apiURL, "/"),
				"api_token":       client.MaskToken(apiToken),
				"backend":         credstore.Describe(canon),
				"config_path":     config.Path(),
				"default_profile": f != nil && f.DefaultProfile == profileName,
			})
		},
	}
	c.Flags().StringVar(&profileName, "profile", "", "profile name (required)")
	c.Flags().StringVar(&apiToken, "api-token", "", "parent API token (required)")
	c.Flags().StringVar(&apiURL, "api-url", "", "FamilyJourney base URL (default: production)")
	c.Flags().StringVar(&userEmail, "user-email", "", "parent email metadata (optional)")
	c.Flags().StringVar(&userName, "user-name", "", "parent name metadata (optional)")
	c.Flags().StringVar(&storage, "storage", "", storageFlagDescription)
	return c
}

func newAuthLoginCmd() *cobra.Command {
	var email, password, profileName, apiURL, storage string
	c := &cobra.Command{
		Use:   "login",
		Short: "Login with parent email/password and save the returned API token",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if email == "" || password == "" || profileName == "" {
				return exitcode.Wrap(exitcode.Usage, fmt.Errorf("--email, --password, and --profile are required"))
			}
			if apiURL == "" {
				apiURL = client.DefaultAPIURL
			}
			backend, err := resolveStorage(storage)
			if err != nil {
				return err
			}

			cli := &client.Client{
				BaseURL:    strings.TrimRight(apiURL, "/"),
				HTTPClient: http.DefaultClient,
				Version:    Version,
			}
			resp, err := cli.Do(http.MethodPost, "/api/v1/auth/login", map[string]any{
				"email":    email,
				"password": password,
			}, nil)
			if err != nil {
				return exitcode.Wrap(exitcode.Network, fmt.Errorf("POST /api/v1/auth/login: %w", err))
			}
			defer resp.Body.Close()
			raw, _ := io.ReadAll(resp.Body)
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				if len(raw) > 0 {
					_, _ = cmd.OutOrStdout().Write(raw)
					_, _ = fmt.Fprintln(cmd.OutOrStdout())
				}
				return exitcode.Wrap(exitcode.FromHTTPStatus(resp.StatusCode), fmt.Errorf("HTTP %d from POST /api/v1/auth/login", resp.StatusCode))
			}
			var parsed struct {
				Token string `json:"token"`
				User  struct {
					Email    string `json:"email"`
					Name     string `json:"name"`
					FamilyID any    `json:"family_id"`
				} `json:"user"`
			}
			if err := json.Unmarshal(raw, &parsed); err != nil {
				return fmt.Errorf("parse login response: %w", err)
			}
			if parsed.Token == "" {
				return fmt.Errorf("login response did not include token")
			}
			canon, err := persistProfile(profileName, config.Profile{
				APIURL:    strings.TrimRight(apiURL, "/"),
				UserEmail: parsed.User.Email,
				UserName:  parsed.User.Name,
				FamilyID:  parsed.User.FamilyID,
			}, parsed.Token, backend)
			if err != nil {
				return err
			}
			return printJSON(cmd.OutOrStdout(), map[string]any{
				"profile":     profileName,
				"api_url":     strings.TrimRight(apiURL, "/"),
				"api_token":   client.MaskToken(parsed.Token),
				"backend":     credstore.Describe(canon),
				"config_path": config.Path(),
				"user_email":  parsed.User.Email,
				"user_name":   parsed.User.Name,
				"family_id":   parsed.User.FamilyID,
			})
		},
	}
	c.Flags().StringVar(&email, "email", "", "parent email (required)")
	c.Flags().StringVar(&password, "password", "", "parent password (required)")
	c.Flags().StringVar(&profileName, "profile", "", "profile name (required)")
	c.Flags().StringVar(&apiURL, "api-url", "", "FamilyJourney base URL (default: production)")
	c.Flags().StringVar(&storage, "storage", "", storageFlagDescription)
	return c
}

func newAuthStatusCmd() *cobra.Command {
	var profile string
	c := &cobra.Command{
		Use:   "status",
		Short: "Show resolved credential status as JSON",
		RunE: func(cmd *cobra.Command, _ []string) error {
			activeProfile := profile
			if activeProfile == "" {
				activeProfile = rootProfile
			}
			cli := client.NewWithProfile(activeProfile)
			cli.Version = Version
			rec := map[string]any{
				"api_url":     cli.BaseURL,
				"api_token":   cli.MaskedAPIToken(),
				"source":      nil,
				"backend":     credstore.Describe(cli.Backend),
				"cli_version": Version,
				"config_path": config.Path(),
			}
			if cli.Source != "" {
				rec["source"] = cli.Source
			}
			if strings.HasPrefix(cli.Source, "profile:") {
				name := strings.TrimPrefix(cli.Source, "profile:")
				rec["profile"] = name
				if f, err := config.Load(); err == nil {
					if p, ok := f.Get(name); ok {
						rec["user_email"] = p.UserEmail
						rec["user_name"] = p.UserName
						rec["family_id"] = p.FamilyID
						rec["saved_at"] = p.CreatedAt
					}
				}
			}
			resp, err := cli.Do(http.MethodGet, "/api/v1/auth/me", nil, nil)
			if err != nil {
				rec["reachable"] = false
				rec["reachable_error"] = err.Error()
			} else {
				defer resp.Body.Close()
				rec["reachable"] = resp.StatusCode < 500
				rec["auth_ok"] = resp.StatusCode >= 200 && resp.StatusCode < 300
				rec["server_status"] = resp.StatusCode
			}
			return printJSON(cmd.OutOrStdout(), rec)
		},
	}
	c.Flags().StringVar(&profile, "profile", "", "show status for a specific profile")
	return c
}

func newAuthListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List saved profiles as JSON",
		RunE: func(cmd *cobra.Command, _ []string) error {
			f, err := config.Load()
			if err != nil {
				return err
			}
			profiles := make([]map[string]any, 0, len(f.Profiles))
			for _, name := range f.Names() {
				p := f.Profiles[name]
				profiles = append(profiles, map[string]any{
					"name":       name,
					"api_url":    p.APIURL,
					"api_token":  client.MaskToken(p.APIToken),
					"backend":    credstore.Describe(p.Backend),
					"is_default": name == f.DefaultProfile,
					"user_email": p.UserEmail,
					"user_name":  p.UserName,
					"family_id":  p.FamilyID,
					"created_at": p.CreatedAt,
				})
			}
			return printJSON(cmd.OutOrStdout(), map[string]any{
				"config_path":     config.Path(),
				"default_profile": f.DefaultProfile,
				"profiles":        profiles,
			})
		},
	}
}

func newAuthUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use <profile>",
		Short: "Set the default saved profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := config.Load()
			if err != nil {
				return err
			}
			if err := f.SetDefault(args[0]); err != nil {
				return exitcode.Wrap(exitcode.NotFound, err)
			}
			if err := f.Save(); err != nil {
				return err
			}
			return printJSON(cmd.OutOrStdout(), map[string]any{
				"default_profile": args[0],
				"config_path":     config.Path(),
			})
		},
	}
}

func newAuthLogoutCmd() *cobra.Command {
	var profile string
	var force bool
	c := &cobra.Command{
		Use:   "logout",
		Short: "Remove a local saved profile without rotating the server token",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_ = force
			f, err := config.Load()
			if err != nil {
				return err
			}
			name := profile
			if name == "" {
				name = f.DefaultProfile
			}
			if name == "" {
				return exitcode.Wrap(exitcode.Usage, fmt.Errorf("no profile to remove"))
			}
			if !f.Delete(name) {
				return exitcode.Wrap(exitcode.NotFound, fmt.Errorf("no profile named %q", name))
			}
			if err := f.Save(); err != nil {
				return err
			}
			_ = credstore.Delete(name)
			return printJSON(cmd.OutOrStdout(), map[string]any{
				"removed_profile": name,
				"default_profile": f.DefaultProfile,
				"config_path":     config.Path(),
			})
		},
	}
	c.Flags().StringVar(&profile, "profile", "", "profile to remove (default: current default)")
	c.Flags().BoolVar(&force, "force", false, "accepted for non-interactive compatibility; logout never calls the server")
	return c
}
