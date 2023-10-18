package frontend

import (
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"path/filepath"
)

func (f *Frontend) prodHandler() http.Handler {
	root := path.Join(f.frontEndFolderPath, f.distFolder)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := tryRead(f.frontAssets, root, r.URL.Path, w)
		if err == nil {
			return
		}
		if f, err := open(f.frontAssets, root, filepath.Join(f.frontEndFolderName, "index.html")); err == nil {
			defer f.Close()
			w.Header().Set("Content-Type", "text/html")
			_, _ = io.Copy(w, f)
		} else {
			panic(err)
		}
	})

	return handler
}

func open(assets fs.FS, prefix, requestedPath string) (io.ReadCloser, error) {
	f, err := assets.Open(path.Join(prefix, requestedPath))
	if err != nil {
		return nil, err
	}

	stat, _ := f.Stat()
	if stat.IsDir() {
		_ = f.Close()
		return nil, ErrDir
	}

	return f, nil
}

func tryRead(assets fs.FS, prefix, requestedPath string, w http.ResponseWriter) error {
	f, err := assets.Open(path.Join(prefix, requestedPath))
	if err != nil {
		return err
	}
	defer f.Close()

	// Go's fs.Open() doesn't return error when reading directory,
	// But it is not needed here
	stat, _ := f.Stat()
	if stat.IsDir() {
		return ErrDir
	}

	contentType := mime.TypeByExtension(filepath.Ext(requestedPath))
	w.Header().Set("Content-Type", contentType)
	_, err = io.Copy(w, f)
	return err
}
