package cmd

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

//go:embed embedded/familyjourney_skill.md
var familyJourneySkill string

func newSkillCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "skill",
		Short: "Print bundled agent skills",
		Long:  "Print bundled skills that teach AI assistants to use the FamilyJourney CLI safely.",
	}
	c.AddCommand(newSkillGetCmd())
	return c
}

func newSkillGetCmd() *cobra.Command {
	var output string
	c := &cobra.Command{
		Use:   "get familyjourney",
		Short: "Get the bundled FamilyJourney agent skill",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 || args[0] != "familyjourney" {
				return fmt.Errorf("usage: familyjourney skill get familyjourney")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			if output == "" || output == "-" {
				_, err := fmt.Fprint(cmd.OutOrStdout(), familyJourneySkill)
				return err
			}
			if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
				return err
			}
			return os.WriteFile(output, []byte(familyJourneySkill), 0o644)
		},
	}
	c.Flags().StringVarP(&output, "output", "o", "", "write skill to this file instead of stdout")
	return c
}
