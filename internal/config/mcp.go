package config

type MCPServerConfig struct {
	Enabled bool
	Command string
	Args    []string
}

type MCPConfig struct {
	NPMPath  string
	NodePath string
	Servers  map[string]MCPServerConfig
}

func (m *MCPConfig) UnmarshalTOML(data interface{}) error {
	raw, ok := data.(map[string]interface{})
	if !ok {
		return nil
	}
	m.Servers = make(map[string]MCPServerConfig)

	for k, v := range raw {
		switch k {
		case "npm_path":
			if str, ok := v.(string); ok {
				m.NPMPath = str
			}
		case "node_path":
			if str, ok := v.(string); ok {
				m.NodePath = str
			}
		default:
			if toolMap, ok := v.(map[string]interface{}); ok {
				var server MCPServerConfig
				if enabled, ok := toolMap["enabled"].(bool); ok {
					server.Enabled = enabled
				}
				if cmd, ok := toolMap["command"].(string); ok {
					server.Command = cmd
				}
				if argsIfc, ok := toolMap["args"].([]interface{}); ok {
					var args []string
					for _, a := range argsIfc {
						if aStr, ok := a.(string); ok {
							args = append(args, aStr)
						}
					}
					server.Args = args
				}
				m.Servers[k] = server
			}
		}
	}
	return nil
}
