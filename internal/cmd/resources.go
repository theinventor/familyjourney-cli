package cmd

import (
	"net/http"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"
)

func newWhoamiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Probe auth with GET /api/v1/auth/me",
		RunE:  methodGet("/api/v1/auth/me"),
	}
}

func newFamilyCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "family",
		Short: "Inspect the current FamilyJourney family",
	}
	c.AddCommand(&cobra.Command{
		Use:   "get",
		Short: "Get family overview and stats",
		RunE:  methodGet("/api/v1/family"),
	})
	return c
}

func newKidsCmd() *cobra.Command {
	c := &cobra.Command{Use: "kids", Short: "Manage kids"}
	c.AddCommand(&cobra.Command{Use: "list", Short: "List kids", RunE: methodGet("/api/v1/kids")})
	c.AddCommand(&cobra.Command{
		Use:   "get <kid-id>",
		Short: "Get one kid",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return apiRequest(cmd, http.MethodGet, "/api/v1/kids/"+args[0], nil, nil)
		},
	})
	c.AddCommand(newKidCreateCmd())
	c.AddCommand(newKidUpdateCmd())
	c.AddCommand(newKidDeleteCmd())
	c.AddCommand(&cobra.Command{
		Use:   "reset-password <kid-id>",
		Short: "Reset a kid password and return the generated password",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return apiRequest(cmd, http.MethodPost, "/api/v1/kids/"+args[0]+"/reset_password", nil, nil)
		},
	})
	return c
}

func newKidCreateCmd() *cobra.Command {
	var name, email string
	var groupIDs []int
	c := &cobra.Command{
		Use:   "create",
		Short: "Create a kid account",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := requireAll(map[string]string{"--name": name, "--email": email}); err != nil {
				return err
			}
			attrs := map[string]any{"name": name, "email": email}
			if len(groupIDs) > 0 {
				attrs["group_ids"] = groupIDs
			}
			return apiRequest(cmd, http.MethodPost, "/api/v1/kids", nested("kid", attrs), nil)
		},
	}
	c.Flags().StringVar(&name, "name", "", "kid name (required)")
	c.Flags().StringVar(&email, "email", "", "kid email (required)")
	c.Flags().IntSliceVar(&groupIDs, "group-id", nil, "group id to assign; repeat or comma-separate")
	return c
}

func newKidUpdateCmd() *cobra.Command {
	var name, email string
	var groupIDs []int
	c := &cobra.Command{
		Use:   "update <kid-id>",
		Short: "Update a kid account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			attrs := map[string]any{}
			addStringChanged(cmd, attrs, "name", "name", name)
			addStringChanged(cmd, attrs, "email", "email", email)
			addIntSliceChanged(cmd, attrs, "group-id", "group_ids", groupIDs)
			if err := requireChanged(attrs); err != nil {
				return err
			}
			return apiRequest(cmd, http.MethodPatch, "/api/v1/kids/"+args[0], nested("kid", attrs), nil)
		},
	}
	c.Flags().StringVar(&name, "name", "", "kid name")
	c.Flags().StringVar(&email, "email", "", "kid email")
	c.Flags().IntSliceVar(&groupIDs, "group-id", nil, "replacement group ids; repeat or comma-separate")
	return c
}

func newKidDeleteCmd() *cobra.Command {
	var force bool
	c := &cobra.Command{
		Use:   "delete <kid-id>",
		Short: "Delete a kid account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireForce(force, "kid"); err != nil {
				return err
			}
			return apiRequest(cmd, http.MethodDelete, "/api/v1/kids/"+args[0], nil, nil)
		},
	}
	addCommonDeleteFlags(c, &force)
	return c
}

func newBadgesCmd() *cobra.Command {
	c := &cobra.Command{Use: "badges", Short: "Manage badges"}
	c.AddCommand(&cobra.Command{Use: "list", Short: "List badges", RunE: methodGet("/api/v1/badges")})
	c.AddCommand(&cobra.Command{
		Use:   "get <badge-id>",
		Short: "Get one badge",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return apiRequest(cmd, http.MethodGet, "/api/v1/badges/"+args[0], nil, nil)
		},
	})
	c.AddCommand(newBadgeCreateCmd())
	c.AddCommand(newBadgeUpdateCmd())
	c.AddCommand(newBadgeDeleteCmd())
	c.AddCommand(newBadgeStateCmd("publish"))
	c.AddCommand(newBadgeStateCmd("unpublish"))
	return c
}

