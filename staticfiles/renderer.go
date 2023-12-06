package staticfiles

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"path/filepath"
	"strings"
	"time"
)

type Renderer struct {
	LeftDelimeter  string
	RightDelimeter string
	template       *template.Template
}

func (r *Renderer) Load(fs embed.FS) error {
	t := template.
		New("template").
		Delims(
			r.LeftDelimeter,
			r.RightDelimeter,
		).
		Funcs(template.FuncMap{
			"str2html": str2html,
			"strlen":   strlen,
			"substr":   substr,
			"timefmt":  timefmt,
		})

	if err := walkEmbed(fs, func(fs embed.FS, fname string) error {
		if strings.HasSuffix(fname, ".html") {
			b, err := fs.ReadFile(fname)
			if err != nil {
				return err
			}
			tplName := fname[len("templates")+1:]
			if _, err := t.New(tplName).Parse(string(b)); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return err
	}

	r.template = t

	return nil
}

func walkEmbed(fs embed.FS, fn func(fs embed.FS, fname string) error) error {
	dirs := []string{"."}
	i := 0

	for {
		if i >= len(dirs) {
			return nil
		}

		path := dirs[i]
		dir, err := fs.ReadDir(path)

		if err != nil {
			return err
		}

		for _, f := range dir {
			if f.IsDir() {
				dirs = append(dirs, filepath.ToSlash(filepath.Join(path, f.Name())))

				continue
			}

			if err := fn(fs, filepath.ToSlash(filepath.Join(path, f.Name()))); err != nil {
				return err
			}
		}
		i++
	}
}

func (r *Renderer) Render(m map[string]interface{}, templates ...string) (string, error) {
	buf := bytes.Buffer{}
	for _, name := range templates {
		if buf.Len() > 0 {
			if m == nil {
				m = make(map[string]interface{}, 1)
			}

			m["InheritedHTML"] = template.HTML(buf.String()) //nolint:gosec
		}

		buf.Reset()

		t := r.template.Lookup(name)
		if t == nil {
			return "", fmt.Errorf("undefined html template: %s", name)
		}

		if err := t.Execute(&buf, m); err != nil {
			return "", err
		}
	}

	return buf.String(), nil
}

// str2html returns a `template.HTML` for the s.
func str2html(s string) template.HTML {
	return template.HTML(s) //nolint:gosec
}

// strlen returns the number of characters of the s.
func strlen(s string) int {
	return len([]rune(s))
}

// substr returns the substring consisting of the characters of the s starting
// at the index i and continuing up to, but not including, the character at the
// index j.
func substr(s string, i, j int) string {
	return string([]rune(s)[i:j])
}

// timefmt returns a textual representation of the t formatted for the layout.
func timefmt(t time.Time, layout string) string {
	return t.Format(layout)
}
