// Package render walks a templates directory and renders Go templates.
package render

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"gopkg.in/yaml.v3"
)

// TemplateCtx holds the data available to every template.
type TemplateCtx struct {
	GitCommit string
	GitBranch string
	GitDirty  bool
	Timestamp time.Time
	Initiator string
	Version   string
	Env       map[string]string
}

// FuncMap returns the helper functions registered for templates.
func FuncMap() template.FuncMap {
	return template.FuncMap{
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"default": func(def, val string) string {
			if val == "" {
				return def
			}
			return val
		},
		"toYaml": func(v any) (string, error) {
			b, err := yaml.Marshal(v)
			if err != nil {
				return "", err
			}
			return strings.TrimSuffix(string(b), "\n"), nil
		},
		"toJson": func(v any) (string, error) {
			b, err := json.Marshal(v)
			if err != nil {
				return "", err
			}
			return string(b), nil
		},
		"env": os.Getenv,
	}
}

// Render walks templatesDir, renders .tmpl files against ctx into outputDir,
// and copies non-.tmpl files verbatim. File modes are preserved.
func Render(templatesDir, outputDir string, ctx TemplateCtx) error {
	return filepath.WalkDir(templatesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(templatesDir, path)
		if err != nil {
			return fmt.Errorf("computing relative path: %w", err)
		}

		destPath := filepath.Join(outputDir, rel)

		if d.IsDir() {
			return os.MkdirAll(destPath, 0o755)
		}

		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("stat %s: %w", path, err)
		}

		if strings.HasSuffix(path, ".tmpl") {
			destPath = strings.TrimSuffix(destPath, ".tmpl")
			return renderTemplate(path, destPath, info.Mode(), ctx)
		}

		return copyFile(path, destPath, info.Mode())
	})
}

func renderTemplate(src, dst string, mode fs.FileMode, ctx TemplateCtx) error {
	tmpl, err := template.New(filepath.Base(src)).Funcs(FuncMap()).ParseFiles(src)
	if err != nil {
		return fmt.Errorf("parsing template %s: %w", src, err)
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("creating parent dir for %s: %w", dst, err)
	}

	f, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("creating %s: %w", dst, err)
	}
	defer f.Close()

	if err := tmpl.ExecuteTemplate(f, filepath.Base(src), ctx); err != nil {
		return fmt.Errorf("executing template %s: %w", src, err)
	}

	return nil
}

func copyFile(src, dst string, mode fs.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening %s: %w", src, err)
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("creating parent dir for %s: %w", dst, err)
	}

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("creating %s: %w", dst, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copying %s to %s: %w", src, dst, err)
	}

	return nil
}
