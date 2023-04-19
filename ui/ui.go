package ui

import (
	"context"
	"embed"
	"io/fs"
	"net/http"
	"os"
	"path"

	"github.com/rs/zerolog/log"
)

var (
	//go:embed dist/*
	Dist embed.FS
)

const root = "dist" // This should match the above embed directive

// fsFunc is short-hand for constructing a http.FileSystem
// implementation
type fsFunc func(name string) (fs.File, error)

func (f fsFunc) Open(name string) (fs.File, error) {
	return f(name)
}

// AssetHandler returns an http.Handler that will serve files from
// the Assets embed.FS.  When locating a file, it will strip the given
// prefix from the request and prepend the root to the filesystem
// lookup: typical prefix might be /web/, and root would be build.
func AssetHandler(ctx context.Context, prefix string) http.Handler {
	handler := fsFunc(func(name string) (fs.File, error) {
		log.Ctx(ctx).Trace().Str("assetPath", name).Msg("Looking up asset")
		assetPath := path.Join(root, name)
		// If we can't find the asset, return the default index.html
		// content
		f, err := Dist.Open(assetPath)
		if err != nil {
			log.Ctx(ctx).Trace().Err(err).Msg("Asset lookup result")
		}
		if os.IsNotExist(err) {
			return Dist.Open("index.html")
		}

		// Otherwise assume this is a legitimate request routed
		// correctly
		return f, err
	})

	return http.StripPrefix(prefix, http.FileServer(http.FS(handler)))
}
