// Command mksyso writes a COFF object with Windows resources, for go build to link into an
// executable: no windres needed. One object holds what the program needs, because a second .syso
// with its own manifest or icon makes the linker fail with a duplicate resource.
//
//	mksyso -arch amd64 -o app_windows_amd64.syso -icon 16.png,32.png,48.png,256.png \
//	    -version 1.2.3.4 -product App -company "Org" -description "App editor" [-manifest app.manifest]
//
// -icon takes an .ico file or square PNG files (at most 256), stored as they are. -manifest is
// RT_MANIFEST; leave it out for a program on the gowt facade, whose winmanifest package has it.
// Any of -version, -product, -company, -description, -copyright, -original and -internal writes a
// VERSIONINFO resource (en-US, Unicode).
package main

import (
	"flag"
	"log"
	"os"
)

func main() {
	arch := flag.String("arch", "amd64", "amd64 or arm64")
	manifest := flag.String("manifest", "", "manifest file (RT_MANIFEST)")
	icon := flag.String("icon", "", "an .ico file, or PNG files separated by commas")
	out := flag.String("o", "", "output .syso")
	var v versionInfo
	flag.StringVar(&v.version, "version", "", "file and product version, such as 1.2.3.4")
	flag.StringVar(&v.product, "product", "", "ProductName")
	flag.StringVar(&v.company, "company", "", "CompanyName")
	flag.StringVar(&v.description, "description", "", "FileDescription, the name Task Manager shows")
	flag.StringVar(&v.copyright, "copyright", "", "LegalCopyright")
	flag.StringVar(&v.original, "original", "", "OriginalFilename")
	flag.StringVar(&v.internal, "internal", "", "InternalName")
	flag.Parse()

	var rs []resource
	if *manifest != "" {
		data, err := os.ReadFile(*manifest)
		if err != nil {
			log.Fatal(err)
		}
		rs = append(rs, resource{rtManifest, 1, data})
	}
	if *icon != "" {
		ics, err := icons(*icon)
		if err != nil {
			log.Fatal(err)
		}
		rs = append(rs, ics...)
	}
	if v != (versionInfo{}) {
		data, err := versionResource(v)
		if err != nil {
			log.Fatal(err)
		}
		rs = append(rs, resource{rtVersion, 1, data})
	}
	obj, err := coff(*arch, rs)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(*out, obj, 0o644); err != nil {
		log.Fatal(err)
	}
}
