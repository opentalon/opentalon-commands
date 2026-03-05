package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/opentalon/opentalon/pkg/plugin"
)

func TestParseCommand(t *testing.T) {
	tests := []struct {
		text      string
		wantCmd   string
		wantRest  string
	}{
		{"/install skill org/repo", "install", "skill org/repo"},
		{"/show config", "show", "config"},
		{"/commands", "commands", ""},
		{"/help", "help", ""},
		{"/set prompt hello world", "set", "prompt hello world"},
		{"/clear", "clear", ""},
		{"/new", "new", ""},
		{"/INSTALL skill x", "install", "skill x"},
		{"/  install  skill x", "install", " skill x"}, // rest is untrimmed; handler trims when needed
		{"/unknown", "unknown", ""},
		{"not a command", "", "not a command"},
		{"", "", ""},
		{"/", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			cmd, rest := parseCommand(tt.text)
			if cmd != tt.wantCmd || rest != tt.wantRest {
				t.Errorf("parseCommand(%q) = %q, %q; want %q, %q", tt.text, cmd, rest, tt.wantCmd, tt.wantRest)
			}
		})
	}
}

func TestCommandsHandler_Execute(t *testing.T) {
	h := commandsHandler{}

	// Capabilities
	caps := h.Capabilities()
	if caps.Name != pluginName {
		t.Errorf("Capabilities().Name = %q; want %q", caps.Name, pluginName)
	}
	if len(caps.Actions) != 1 || caps.Actions[0].Name != actionPrepare {
		t.Errorf("Capabilities().Actions = %v; want single action %q", caps.Actions, actionPrepare)
	}

	// Unknown action
	resp := h.Execute(plugin.Request{ID: "1", Action: "other", Args: map[string]string{"text": "foo"}})
	if resp.Error == "" {
		t.Error("Execute(unknown action): expected Error")
	}

	// Pass-through: empty or non-command
	for _, text := range []string{"", "hello", "what is 2+2"} {
		resp := h.Execute(plugin.Request{ID: "1", Action: actionPrepare, Args: map[string]string{"text": text}})
		if resp.Error != "" {
			t.Errorf("Execute(%q): unexpected Error: %s", text, resp.Error)
		}
		if resp.Content != text {
			t.Errorf("Execute(%q): Content = %q; want %q (pass-through)", text, resp.Content, text)
		}
	}

	// /install skill
	resp = h.Execute(plugin.Request{ID: "1", Action: actionPrepare, Args: map[string]string{"text": "/install skill https://github.com/org/repo"}})
	assertPreparerInvoke(t, resp, "install_skill", map[string]string{"url": "https://github.com/org/repo"})

	resp = h.Execute(plugin.Request{ID: "1", Action: actionPrepare, Args: map[string]string{"text": "/install skill org/repo main"}})
	assertPreparerInvoke(t, resp, "install_skill", map[string]string{"url": "org/repo", "ref": "main"})

	resp = h.Execute(plugin.Request{ID: "1", Action: actionPrepare, Args: map[string]string{"text": "/install skill org/repo"}})
	assertPreparerInvoke(t, resp, "install_skill", map[string]string{"url": "org/repo"})

	resp = h.Execute(plugin.Request{ID: "1", Action: actionPrepare, Args: map[string]string{"text": "/install foo"}})
	assertPreparerMessage(t, resp, "Usage: /install skill")

	resp = h.Execute(plugin.Request{ID: "1", Action: actionPrepare, Args: map[string]string{"text": "/install"}})
	assertPreparerMessage(t, resp, "Usage: /install skill")

	// /show config
	resp = h.Execute(plugin.Request{ID: "1", Action: actionPrepare, Args: map[string]string{"text": "/show config"}})
	assertPreparerInvoke(t, resp, "show_config", nil)

	resp = h.Execute(plugin.Request{ID: "1", Action: actionPrepare, Args: map[string]string{"text": "/show other"}})
	assertPreparerMessage(t, resp, "Unknown command")

	// /commands and /help
	resp = h.Execute(plugin.Request{ID: "1", Action: actionPrepare, Args: map[string]string{"text": "/commands"}})
	assertPreparerInvoke(t, resp, "list_commands", nil)

	resp = h.Execute(plugin.Request{ID: "1", Action: actionPrepare, Args: map[string]string{"text": "/help"}})
	assertPreparerInvoke(t, resp, "list_commands", nil)

	// /set prompt
	resp = h.Execute(plugin.Request{ID: "1", Action: actionPrepare, Args: map[string]string{"text": "/set prompt Be concise."}})
	assertPreparerInvoke(t, resp, "set_prompt", map[string]string{"text": "Be concise."})

	resp = h.Execute(plugin.Request{ID: "1", Action: actionPrepare, Args: map[string]string{"text": "/set prompt multi\nline"}})
	assertPreparerInvoke(t, resp, "set_prompt", map[string]string{"text": "multi\nline"})

	resp = h.Execute(plugin.Request{ID: "1", Action: actionPrepare, Args: map[string]string{"text": "/set other"}})
	assertPreparerMessage(t, resp, "Usage: /set prompt")

	// /clear and /new
	resp = h.Execute(plugin.Request{ID: "1", Action: actionPrepare, Args: map[string]string{"text": "/clear"}})
	assertPreparerInvoke(t, resp, "clear_session", nil)

	resp = h.Execute(plugin.Request{ID: "1", Action: actionPrepare, Args: map[string]string{"text": "/new"}})
	assertPreparerInvoke(t, resp, "clear_session", nil)

	// Unknown command
	resp = h.Execute(plugin.Request{ID: "1", Action: actionPrepare, Args: map[string]string{"text": "/unknown"}})
	assertPreparerMessage(t, resp, "Unknown command: /unknown")

	resp = h.Execute(plugin.Request{ID: "1", Action: actionPrepare, Args: map[string]string{"text": "/"}})
	assertPreparerMessage(t, resp, "Unknown command")
}

