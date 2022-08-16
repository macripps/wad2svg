package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: wad2svg file map [imageWidth=1280] [imageHeight=1024]")
		os.Exit(1)
	}
	var fileName = os.Args[1]
	var wadName = filepath.Base(fileName)
	var mapName = os.Args[2]
	imageWidth := 1280
	imageHeight := 1024
	var err error
	if len(os.Args) > 3 {
		imageWidth, err = strconv.Atoi(os.Args[3])
		if err != nil {
			panic(err)
		}
	}
	if len(os.Args) > 4 {
		imageHeight, err = strconv.Atoi(os.Args[4])
		if err != nil {
			panic(err)
		}
	}
	f, err := os.Open(fileName)
	if err != nil {
		panic(err)
	}
	m := ReadMap(f, mapName)
	m.render(os.Stdout, wadName, mapName, imageWidth, imageHeight)
}

type Map struct {
	LineDefs []LineDef
	SideDefs []SideDef
	Vertexes []Vertex
	Sectors  []Sector
}

func (m *Map) parseLineDefs(r io.ReaderAt, size uint32, offset int64) {
	numLineDefs := size / 14
	m.LineDefs = make([]LineDef, 0, numLineDefs)
	var l LineDef
	fmt.Fprintf(os.Stderr, "Reading %d linedefs\n", numLineDefs)
	for numLineDefs > 0 {
		l, offset = ReadLineDefFrom(r, offset)
		m.LineDefs = append(m.LineDefs, l)
		numLineDefs--
	}
}

func (m *Map) parseSideDefs(r io.ReaderAt, size uint32, offset int64) {
	numSideDefs := size / 30
	m.SideDefs = make([]SideDef, 0, numSideDefs)
	var s SideDef
	fmt.Fprintf(os.Stderr, "Reading %d sidedefs\n", numSideDefs)
	for numSideDefs > 0 {
		s, offset = ReadSideDefFrom(r, offset)
		m.SideDefs = append(m.SideDefs, s)
		numSideDefs--
	}
}

func (m *Map) parseVertexes(r io.ReaderAt, size uint32, offset int64) {
	numVertexes := size / 4
	m.Vertexes = make([]Vertex, 0, numVertexes)
	fmt.Fprintf(os.Stderr, "Reading %d vertexes\n", numVertexes)
	var v Vertex
	for numVertexes > 0 {
		v, offset = ReadVertexFrom(r, offset)
		m.Vertexes = append(m.Vertexes, v)
		numVertexes--
	}
}

func (m *Map) parseSectors(r io.ReaderAt, size uint32, offset int64) {
	numSectors := size / 26
	m.Sectors = make([]Sector, 0, numSectors)
	fmt.Fprintf(os.Stderr, "Reading %d sectors\n", numSectors)
	var s Sector
	for numSectors > 0 {
		s, offset = ReadSectorFrom(r, int(size/26-numSectors), offset)
		m.Sectors = append(m.Sectors, s)
		numSectors--
	}

}

type LineDefFlag uint16

const (
	BLOCKS_MONSTERS_AND_PLAYERS LineDefFlag = 1 << iota
	BLOCKS_MONSTERS
	TWO_SIDED
	UPPER_TEXTURE_UNPEGGED
	LOWER_TEXTURE_UNPEGGED
	SECRET
	BLOCKS_SOUND
	NEVER_SHOWN_ON_AUTOMAP
	ALWAYS_SHOWN_ON_AUTOMAP
	CAN_ACTIVATE_MORE_THAN_ONCE
	ACTIVATED_WHEN_USED_BY_PLAYER
	ACTIVATED_WHEN_CROSSED_BY_MONSTER
	ACTIVATED_WHEN_BUMPED_BY_PLAYER
	CAN_BE_ACTIVATED_BY_MONSTERS_OR_PLAYER
	UNUSED
	BLOCKS_EVERYTHING

	ACTIVATED_WHEN_HIT_BY_PROJECTILE         = 0x0c00
	ACTIVATED_WHEN_CROSSED_BY_PROJECTILE     = 0x1400
	ACTIVATED_WHEN_USED_OR_CROSSED_BY_PLAYER = 0x1800
)

// TODO(macripps): Support HEXEN/ZDoom LineDef layout
type LineDef struct {
	start        uint16
	end          uint16
	flags        uint16
	specialType  uint16
	sectorTag    uint16
	rightSideDef uint16
	leftSideDef  uint16
}