func newBadgeCreateCmd() *cobra.Command {
	var title, description, status string
	var points, categoryID int
	var groupIDs []int
	var challenges []string
	c := &cobra.Command{
		Use:   "create",
		Short: "Create a badge",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := requireAll(map[string]string{"--title": title}); err != nil {
				return err
			}
			if err := requireNonNegativeIntFlag(cmd, "points", points); err != nil {
				return err
			}
			attrs := map[string]any{"title": title, "points": points}
			addString(attrs, "description", description)
			addString(attrs, "status", status)
			if categoryID > 0 {
				attrs["badge_category_id"] = categoryID
			}
			if len(groupIDs) > 0 {
				attrs["group_ids"] = groupIDs
			}
			if len(challenges) > 0 {
				attrs["badge_challenges_attributes"] = challengeAttributeList(challenges)
			}
			return apiRequest(cmd, http.MethodPost, "/api/v1/badges", nested("badge", attrs), nil)
		},
	}
	c.Flags().StringVar(&title, "title", "", "badge title (required)")
	c.Flags().StringVar(&description, "description", "", "badge description")
	c.Flags().IntVar(&points, "points", 0, "points awarded (required)")
	c.Flags().StringVar(&status, "status", "", "badge status, usually draft or published")
	c.Flags().IntVar(&categoryID, "badge-category-id", 0, "badge category id")
	c.Flags().IntSliceVar(&groupIDs, "group-id", nil, "group id allowed to earn this badge; repeat or comma-separate")
	c.Flags().StringArrayVar(&challenges, "challenge", nil, "challenge title; repeat to create a multi-challenge badge")
	return c
}

func newBadgeUpdateCmd() *cobra.Command {
	var title, description, status string
	var points, categoryID int
	var groupIDs []int
	var challenges []string
	c := &cobra.Command{
		Use:   "update <badge-id>",
		Short: "Update a badge",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			attrs := map[string]any{}
			addStringChanged(cmd, attrs, "title", "title", title)
			addStringChanged(cmd, attrs, "description", "description", description)
			addStringChanged(cmd, attrs, "status", "status", status)
			addIntChanged(cmd, attrs, "points", "points", points)
			addIntChanged(cmd, attrs, "badge-category-id", "badge_category_id", categoryID)
			addIntSliceChanged(cmd, attrs, "group-id", "group_ids", groupIDs)
			if flagChanged(cmd, "challenge") {
				attrs["badge_challenges_attributes"] = challengeAttributeList(challenges)
			}
			if err := requireChanged(attrs); err != nil {
				return err
			}
			return apiRequest(cmd, http.MethodPatch, "/api/v1/badges/"+args[0], nested("badge", attrs), nil)
		},
	}
	c.Flags().StringVar(&title, "title", "", "badge title")
	c.Flags().StringVar(&description, "description", "", "badge description")
	c.Flags().IntVar(&points, "points", 0, "points awarded")
	c.Flags().StringVar(&status, "status", "", "badge status")
	c.Flags().IntVar(&categoryID, "badge-category-id", 0, "badge category id")
	c.Flags().IntSliceVar(&groupIDs, "group-id", nil, "replacement group ids; repeat or comma-separate")
	c.Flags().StringArrayVar(&challenges, "challenge", nil, "replacement challenge titles")
	return c
}

func newBadgeDeleteCmd() *cobra.Command {
	var force bool
	c := &cobra.Command{
		Use:   "delete <badge-id>",
		Short: "Delete a badge",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireForce(force, "badge"); err != nil {
				return err
			}
			return apiRequest(cmd, http.MethodDelete, "/api/v1/badges/"+args[0], nil, nil)
		},
	}
	addCommonDeleteFlags(c, &force)
	return c
}

func newBadgeStateCmd(action string) *cobra.Command {
	return &cobra.Command{
		Use:   action + " <badge-id>",
		Short: action + " a badge",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return apiRequest(cmd, http.MethodPost, "/api/v1/badges/"+args[0]+"/"+action, nil, nil)
		},
	}
}

