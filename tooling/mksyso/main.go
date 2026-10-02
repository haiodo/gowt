// Command mksyso writes a COFF object with one RT_MANIFEST resource (ID 1), for go build to link into a
// Windows executable: mksyso -arch amd64|arm64 -manifest app.manifest -o out.syso. No windres needed.
package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"log"
	"os"
)

func main() {
	arch := flag.String("arch", "amd64", "amd64 or arm64")
	manifest := flag.String("manifest", "", "manifest file")
	out := flag.String("o", "", "output .syso")
	flag.Parse()
	var machine, addr32nb uint16
	switch *arch {
	case "amd64":
		machine, addr32nb = 0x8664, 3 // IMAGE_REL_AMD64_ADDR32NB
	case "arm64":
		machine, addr32nb = 0xAA64, 2 // IMAGE_REL_ARM64_ADDR32NB
	default:
		log.Fatalf("unknown -arch %q", *arch)
	}
	data, err := os.ReadFile(*manifest)
	if err != nil {
		log.Fatal(err)
	}

	le := binary.LittleEndian
	var rsrc bytes.Buffer
	w := func(v ...any) {
		for _, x := range v {
			binary.Write(&rsrc, le, x)
		}
	}
	dir := func(entryID, offset uint32) { // directory with one ID entry; offset has the subdirectory bit
		w(uint32(0), uint32(0), uint32(0), uint16(0), uint16(1), entryID, offset)
	}
	dir(24, 0x80000000|24) // type RT_MANIFEST
	dir(1, 0x80000000|48)  // name 1
	dir(0x0409, 72)        // language en-US -> data entry
	const entryOff = 72    // the data entry's OffsetToData needs a relocation
	w(uint32(88), uint32(len(data)), uint32(0), uint32(0))
	rsrc.Write(data)
	for rsrc.Len()%4 != 0 {
		rsrc.WriteByte(0)
	}

	const headers = 20 + 40
	relocOff := headers + rsrc.Len()
	symOff := relocOff + 10
	var f bytes.Buffer
	fw := func(v ...any) {
		for _, x := range v {
			binary.Write(&f, le, x)
		}
	}
	fw(machine, uint16(1), uint32(0), uint32(symOff), uint32(1), uint16(0), uint16(0)) // file header
	f.WriteString(".rsrc\x00\x00\x00")
	fw(uint32(0), uint32(0), uint32(rsrc.Len()), uint32(headers), uint32(relocOff), uint32(0), uint16(1), uint16(0), uint32(0x40000040))
	f.Write(rsrc.Bytes())
	fw(uint32(entryOff), uint32(0), addr32nb) // relocation against symbol 0
	f.WriteString(".rsrc\x00\x00\x00")
	fw(uint32(0), uint16(1), uint16(0), uint8(3), uint8(0)) // static section symbol
	fw(uint32(4))                                           // empty string table
	if err := os.WriteFile(*out, f.Bytes(), 0o644); err != nil {
		log.Fatal(err)
	}
}
