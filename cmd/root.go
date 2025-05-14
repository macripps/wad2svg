package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/macripps/wad2svg/svg"
	"github.com/macripps/wad2svg/wad"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "wad2svg wad_file map_name",
	Short: "wad2svg generates SVG files from Doom and Doom2 WAD files",
	Args:  cobra.ExactArgs(2),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		var comps []string
		if len(args) == 0 {
			comps = cobra.AppendActiveHelp(comps, "requires a path to a WAD file")
		}
		if len(args) < 2 {
			comps = cobra.AppendActiveHelp(comps, "requires a map name")
		}
		return comps, cobra.ShellCompDirectiveDefault
	},
	Run: func(cmd *cobra.Command, args []string) {
		var fileName = args[0]
		opts.WadName = filepath.Base(fileName)
		opts.MapName = args[1]
		f, err := os.Open(fileName)
		if err != nil {
			panic(err)
		}
		m := &wad.Map{}
		m.ReadFrom(f, opts.MapName)
		svg.Render(os.Stdout, m, opts)
	},
}

var opts *svg.RenderOpts = &svg.RenderOpts{}

func init() {
	rootCmd.PersistentFlags().IntVar(&opts.ImageWidth, "image_width", 1280, "Width of generated SVG image")
	rootCmd.PersistentFlags().IntVar(&opts.ImageHeight, "image_height", 1024, "Height of generated SVG image")
	rootCmd.PersistentFlags().BoolVar(&opts.ListMaps, "list_maps", false, "If true, print a list of maps to stderr")
	rootCmd.PersistentFlags().BoolVar(&opts.RenderAmmo, "show_ammo", true, "Whether or not to show ammunition")
	rootCmd.PersistentFlags().BoolVar(&opts.RenderArtifacts, "show_artifacts", true, "Whether or not to show items")
	rootCmd.PersistentFlags().BoolVar(&opts.RenderKeys, "show_keys", true, "Whether or not to show keys")
	rootCmd.PersistentFlags().BoolVar(&opts.RenderMonsters, "show_monsters", true, "Whether or not to show monsters")
	rootCmd.PersistentFlags().BoolVar(&opts.RenderPowerups, "show_powerups", true, "Whether or not to show powerups")
	rootCmd.PersistentFlags().BoolVar(&opts.RenderWeapons, "show_weapons", true, "Whether or not to show weapons")
	rootCmd.PersistentFlags().BoolVar(&opts.RenderMultiplayer, "show_mp", false, "Whether or not to show multiplayer items")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
