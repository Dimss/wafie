package assets

import (
	"bytes"
	"embed"
	"text/template"
)

//go:embed assets/*
var assetsFs embed.FS

type Assets struct {
	AssetsFs embed.FS
}

func NewAssets() *Assets {
	return &Assets{
		AssetsFs: assetsFs,
	}
}

func (a *Assets) RenderVirtualHost(data interface{}) (string, error) {
	content, err := a.AssetsFs.ReadFile("assets/virtualhost.tpl")
	if err != nil {
		return "", err
	}
	var tpl bytes.Buffer
	tmpl, err := template.
		New("virtualhost").
		Option("missingkey=error").
		Parse(string(content))
	if err != nil {
		return "", err
	}
	if err := tmpl.Execute(&tpl, data); err != nil {
		return "", err
	}
	return tpl.String(), nil

}