func challengeAttributeList(values []string) []map[string]any {
	out := make([]map[string]any, 0, len(values))
	for i, value := range values {
		out = append(out, map[string]any{
			"title":       value,
			"description": value,
			"position":    i + 1,
		})
	}
	return out
}

func newSubmissionsCmd() *cobra.Command {
	c := &cobra.Command{Use: "submissions", Aliases: []string{"badge-submissions"}, Short: "Review badge submissions"}
	var listStatus string
	list := &cobra.Command{
		Use:   "list",
		Short: "List badge submissions",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return apiRequest(cmd, http.MethodGet, "/api/v1/badge_submissions", nil, statusQuery(listStatus))
		},
	}
	list.Flags().StringVar(&listStatus, "status", "", "filter by status, e.g. pending_review")
	c.AddCommand(list)
	c.AddCommand(&cobra.Command{
		Use:   "get <submission-id>",
		Short: "Get one badge submission",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return apiRequest(cmd, http.MethodGet, "/api/v1/badge_submissions/"+args[0], nil, nil)
		},
	})
	var approveFeedback string
	approve := &cobra.Command{
		Use:   "approve <submission-id>",
		Short: "Approve a badge submission",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var body any
			if approveFeedback != "" {
				body = map[string]any{"feedback": approveFeedback}
			}
			return apiRequest(cmd, http.MethodPost, "/api/v1/badge_submissions/"+args[0]+"/approve", body, nil)
		},
	}
	approve.Flags().StringVar(&approveFeedback, "feedback", "", "optional parent feedback")
	c.AddCommand(approve)

	var reason, denyFeedback string
	deny := &cobra.Command{
		Use:   "deny <submission-id>",
		Short: "Deny a badge submission",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var body any
			switch {
			case denyFeedback != "":
				body = map[string]any{"feedback": denyFeedback}
			case reason != "":
				body = map[string]any{"reason": reason}
			}
			return apiRequest(cmd, http.MethodPost, "/api/v1/badge_submissions/"+args[0]+"/deny", body, nil)
		},
	}
	deny.Flags().StringVar(&reason, "reason", "", "denial reason sent to the API")
	deny.Flags().StringVar(&denyFeedback, "feedback", "", "parent feedback; preferred over --reason when both are set")
	c.AddCommand(deny)
	return c
}

func newPrizesCmd() *cobra.Command {
	c := &cobra.Command{Use: "prizes", Short: "Manage prizes"}
	c.AddCommand(&cobra.Command{Use: "list", Short: "List prizes", RunE: methodGet("/api/v1/prizes")})
	c.AddCommand(&cobra.Command{
		Use:   "get <prize-id>",
		Short: "Get one prize",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return apiRequest(cmd, http.MethodGet, "/api/v1/prizes/"+args[0], nil, nil)
		},
	})
	c.AddCommand(newPrizeCreateCmd())
	c.AddCommand(newPrizeUpdateCmd())
	c.AddCommand(newPrizeDeleteCmd())
	return c
}

func newPrizeCreateCmd() *cobra.Command {
	var name, description string
	var pointCost int
	var active bool
	c := &cobra.Command{
		Use:   "create",
		Short: "Create a prize",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := requireAll(map[string]string{"--name": name}); err != nil {
				return err
			}
			if err := requireNonNegativeIntFlag(cmd, "point-cost", pointCost); err != nil {
				return err
			}
			attrs := map[string]any{"name": name, "point_cost": pointCost, "active": active}
			addString(attrs, "description", description)
			return apiRequest(cmd, http.MethodPost, "/api/v1/prizes", nested("prize", attrs), nil)
		},
	}
	c.Flags().StringVar(&name, "name", "", "prize name (required)")
	c.Flags().StringVar(&description, "description", "", "prize description")
	c.Flags().IntVar(&pointCost, "point-cost", 0, "point cost (required)")
	c.Flags().BoolVar(&active, "active", true, "whether the prize is active")
	return c
}

