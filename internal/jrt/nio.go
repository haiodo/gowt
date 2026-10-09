package jrt

import (
	"bytes"
	"io"
	"net/url"
	"os"
	"path/filepath"
)

// Path is java.nio.file.Path as its string form. Options arguments (LinkOption, CopyOption,
// FileAttribute) arrive as an ignored []any: only the default behaviour is implemented.
type Path struct{ s string }

func PathOf(first string, more []string) *Path {
	return &Path{filepath.Join(append([]string{first}, more...)...)}
}

func (p *Path) Resolve(other any) *Path {
	switch o := other.(type) {
	case *Path:
		return &Path{filepath.Join(p.s, o.s)}
	case string:
		return &Path{filepath.Join(p.s, o)}
	}
	panic(NewIllegalArgumentException("Path.resolve: unsupported argument"))
}

func (p *Path) GetParent() *Path   { return &Path{filepath.Dir(p.s)} }
func (p *Path) ToString() string   { return p.s }
func (p *Path) String() string     { return p.s }
func (p *Path) GetFileName() *Path { return &Path{filepath.Base(p.s)} }

// ToUri is Path.toUri(): an absolute file:// URI.
func (p *Path) ToUri() *URI {
	abs, err := filepath.Abs(p.s)
	if err != nil {
		abs = p.s
	}
	return &URI{(&url.URL{Scheme: "file", Path: filepath.ToSlash(abs)}).String()}
}

func ioFail(err error) {
	if err != nil {
		panic(NewIOException())
	}
}

func FilesIsRegularFile(p *Path, _ []any) bool {
	st, err := os.Stat(p.s)
	return err == nil && st.Mode().IsRegular()
}

func FilesCreateDirectories(p *Path, _ []any) *Path {
	ioFail(os.MkdirAll(p.s, 0o755))
	return p
}

// FilesCopy is copy(InputStream, Path) and copy(Path, Path); like Java it refuses to overwrite.
func FilesCopy(src any, dst *Path, _ []any) int64 {
	var r io.Reader
	switch s := src.(type) {
	case *Path:
		f, err := os.Open(s.s)
		ioFail(err)
		defer f.Close()
		r = f
	case InputStream:
		b := ReadAllBytes(s)
		raw := make([]byte, len(b))
		for i, v := range b {
			raw[i] = byte(v)
		}
		r = bytes.NewReader(raw)
	}
	out, err := os.OpenFile(dst.s, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	ioFail(err)
	defer out.Close()
	n, err := io.Copy(out, r)
	ioFail(err)
	return n
}

func FilesReadAllBytes(p *Path) []int8 {
	b, err := os.ReadFile(p.s)
	ioFail(err)
	out := make([]int8, len(b))
	for i, v := range b {
		out[i] = int8(v)
	}
	return out
}

func FilesNewInputStream(p *Path, _ []any) InputStream { return NewFileInputStream(p.s) }

func FilesDeleteIfExists(p *Path) bool {
	err := os.Remove(p.s)
	if os.IsNotExist(err) {
		return false
	}
	ioFail(err)
	return true
}

// CreateTempDir backs an @TempDir field; RemoveTempDir deletes it after the class.
func CreateTempDir() *Path {
	d, err := os.MkdirTemp("", "junit")
	ioFail(err)
	return &Path{d}
}

func RemoveTempDir(p *Path) { os.RemoveAll(p.s) }

func FilesDelete(p *Path) { ioFail(os.Remove(p.s)) }

func FilesIsSymbolicLink(p *Path) bool {
	fi, err := os.Lstat(p.s)
	return err == nil && fi.Mode()&os.ModeSymlink != 0
}

func FilesReadSymbolicLink(p *Path) *Path {
	target, err := os.Readlink(p.s)
	ioFail(err)
	return &Path{target}
}
