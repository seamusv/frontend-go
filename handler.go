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
	"strings"
)

var (
	ErrDir              = errors.New("path is dir")
	ErrNoFrontendFolder = errors.New("no frontend folder")
)

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

func tryRead(prefix, requestedPath string, w http.ResponseWriter, r *http.Request) error {
	f, err := frontAssets.Open(path.Join(prefix, requestedPath))
	if err != nil {
		if paths := strings.Split(strings.TrimPrefix(requestedPath, "/"), "/"); len(paths) > 1 {
			f, err := frontAssets.Open(path.Join(prefix, paths[0]))
			if err != nil {
				// if not found, return nil to use index.html
				return nil
			}
			defer f.Close()
			fi, err := f.Stat()
			if err != nil {
				// should never error, but return nil to use index.html
				return nil
			}
			if fi.IsDir() {
				return ErrNoFrontendFolder
			}
			// may be a hack attack to sniff files, but return nil to use index.html
			return nil
		}
		if requestedPath == "/favicon.ico" {
			return ErrNoFrontendFolder
		}
		return err
	}
	defer f.Close()

	// Go's fs.Open() doesn't return error when reading directory,
	// But it is not needed here
	stat, _ := f.Stat()
	if stat.IsDir() {
		if !strings.HasSuffix(requestedPath, "/") {
			http.Redirect(w, r, requestedPath+"/", http.StatusTemporaryRedirect)
			return nil
		}
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
func NewSPAHandler(ctx context.Context, indexMiddleware func(handler http.Handler) http.Handler) (http.Handler, error) {
	if indexMiddleware == nil {
		indexMiddleware = func(handler http.Handler) http.Handler {
			return handler
		}
	}
	var handler http.Handler
	switch mode {
	case Release:
		o := normalizeRelOpt(opt)
		root := path.Join(o.FrontEndFolderPath, o.DistFolder)
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := tryRead(root, r.URL.Path, w, r)
			if err == nil {
				return
			}
			if errors.Is(err, ErrNoFrontendFolder) {
				http.NotFound(w, r)
				return
			}
			if errors.Is(err, ErrDir) {
				err = tryRead(root, filepath.Join(r.URL.Path, "index.html"), w, r)
				if err == nil {
					return
				}
			}
			if o.FrameworkType == NextJS {
				// SSG generates .html but request URL may not have extensions
				err = tryRead(root, r.URL.Path+".html", w, r)
				if err == nil {
					return
				}
			}
			indexMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				err := tryRead(root, "index.html", w, r)
				if err != nil {
					log.Println(err)
					http.NotFound(w, r)
				}
			})).ServeHTTP(w, r)
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
func NewSPAHandlerFunc(ctx context.Context, indexMiddleware func(handler http.Handler) http.Handler) (http.HandlerFunc, error) {
	h, err := NewSPAHandler(ctx, indexMiddleware)
	if err != nil {
		return nil, err
	}
	return func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r)
	}, nil
}

// MustNewSPAHandler is similar to [NewSPAHandler] but this calls panic when error.
func MustNewSPAHandler(ctx context.Context, indexMiddleware func(handler http.Handler) http.Handler) http.Handler {
	h, err := NewSPAHandler(ctx, indexMiddleware)
	if err != nil {
		panic(err)
	}
	return h
}

// MustNewSPAHandlerFunc is similar to [NewSPAHandlerFunc] but this calls panic when error.
func MustNewSPAHandlerFunc(ctx context.Context, indexMiddleware func(handler http.Handler) http.Handler) http.HandlerFunc {
	h, err := NewSPAHandlerFunc(ctx, indexMiddleware)
	if err != nil {
		panic(err)
	}
	return h
}