func assertPreparerInvoke(t *testing.T, resp plugin.Response, action string, args map[string]string) {
	t.Helper()
	if resp.Error != "" {
		t.Fatalf("unexpected Error: %s", resp.Error)
	}
	var pr preparerResponse
	if err := json.Unmarshal([]byte(resp.Content), &pr); err != nil {
		t.Fatalf("Content is not valid preparer JSON: %v\ncontent: %s", err, resp.Content)
	}
	if pr.SendToLLM == nil || *pr.SendToLLM {
		t.Errorf("expected send_to_llm: false; got %v", pr.SendToLLM)
	}
	if len(pr.Invoke) != 1 {
		t.Fatalf("expected one invoke step; got %d", len(pr.Invoke))
	}
	step := pr.Invoke[0]
	if step.Plugin != opentalonPlugin {
		t.Errorf("invoke.plugin = %q; want %q", step.Plugin, opentalonPlugin)
	}
	if step.Action != action {
		t.Errorf("invoke.action = %q; want %q", step.Action, action)
	}
	for k, v := range args {
		if step.Args[k] != v {
			t.Errorf("invoke.args[%q] = %q; want %q", k, step.Args[k], v)
		}
	}
}

func assertPreparerMessage(t *testing.T, resp plugin.Response, substring string) {
	t.Helper()
	if resp.Error != "" {
		t.Fatalf("unexpected Error: %s", resp.Error)
	}
	var pr preparerResponse
	if err := json.Unmarshal([]byte(resp.Content), &pr); err != nil {
		t.Fatalf("Content is not valid preparer JSON: %v\ncontent: %s", err, resp.Content)
	}
	if pr.SendToLLM == nil || *pr.SendToLLM {
		t.Errorf("expected send_to_llm: false; got %v", pr.SendToLLM)
	}
	if substring != "" && (len(pr.Message) == 0 || !strings.Contains(pr.Message, substring)) {
		t.Errorf("expected message containing %q; got %q", substring, pr.Message)
	}
}
