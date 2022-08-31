package wad

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

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
	Start        uint16
	End          uint16
	Flags        uint16
	SpecialType  uint16
	SectorTag    uint16
	RightSideDef uint16
	LeftSideDef  uint16
}

func (l *LineDef) IsDoor() bool {
	return l.SpecialType == 1 || l.SpecialType == 2 || l.SpecialType == 3 || l.SpecialType == 4 || l.SpecialType == 16
}
func (l *LineDef) IsTeleporter() bool {
	return l.SpecialType == 39 || l.SpecialType == 97 || l.SpecialType == 125 || l.SpecialType == 126 || l.SpecialType == 174 || l.SpecialType == 195 || l.SpecialType == 207 || l.SpecialType == 208 || l.SpecialType == 209 || l.SpecialType == 210 || l.SpecialType == 243 || l.SpecialType == 244 || l.SpecialType == 262 || l.SpecialType == 263 || l.SpecialType == 264 || l.SpecialType == 265 || l.SpecialType == 266 || l.SpecialType == 267 || l.SpecialType == 268 || l.SpecialType == 269
}
func (l *LineDef) IsLift() bool {
	return l.SpecialType == 10 || l.SpecialType == 21 || l.SpecialType == 62 || l.SpecialType == 88 || l.SpecialType == 120 || l.SpecialType == 121 || l.SpecialType == 123
}
func (l *LineDef) IsExit() bool {
	return l.SpecialType == 11 || l.SpecialType == 51 || l.SpecialType == 52 || l.SpecialType == 124 || l.SpecialType == 197 || l.SpecialType == 198
}
func (l *LineDef) IsSecret() bool {
	return l.Flags&uint16(SECRET) == uint16(SECRET)
}
func (l LineDef) Flip() LineDef {
	return LineDef{
		Start:        l.End,
		End:          l.Start,
		Flags:        l.Flags,
		SpecialType:  l.SpecialType,
		SectorTag:    l.SectorTag,
		RightSideDef: l.LeftSideDef,
		LeftSideDef:  l.RightSideDef,
	}
}

type SideDef struct {
	XOffset           int16
	YOffset           int16
	UpperTextureName  string
	LowerTextureName  string
	MiddleTextureName string
	SectorNumber      uint16
}

type Vertex struct {
	X int16
	Y int16
}

type Sector struct {
	FloorHeight    int16
	CeilingHeight  int16
	FloorTexture   string
	CeilingTexture string
	LightLevel     uint16
	SectorType     uint16
	TagNumber      uint16
}

func (s *Sector) isSecret() bool {
	return s.SectorType == 9
}

func (s *Sector) isDamage() bool {
	return s.SectorType == 4 || s.SectorType == 5 || s.SectorType == 7 || s.SectorType == 16
}

var sectorFill = []string{"white", "white", "white", "white", "red", "red", "unused", "red", "white", "aqua", "green", "purple", "white", "white", "green", "unused", "red", "white"}
var sectorStroke = []string{"black", "black", "black", "black", "red", "red", "unused", "red", "black", "aqua", "green", "purple", "black", "black", "green", "unused", "red", "black"}
var sectorOpacity = []string{"1.0", "1.0", "1.0", "1.0", "0.2", "0.1", "unused", "0.05", "1.0", "0.5", "1.0", "1.0", "1.0", "1.0", "1.0", "unused", "0.2", "1.0"}

func (s *Sector) ToSvgAttributeString() string {
	if int(s.SectorType) < len(sectorFill) {
		return fmt.Sprintf("fill=\"%s\" stroke=\"%s\" fill-opacity=\"%s\" stroke-width=\"1\"", sectorFill[s.SectorType], sectorStroke[s.SectorType], sectorOpacity[s.SectorType])
	}
	return "fill=\"white\" stroke=\"black\" fill-opacity=\"1.0\" stroke-width=\"1\""
}

type Thing struct {
	XPosition int16
	YPosition int16
	Angle     uint16
	ThingType uint16
	Flags     uint16
}

type Map struct {
	LineDefs []LineDef
	SideDefs []SideDef
	Vertexes []Vertex
	Sectors  []Sector
	Things   []Thing
}

