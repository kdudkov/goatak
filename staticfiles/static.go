package staticfiles

import (
	"embed"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
)

//go:embed static
var staticFiles embed.FS

func Embed(f *fiber.App) {
	f.Use("/static", filesystem.New(filesystem.Config{
		Root:       http.FS(staticFiles),
		PathPrefix: "static",
	}))
}
