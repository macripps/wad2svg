package svg

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/macripps/wad2svg/wad"
)

type RenderOpts struct {
	WadName           string
	MapName           string
	ImageWidth        int
	ImageHeight       int
	ListMaps          bool
	RenderArtifacts   bool
	RenderAmmo        bool
	RenderKeys        bool
	RenderMonsters    bool
	RenderPowerups    bool
	RenderWeapons     bool
	RenderMultiplayer bool
}

func Render(w io.Writer, m *wad.Map, opts *RenderOpts) {
	minX, minY, maxX, maxY := int16(32767), int16(32767), int16(-32768), int16(-32768)
	for i := 0; i < len(m.Vertexes); i++ {
		x := m.Vertexes[i].X
		y := m.Vertexes[i].Y
		if x < minX {
			minX = x
		}
		if y < minY {
			minY = y
		}
		if x > maxX {
			maxX = x
		}
		if y > maxY {
			maxY = y
		}
	}
	width := int32(maxX) - int32(minX)
	height := int32(maxY) - int32(minY)
	fmt.Fprintf(os.Stderr, "MinX: %d MaxX: %d Width: %d\nMinY: %d MaxY: %d Height: %d\n", minX, maxX,width, minY, maxY, height)
	fmt.Fprintln(w, "<?xml version=\"1.0\" standalone=\"no\"?>")
	fmt.Fprintf(w, "<svg width=\"%d\" height=\"%d\" viewBox=\"%d %d %d %d\" xmlns=\"http://www.w3.org/2000/svg\">\n", opts.ImageWidth, opts.ImageHeight, minX, minY, width, height)
	fmt.Fprintf(w, "  <title>%s - %s</title>\n", opts.WadName, opts.MapName)
	fmt.Fprintln(w, "  <g fill-rule=\"evenodd\">")

	sectors := m.Sectors
	for i, sector := range sectors {
// 		fmt.Fprintf(os.Stderr, "Rendering sector #%d/%d\n", i+1, len(sectors))
		renderSector(w, m, sector, i)
	}
	things := m.Things
	for i, thing := range things {
// 		fmt.Fprintf(os.Stderr, "Rendering thing #%d/%d\n", i+1, len(things))
		renderThing(w, thing, i, opts)
	}
	fmt.Fprintln(w, "  </g>")
	fmt.Fprintln(w, "</svg>")
}

func renderSector(w io.Writer, m *wad.Map, s wad.Sector, i int) {
	fmt.Fprintf(w, "    <g %s>\n", s.ToSvgAttributeString())
	fmt.Fprintf(w, "      <title>Sector %d</title>\n", i)
	fmt.Fprintf(w, "      <desc>Sector Type: %d</desc>\n", s.SectorType)
	sectorLineDefs := make([]wad.LineDef, 0)
	for sd := 0; sd < len(m.SideDefs); sd++ {
		sidedef := m.SideDefs[sd]
		if int(sidedef.SectorNumber) == i {
			for ld := 0; ld < len(m.LineDefs); ld++ {
				linedef := m.LineDefs[ld]
				if int(linedef.LeftSideDef) == sd || int(linedef.RightSideDef) == sd {
					sectorLineDefs = append(sectorLineDefs, linedef)
				}
			}
		}
	}
	// Render all sector linedefs with appropriate fill
	renderAllLineDefs(w, m, sectorLineDefs)
	// Now render special linedefs again, with colours
	renderSpecialLineDefs(w, m, sectorLineDefs)
	fmt.Fprintf(w, "    </g>\n")
}

