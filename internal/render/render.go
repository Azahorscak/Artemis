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
// All fields are exported so text/template can access them by name (e.g. {{.GitCommit}}).
type TemplateCtx struct {
	GitCommit string            // full SHA-1 commit hash, or empty if unavailable
	GitBranch string            // branch name (e.g. "main"), or empty on detached HEAD
	GitDirty  bool              // true when the working tree has uncommitted changes
	Timestamp time.Time         // UTC build time; use {{.Timestamp.Format "2006-01-02"}} in templates
	Initiator string            // identity of whoever triggered the build
	Version   string            // binary version string set via ldflags
	Env       map[string]string // optional allowlisted env vars; nil means none pre-populated
}

// FuncMap returns the helper functions registered for templates.
// These are available in every .tmpl file via the standard {{call}} syntax.
func FuncMap() template.FuncMap {
	return template.FuncMap{
		// upper / lower: {{.GitBranch | upper}}
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		// default: returns def when val is empty — {{.Version | default "dev"}}
		"default": func(def, val string) string {
			if val == "" {
				return def
			}
			return val
		},
		// toYaml: marshals any value to a YAML string without a trailing newline.
		// Useful for embedding structured data in YAML templates.
		"toYaml": func(v any) (string, error) {
			b, err := yaml.Marshal(v)
			if err != nil {
				return "", err
			}
			return strings.TrimSuffix(string(b), "\n"), nil
		},
		// toJson: marshals any value to a compact JSON string.
		// Useful for embedding structured data in JSON or shell templates.
		"toJson": func(v any) (string, error) {
			b, err := json.Marshal(v)
			if err != nil {
				return "", err
			}
			return string(b), nil
		},
		// env: reads an environment variable at render time — {{env "HOME"}}
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
			// Strip the .tmpl extension so "foo.txt.tmpl" becomes "foo.txt" in the output.
			destPath = strings.TrimSuffix(destPath, ".tmpl")
			return renderTemplate(path, destPath, info.Mode(), ctx)
		}

		// Non-template files are copied verbatim (images, scripts, etc.).
		return copyFile(path, destPath, info.Mode())
	})
}

// renderTemplate parses src as a Go text/template and executes it with ctx,
// writing the result to dst with the original file mode preserved.
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

// copyFile copies src to dst byte-for-byte, preserving the original file mode.
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
