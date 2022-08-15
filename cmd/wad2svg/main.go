package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	var w *WAD = &WAD{}
	w.ReadFrom(f)

	linedefs, vertexes := w.GetDefs(mapName)
	render(wadName, mapName, linedefs, vertexes)
}

type Vertex struct {
	x int16
	y int16
}

func render(wadName string, mapName string, linedefs *LumpPtr, vertexes *LumpPtr) {
	var numLineDefs = linedefs.size / 14
	var numVertexes = vertexes.size / 4
	var parsedVertexes = make([]Vertex, numVertexes)
	var minX, minY, maxX, maxY = int16(32767), int16(32767), int16(-32768), int16(-32768)
	for i := 0; i < int(numVertexes); i++ {
		var x = int16(vertexes.Lump.data[i*4]) | int16(vertexes.Lump.data[(i*4)+1])<<8
		var y = -(int16(vertexes.Lump.data[(i*4)+2]) | int16(vertexes.Lump.data[(i*4)+3])<<8)
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
		parsedVertexes[i] = Vertex{x, y}
	}
	fmt.Println("<?xml version=\"1.0\" standalone=\"no\"?>")
	fmt.Printf("<svg width=\"640px\" height=\"480px\" viewBox=\"%d %d %d %d\" xmlns=\"http://www.w3.org/2000/svg\">\n", minX, minY, maxX-minX, maxY-minY)
	fmt.Printf("  <title>%s - %s</title>\n", wadName, mapName)
	fmt.Println("  <g>")
	for i := 0; i < int(numLineDefs); i++ {
		var linedef = linedefs.Lump.data[14*i : 14*(i+1)]
		var vertex1 = parsedVertexes[binary.LittleEndian.Uint16(linedef[0:2])]
		var vertex2 = parsedVertexes[binary.LittleEndian.Uint16(linedef[2:4])]
		var lineType = binary.LittleEndian.Uint16(linedef[6:8])
		color := "black"
		if lineType == 1 || lineType == 2 || lineType == 3 || lineType == 4 || lineType == 16 {
			// Door
			color = "green"
		} else if lineType == 39 || lineType == 97 || lineType == 125 || lineType == 126 || lineType == 174 || lineType == 195 || lineType == 207 || lineType == 208 || lineType == 209 || lineType == 210 || lineType == 243 || lineType == 244 || lineType == 262 || lineType == 263 || lineType == 264 || lineType == 265 || lineType == 266 || lineType == 267 || lineType == 268 || lineType == 269 {
			// Teleporter
			color = "red"
		} else if lineType == 10 || lineType == 21 || lineType == 62 || lineType == 88 || lineType == 120 || lineType == 121 || lineType == 123 {
			// Lift
			color = "blue"
		} else if lineType == 11 || lineType == 51 || lineType == 52 || lineType == 124 || lineType == 197 || lineType == 198 {
			// Exit
			color = "purple"
		} else if lineType != 0 {
			// Some other type
			color = "orange"
		}
		fmt.Printf("    <path d=\"M %d %d L %d %d\" stroke=\"%s\" stroke-width=\"3\"/>\n", vertex1.x, vertex1.y, vertex2.x, vertex2.y, color)
	}
	fmt.Println("  </g>")
	fmt.Println("</svg>")
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

type Lump struct {
	data []byte
}

func (l *Lump) ReadFrom(r io.ReaderAt, size int, offset int64) {
	l.data = make([]byte, size)
	r.ReadAt(l.data, offset)
}

type LumpPtr struct {
	Lump *Lump
	size uint32
	name string
}

type InfoTable struct {
	lumps []*LumpPtr
}

func readLump(r io.ReaderAt, offset int64) (*LumpPtr, int64) {
	var buffer = make([]byte, 4)
	var nameBuffer = make([]byte, 8)
	var lump = &LumpPtr{}
	lump.Lump = &Lump{}
	r.ReadAt(buffer, offset)
	var lumpOffset = binary.LittleEndian.Uint32(buffer)
	r.ReadAt(buffer, offset+4)
	var size = binary.LittleEndian.Uint32(buffer)
	lump.size = size
	lump.Lump.ReadFrom(r, int(size), int64(lumpOffset))
	r.ReadAt(nameBuffer, offset+8)
	lump.name = string(nameBuffer)
	lump.Lump = &Lump{}
	lump.Lump.ReadFrom(r, int(size), int64(lumpOffset))
	return lump, offset + 16
}

func (i *InfoTable) ReadFrom(r io.ReaderAt, numLumps int, offset int64) {
	i.lumps = make([]*LumpPtr, 0, numLumps)
	var lump *LumpPtr
	for numLumps > 0 {
		lump, offset = readLump(r, offset)
		i.lumps = append(i.lumps, lump)
		numLumps--
	}
}

type WAD struct {
	identification string
	numlumps       uint32
	infoTable      *InfoTable
}

func (w *WAD) ReadFrom(r io.ReaderAt) {
	var buffer = make([]byte, 4)
	r.ReadAt(buffer, 0)
	w.identification = string(buffer)
	r.ReadAt(buffer, 4)
	w.numlumps = binary.LittleEndian.Uint32(buffer)
	r.ReadAt(buffer, 8)
	var infoTableOffset = binary.LittleEndian.Uint32(buffer)
	var infoTable = &InfoTable{}
	infoTable.ReadFrom(r, int(w.numlumps), int64(infoTableOffset))
	w.infoTable = infoTable
}

func (w *WAD) GetDefs(mapName string) (*LumpPtr, *LumpPtr) {
	var l = w.infoTable.lumps
	var padded = (mapName + "\x00\x00\x00\x00\x00\x00\x00\x00\x00")[0:8]
	var linedefs *LumpPtr
	var vertexes *LumpPtr
	found := false
	for idx := 0; idx < len(l); idx++ {
		if l[idx].name == padded {
			found = true
		}
		if found && l[idx].name == "LINEDEFS" {
			linedefs = l[idx]
		}
		if found && l[idx].name == "VERTEXES" {
			vertexes = l[idx]
			break
		}
	}
	return linedefs, vertexes
}
