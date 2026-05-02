package server

import "embed"

//go:embed web/dist/*
var staticFiles embed.FS
