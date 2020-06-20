package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alecthomas/kingpin"
	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/mknote"
	"github.com/saracen/walker"
)

var (
	flag struct {
		Source string
		Dest   string
	}
)

func main() {
	app := kingpin.New("photo", "Sort my photos").Author("kyoh86")
	app.Arg("source", "Target file or directory").Required().ExistingFileOrDirVar(&flag.Source)
	app.Arg("dest", "Output directory").Required().StringVar(&flag.Dest)

	kingpin.MustParse(app.Parse(os.Args[1:]))

	if err := walker.Walk(flag.Source, func(path string, fi os.FileInfo) error {
		if fi.IsDir() {
			return nil
		}

		ext := strings.ToUpper(filepath.Ext(path))
		if !strings.Contains(".NEF.CR2.CR.JPG.JPEG", ext) {
			return nil // unsupported file
		}

		rel, err := filepath.Rel(flag.Source, path)
		if err != nil {
			return fmt.Errorf("parsing path as relative %s: %w", path, err)
		}

		relDir := filepath.Dir(rel)
		relBase := filepath.Base(rel)

		t, err := getTime(path)
		if err != nil {
			log.Printf("failed to parse %s (%s)\n", path, err)
			return nil
		}

		dirAbs := filepath.Join(flag.Dest, relDir, t.Format("2006"), t.Format("2006-01-02"))
		if err := os.MkdirAll(dirAbs, 0755); err != nil {
			return fmt.Errorf("making dest dir %s: %w", dirAbs, err)
		}
		abs := filepath.Join(dirAbs, t.Format("2006-01-02_15-04-05_")+relBase)

		return copyFile(path, fi, abs)
	}); err != nil {
		log.Fatalln(err)
	}
}

func copyFile(source string, sourceInfo os.FileInfo, dest string) error {
	sourceFile, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("opening source file to copy %s: %w", source, err)
	}
	defer sourceFile.Close()

	destFile, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, sourceInfo.Mode())
	if err != nil {
		return fmt.Errorf("opening dest file to copy %s: %w", dest, err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("copying file %s to %s: %w", source, dest, err)
	}
	return nil
}

func getTime(filename string) (*time.Time, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Optionally register camera makenote data parsing - currently Nikon and
	// Canon are supported.
	exif.RegisterParsers(mknote.All...)

	x, err := exif.Decode(file)
	if err != nil {
		return nil, err
	}

	dto, err := x.Get(exif.DateTimeOriginal)
	if err != nil {
		return nil, err
	}
	str, err := dto.StringVal()
	if err != nil {
		return nil, err
	}
	t, err := time.Parse("2006:01:02 15:04:05", str)
	if err != nil {
		return nil, err
	}
	return &t, err
}
