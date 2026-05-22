// Package cmd builds the FamilyJourney CLI command tree.
package cmd

import (
	"runtime/debug"

	"github.com/spf13/cobra"
	"github.com/theinventor/familyjourney-cli/internal/client"
)

var Version = "dev"

var rootProfile string

func init() {
	if Version != "dev" {
		return
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	v := info.Main.Version
	if v == "" || v == "(devel)" {
		return
	}
	Version = v
}

func NewRootCmd() *cobra.Command {
	rootProfile = ""
	root := &cobra.Command{
		Use:   "familyjourney",
		Short: "FamilyJourney CLI for AI agents and parent API workflows",
		Long: `familyjourney is the official CLI for FamilyJourney's parent-only REST API.

It is designed for non-interactive agents: JSON goes to stdout by default,
API errors return useful non-zero exit codes, and saved profiles can be
overridden with FAMILYJOURNEY_API_TOKEN and FAMILYJOURNEY_API_URL.`,
		Version:       Version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().StringVar(&rootProfile, "profile", "", "saved auth profile to use for this invocation")

	root.AddCommand(newAuthCmd())
	root.AddCommand(newSkillCmd())
	root.AddCommand(newWhoamiCmd())
	root.AddCommand(newFamilyCmd())
	root.AddCommand(newKidsCmd())
	root.AddCommand(newBadgesCmd())
	root.AddCommand(newSubmissionsCmd())
	root.AddCommand(newPrizesCmd())
	root.AddCommand(newRedemptionsCmd())
	root.AddCommand(newCategoriesCmd())
	root.AddCommand(newGroupsCmd())
	root.AddCommand(newChallengesCmd())

	return root
}

func newAPIClient() *client.Client {
	c := client.NewWithProfile(rootProfile)
	c.Version = Version
	return c
}
