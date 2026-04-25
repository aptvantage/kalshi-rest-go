package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aptvantage/kalshi-rest-go/kalshi"
	"github.com/spf13/cobra"
)

func newSeriesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "series",
		Short: "Browse series (contract templates)",
		Long: `Series are recurring contract templates (e.g. KXHIGHNY = "NYC High Temperature").
Each series spawns Events (one per date/period), which contain individual Markets.

Hierarchy:  Series → Event(s) → Market(s)
Example:    KXHIGHNY → KXHIGHNY-26APR25 → KXHIGHNY-26APR25-T51`,
	}

	var (
		listCategory      string
		listTags          string
		listIncludeVolume bool
	)
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all series",
		Long: `List series (contract families). Use --category to narrow results.

Examples:
  kalshi-cli series list
  kalshi-cli series list --category weather
  kalshi-cli series list --category financials --include-volume
  kalshi-cli series list -o wide`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newUnauthClient()
			if err != nil {
				return err
			}
			params := kalshi.GetSeriesListParams{}
			if listCategory != "" {
				params.Category = &listCategory
			}
			if listTags != "" {
				params.Tags = &listTags
			}
			if listIncludeVolume {
				params.IncludeVolume = &listIncludeVolume
			}
			resp, err := client.GetSeriesListWithResponse(context.Background(), &params)
			if err != nil {
				return fmt.Errorf("request failed: %w", err)
			}
			if resp.StatusCode() != 200 {
				fmt.Fprintf(os.Stderr, "HTTP %d: %s\n", resp.StatusCode(), string(resp.Body))
				os.Exit(1)
			}
			switch flagOutput {
			case "json", "yaml":
				return render(resp.JSON200, func(wide bool) ([]string, [][]string) {
					return seriesTable(resp.JSON200.Series, wide)
				})
			default:
				headers, rows := seriesTable(resp.JSON200.Series, isWide())
				if err := printTable(headers, rows); err != nil {
					return err
				}
				fmt.Fprintf(os.Stderr, "\n# %d series\n", len(resp.JSON200.Series))
				return nil
			}
		},
	}
	listCmd.Flags().StringVar(&listCategory, "category", "", "Filter by category (e.g. weather, financials, crypto, politics)")
	listCmd.Flags().StringVar(&listTags, "tags", "", "Filter by tag (comma-separated)")
	listCmd.Flags().BoolVar(&listIncludeVolume, "include-volume", false, "Include total all-time volume in output")

	getCmd := &cobra.Command{
		Use:   "get <ticker>",
		Short: "Get details for a single series",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newUnauthClient()
			if err != nil {
				return err
			}
			incVol := true
			resp, err := client.GetSeriesWithResponse(context.Background(), args[0], &kalshi.GetSeriesParams{
				IncludeVolume: &incVol,
			})
			if err != nil {
				return fmt.Errorf("request failed: %w", err)
			}
			if resp.StatusCode() != 200 {
				fmt.Fprintf(os.Stderr, "HTTP %d: %s\n", resp.StatusCode(), string(resp.Body))
				os.Exit(1)
			}
			return render(resp.JSON200, func(wide bool) ([]string, [][]string) {
				return seriesTable([]kalshi.Series{resp.JSON200.Series}, wide)
			})
		},
	}

	categoriesCmd := &cobra.Command{
		Use:   "categories",
		Short: "List all series categories and their tags",
		Long: `List all categories that series can belong to, with their associated tags.
Use a category name with 'series list --category <name>' to filter series.

Example:
  kalshi-cli series categories
  kalshi-cli series categories -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newUnauthClient()
			if err != nil {
				return err
			}
			resp, err := client.GetTagsForSeriesCategoriesWithResponse(context.Background())
			if err != nil {
				return fmt.Errorf("request failed: %w", err)
			}
			if resp.StatusCode() != 200 {
				fmt.Fprintf(os.Stderr, "HTTP %d: %s\n", resp.StatusCode(), string(resp.Body))
				os.Exit(1)
			}
			return render(resp.JSON200, func(wide bool) ([]string, [][]string) {
				return categoriesTable(resp.JSON200.TagsByCategories, wide)
			})
		},
	}

	cmd.AddCommand(listCmd, getCmd, categoriesCmd)
	return cmd
}

func seriesTable(series []kalshi.Series, wide bool) ([]string, [][]string) {
	// Default: identity + trading cost + cadence — enough to evaluate LP fit at a glance.
	headers := []string{"TICKER", "TITLE", "CATEGORY", "FEE_TYPE", "FEE_MULT", "FREQUENCY"}
	if wide {
		// Wide adds tags and volume (if available).
		headers = append(headers, "VOLUME", "TAGS")
	}
	rows := make([][]string, 0, len(series))
	for _, s := range series {
		row := []string{
			s.Ticker,
			truncate(s.Title, 40),
			s.Category,
			string(s.FeeType),
			fmt.Sprintf("%.2f", s.FeeMultiplier),
			s.Frequency,
		}
		if wide {
			vol := "-"
			if s.VolumeFp != nil {
				vol = string(*s.VolumeFp)
			}
			row = append(row, vol, strings.Join(s.Tags, ","))
		}
		rows = append(rows, row)
	}
	return headers, rows
}

func categoriesTable(tagsByCategory map[string][]string, wide bool) ([]string, [][]string) {
	// Sort categories for stable output.
	categories := make([]string, 0, len(tagsByCategory))
	for c := range tagsByCategory {
		categories = append(categories, c)
	}
	// Simple alphabetical sort inline.
	for i := 0; i < len(categories); i++ {
		for j := i + 1; j < len(categories); j++ {
			if categories[i] > categories[j] {
				categories[i], categories[j] = categories[j], categories[i]
			}
		}
	}

	headers := []string{"CATEGORY", "TAGS"}
	rows := make([][]string, 0, len(categories))
	for _, cat := range categories {
		tags := tagsByCategory[cat]
		tagStr := strings.Join(tags, ", ")
		if !wide {
			tagStr = truncate(tagStr, 70)
		}
		rows = append(rows, []string{cat, tagStr})
	}
	return headers, rows
}
