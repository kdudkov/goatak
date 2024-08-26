package staticfiles

import (
	"embed"
	"net/http"
	"net/url"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
)

//go:embed static
var staticFiles embed.FS

//go:embed webtak
var webtakFiles embed.FS

type PathUnescapeFs struct {
	fs http.FileSystem
}

func (c *PathUnescapeFs) Open(name string) (http.File, error) {
	// 解码路径中的空格
	decodedName, err := url.PathUnescape(name)
	if err != nil {
		return nil, err
	}
	// 使用解码后的路径打开文件
	return c.fs.Open(decodedName)
}

func Embed(f *fiber.App) {
	f.Use("/static", filesystem.New(filesystem.Config{
		Root:       http.FS(staticFiles),
		PathPrefix: "static",
	}))
}

func EmbedWebTak(f *fiber.App) {
	f.Use("/webtak", filesystem.New(filesystem.Config{
		Root:       &PathUnescapeFs{fs: http.FS(webtakFiles)},
		PathPrefix: "webtak",
	}))
}
