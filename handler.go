package frontend

import (
	"context"
	"embed"
	"errors"
	"io"
	"log"
	"mime"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"path/filepath"
	"strconv"
)

var ErrDir = errors.New("path is dir")

var mode Mode = Development

var frontAssets embed.FS
var opt Opt

func SetFrontAsset(assets embed.FS, o Opt) {
	frontAssets = assets
	mode = Release
	opt = o
}

func SetOption(o Opt) {
	opt = o
}

type FallbackInterceptor func(r *http.Request, reader io.Reader) io.Reader

var fallbackInterceptor FallbackInterceptor = func(r *http.Request, reader io.Reader) io.Reader {
	return reader
}

func SetFallbackInterceptor(f FallbackInterceptor) {
	fallbackInterceptor = f
}

func open(prefix, requestedPath string) (io.ReadCloser, error) {
	f, err := frontAssets.Open(path.Join(prefix, requestedPath))
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

func tryRead(prefix, requestedPath string, w http.ResponseWriter) error {
	f, err := frontAssets.Open(path.Join(prefix, requestedPath))
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

// NewSPAHandler is handler that handles SPA contents.
//
// Use with net/http:
//
//	h, err := NewSPAHandler(ctx)
//	http.Handle("/", h)
func NewSPAHandler(ctx context.Context) (http.Handler, error) {
	var handler http.Handler
	switch mode {
	case Release:
		o := normalizeRelOpt(opt)
		root := path.Join(o.FrontEndFolderPath, o.DistFolder)
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := tryRead(root, r.URL.Path, w)
			if err == nil {
				return
			}
			if o.FrameworkType == NextJS {
				// SSG generates .html but request URL may not have extensions
				err = tryRead(root, r.URL.Path+".html", w)
				if err == nil {
					return
				}
			}
			if f, err := open(root, "index.html"); err == nil {
				defer f.Close()
				w.Header().Set("Content-Type", "text/html")
				_, _ = io.Copy(w, fallbackInterceptor(r, f))
			} else {
				panic(err)
			}
		})
	case Development:
		o, err := normalizeDevOpt(".", opt)
		if err != nil {
			return nil, err
		}
		if !o.SkipRunningDevServer {
			_, host, err := startDevServer(ctx, o.FrontEndFolderPath, o.DevServerCommand)
			if err != nil {
				return nil, err
			}
			u, err := url.Parse(host)
			if err != nil {
				log.Fatal(err)
			}
			handler = httputil.NewSingleHostReverseProxy(u)
		} else if o.Port != 0 {
			// todo: test
			u, _ := url.Parse("http://localhost:" + strconv.Itoa(int(o.Port)))
			handler = httputil.NewSingleHostReverseProxy(u)
		} else {
			// todo: test
			handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// do nothing
			})
		}
	}

	return handler, nil
}

// NewSPAHandlerFunc is handler function that handles SPA contents.
//
// Use with chi:
//
//	r := chi.NewRouter()
//	c, err := NewSPAHandlerFunc(ctx)
//	http.NotFound(h)
func NewSPAHandlerFunc(ctx context.Context) (http.HandlerFunc, error) {
	h, err := NewSPAHandler(ctx)
	if err != nil {
		return nil, err
	}
	return func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r)
	}, nil
}

// MustNewSPAHandler is similar to [NewSPAHandler] but this calls panic when error.
func MustNewSPAHandler(ctx context.Context) http.Handler {
	h, err := NewSPAHandler(ctx)
	if err != nil {
		panic(err)
	}
	return h
}

// MustNewSPAHandlerFunc is similar to [NewSPAHandlerFunc] but this calls panic when error.
func MustNewSPAHandlerFunc(ctx context.Context) http.HandlerFunc {
	h, err := NewSPAHandlerFunc(ctx)
	if err != nil {
		panic(err)
	}
	return h
}