func (l *LineDef) isDoor() bool {
	return l.specialType == 1 || l.specialType == 2 || l.specialType == 3 || l.specialType == 4 || l.specialType == 16
}
func (l *LineDef) isTeleporter() bool {
	return l.specialType == 39 || l.specialType == 97 || l.specialType == 125 || l.specialType == 126 || l.specialType == 174 || l.specialType == 195 || l.specialType == 207 || l.specialType == 208 || l.specialType == 209 || l.specialType == 210 || l.specialType == 243 || l.specialType == 244 || l.specialType == 262 || l.specialType == 263 || l.specialType == 264 || l.specialType == 265 || l.specialType == 266 || l.specialType == 267 || l.specialType == 268 || l.specialType == 269
}
func (l *LineDef) isLift() bool {
	return l.specialType == 10 || l.specialType == 21 || l.specialType == 62 || l.specialType == 88 || l.specialType == 120 || l.specialType == 121 || l.specialType == 123
}
func (l *LineDef) isExit() bool {
	return l.specialType == 11 || l.specialType == 51 || l.specialType == 52 || l.specialType == 124 || l.specialType == 197 || l.specialType == 198
}
func (l *LineDef) isSecret() bool {
	return l.flags&uint16(SECRET) == uint16(SECRET)
}
func (l LineDef) flip() LineDef {
	return LineDef{
		start:        l.end,
		end:          l.start,
		flags:        l.flags,
		specialType:  l.specialType,
		sectorTag:    l.sectorTag,
		rightSideDef: l.leftSideDef,
		leftSideDef:  l.rightSideDef,
	}
}

type SideDef struct {
	xOffset           int16
	yOffset           int16
	upperTextureName  string
	lowerTextureName  string
	middleTextureName string
	sectorNumber      uint16
}

type Vertex struct {
	x int16
	y int16
}

type Sector struct {
	sectorNumber   int
	floorHeight    int16
	ceilingHeight  uint16
	floorTexture   string
	ceilingTexture string
	lightLevel     uint16
	sectorType     uint16
	tagNumber      uint16
}

func (s *Sector) isSecret() bool {
	return s.sectorType == 9
}

func (s *Sector) isDamage() bool {
	return s.sectorType == 4 || s.sectorType == 5 || s.sectorType == 7 || s.sectorType == 16
}

var sectorFill = []string{"white", "white", "white", "white", "red", "red", "unused", "red", "white", "aqua", "green", "purple", "white", "white", "green", "unused", "red", "white"}
var sectorStroke = []string{"black", "black", "black", "black", "red", "red", "unused", "red", "black", "aqua", "green", "purple", "black", "black", "green", "unused", "red", "black"}
var sectorOpacity = []string{"1.0", "1.0", "1.0", "1.0", "0.2", "0.1", "unused", "0.05", "1.0", "0.5", "1.0", "1.0", "1.0", "1.0", "1.0", "unused", "0.2", "1.0"}

func (s *Sector) ToSvgAttributeString() string {
	return fmt.Sprintf("fill=\"%s\" stroke=\"%s\" opacity=\"%s\" stroke-width=\"1\"", sectorFill[s.sectorType], sectorStroke[s.sectorType], sectorOpacity[s.sectorType])
}

