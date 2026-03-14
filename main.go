package main

import (
	"encoding/json"
	"log"
	"strings"

	"github.com/opentalon/opentalon/pkg/plugin"
)

const (
	pluginName        = "opentalon-commands"
	actionPrepare     = "prepare"
	argKeyText        = "text"
	opentalonPlugin   = "opentalon"
	actionInstallSkill = "install_skill"
	actionShowConfig   = "show_config"
	actionListCommands = "list_commands"
	actionSetPrompt    = "set_prompt"
	actionClearSession = "clear_session"
	actionReloadMCP    = "reload_mcp"
)

// preparerResponse is the JSON shape returned when the message is a command (send_to_llm: false).
type preparerResponse struct {
	SendToLLM *bool       `json:"send_to_llm"`
	Message   string      `json:"message,omitempty"`
	Invoke    []invokeStep `json:"invoke,omitempty"`
}

type invokeStep struct {
	Plugin string            `json:"plugin"`
	Action string            `json:"action"`
	Args   map[string]string `json:"args"`
}

func main() {
	h := &commandsHandler{}
	if err := plugin.Serve(h); err != nil {
		log.Fatal(err)
	}
}

type commandsHandler struct{}

func (commandsHandler) Capabilities() plugin.CapabilitiesMsg {
	return plugin.CapabilitiesMsg{
		Name:        pluginName,
		Description: "Parses slash commands (/install, /show config, /commands, /set prompt, /clear, /reload mcp) and returns invoke or message for the core.",
		Actions: []plugin.ActionMsg{
			{Name: actionPrepare, Description: "Parse user message; if it starts with /, return send_to_llm: false and invoke or message.", Parameters: []plugin.ParameterMsg{{Name: argKeyText, Description: "User message content", Type: "string", Required: true}}},
		},
	}
}

func (commandsHandler) Execute(req plugin.Request) plugin.Response {
	if req.Action != actionPrepare {
		return plugin.Response{CallID: req.ID, Error: "unknown action: " + req.Action}
	}
	text := strings.TrimSpace(req.Args[argKeyText])
	if text == "" || !strings.HasPrefix(text, "/") {
		return plugin.Response{CallID: req.ID, Content: text}
	}

	// Parse slash command
	cmd, rest := parseCommand(text)
	switch cmd {
	case "install":
		// /install skill <url> [ref]
		rest = strings.TrimSpace(rest)
		if !strings.HasPrefix(strings.ToLower(rest), "skill ") {
			return plugin.Response{CallID: req.ID, Content: preparerJSON(false, "Usage: /install skill <url> [ref]", nil)}
		}
		parts := strings.Fields(rest[6:])
		if len(parts) < 1 {
			return plugin.Response{CallID: req.ID, Content: preparerJSON(false, "Usage: /install skill <url> [ref]", nil)}
		}
		args := map[string]string{"url": parts[0]}
		if len(parts) >= 2 {
			args["ref"] = parts[1]
		}
		return plugin.Response{CallID: req.ID, Content: preparerJSON(false, "", []invokeStep{{Plugin: opentalonPlugin, Action: actionInstallSkill, Args: args}})}
	case "show":
		if strings.TrimSpace(strings.ToLower(rest)) == "config" {
			return plugin.Response{CallID: req.ID, Content: preparerJSON(false, "", []invokeStep{{Plugin: opentalonPlugin, Action: actionShowConfig, Args: map[string]string{}}})}
		}
		return plugin.Response{CallID: req.ID, Content: preparerJSON(false, "Unknown command. Try /show config or /commands.", nil)}
	case "commands", "help":
		return plugin.Response{CallID: req.ID, Content: preparerJSON(false, "", []invokeStep{{Plugin: opentalonPlugin, Action: actionListCommands, Args: map[string]string{}}})}
	case "set":
		// /set prompt <text>
		rest = strings.TrimSpace(rest)
		if !strings.HasPrefix(strings.ToLower(rest), "prompt ") {
			return plugin.Response{CallID: req.ID, Content: preparerJSON(false, "Usage: /set prompt <text>", nil)}
		}
		promptText := strings.TrimSpace(rest[7:])
		return plugin.Response{CallID: req.ID, Content: preparerJSON(false, "", []invokeStep{{Plugin: opentalonPlugin, Action: actionSetPrompt, Args: map[string]string{"text": promptText}}})}
	case "clear", "new":
		return plugin.Response{CallID: req.ID, Content: preparerJSON(false, "", []invokeStep{{Plugin: opentalonPlugin, Action: actionClearSession, Args: map[string]string{}}})}
	case "reload":
		// /reload mcp [server]
		rest = strings.TrimSpace(rest)
		sub, server := parseCommand("/" + rest)
		if sub != "mcp" {
			return plugin.Response{CallID: req.ID, Content: preparerJSON(false, "Usage: /reload mcp [server]", nil)}
		}
		args := map[string]string{"server": strings.TrimSpace(server)}
		return plugin.Response{CallID: req.ID, Content: preparerJSON(false, "", []invokeStep{{Plugin: opentalonPlugin, Action: actionReloadMCP, Args: args}})}
	case "":
		return plugin.Response{CallID: req.ID, Content: preparerJSON(false, "Unknown command. Try /commands.", nil)}
	default:
		return plugin.Response{CallID: req.ID, Content: preparerJSON(false, "Unknown command: /"+cmd+". Try /commands.", nil)}
	}
}

// parseCommand returns the first word (command) and the rest of the line after the leading /.
func parseCommand(text string) (cmd, rest string) {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "/") {
		return "", text
	}
	text = strings.TrimSpace(text[1:])
	i := strings.IndexFunc(text, func(r rune) bool { return r == ' ' || r == '\t' })
	if i < 0 {
		return strings.ToLower(text), ""
	}
	return strings.ToLower(text[:i]), text[i+1:]
}

func preparerJSON(sendToLLM bool, message string, invoke []invokeStep) string {
	s := sendToLLM
	resp := preparerResponse{SendToLLM: &s, Message: message, Invoke: invoke}
	data, _ := json.Marshal(resp)
	return string(data)
}