func newPrizeUpdateCmd() *cobra.Command {
	var name, description string
	var pointCost int
	var active bool
	c := &cobra.Command{
		Use:   "update <prize-id>",
		Short: "Update a prize",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			attrs := map[string]any{}
			addStringChanged(cmd, attrs, "name", "name", name)
			addStringChanged(cmd, attrs, "description", "description", description)
			addIntChanged(cmd, attrs, "point-cost", "point_cost", pointCost)
			addBoolChanged(cmd, attrs, "active", "active", active)
			if err := requireChanged(attrs); err != nil {
				return err
			}
			return apiRequest(cmd, http.MethodPatch, "/api/v1/prizes/"+args[0], nested("prize", attrs), nil)
		},
	}
	c.Flags().StringVar(&name, "name", "", "prize name")
	c.Flags().StringVar(&description, "description", "", "prize description")
	c.Flags().IntVar(&pointCost, "point-cost", 0, "point cost")
	c.Flags().BoolVar(&active, "active", false, "whether the prize is active")
	return c
}

func newPrizeDeleteCmd() *cobra.Command {
	var force bool
	c := &cobra.Command{
		Use:   "delete <prize-id>",
		Short: "Delete a prize",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireForce(force, "prize"); err != nil {
				return err
			}
			return apiRequest(cmd, http.MethodDelete, "/api/v1/prizes/"+args[0], nil, nil)
		},
	}
	addCommonDeleteFlags(c, &force)
	return c
}

func newRedemptionsCmd() *cobra.Command {
	c := &cobra.Command{Use: "redemptions", Short: "Review prize redemptions"}
	var listStatus string
	list := &cobra.Command{
		Use:   "list",
		Short: "List redemptions",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return apiRequest(cmd, http.MethodGet, "/api/v1/redemptions", nil, statusQuery(listStatus))
		},
	}
	list.Flags().StringVar(&listStatus, "status", "", "filter by status, e.g. pending")
	c.AddCommand(list)
	c.AddCommand(&cobra.Command{
		Use:   "get <redemption-id>",
		Short: "Get one redemption",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return apiRequest(cmd, http.MethodGet, "/api/v1/redemptions/"+args[0], nil, nil)
		},
	})
	var approveFeedback string
	approve := &cobra.Command{
		Use:   "approve <redemption-id>",
		Short: "Approve a redemption",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var body any
			if approveFeedback != "" {
				body = map[string]any{"feedback": approveFeedback}
			}
			return apiRequest(cmd, http.MethodPost, "/api/v1/redemptions/"+args[0]+"/approve", body, nil)
		},
	}
	approve.Flags().StringVar(&approveFeedback, "feedback", "", "optional parent feedback")
	c.AddCommand(approve)

	var denyFeedback, denyReason string
	deny := &cobra.Command{
		Use:   "deny <redemption-id>",
		Short: "Deny a redemption",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var body any
			switch {
			case denyFeedback != "":
				body = map[string]any{"feedback": denyFeedback}
			case denyReason != "":
				body = map[string]any{"reason": denyReason}
			}
			return apiRequest(cmd, http.MethodPost, "/api/v1/redemptions/"+args[0]+"/deny", body, nil)
		},
	}
	deny.Flags().StringVar(&denyFeedback, "feedback", "", "parent feedback; preferred over --reason when both are set")
	deny.Flags().StringVar(&denyReason, "reason", "", "denial reason sent to the API")
	c.AddCommand(deny)
	return c
}

func newCategoriesCmd() *cobra.Command {
	c := &cobra.Command{Use: "categories", Aliases: []string{"badge-categories"}, Short: "Manage badge categories"}
	c.AddCommand(&cobra.Command{Use: "list", Short: "List badge categories", RunE: methodGet("/api/v1/badge_categories")})
	c.AddCommand(&cobra.Command{
		Use:   "get <category-id>",
		Short: "Get one badge category",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return apiRequest(cmd, http.MethodGet, "/api/v1/badge_categories/"+args[0], nil, nil)
		},
	})
	c.AddCommand(newCategoryCreateCmd())
	c.AddCommand(newCategoryUpdateCmd())
	c.AddCommand(newCategoryDeleteCmd())
	c.AddCommand(newCategoryMoveCmd("move-up", "move_up"))
	c.AddCommand(newCategoryMoveCmd("move-down", "move_down"))
	return c
}

