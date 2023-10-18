package frontend

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
)

func (f *Frontend) StartDevServer() error {
	if f.devStopFunc != nil {
		return fmt.Errorf("dev server is already running")
	}

	if _, err := os.Stat(filepath.Join(f.frontEndFolderPath, "package.json")); os.IsNotExist(err) {
		return ErrPackageJsonNotFound
	}

	ds, host, err := startDevServer(context.Background(), f.frontEndFolderPath, f.devServerCommand)
	if err != nil {
		return err
	}
	f.devStopFunc = ds.Stop
	u, err := url.Parse(host)
	if err != nil {
		return err
	}
	f.devUrl = u
	return nil
}

func (f *Frontend) StopDevServer() error {
	if f.devStopFunc == nil {
		return fmt.Errorf("dev server is not running")
	}
	f.devStopFunc()
	return nil
}

func (f *Frontend) devHandler() http.Handler {
	return httputil.NewSingleHostReverseProxy(f.devUrl)
}
