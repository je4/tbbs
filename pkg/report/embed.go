package report

import "embed"

//go:embed template/*
var TemplateFS embed.FS

//go:embed static/css/*
//go:embed static/thirdparty/pagedjs/css/*
//go:embed static/thirdparty/pagedjs/js/*
var StaticFS embed.FS