func newCategoryCreateCmd() *cobra.Command {
	var name, description string
	c := &cobra.Command{
		Use:   "create",
		Short: "Create a badge category",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := requireAll(map[string]string{"--name": name}); err != nil {
				return err
			}
			attrs := map[string]any{"name": name}
			addString(attrs, "description", description)
			return apiRequest(cmd, http.MethodPost, "/api/v1/badge_categories", nested("badge_category", attrs), nil)
		},
	}
	c.Flags().StringVar(&name, "name", "", "category name (required)")
	c.Flags().StringVar(&description, "description", "", "category description")
	return c
}

func newCategoryUpdateCmd() *cobra.Command {
	var name, description string
	c := &cobra.Command{
		Use:   "update <category-id>",
		Short: "Update a badge category",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			attrs := map[string]any{}
			addStringChanged(cmd, attrs, "name", "name", name)
			addStringChanged(cmd, attrs, "description", "description", description)
			if err := requireChanged(attrs); err != nil {
				return err
			}
			return apiRequest(cmd, http.MethodPatch, "/api/v1/badge_categories/"+args[0], nested("badge_category", attrs), nil)
		},
	}
	c.Flags().StringVar(&name, "name", "", "category name")
	c.Flags().StringVar(&description, "description", "", "category description")
	return c
}

func newCategoryDeleteCmd() *cobra.Command {
	var force bool
	c := &cobra.Command{
		Use:   "delete <category-id>",
		Short: "Delete a badge category",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireForce(force, "category"); err != nil {
				return err
			}
			return apiRequest(cmd, http.MethodDelete, "/api/v1/badge_categories/"+args[0], nil, nil)
		},
	}
	addCommonDeleteFlags(c, &force)
	return c
}

func newCategoryMoveCmd(use, action string) *cobra.Command {
	return &cobra.Command{
		Use:   use + " <category-id>",
		Short: use + " a badge category",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return apiRequest(cmd, http.MethodPost, "/api/v1/badge_categories/"+args[0]+"/"+action, nil, nil)
		},
	}
}

func newGroupsCmd() *cobra.Command {
	c := &cobra.Command{Use: "groups", Short: "Manage groups"}
	c.AddCommand(&cobra.Command{Use: "list", Short: "List groups", RunE: methodGet("/api/v1/groups")})
	c.AddCommand(&cobra.Command{
		Use:   "get <group-id>",
		Short: "Get one group",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return apiRequest(cmd, http.MethodGet, "/api/v1/groups/"+args[0], nil, nil)
		},
	})
	c.AddCommand(newGroupCreateCmd())
	c.AddCommand(newGroupUpdateCmd())
	c.AddCommand(newGroupDeleteCmd())
	c.AddCommand(newGroupMemberCmd("add-member", "add_member"))
	c.AddCommand(newGroupMemberCmd("remove-member", "remove_member"))
	return c
}

func newGroupCreateCmd() *cobra.Command {
	var name, description string
	c := &cobra.Command{
		Use:   "create",
		Short: "Create a group",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := requireAll(map[string]string{"--name": name}); err != nil {
				return err
			}
			attrs := map[string]any{"name": name}
			addString(attrs, "description", description)
			return apiRequest(cmd, http.MethodPost, "/api/v1/groups", nested("group", attrs), nil)
		},
	}
	c.Flags().StringVar(&name, "name", "", "group name (required)")
	c.Flags().StringVar(&description, "description", "", "group description")
	return c
}

func newGroupUpdateCmd() *cobra.Command {
	var name, description string
	c := &cobra.Command{
		Use:   "update <group-id>",
		Short: "Update a group",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			attrs := map[string]any{}
			addStringChanged(cmd, attrs, "name", "name", name)
			addStringChanged(cmd, attrs, "description", "description", description)
			if err := requireChanged(attrs); err != nil {
				return err
			}
			return apiRequest(cmd, http.MethodPatch, "/api/v1/groups/"+args[0], nested("group", attrs), nil)
		},
	}
	c.Flags().StringVar(&name, "name", "", "group name")
	c.Flags().StringVar(&description, "description", "", "group description")
	return c
}

