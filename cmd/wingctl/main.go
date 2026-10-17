package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/HP-jh/tars-update"

	"github.com/HP-jh/TarsWingGuard/internal/version"
	"github.com/HP-jh/TarsWingGuard/pkg/catalog"
	"github.com/HP-jh/TarsWingGuard/pkg/config"
	"github.com/HP-jh/TarsWingGuard/pkg/tier"
)

func main() {
	// Subcommand routing
	if len(os.Args) >= 2 && os.Args[1] == "update" {
		handleUpdate(os.Args[2:])
		return
	}

	var (
		catalogPath = flag.String("catalog", "wing.yaml", "Path to wing catalog YAML")
		tsghConfig  = flag.String("tsg-config", "", "Path to TSG config.json (loads wing section)")
		list        = flag.Bool("list", false, "List all modules")
		resolve     = flag.String("resolve", "", "Resolve modules for tier: basic|pro|ultimate")
		showTier    = flag.Bool("tiers", false, "Show tier summary")
		dryRun      = flag.Bool("dry-run", false, "Dry-run install (verify catalog + tier resolve)")
		jsonOut     = flag.Bool("json", false, "Output as JSON")
		ver         = flag.Bool("version", false, "Show version")
	)
	flag.Parse()

	if *ver {
		fmt.Printf("wingctl %s (commit %s)\n", version.Version, version.Commit)
		os.Exit(0)
	}

	cat, err := catalog.Parse(*catalogPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if *dryRun {
		if err := runDryRun(cat, *tsghConfig); err != nil {
			fmt.Fprintf(os.Stderr, "Dry-run failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Dry-run OK: catalog valid, all tiers resolvable")
		os.Exit(0)
	}

	if *list {
		printModules(cat, *jsonOut)
		os.Exit(0)
	}

	if *resolve != "" {
		t := catalog.Tier(*resolve)
		if !t.Valid() {
			fmt.Fprintf(os.Stderr, "Error: invalid tier %q\n", *resolve)
			os.Exit(1)
		}
		mods := cat.ResolveTier(t)
		printResolved(mods, *jsonOut)
		os.Exit(0)
	}

	if *showTier {
		printTierSummary(cat, *jsonOut)
		os.Exit(0)
	}

	if *tsghConfig != "" {
		wingCfg, err := config.LoadTSGConfig(*tsghConfig)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading TSG config: %v\n", err)
			os.Exit(1)
		}
		if err := wingCfg.Validate(); err != nil {
			fmt.Fprintf(os.Stderr, "Validation error: %v\n", err)
			os.Exit(1)
		}
		catPath := config.ResolveCatalogPath(wingCfg, *tsghConfig)
		if catPath != *catalogPath {
			cat, err = catalog.Parse(catPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error loading catalog from config: %v\n", err)
				os.Exit(1)
			}
		}
		resolver := tier.NewResolver(cat)
		mods, err := resolver.Resolve(wingCfg.ToTierConfig())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Resolve error: %v\n", err)
			os.Exit(1)
		}
		printResolved(mods, *jsonOut)
		os.Exit(0)
	}

	flag.Usage()
	fmt.Fprintf(os.Stderr, "\nSubcommands:\n  update check    Check for newer version\n  update apply    Download and apply update\n  update rollback Rollback to previous version\n")
	os.Exit(1)
}

func handleUpdate(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: wingctl update <check|apply|rollback>")
		os.Exit(1)
	}
	cmd := args[0]

	u := update.NewUpdater("twg", version.Version, "HP-jh", "TarsWingGuard")

	switch cmd {
	case "check":
		result := u.Check()
		if result.Error != nil {
			// Try fallback
			stable, err := u.FallbackToStable()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Check failed and no stable version cached: %v\n", result.Error)
				os.Exit(1)
			}
			fmt.Printf("GitHub unreachable. Cached stable version: %s\n", stable)
			os.Exit(0)
		}
		if result.HasUpdate {
			fmt.Printf("Update available: %s -> %s\n", result.Current.String(), result.Latest.String())
			fmt.Printf("Asset: %s\n", result.AssetName)
			fmt.Printf("Release: %s\n", result.Release.HTMLURL)
		} else {
			fmt.Printf("Already up to date: %s\n", result.Current.String())
		}
	case "apply":
		result := u.Check()
		if result.Error != nil {
			fmt.Fprintf(os.Stderr, "Check failed: %v\n", result.Error)
			os.Exit(1)
		}
		if !result.HasUpdate {
			fmt.Println("No update available.")
			os.Exit(0)
		}
		fmt.Printf("Downloading %s -> %s ...\n", result.Current.String(), result.Latest.String())
		if err := u.Apply(result); err != nil {
			fmt.Fprintf(os.Stderr, "Apply failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Update applied successfully. Please restart wingctl.")
	case "rollback":
		if err := u.Rollback(); err != nil {
			fmt.Fprintf(os.Stderr, "Rollback failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Rollback successful. Please restart wingctl.")
	default:
		fmt.Fprintf(os.Stderr, "Unknown update command: %q\n", cmd)
		fmt.Fprintln(os.Stderr, "Usage: wingctl update <check|apply|rollback>")
		os.Exit(1)
	}
}

func printModules(cat *catalog.Catalog, asJSON bool) {
	if asJSON {
		b, _ := json.MarshalIndent(cat.Modules, "", "  ")
		fmt.Println(string(b))
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tVERSION\tTIER\tTRANSPORT\tSTATUS")
	for _, m := range cat.Modules {
		status := "enabled"
		if m.Disabled {
			status = "disabled"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", m.ID, m.Name, m.Version, m.Tier, m.Transport, status)
	}
	w.Flush()
}

func printResolved(mods []catalog.Module, asJSON bool) {
	if asJSON {
		b, _ := json.MarshalIndent(mods, "", "  ")
		fmt.Println(string(b))
		return
	}
	fmt.Printf("Resolved %d module(s):\n", len(mods))
	for _, m := range mods {
		fmt.Printf("  - %s (%s) [%s]\n", m.Name, m.ID, m.Tier)
	}
}

func printTierSummary(cat *catalog.Catalog, asJSON bool) {
	summary := tier.TierSummary(cat)
	if asJSON {
		b, _ := json.MarshalIndent(summary, "", "  ")
		fmt.Println(string(b))
		return
	}
	for _, t := range []catalog.Tier{catalog.TierBasic, catalog.TierPro, catalog.TierUltimate} {
		info := summary[t]
		fmt.Printf("\n[%s] %d module(s)\n", strings.ToUpper(string(t)), info.ModuleCount)
		for _, n := range info.Modules {
			fmt.Printf("  - %s\n", n)
		}
	}
}

func runDryRun(cat *catalog.Catalog, tsgConfigPath string) error {
	// Verify all tiers can resolve
	for _, t := range []catalog.Tier{catalog.TierBasic, catalog.TierPro, catalog.TierUltimate} {
		mods := cat.ResolveTier(t)
		if t == catalog.TierBasic && len(mods) == 0 {
			return fmt.Errorf("basic tier has zero modules")
		}
	}
	// If tsg config given, validate it too
	if tsgConfigPath != "" {
		wingCfg, err := config.LoadTSGConfig(tsgConfigPath)
		if err != nil {
			return err
		}
		if err := wingCfg.Validate(); err != nil {
			return err
		}
		resolver := tier.NewResolver(cat)
		_, err = resolver.Resolve(wingCfg.ToTierConfig())
		if err != nil {
			return err
		}
	}
	return nil
}
