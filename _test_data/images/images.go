package images

import (
	"embed"
	"io/fs"
)

//go:embed *.jpeg *.tif *.webp *.png
var contents embed.FS

func List() []string {
	result := make([]string, 0)
	_ = fs.WalkDir(contents, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			result = append(result, path)
		}
		return nil
	})
	return result
}

func Open(name string) (fs.File, error) {
	return contents.Open(name)
}