func (m *Map) render(out io.Writer, wadName string, mapName string, imageWidth, imageHeight int) {
	minX, minY, maxX, maxY := int16(32767), int16(32767), int16(-32768), int16(-32768)
	for i := 0; i < len(m.Vertexes); i++ {
		x := m.Vertexes[i].x
		y := m.Vertexes[i].y
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
	fmt.Fprintln(os.Stdout, "<?xml version=\"1.0\" standalone=\"no\"?>")
	fmt.Fprintf(os.Stdout, "<svg width=\"%d\" height=\"%d\" viewBox=\"%d %d %d %d\" xmlns=\"http://www.w3.org/2000/svg\">\n", imageWidth, imageHeight, minX, minY, maxX-minX, maxY-minY)
	fmt.Fprintf(os.Stdout, "  <title>%s - %s</title>\n", wadName, mapName)
	fmt.Fprintln(os.Stdout, "  <g>")

	sectors := m.Sectors
	for i, sector := range sectors {
		fmt.Fprintf(os.Stderr, "Rendering sector #%d/%d\n", i+1, len(sectors))
		m.renderSector(sector, i)
	}

	linedefsToDraw := make([]LineDef, len(m.LineDefs))
	copy(linedefsToDraw, m.LineDefs)

	fmt.Fprintln(os.Stdout, "  </g>")
	fmt.Fprintln(os.Stdout, "</svg>")
}

func (m *Map) renderSector(s Sector, i int) {
	fmt.Fprintf(os.Stdout, "    <!-- Sector type %d -->\n", s.sectorType)
	fmt.Fprintf(os.Stdout, "    <g %s>\n", s.ToSvgAttributeString())
	sectorLineDefs := make([]LineDef, 0)
	for sd := 0; sd < len(m.SideDefs); sd++ {
		sidedef := m.SideDefs[sd]
		if int(sidedef.sectorNumber) == i {
			for ld := 0; ld < len(m.LineDefs); ld++ {
				linedef := m.LineDefs[ld]
				if int(linedef.leftSideDef) == sd || int(linedef.rightSideDef) == sd {
					sectorLineDefs = append(sectorLineDefs, linedef)
				}
			}
		}
	}
	// Render all sector linedefs with appropriate fill
	m.renderAllLineDefs(sectorLineDefs)
	// Now render special linedefs again, with colors
	m.renderSpecialLineDefs(sectorLineDefs)
	fmt.Fprintf(os.Stdout, "    </g>\n")
}

func (m *Map) renderAllLineDefs(lds []LineDef) {
	sectorLineDefs := make([]LineDef, len(lds))
	copy(sectorLineDefs, lds)
	var linedefs []LineDef
	path := strings.Builder{}
	for len(sectorLineDefs) > 0 {
		linedefs, sectorLineDefs = selectLineDef(sectorLineDefs, func(ld1, ld2 LineDef) bool {
			return true
		})
		linedef := linedefs[0]
		start := m.Vertexes[linedef.start]
		end := m.Vertexes[linedef.end]
		path.WriteString(fmt.Sprintf(" M %d %d L %d,%d", start.x, start.y, end.x, end.y))
		for i := 1; i < len(linedefs); i++ {
			v := m.Vertexes[linedefs[i].end]
			path.WriteString(fmt.Sprintf(" %d,%d", v.x, v.y))
		}
	}
	fmt.Fprintf(os.Stdout, "    <path d=\"%s\" fill-rule=\"evenodd\"/>\n", path.String())
}

func (m *Map) renderSpecialLineDefs(lds []LineDef) {
	sectorLineDefs := make([]LineDef, 0, len(lds))
	for _, s := range lds {
		if s.specialType != 0 {
			sectorLineDefs = append(sectorLineDefs, s)
		}
	}
	for _, linedef := range sectorLineDefs {
		lineType := linedef.specialType
		if lineType != 0 {
			stroke := "orange"
			strokeWidth := 3
			if linedef.isDoor() {
				stroke = "green"
			} else if linedef.isTeleporter() {
				stroke = "red"
			} else if linedef.isLift() {
				stroke = "blue"
			} else if linedef.isExit() {
				stroke = "purple"
			}
			start := m.Vertexes[linedef.start]
			end := m.Vertexes[linedef.end]

			fmt.Fprintf(os.Stdout, "    <!-- Type %d -->\n", linedef.specialType)
			fmt.Fprintf(os.Stdout, "    <path d=\"M %d %d L %d %d\" stroke=\"%s\" stroke-width=\"%d\" />", start.x, start.y, end.x, end.y, stroke, strokeWidth)
		}
	}
}

func selectLineDef(lineDefs []LineDef, shouldInclude func(LineDef, LineDef) bool) ([]LineDef, []LineDef) {
	if len(lineDefs) == 1 {
		return lineDefs, []LineDef{}
	}
	nextLineDef := lineDefs[0]
	lineDefGroup := []LineDef{nextLineDef}
	lineDefs = lineDefs[1:]
	for i := 0; i < len(lineDefs); i++ {
		head := lineDefGroup[len(lineDefGroup)-1]
		tail := lineDefGroup[0]
		l := lineDefs[i]
		if l.start == head.end &&
			shouldInclude(l, head) {
			lineDefGroup = append(lineDefGroup, l)
			lineDefs[i] = lineDefs[len(lineDefs)-1]
			lineDefs = lineDefs[:len(lineDefs)-1]
			i = -1
		} else if l.end == tail.start &&
			shouldInclude(l, tail) {
			lineDefGroup = append([]LineDef{l}, lineDefGroup...)
			lineDefs[i] = lineDefs[len(lineDefs)-1]
			lineDefs = lineDefs[:len(lineDefs)-1]
			i = -1
		} else if l.end == head.end && shouldInclude(l, head) {
			lineDefGroup = append(lineDefGroup, l.flip())
			lineDefs[i] = lineDefs[len(lineDefs)-1]
			lineDefs = lineDefs[:len(lineDefs)-1]
			i = -1
		} else if l.start == tail.start && shouldInclude(l, tail) {
			lineDefGroup = append([]LineDef{l.flip()}, lineDefGroup...)
			lineDefs[i] = lineDefs[len(lineDefs)-1]
			lineDefs = lineDefs[:len(lineDefs)-1]
			i = -1
		}
	}
	return lineDefGroup, lineDefs
}

type LumpType uint32

const (
	THINGS LumpType = iota
	LINEDEFS
	SIDEDEFS
	VERTEXES
	SEGS
	SSECTORS
	NODES
	SECTORS
	REJECT
	BLOCKMAP
	BEHAVIOR
)

type LumpPtr struct {
	size   uint32
	offset uint32
	name   string
}

func readLump(r io.ReaderAt, offset int64) (*LumpPtr, int64) {
	var buffer = make([]byte, 8)
	var lump = &LumpPtr{}
	r.ReadAt(buffer, offset)
	lump.offset = binary.LittleEndian.Uint32(buffer[0:4])
	lump.size = binary.LittleEndian.Uint32(buffer[4:8])
	r.ReadAt(buffer, offset+8)
	lump.name = string(buffer)
	return lump, offset + 16
}

func ReadMap(r io.ReaderAt, mapName string) *Map {
	var buffer = make([]byte, 4)
	r.ReadAt(buffer, 4)
	numLumps := binary.LittleEndian.Uint32(buffer)
	r.ReadAt(buffer, 8)
	var infoTableOffset = binary.LittleEndian.Uint32(buffer)
	return readMapFromInfoTable(r, mapName, int(numLumps), int64(infoTableOffset))
}

func readMapFromInfoTable(r io.ReaderAt, mapName string, numLumps int, offset int64) *Map {
	mapNamePadded := (mapName + "\x00\x00\x00\x00\x00\x00\x00\x00")[:8]
	m := &Map{}
	var lump *LumpPtr
	found := false
	for numLumps > 0 {
		lump, offset = readLump(r, offset)
		if lump.name == mapNamePadded {
			fmt.Fprintf(os.Stderr, "Found map %s\n", lump.name)
			found = true
		}
		if found && lump.name == "LINEDEFS" {
			m.parseLineDefs(r, lump.size, int64(lump.offset))
		}
		if found && lump.name == "SIDEDEFS" {
			m.parseSideDefs(r, lump.size, int64(lump.offset))
		}
		if found && lump.name == "VERTEXES" {
			m.parseVertexes(r, lump.size, int64(lump.offset))
		}
		if found && lump.name == "SECTORS\x00" {
			m.parseSectors(r, lump.size, int64(lump.offset))
			break
		}
		numLumps--
	}
	return m
}

var linedef = make([]byte, 14)

func ReadLineDefFrom(r io.ReaderAt, offset int64) (LineDef, int64) {
	r.ReadAt(linedef, offset)
	l := LineDef{
		start:        binary.LittleEndian.Uint16(linedef[0:2]),
		end:          binary.LittleEndian.Uint16(linedef[2:4]),
		flags:        binary.LittleEndian.Uint16(linedef[4:6]),
		specialType:  binary.LittleEndian.Uint16(linedef[6:8]),
		sectorTag:    binary.LittleEndian.Uint16(linedef[8:10]),
		rightSideDef: binary.LittleEndian.Uint16(linedef[10:12]),
		leftSideDef:  binary.LittleEndian.Uint16(linedef[12:14]),
	}

	return l, offset + 14
}

var sidedef = make([]byte, 30)

func ReadSideDefFrom(r io.ReaderAt, offset int64) (SideDef, int64) {
	r.ReadAt(sidedef, offset)
	l := SideDef{
		sectorNumber: binary.LittleEndian.Uint16(sidedef[28:30]),
	}

	return l, offset + 30
}

var vertex = make([]byte, 4)

func ReadVertexFrom(r io.ReaderAt, offset int64) (Vertex, int64) {
	r.ReadAt(vertex, offset)
	v := Vertex{
		x: int16(vertex[0]) | int16(vertex[1])<<8,
		y: -(int16(vertex[2]) | int16(vertex[3])<<8),
	}

	return v, offset + 4
}

var sector = make([]byte, 26)

func ReadSectorFrom(r io.ReaderAt, idx int, offset int64) (Sector, int64) {
	r.ReadAt(sector, offset)
	s := Sector{
		sectorNumber: idx,
		floorHeight:  int16(sector[0]) | int16(sector[1])<<8,
		sectorType:   binary.LittleEndian.Uint16(sector[22:24]),
	}
	return s, offset + 26
}
