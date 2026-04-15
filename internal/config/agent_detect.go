package config

import (
	"fmt"
	"os/exec"
)

type DetectedAgent struct {
	Name    string
	Binary  string
	Path    string
	Found   bool
	Logo    string
	LogoClr string
}

type agentSpec struct {
	name    string
	binary  string
	logo    string
	logoClr string
}

var agentSpecs = []agentSpec{
	{name: "claude-code", binary: "claude", logo: " ▐▛███▜▌\n▝▜█████▛▘\n  ▘▘ ▝▝", logoClr: "#D97757"},
	{name: "opencode", binary: "opencode", logo: " ██████\n██\n██\n ██████\n     ██\n     ██\n ██████\n", logoClr: "#7C3AED"},
	{name: "codex", binary: "codex", logo: " ██████\n██\n██  ████\n██    ██\n ██████\n", logoClr: "#10A37F"},
	{name: "cursor", binary: "cursor", logo: "████████\n  ██  ██\n  ██  ██\n  ██████\n  ██  ██\n  ██  ██\n████████\n", logoClr: "#F0DB4F"},
}

func DetectAgents() []DetectedAgent {
	agents := make([]DetectedAgent, len(agentSpecs))
	for i, spec := range agentSpecs {
		path, err := exec.LookPath(spec.binary)
		agents[i] = DetectedAgent{
			Name:    spec.name,
			Binary:  spec.binary,
			Path:    path,
			Found:   err == nil,
			Logo:    spec.logo,
			LogoClr: spec.logoClr,
		}
		if err == nil {
			agents[i].Path = fmt.Sprintf("%s (%s)", spec.binary, path)
		}
	}
	return agents
}
