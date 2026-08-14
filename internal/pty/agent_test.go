package pty

import (
	"reflect"
	"testing"
)

func TestFreeCodeReadyPatternMatchesBanner(t *testing.T) {
	cfg := NewFreeCode()
	readyRe := cfg.ReadyPattern

	banner := StripANSI("                       >_ FreeCode (v0.24.6)                      ")
	if !readyRe.MatchString(banner) {
		t.Fatal("FreeCode banner subtitle should mark the agent as ready")
	}

	if readyRe.MatchString("I loaded the FreeCode config from disk") {
		t.Fatal("Ready pattern is too loose — matched free-form prose")
	}
}

func TestFreeCodeIsInteractiveTUI(t *testing.T) {
	cfg := NewFreeCode()

	for _, a := range cfg.Args {
		if a == "run" {
			t.Fatal("FreeCode must open its TUI, not the headless 'run' subcommand")
		}
	}

	if cfg.SendPrompt == nil {
		t.Fatal("FreeCode needs SendPrompt set to type the prompt into the TUI editor")
	}

	if reflect.ValueOf(cfg.SendPrompt).Pointer() != reflect.ValueOf(SendPromptSingleLine).Pointer() {
		t.Fatalf("expected SendPromptSingleLine, got %T", cfg.SendPrompt)
	}

	if cfg.Bin != "freecode" {
		t.Fatalf("expected Bin=freecode, got %q", cfg.Bin)
	}
}

func TestFreeCodeIdlePatternMatchesIdlePrompt(t *testing.T) {
	cfg := NewFreeCode()
	if len(cfg.IdlePatterns) == 0 {
		t.Fatal("FreeCode needs IdlePatterns to detect completion")
	}

	emptyPrompt := StripANSI("❯ ")
	matched := false
	for _, re := range cfg.IdlePatterns {
		if re.MatchString(emptyPrompt) {
			matched = true
			break
		}
	}
	if !matched {
		t.Fatalf("no idle pattern matched the empty TUI prompt %q", emptyPrompt)
	}
}

func TestFreeCodeIsRegistered(t *testing.T) {
	cfg := GetConfig("freecode")
	if cfg == nil {
		t.Fatal("FreeCode config not registered under key \"freecode\"")
	}
	if cfg.Bin != "freecode" {
		t.Fatalf("registered freecode config has Bin=%q, want \"freecode\"", cfg.Bin)
	}
}