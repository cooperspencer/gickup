package zip

import (
	"archive/zip"
	"compress/flate"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Create a Zip file `{repository}.zip` and recursively add the contents of all paths in the `tozip` array to it.
// Deletes the original contents of `repository` such that only the newly created Zip file remains.
func Zip(repository string, tozip []string) error {
	file, err := os.Create(fmt.Sprintf("%s.zip", repository))
	if err != nil {
		return err
	}
	defer file.Close()

	w := zip.NewWriter(file)
	w.RegisterCompressor(zip.Deflate, func(out io.Writer) (io.WriteCloser, error) {
		return flate.NewWriter(out, flate.BestCompression)
	})
	defer w.Close()

	parentDir := filepath.Dir(repository)

	walker := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		abspath, err := filepath.Rel(parentDir, path)
		if err != nil {
			return err
		}
		f, err := w.Create(abspath)
		if err != nil {
			return err
		}

		_, err = io.Copy(f, file)
		if err != nil {
			return err
		}

		return nil
	}

	for _, path := range tozip {
		err = filepath.Walk(path, walker)
		if err != nil {
			return err
		}
	}

	for _, dir := range tozip {
		err = os.RemoveAll(dir)
		if err != nil {
			return err
		}
	}
	return nil
}
