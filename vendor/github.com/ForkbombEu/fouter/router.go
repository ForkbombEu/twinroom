package fouter

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type SlangFile struct {
	Path       string
	Content    string
	FileName   string
	Dir        string
	IsEmbedded bool
}

// CreateFileRouter discovers all .slang files in both the embedded FS and the base directory FS, applying the RouteHandler for each file.
func CreateFileRouter(baseDir string, embeddedFS *embed.FS, embeddedDir string, handler func(SlangFile)) error {
	// Handle files from the embedded FS
	if embeddedFS != nil {
		err := fs.WalkDir(embeddedFS, embeddedDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && strings.HasSuffix(path, ".slang") {
				content, err := embeddedFS.ReadFile(path)
				if err != nil {
					return err
				}
				relativePath := filepath.Dir(path)

				slangFile := SlangFile{
					Path:       path,
					Content:    string(content),
					FileName:   filepath.Base(path),
					Dir:        relativePath,
					IsEmbedded: true,
				}
				handler(slangFile)
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	// Handle files from the filesystem, but skip the embedded directory if it's a subdirectory of baseDir
	if baseDir != "" {
		err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			// Skip the embedded directory
			if info.IsDir() && filepath.Base(path) == filepath.Base(embeddedDir) {
				return filepath.SkipDir
			}

			if !info.IsDir() && strings.HasSuffix(info.Name(), ".slang") {
				content, err := os.ReadFile(path)
				if err != nil {
					return err
				}

				relativePath, err := filepath.Rel(baseDir, path)
				if err != nil {
					return err
				}

				slangFile := SlangFile{
					Path:       path,
					Content:    string(content),
					FileName:   info.Name(),
					Dir:        filepath.Dir(relativePath),
					IsEmbedded: false,
				}

				handler(slangFile)
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}
