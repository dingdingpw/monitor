package server

import "embed"

//go:embed agent_bins/*
var agentBinaries embed.FS
