package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/theinventor/familyjourney-cli/internal/exitcode"
)

func printJSON(out io.Writer, v any) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func apiRequest(cmd *cobra.Command, method, path string, body any, query url.Values) error {
	cli := newAPIClient()
	resp, err := cli.Do(method, path, body, query)
	if err != nil {
		return exitcode.Wrap(exitcode.Network, fmt.Errorf("%s %s: %w", method, path, err))
	}
	defer resp.Body.Close()

	raw, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return exitcode.Wrap(exitcode.Network, fmt.Errorf("read response body: %w", readErr))
	}

	if len(raw) > 0 {
		if _, err := cmd.OutOrStdout().Write(raw); err != nil {
			return err
		}
		if !strings.HasSuffix(string(raw), "\n") {
			if _, err := fmt.Fprintln(cmd.OutOrStdout()); err != nil {
				return err
			}
		}
	} else if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if err := printJSON(cmd.OutOrStdout(), map[string]any{"ok": true}); err != nil {
			return err
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := fmt.Sprintf("HTTP %d from %s %s", resp.StatusCode, method, path)
		if len(raw) == 0 {
			msg += "; empty response body, check FAMILYJOURNEY_API_URL"
		}
		return exitcode.Wrap(exitcode.FromHTTPStatus(resp.StatusCode), fmt.Errorf("%s", msg))
	}
	return nil
}

func requireAll(values map[string]string) error {
	for name, value := range values {
		if strings.TrimSpace(value) == "" {
			return exitcode.Wrap(exitcode.Usage, fmt.Errorf("%s is required", name))
		}
	}
	return nil
}

func requirePositiveInt(name string, value int) error {
	if value <= 0 {
		return exitcode.Wrap(exitcode.Usage, fmt.Errorf("%s must be greater than zero", name))
	}
	return nil
}

func requireNonNegativeIntFlag(cmd *cobra.Command, flagName string, value int) error {
	if !flagChanged(cmd, flagName) {
		return exitcode.Wrap(exitcode.Usage, fmt.Errorf("--%s is required", flagName))
	}
	if value < 0 {
		return exitcode.Wrap(exitcode.Usage, fmt.Errorf("--%s must be zero or greater", flagName))
	}
	return nil
}

func flagChanged(cmd *cobra.Command, name string) bool {
	f := cmd.Flags().Lookup(name)
	return f != nil && f.Changed
}

func changedMapEmpty(attrs map[string]any) bool {
	return len(attrs) == 0
}

func requireChanged(attrs map[string]any) error {
	if changedMapEmpty(attrs) {
		return exitcode.Wrap(exitcode.Usage, fmt.Errorf("at least one field flag is required"))
	}
	return nil
}

func requireForce(force bool, resource string) error {
	if force {
		return nil
	}
	return exitcode.Wrap(exitcode.Usage, fmt.Errorf("%s delete requires --force", resource))
}

func addString(attrs map[string]any, name, value string) {
	if value != "" {
		attrs[name] = value
	}
}

func addStringChanged(cmd *cobra.Command, attrs map[string]any, flagName, jsonName, value string) {
	if flagChanged(cmd, flagName) {
		attrs[jsonName] = value
	}
}

func addIntChanged(cmd *cobra.Command, attrs map[string]any, flagName, jsonName string, value int) {
	if flagChanged(cmd, flagName) {
		attrs[jsonName] = value
	}
}

func addBoolChanged(cmd *cobra.Command, attrs map[string]any, flagName, jsonName string, value bool) {
	if flagChanged(cmd, flagName) {
		attrs[jsonName] = value
	}
}

func addIntSliceChanged(cmd *cobra.Command, attrs map[string]any, flagName, jsonName string, value []int) {
	if flagChanged(cmd, flagName) {
		attrs[jsonName] = value
	}
}

func nested(key string, attrs map[string]any) map[string]any {
	return map[string]any{key: attrs}
}

func oneIDArg(cmd *cobra.Command, args []string) (string, error) {
	if len(args) != 1 {
		return "", fmt.Errorf("usage: %s", cmd.UseLine())
	}
	id := strings.TrimSpace(args[0])
	if id == "" {
		return "", exitcode.Wrap(exitcode.Usage, fmt.Errorf("id is required"))
	}
	return id, nil
}

func addCommonDeleteFlags(c *cobra.Command, force *bool) {
	c.Flags().BoolVar(force, "force", false, "required confirmation flag for destructive deletes")
}

func stringArrayFromFlags(flags *pflag.FlagSet, name string) []string {
	values, err := flags.GetStringArray(name)
	if err != nil {
		return nil
	}
	return values
}

func statusQuery(status string) url.Values {
	q := url.Values{}
	if status != "" {
		q.Set("status", status)
	}
	return q
}

func methodGet(path string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		return apiRequest(cmd, http.MethodGet, path, nil, nil)
	}
}
