package workspace

import (
	"encoding/json"
	"fmt"

	"bennypowers.dev/asimonim/lsp/internal/log"
	"bennypowers.dev/asimonim/lsp/types"
	"github.com/go-json-experiment/json/jsontext"
	"go.lsp.dev/protocol"
)

// DidChangeConfiguration handles the workspace/didChangeConfiguration notification
func DidChangeConfiguration(req *types.RequestContext, params *protocol.DidChangeConfigurationParams) error {
	log.Info("Configuration changed")

	// Parse the settings
	config, err := parseConfiguration(params.Settings)
	if err != nil {
		log.Info("Warning: failed to parse configuration: %v", err)
		return nil // Don't fail, just use defaults
	}

	// Update server configuration
	req.Server.SetConfig(config)

	log.Info("New configuration: %+v", config)

	// Reload tokens with new configuration
	if err := req.Server.LoadTokensFromConfig(); err != nil {
		log.Info("Warning: failed to reload tokens: %v", err)
	}

	// Refresh diagnostics for all open documents
	if req.Server.UsePullDiagnostics() {
		req.Server.NotifyDiagnosticRefresh()
	} else if req.Ctx != nil {
		for _, doc := range req.Server.AllDocuments() {
			if err := req.Server.PublishDiagnostics(req.Ctx, doc.URI()); err != nil {
				log.Info("Warning: failed to publish diagnostics for %s: %v", doc.URI(), err)
			}
		}
	}

	return nil
}

// parseConfiguration parses the configuration from the settings
func parseConfiguration(settings any) (types.ServerConfig, error) {
	// Default configuration
	config := types.DefaultConfig()

	if settings == nil {
		return config, nil
	}

	// Settings may arrive as jsontext.Value (raw JSON bytes) from the new
	// go.lsp.dev/protocol, or as map[string]any from tests / older callers.
	var settingsMap map[string]any
	switch v := settings.(type) {
	case jsontext.Value:
		if len(v) == 0 {
			return config, nil
		}
		if err := json.Unmarshal(v, &settingsMap); err != nil {
			return config, fmt.Errorf("settings is not a map")
		}
	case map[string]any:
		settingsMap = v
	default:
		return config, fmt.Errorf("settings is not a map")
	}

	// Look for our configuration: prefer "asimonim", fall back to legacy keys
	var ourSettings any
	if val, exists := settingsMap["asimonim"]; exists {
		ourSettings = val
	} else if val, exists := settingsMap["designTokensLanguageServer"]; exists {
		ourSettings = val
	} else if val, exists := settingsMap["design-tokens-language-server"]; exists {
		ourSettings = val
	} else {
		// No configuration provided, use defaults
		return config, nil
	}

	// Validate that the settings value is an object
	settingsObj, ok := ourSettings.(map[string]any)
	if !ok {
		return config, fmt.Errorf("configuration value must be an object, got %T", ourSettings)
	}

	// Convert to JSON and back to parse into struct
	jsonBytes, err := json.Marshal(settingsObj)
	if err != nil {
		return config, fmt.Errorf("failed to marshal settings: %w", err)
	}

	if err := json.Unmarshal(jsonBytes, &config); err != nil {
		return config, fmt.Errorf("failed to unmarshal settings: %w", err)
	}

	// Track whether groupMarkers was explicitly provided
	if _, hasGM := settingsObj["groupMarkers"]; hasGM {
		config.GroupMarkersSet = true
	}

	return config, nil
}