func (m *Map) parseThings(r io.ReaderAt, size uint32, offset int64) {
	numThings := size / 10
	m.Things = make([]Thing, 0, numThings)
	var t Thing
	fmt.Fprintf(os.Stderr, "Reading %d things\n", numThings)
	for numThings > 0 {
		t, offset = ReadThingFrom(r, offset)
		m.Things = append(m.Things, t)
		numThings--
	}
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
		s, offset = ReadSectorFrom(r, offset)
		m.Sectors = append(m.Sectors, s)
		numSectors--
	}
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

func (m *Map) ReadFrom(r io.ReaderAt, mapName string) {
	var buffer = make([]byte, 4)
	r.ReadAt(buffer, 4)
	numLumps := binary.LittleEndian.Uint32(buffer)
	r.ReadAt(buffer, 8)
	var infoTableOffset = binary.LittleEndian.Uint32(buffer)
	m.readMapFromInfoTable(r, mapName, int(numLumps), int64(infoTableOffset))
}

func (m *Map) readMapFromInfoTable(r io.ReaderAt, mapName string, numLumps int, offset int64) {
	mapNamePadded := (mapName + "\x00\x00\x00\x00\x00\x00\x00\x00")[:8]
	var lump *LumpPtr
	found := false
	for numLumps > 0 {
		lump, offset = readLump(r, offset)
		if lump.name == mapNamePadded {
			fmt.Fprintf(os.Stderr, "Found map %s\n", lump.name)
			found = true
		}
		if found && lump.name == "THINGS\x00\x00" {
			m.parseThings(r, lump.size, int64(lump.offset))
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
}

var linedef = make([]byte, 14)

func ReadLineDefFrom(r io.ReaderAt, offset int64) (LineDef, int64) {
	r.ReadAt(linedef, offset)
	l := LineDef{
		Start:        binary.LittleEndian.Uint16(linedef[0:2]),
		End:          binary.LittleEndian.Uint16(linedef[2:4]),
		Flags:        binary.LittleEndian.Uint16(linedef[4:6]),
		SpecialType:  binary.LittleEndian.Uint16(linedef[6:8]),
		SectorTag:    binary.LittleEndian.Uint16(linedef[8:10]),
		RightSideDef: binary.LittleEndian.Uint16(linedef[10:12]),
		LeftSideDef:  binary.LittleEndian.Uint16(linedef[12:14]),
	}

	return l, offset + 14
}

var sidedef = make([]byte, 30)

func ReadSideDefFrom(r io.ReaderAt, offset int64) (SideDef, int64) {
	r.ReadAt(sidedef, offset)
	l := SideDef{
		SectorNumber: binary.LittleEndian.Uint16(sidedef[28:30]),
	}

	return l, offset + 30
}

var vertex = make([]byte, 4)

func ReadVertexFrom(r io.ReaderAt, offset int64) (Vertex, int64) {
	r.ReadAt(vertex, offset)
	v := Vertex{
		X: int16(vertex[0]) | int16(vertex[1])<<8,
		Y: -(int16(vertex[2]) | int16(vertex[3])<<8),
	}

	return v, offset + 4
}

var sector = make([]byte, 26)

func ReadSectorFrom(r io.ReaderAt, offset int64) (Sector, int64) {
	r.ReadAt(sector, offset)
	s := Sector{
		FloorHeight:    int16(sector[0]) | int16(sector[1])<<8,
		CeilingHeight:  int16(sector[2]) | int16(sector[3])<<8,
		FloorTexture:   string(sector[4:12]),
		CeilingTexture: string(sector[12:20]),
		LightLevel:     binary.LittleEndian.Uint16(sector[20:22]),
		SectorType:     binary.LittleEndian.Uint16(sector[22:24]),
		TagNumber:      binary.LittleEndian.Uint16(sector[24:26]),
	}
	return s, offset + 26
}

var thing = make([]byte, 10)

func ReadThingFrom(r io.ReaderAt, offset int64) (Thing, int64) {
	r.ReadAt(thing, offset)
	t := Thing{
		XPosition: int16(thing[0]) | int16(thing[1])<<8,
		YPosition: -(int16(thing[2]) | int16(thing[3])<<8),
		Angle:     binary.LittleEndian.Uint16(thing[4:6]),
		ThingType: binary.LittleEndian.Uint16(thing[6:8]),
		Flags:     binary.LittleEndian.Uint16(thing[8:10]),
	}
	return t, offset + 10
}
