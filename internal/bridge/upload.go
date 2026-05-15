package bridge

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const maxProfileImageUploadBytes = 8 << 20

type toolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

func preparePayload(payload []byte) ([]byte, error) {
	var req rpcRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, err
	}
	if req.Method != "tools/call" || len(req.Params) == 0 {
		return payload, nil
	}

	var params toolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, err
	}
	if params.Name != "upload_profile_image" || len(params.Arguments) == 0 {
		return payload, nil
	}

	arguments := map[string]any{}
	if err := json.Unmarshal(params.Arguments, &arguments); err != nil {
		return nil, err
	}
	imagePath, _ := arguments["imagePath"].(string)
	imagePath = strings.TrimSpace(imagePath)
	if imagePath == "" {
		return payload, nil
	}
	if imageBase64, _ := arguments["imageBase64"].(string); strings.TrimSpace(imageBase64) != "" {
		return nil, errors.New("upload_profile_image accepts imagePath or imageBase64, not both")
	}
	imagePath, err := expandLocalImagePath(imagePath)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(imagePath)
	if err != nil {
		return nil, fmt.Errorf("read imagePath: %w", err)
	}
	if info.IsDir() {
		return nil, errors.New("imagePath must point to a file")
	}
	if info.Size() == 0 {
		return nil, errors.New("imagePath file is empty")
	}
	if info.Size() > maxProfileImageUploadBytes {
		return nil, errors.New("imagePath file must be under 8MB")
	}

	content, err := os.ReadFile(imagePath)
	if err != nil {
		return nil, fmt.Errorf("read imagePath: %w", err)
	}
	if len(content) == 0 {
		return nil, errors.New("imagePath file is empty")
	}
	if len(content) > maxProfileImageUploadBytes {
		return nil, errors.New("imagePath file must be under 8MB")
	}

	arguments["imageBase64"] = base64.StdEncoding.EncodeToString(content)
	delete(arguments, "imagePath")

	nextArguments, err := json.Marshal(arguments)
	if err != nil {
		return nil, err
	}
	params.Arguments = nextArguments

	nextParams, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	req.Params = nextParams

	return json.Marshal(req)
}

func expandLocalImagePath(imagePath string) (string, error) {
	if imagePath == "~" || strings.HasPrefix(imagePath, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("expand imagePath: %w", err)
		}
		if imagePath == "~" {
			return home, nil
		}
		return filepath.Join(home, strings.TrimPrefix(imagePath, "~/")), nil
	}
	return imagePath, nil
}

func prepareResponse(method string, payload []byte) []byte {
	if method != "tools/list" {
		return payload
	}

	var response map[string]any
	if err := json.Unmarshal(payload, &response); err != nil {
		return payload
	}
	result, ok := response["result"].(map[string]any)
	if !ok {
		return payload
	}
	tools, ok := result["tools"].([]any)
	if !ok {
		return payload
	}

	changed := false
	for _, item := range tools {
		tool, ok := item.(map[string]any)
		if !ok || tool["name"] != "upload_profile_image" {
			continue
		}
		tool["description"] = "Upload profile or header image bytes, or a local imagePath when using this bridge, and return a public image URL for a review-only profile edit proposal."
		schema, ok := tool["inputSchema"].(map[string]any)
		if !ok {
			continue
		}
		schema["required"] = []any{"kind"}
		properties, ok := schema["properties"].(map[string]any)
		if !ok {
			properties = map[string]any{}
			schema["properties"] = properties
		}
		properties["imageBase64"] = map[string]any{
			"type":        "string",
			"description": "Raw base64 image bytes, or a data:image/...;base64 URL. Maximum decoded size is 8MB.",
		}
		properties["imagePath"] = map[string]any{
			"type":        "string",
			"description": "Local image file path readable by profilescribe-mcp. The bridge converts it to imageBase64 before forwarding. Maximum file size is 8MB.",
		}
		changed = true
	}
	if !changed {
		return payload
	}

	nextPayload, err := json.Marshal(response)
	if err != nil {
		return payload
	}
	return nextPayload
}
