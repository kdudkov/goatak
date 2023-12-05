package staticfiles

import (
	"embed"
	"errors"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/aofei/air"
)

//go:embed static
var staticFiles embed.FS

var hfs = http.FS(staticFiles)

func EmbedFiles(a *air.Air, prefix string) {
	if strings.HasSuffix(prefix, "/") {
		prefix += "*"
	} else {
		prefix += "/*"
	}

	h := func(req *air.Request, res *air.Response) error {
		path := req.Param("*").Value().String()
		path = filepath.FromSlash("/static/" + path)
		path = filepath.Clean(path)

		f, err := hfs.Open(path)
		if err != nil {
			var _t2 *fs.PathError
			if ok := errors.Is(err, _t2); ok {
				return a.NotFoundHandler(req, res)
			}

			return err
		}

		if res.Header.Get("Content-Type") == "" {
			res.Header.Set("Content-Type", mime.TypeByExtension(filepath.Ext(path)))
		}

		return res.Write(f)
	}

	a.BATCH([]string{http.MethodGet, http.MethodHead}, prefix, h)
}

func EmbedFile(a *air.Air, path string, file string) {
	a.BATCH([]string{http.MethodGet, http.MethodHead}, path, func(request *air.Request, response *air.Response) error {
		f, err := hfs.Open(file)
		if err != nil {
			var _t1 *fs.PathError
			if ok := errors.Is(err, _t1); ok {
				return a.NotFoundHandler(request, response)
			}

			return err
		}

		if response.Header.Get("Content-Type") == "" {
			response.Header.Set("Content-Type", mime.TypeByExtension(filepath.Ext(file)))
		}

		return response.Write(f)
	})
}