func newGroupDeleteCmd() *cobra.Command {
	var force bool
	c := &cobra.Command{
		Use:   "delete <group-id>",
		Short: "Delete a group",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireForce(force, "group"); err != nil {
				return err
			}
			return apiRequest(cmd, http.MethodDelete, "/api/v1/groups/"+args[0], nil, nil)
		},
	}
	addCommonDeleteFlags(c, &force)
	return c
}

func newGroupMemberCmd(use, action string) *cobra.Command {
	var kidID int
	c := &cobra.Command{
		Use:   use + " <group-id> --kid-id <kid-id>",
		Short: use + " for a group",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requirePositiveInt("--kid-id", kidID); err != nil {
				return err
			}
			return apiRequest(cmd, http.MethodPost, "/api/v1/groups/"+args[0]+"/"+action, map[string]any{"user_id": kidID}, nil)
		},
	}
	c.Flags().IntVar(&kidID, "kid-id", 0, "kid id (required)")
	return c
}

func newChallengesCmd() *cobra.Command {
	c := &cobra.Command{Use: "challenges", Short: "Manage badge challenges"}
	var badgeID int
	list := &cobra.Command{
		Use:   "list",
		Short: "List badge challenges",
		RunE: func(cmd *cobra.Command, _ []string) error {
			q := url.Values{}
			if badgeID > 0 {
				q.Set("badge_id", strconv.Itoa(badgeID))
			}
			return apiRequest(cmd, http.MethodGet, "/api/v1/challenges", nil, q)
		},
	}
	list.Flags().IntVar(&badgeID, "badge-id", 0, "filter by badge id")
	c.AddCommand(list)
	c.AddCommand(&cobra.Command{
		Use:   "get <challenge-id>",
		Short: "Get one challenge",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return apiRequest(cmd, http.MethodGet, "/api/v1/challenges/"+args[0], nil, nil)
		},
	})
	c.AddCommand(newChallengeCreateCmd())
	c.AddCommand(newChallengeUpdateCmd())
	c.AddCommand(newChallengeDeleteCmd())
	return c
}

func newChallengeCreateCmd() *cobra.Command {
	var badgeID, position int
	var title, description string
	c := &cobra.Command{
		Use:   "create",
		Short: "Create a challenge on a badge",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := requirePositiveInt("--badge-id", badgeID); err != nil {
				return err
			}
			if err := requireAll(map[string]string{"--title": title}); err != nil {
				return err
			}
			attrs := map[string]any{"title": title}
			addString(attrs, "description", description)
			if position > 0 {
				attrs["position"] = position
			}
			body := map[string]any{"badge_id": badgeID, "challenge": attrs}
			return apiRequest(cmd, http.MethodPost, "/api/v1/challenges", body, nil)
		},
	}
	c.Flags().IntVar(&badgeID, "badge-id", 0, "badge id (required)")
	c.Flags().StringVar(&title, "title", "", "challenge title (required)")
	c.Flags().StringVar(&description, "description", "", "challenge description")
	c.Flags().IntVar(&position, "position", 0, "challenge position")
	return c
}

func newChallengeUpdateCmd() *cobra.Command {
	var title, description string
	var position int
	c := &cobra.Command{
		Use:   "update <challenge-id>",
		Short: "Update a challenge",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			attrs := map[string]any{}
			addStringChanged(cmd, attrs, "title", "title", title)
			addStringChanged(cmd, attrs, "description", "description", description)
			addIntChanged(cmd, attrs, "position", "position", position)
			if err := requireChanged(attrs); err != nil {
				return err
			}
			return apiRequest(cmd, http.MethodPatch, "/api/v1/challenges/"+args[0], nested("challenge", attrs), nil)
		},
	}
	c.Flags().StringVar(&title, "title", "", "challenge title")
	c.Flags().StringVar(&description, "description", "", "challenge description")
	c.Flags().IntVar(&position, "position", 0, "challenge position")
	return c
}

func newChallengeDeleteCmd() *cobra.Command {
	var force bool
	c := &cobra.Command{
		Use:   "delete <challenge-id>",
		Short: "Delete a challenge",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireForce(force, "challenge"); err != nil {
				return err
			}
			return apiRequest(cmd, http.MethodDelete, "/api/v1/challenges/"+args[0], nil, nil)
		},
	}
	addCommonDeleteFlags(c, &force)
	return c
}
