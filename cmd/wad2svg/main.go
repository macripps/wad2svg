package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: wad2svg file map")
		os.Exit(1)
	}
	var fileName = os.Args[1]
	var wadName = filepath.Base(fileName)
	var mapName = os.Args[2]
	f, err := os.Open(fileName)
	if err != nil {
		panic(err)
	}
	m := ReadMap(f, mapName)
	m.render(wadName, mapName, os.Stdout)
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

func (m *Map) parseSectorDefs(r io.ReaderAt, size uint32, offset int64) {
	numSectors := size / 26
	m.Sectors = make([]Sector, 0, numSectors)
	fmt.Fprintf(os.Stderr, "Reading %d sectors\n", numSectors)
	var s Sector
	for numSectors > 0 {
		s, offset = ReadSectorFrom(r, offset)
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
	floorHeight    uint16
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

func (m *Map) render(wadName string, mapName string, out io.Writer) {
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
	fmt.Println("<?xml version=\"1.0\" standalone=\"no\"?>")
	fmt.Printf("<svg width=\"2560\" height=\"2048\" viewBox=\"%d %d %d %d\" xmlns=\"http://www.w3.org/2000/svg\">\n", minX, minY, maxX-minX, maxY-minY)
	fmt.Printf("  <title>%s - %s</title>\n", wadName, mapName)
	fmt.Println("  <g>")
	linedefsToDraw := make([]LineDef, len(m.LineDefs))
	copy(linedefsToDraw, m.LineDefs)
	var linedefs []LineDef
	for len(linedefsToDraw) > 0 {
		linedefs, linedefsToDraw = selectLineDef(linedefsToDraw)
		linedef := linedefs[0]
		lineType := linedef.specialType
		color := "black"
		width := 1
		if linedef.isDoor() {
			color = "green"
			width = 3
		} else if linedef.isTeleporter() {
			color = "red"
			width = 3
		} else if linedef.isLift() {
			color = "blue"
			width = 3
		} else if linedef.isExit() {
			color = "purple"
			width = 3
		} else if lineType != 0 {
			// Some other type
			color = "orange"
			width = 3
		}
		m.renderPolyline(linedefs, color, "none", width)
	}
	sectors := m.Sectors
	for i := 0; i < len(sectors); i++ {
		sector := sectors[i]
		if sector.isSecret() {
			m.renderSector(sector, i, "aqua")
		} else if sector.isDamage() {
			m.renderSector(sector, i, "red")
		}
	}
	fmt.Println("  </g>")
	fmt.Println("</svg>")
}

func (m *Map) renderSector(s Sector, i int, color string) {
	linedefs := make([]LineDef, 0)
	for sd := 0; sd < len(m.SideDefs); sd++ {
		sidedef := m.SideDefs[sd]
		if int(sidedef.sectorNumber) == i {
			for ld := 0; ld < len(m.LineDefs); ld++ {
				linedef := m.LineDefs[ld]
				if int(linedef.leftSideDef) == sd || int(linedef.rightSideDef) == sd {
					linedefs = append(linedefs, linedef)
				}
			}
		}
	}
	m.renderPolyline(topoSort(linedefs), color, color, 3)
}

func topoSort(lineDefs []LineDef) []LineDef {
	lineDefGroup := make([]LineDef, 0, len(lineDefs))
	nextLineDef := lineDefs[0]
	lineDefGroup = append(lineDefGroup, nextLineDef)
	lineDefs[0] = lineDefs[len(lineDefs)-1]
	lineDefs = lineDefs[:len(lineDefs)-1]
	head := nextLineDef
	tail := nextLineDef
	for i := 0; i < len(lineDefs); i++ {
		l := lineDefs[i]
		if l.start == head.end {
			lineDefGroup = append(lineDefGroup, l)
			lineDefs[i] = lineDefs[len(lineDefs)-1]
			lineDefs = lineDefs[:len(lineDefs)-1]
			i = 0
			head = l
		} else if l.end == tail.start {
			lineDefGroup = append([]LineDef{l}, lineDefGroup...)
			lineDefs[i] = lineDefs[len(lineDefs)-1]
			lineDefs = lineDefs[:len(lineDefs)-1]
			i = 0
			tail = l
		}
	}
	return lineDefGroup
}

func (m *Map) renderPolyline(linedefs []LineDef, stroke, fill string, strokeWidth int) {
	linedef := linedefs[0]
	path := strings.Builder{}
	start := m.Vertexes[linedef.start]
	end := m.Vertexes[linedef.end]
	path.WriteString(fmt.Sprintf("%d,%d %d,%d", start.x, start.y, end.x, end.y))
	for i := 1; i < len(linedefs); i++ {
		v := m.Vertexes[linedefs[i].end]
		path.WriteString(fmt.Sprintf(" %d,%d", v.x, v.y))
	}
	fmt.Fprintf(os.Stdout, "    <polyline points=\"%s\" fill=\"%s\" stroke=\"%s\" stroke-width=\"%d\"/>\n", path.String(), fill, stroke, strokeWidth)
}

func selectLineDef(lineDefs []LineDef) ([]LineDef, []LineDef) {
	lineDefGroup := make([]LineDef, 0)
	nextLineDef := lineDefs[0]
	lineDefGroup = append(lineDefGroup, nextLineDef)
	lineDefs[0] = lineDefs[len(lineDefs)-1]
	lineDefs = lineDefs[:len(lineDefs)-1]
	head := nextLineDef
	tail := nextLineDef
	for i := 0; i < len(lineDefs); i++ {
		l := lineDefs[i]
		if l.start == head.end &&
			l.flags == head.flags &&
			l.specialType == head.specialType {
			lineDefGroup = append(lineDefGroup, l)
			lineDefs[i] = lineDefs[len(lineDefs)-1]
			lineDefs = lineDefs[:len(lineDefs)-1]
			i = 0
			head = l
		} else if l.end == tail.start &&
			l.flags == tail.flags &&
			l.specialType == tail.specialType {
			lineDefGroup = append([]LineDef{l}, lineDefGroup...)
			lineDefs[i] = lineDefs[len(lineDefs)-1]
			lineDefs = lineDefs[:len(lineDefs)-1]
			i = 0
			tail = l
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
			m.parseSectorDefs(r, lump.size, int64(lump.offset))
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

	if l.leftSideDef == 600 || l.rightSideDef == 600 {
		fmt.Fprintf(os.Stderr, "Found linedef for sidedef 600 - %v\n", l)
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

func ReadSectorFrom(r io.ReaderAt, offset int64) (Sector, int64) {
	r.ReadAt(sector, offset)
	s := Sector{
		sectorType: binary.LittleEndian.Uint16(sector[22:24]),
	}
	return s, offset + 26
}
