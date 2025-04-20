package profiles

import (
	"embed"
	"io/fs"
	"path/filepath"
)

//go:embed default/*.icc
var contents embed.FS

// default/*.icc adobe/CMYK/*.icc adobe/RGB/*.icc basiccolor/gcr/*.icc basiccolor/2009/*.icc eci/*.icc icc/*.icc icc-srgb/*.icc

func List() []string {
	result := make([]string, 0)
	_ = fs.WalkDir(contents, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(path) == ".icc" {
			result = append(result, path)
		}
		return nil
	})
	return result
}

func Open(name string) (fs.File, error) {
	return contents.Open(name)
}
