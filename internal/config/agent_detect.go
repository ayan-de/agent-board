package config

import (
	"fmt"
	"os/exec"
)

type DetectedAgent struct {
	Name   string
	Binary string
	Path   string
	Found  bool
}

type agentSpec struct {
	name   string
	binary string
}

var agentSpecs = []agentSpec{
	{name: "claude-code", binary: "claude"},
	{name: "opencode", binary: "opencode"},
	{name: "cursor", binary: "cursor"},
}

func DetectAgents() []DetectedAgent {
	agents := make([]DetectedAgent, len(agentSpecs))
	for i, spec := range agentSpecs {
		path, err := exec.LookPath(spec.binary)
		agents[i] = DetectedAgent{
			Name:   spec.name,
			Binary: spec.binary,
			Path:   path,
			Found:  err == nil,
		}
		if err == nil {
			agents[i].Path = fmt.Sprintf("%s (%s)", spec.binary, path)
		}
	}
	return agents
}
