package frontend

import (
	"embed"
	"errors"
	"net/http"
	"net/url"
	"path/filepath"
)

type Frontend struct {
	devServerCommand   string
	distFolder         string
	fallbackPath       string
	frontAssets        embed.FS
	frontEndFolderName string
	frontEndFolderPath string
	devUrl             *url.URL
	devStopFunc        func()
}

func NewFrontend(opts ...Option) (*Frontend, error) {
	f := &Frontend{
		devServerCommand:   "yarn dev",
		distFolder:         "dist",
		frontEndFolderName: "frontend",
		fallbackPath:       "index.html",
	}

	for _, o := range opts {
		if err := o(f); err != nil {
			return nil, err
		}
	}

	return f, nil
}

func (f *Frontend) Handler() http.Handler {
	if f.devUrl != nil {
		return f.devHandler()
	}
	return f.prodHandler()
}

func (f *Frontend) HandlerFunc() http.HandlerFunc {
	return f.Handler().ServeHTTP
}

var (
	ErrDir                 = errors.New("path is dir")
	ErrPackageJsonNotFound = errors.New("package.json not found")
)

type Option func(*Frontend) error

func WithDevServerCommand(devServerCommand string) Option {
	return func(f *Frontend) error {
		f.devServerCommand = devServerCommand
		return nil
	}
}

func WithDistFolder(distFolder string) Option {
	return func(f *Frontend) error {
		f.distFolder = distFolder
		return nil
	}
}

func WithFallbackPath(fallbackPath string) Option {
	return func(f *Frontend) error {
		f.fallbackPath = fallbackPath
		return nil
	}
}

func WithFrontAssets(frontAssets embed.FS) Option {
	return func(f *Frontend) error {
		f.frontAssets = frontAssets
		return nil
	}
}

func WithFrontEndFolderName(frontEndFolderName, frontEndFolderPath string) Option {
	return func(f *Frontend) error {
		f.frontEndFolderName = frontEndFolderName
		if frontEndFolderPath == "" || !filepath.IsAbs(frontEndFolderPath) {
			p, err := filepath.Abs(frontEndFolderPath)
			if err != nil {
				return err
			}
			f.frontEndFolderPath = filepath.Join(p, frontEndFolderName)
		} else {
			_, f.frontEndFolderName = filepath.Split(frontEndFolderPath)
		}
		return nil
	}
}