func renderAllLineDefs(w io.Writer, m *wad.Map, lds []wad.LineDef) {
	sectorLineDefs := make([]wad.LineDef, len(lds))
	copy(sectorLineDefs, lds)
	var linedefs []wad.LineDef
	path := strings.Builder{}
	for len(sectorLineDefs) > 0 {
		linedefs, sectorLineDefs = selectLineDef(sectorLineDefs, func(ld1, ld2 wad.LineDef) bool {
			return true
		})
		linedef := linedefs[0]
		start := m.Vertexes[linedef.Start]
		end := m.Vertexes[linedef.End]
		path.WriteString(fmt.Sprintf("M %d %d L %d %d ", start.X, start.Y, end.X, end.Y))
		for i := 1; i < len(linedefs); i++ {
			v := m.Vertexes[linedefs[i].End]
			path.WriteString(fmt.Sprintf("%d %d ", v.X, v.Y))
		}
	}
	fmt.Fprintf(w, "    <path d=\"%s\"/>\n", path.String())
}

func renderSpecialLineDefs(w io.Writer, m *wad.Map, lds []wad.LineDef) {
	sectorLineDefs := make([]wad.LineDef, 0, len(lds))
	for _, s := range lds {
		if s.SpecialType != 0 {
			sectorLineDefs = append(sectorLineDefs, s)
		}
	}
	for _, linedef := range sectorLineDefs {
		lineType := linedef.SpecialType
		if lineType != 0 {
			stroke := "orange"
			strokeWidth := 3
			if linedef.IsDoor() {
				stroke = "green"
			} else if linedef.IsTeleporter() {
				stroke = "red"
			} else if linedef.IsLift() {
				stroke = "blue"
			} else if linedef.IsExit() {
				stroke = "purple"
			} else if linedef.IsSecret() {
				stroke = "aqua"
			}
			start := m.Vertexes[linedef.Start]
			end := m.Vertexes[linedef.End]

			fmt.Fprintf(w, "    <!-- Type %d -->\n", linedef.SpecialType)
			fmt.Fprintf(w, "    <path d=\"M %d %d L %d %d\" stroke=\"%s\" stroke-width=\"%d\" />", start.X, start.Y, end.X, end.Y, stroke, strokeWidth)
		}
	}
}

func selectLineDef(lineDefs []wad.LineDef, shouldInclude func(wad.LineDef, wad.LineDef) bool) ([]wad.LineDef, []wad.LineDef) {
	if len(lineDefs) == 1 {
		return lineDefs, []wad.LineDef{}
	}
	nextLineDef := lineDefs[0]
	lineDefGroup := []wad.LineDef{nextLineDef}
	lineDefs = lineDefs[1:]
	for i := 0; i < len(lineDefs); i++ {
		head := lineDefGroup[len(lineDefGroup)-1]
		tail := lineDefGroup[0]
		l := lineDefs[i]
		if l.Start == head.End &&
			shouldInclude(l, head) {
			lineDefGroup = append(lineDefGroup, l)
			lineDefs[i] = lineDefs[len(lineDefs)-1]
			lineDefs = lineDefs[:len(lineDefs)-1]
			i = -1
		} else if l.End == tail.Start &&
			shouldInclude(l, tail) {
			lineDefGroup = append([]wad.LineDef{l}, lineDefGroup...)
			lineDefs[i] = lineDefs[len(lineDefs)-1]
			lineDefs = lineDefs[:len(lineDefs)-1]
			i = -1
		} else if l.End == head.End && shouldInclude(l, head) {
			lineDefGroup = append(lineDefGroup, l.Flip())
			lineDefs[i] = lineDefs[len(lineDefs)-1]
			lineDefs = lineDefs[:len(lineDefs)-1]
			i = -1
		} else if l.Start == tail.Start && shouldInclude(l, tail) {
			lineDefGroup = append([]wad.LineDef{l.Flip()}, lineDefGroup...)
			lineDefs[i] = lineDefs[len(lineDefs)-1]
			lineDefs = lineDefs[:len(lineDefs)-1]
			i = -1
		}
	}
	return lineDefGroup, lineDefs
}

