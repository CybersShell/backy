package backy

import (
	"bytes"
	"os"
	"path/filepath"
	"text/template"

	"gopkg.in/yaml.v3"
)

func LoadVarsYAML(path string) (map[string]interface{}, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	if err := yaml.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func RenderTemplateFile(templatePath string, vars map[string]interface{}) ([]byte, error) {
	tmplText, err := os.ReadFile(templatePath)
	if err != nil {
		return nil, err
	}
	funcs := template.FuncMap{
		"env": func(k, d string) string {
			if v := os.Getenv(k); v != "" {
				return v
			}
			return d
		},
		"default": func(def interface{}, v interface{}) interface{} {
			if v == nil {
				return def
			}
			if s, ok := v.(string); ok && s == "" {
				return def
			}
			return v
		},
		"toYaml": func(v interface{}) string {
			b, _ := yaml.Marshal(v)
			return string(b)
		},
	}
	t := template.New(filepath.Base(templatePath)).Funcs(funcs)
	t, err = t.Parse(string(tmplText))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, vars); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func WriteRenderedFile(templatePath string, vars map[string]interface{}, dest string, perm os.FileMode) error {
	out, err := RenderTemplateFile(templatePath, vars)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dest, out, perm)
}
