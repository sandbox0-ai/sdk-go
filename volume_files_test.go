package sandbox0

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteVolumeDirectoryTar(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "dir"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "dir", "hello.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("hello.txt", filepath.Join(root, "dir", "link.txt")); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := writeVolumeDirectoryTar(&buf, root); err != nil {
		t.Fatalf("write tar: %v", err)
	}

	entries := map[string]*tar.Header{}
	tr := tar.NewReader(&buf)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("read tar: %v", err)
		}
		entries[hdr.Name] = hdr
		if hdr.Name == "dir/hello.txt" {
			content, err := io.ReadAll(tr)
			if err != nil {
				t.Fatalf("read file: %v", err)
			}
			if string(content) != "hello" {
				t.Fatalf("file content = %q, want %q", content, "hello")
			}
		}
	}

	if entries["dir/"] == nil {
		t.Fatalf("missing dir entry")
	}
	if entries["dir/hello.txt"] == nil {
		t.Fatalf("missing file entry")
	}
	link := entries["dir/link.txt"]
	if link == nil {
		t.Fatalf("missing symlink entry")
	}
	if link.Typeflag != tar.TypeSymlink || link.Linkname != "hello.txt" {
		t.Fatalf("symlink = type %q target %q, want symlink to hello.txt", link.Typeflag, link.Linkname)
	}
}