func renderThing(w io.Writer, thing wad.Thing, i int, opts *RenderOpts) {
	if (thing.Flags&16 == 16) && !opts.RenderMultiplayer {
		return
	}
	flags := ""
	if thing.Flags&1 == 1 {
		flags = flags + "12"
	}
	if thing.Flags&2 == 2 {
		flags = flags + "3"
	}
	if thing.Flags&4 == 4 {
		flags = flags + "45"
	}
	if thing.Flags&8 == 8 {
		flags = flags + "D"
	}
	if thing.Flags&16 == 16 {
		flags = flags + "M"
	}

	if opts.RenderAmmo && (thing.ThingType == 17 || thing.ThingType == 2007 || thing.ThingType == 2008 || thing.ThingType == 2010 || thing.ThingType == 2046 || thing.ThingType == 2047 || thing.ThingType == 2048 || thing.ThingType == 2049) {
		colour := "aqua"
		var ammoType string
		switch thing.ThingType {
		case 17:
			ammoType = "Energy cell pack"
		case 2007:
			ammoType = "Clip"
		case 2008:
			ammoType = "4 shotgun shells"
		case 2010:
			ammoType = "Rocket"
		case 2046:
			ammoType = "Box of rockets"
		case 2047:
			ammoType = "Energy cell"
		case 2048:
			ammoType = "Box of bullets"
		case 2049:
			ammoType = "Box of shotgun shells"
		}
		fmt.Fprintf(w, "    <rect x=\"%d\" y=\"%d\" stroke=\"black\" width=\"20\" height=\"20\" fill=\"%s\"><title>%s [%s]</title></rect>\n", thing.XPosition-10, thing.YPosition-10, colour, ammoType, flags)
	}

	if opts.RenderArtifacts && (thing.ThingType == 83 || thing.ThingType == 2013 || thing.ThingType == 2014 || thing.ThingType == 2015 || thing.ThingType == 2022 || thing.ThingType == 2023 || thing.ThingType == 2024 || thing.ThingType == 2026 || thing.ThingType == 2045) {
		colour := "green"
		var artifactType string
		switch thing.ThingType {
		case 83:
			artifactType = "Megasphere"
		case 2013:
			artifactType = "Supercharge"
		case 2014:
			artifactType = "Health bonus"
		case 2015:
			artifactType = "Armor bonus"
		case 2022:
			artifactType = "Invulnerability"
		case 2023:
			artifactType = "Berserk"
		case 2024:
			artifactType = "Partial invisibility"
		case 2026:
			artifactType = "Computer area map"
		case 2045:
			artifactType = "Light amplification visor"
		}
		fmt.Fprintf(w, "    <rect x=\"%d\" y=\"%d\" stroke=\"black\" width=\"20\" height=\"20\" fill=\"%s\"><title>%s [%s]</title></rect>\n", thing.XPosition-10, thing.YPosition-10, colour, artifactType, flags)
	}

	if opts.RenderKeys && (thing.ThingType == 5 || thing.ThingType == 6 || thing.ThingType == 13 || thing.ThingType == 38 || thing.ThingType == 39 || thing.ThingType == 40) {
		var colour string
		var keyType string
		switch thing.ThingType {
		case 5:
			colour = "blue"
			keyType = "Blue keycard"
		case 6:
			colour = "yellow"
			keyType = "Yellow keycard"
		case 13:
			colour = "red"
			keyType = "Red keycard"
		case 38:
			colour = "red"
			keyType = "Red skull key"
		case 39:
			colour = "yellow"
			keyType = "Yellow skull key"
		case 40:
			colour = "blue"
			keyType = "Blue skull key"
		}
		fmt.Fprintf(w, "    <rect x=\"%d\" y=\"%d\" width=\"20\" height=\"20\" fill=\"%s\"><title>%s [%s]</title></rect>\n", thing.XPosition-10, thing.YPosition-10, colour, keyType, flags)
	}

	if opts.RenderMonsters && (thing.ThingType == 7 || thing.ThingType == 9 || thing.ThingType == 16 || thing.ThingType == 58 || thing.ThingType == 64 || thing.ThingType == 65 || thing.ThingType == 66 || thing.ThingType == 67 || thing.ThingType == 68 || thing.ThingType == 69 || thing.ThingType == 71 || thing.ThingType == 72 || thing.ThingType == 84 || thing.ThingType == 3001 || thing.ThingType == 3002 || thing.ThingType == 3003 || thing.ThingType == 3004 || thing.ThingType == 3005 || thing.ThingType == 3006) {
		colour := "black"
		var radius int
		var monsterType string
		switch thing.ThingType {
		case 7:
			radius = 128
			monsterType = "Spiderdemon"
		case 9:
			radius = 20
			monsterType = "Shotgun guy"
		case 16:
			radius = 40
			monsterType = "Cyberdemon"
		case 58:
			radius = 30
			monsterType = "Spectre"
		case 64:
			radius = 20
			monsterType = "Arch-vile"
		case 65:
			radius = 20
			monsterType = "Heavy weapon dude"
		case 66:
			radius = 20
			monsterType = "Revenant"
		case 67:
			radius = 48
			monsterType = "Mancubus"
		case 68:
			radius = 64
			monsterType = "Arachnotron"
		case 69:
			radius = 24
			monsterType = "Hell knight"
		case 71:
			radius = 31
			monsterType = "Pain elemental"
		case 72:
			radius = 16
			monsterType = "Commander Keen"
		case 84:
			radius = 20
			monsterType = "Wolfenstein SS"
		case 3001:
			radius = 20
			monsterType = "Imp"
		case 3002:
			radius = 30
			monsterType = "Demon"
		case 3003:
			radius = 24
			monsterType = "Baron of Hell"
		case 3004:
			radius = 20
			monsterType = "Zombieman"
		case 3005:
			radius = 31
			monsterType = "Cacodemon"
		case 3006:
			radius = 16
			monsterType = "Lost soul"
		}
		fmt.Fprintf(w, "    <circle cx=\"%d\" cy=\"%d\" r=\"%d\" fill=\"%s\"><title>%s [%s]</title></circle>\n", thing.XPosition-10, thing.YPosition-10, radius, colour, monsterType, flags)
	}

	if opts.RenderPowerups && (thing.ThingType == 8 || thing.ThingType == 2011 || thing.ThingType == 2012 || thing.ThingType == 2018 || thing.ThingType == 2019 || thing.ThingType == 2025) {
		colour := "yellow"
		var powerUpType string
		switch thing.ThingType {
		case 8:
			powerUpType = "Backpack"
		case 2011:
			powerUpType = "Stimpack"
		case 2012:
			powerUpType = "Medikit"
		case 2018:
			powerUpType = "Armor"
		case 2019:
			powerUpType = "Megaarmor"
		case 2025:
			powerUpType = "Radiation shielding suit"
		}
		fmt.Fprintf(w, "    <rect x=\"%d\" y=\"%d\" width=\"20\" height=\"20\" stroke=\"black\" fill=\"%s\"><title>%s [%s]</title></rect>\n", thing.XPosition-10, thing.YPosition-10, colour, powerUpType, flags)
	}

	if opts.RenderWeapons && (thing.ThingType == 82 || thing.ThingType == 2001 || thing.ThingType == 2002 || thing.ThingType == 2003 || thing.ThingType == 2004 || thing.ThingType == 2005 || thing.ThingType == 2006) {
		colour := "red"
		var weaponType string
		switch thing.ThingType {
		case 82:
			weaponType = "Super shotgun"
		case 2001:
			weaponType = "Shotgun"
		case 2002:
			weaponType = "Chaingun"
		case 2003:
			weaponType = "Rocket launcher"
		case 2004:
			weaponType = "Plasma gun"
		case 2005:
			weaponType = "Chainsaw"
		case 2006:
			weaponType = "BFG9000"
		}
		fmt.Fprintf(w, "    <rect x=\"%d\" y=\"%d\" width=\"20\" height=\"20\" stroke=\"black\" fill=\"%s\"><title>%s [%s]</title></rect>\n", thing.XPosition-10, thing.YPosition-10, colour, weaponType, flags)
	}
}
