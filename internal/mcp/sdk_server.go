package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *Server) newSDKServer() *sdkmcp.Server {
	server := sdkmcp.NewServer(&sdkmcp.Implementation{
		Name:    "pipeops-mcp-server",
		Version: "1.1.0",
	}, nil)

	for _, definition := range s.toolDefinitions() {
		definition := definition
		annotations := annotationsForTool(definition.tool.Name)
		server.AddTool(&sdkmcp.Tool{
			Name:        definition.tool.Name,
			Description: definition.tool.Description,
			InputSchema: definition.tool.InputSchema,
			Annotations: &sdkmcp.ToolAnnotations{
				ReadOnlyHint:    annotations.ReadOnlyHint,
				DestructiveHint: annotations.DestructiveHint,
				IdempotentHint:  annotations.IdempotentHint,
				OpenWorldHint:   annotations.OpenWorldHint,
			},
		}, func(ctx context.Context, request *sdkmcp.CallToolRequest) (*sdkmcp.CallToolResult, error) {
			arguments := map[string]interface{}{}
			if request.Params != nil && len(request.Params.Arguments) > 0 {
				if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
					return toolErrorResult(fmt.Errorf("invalid parameters: %w", err)), nil
				}
			}

			result, err := definition.handler(ctx, arguments)
			if err != nil {
				return toolErrorResult(err), nil
			}
			return convertLegacyToolResult(result)
		})
	}

	return server
}

func convertLegacyToolResult(result interface{}) (*sdkmcp.CallToolResult, error) {
	encoded, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("encode tool result: %w", err)
	}

	var converted sdkmcp.CallToolResult
	if err := json.Unmarshal(encoded, &converted); err != nil {
		return nil, fmt.Errorf("convert tool result: %w", err)
	}
	if len(converted.Content) == 0 {
		converted.Content = []sdkmcp.Content{&sdkmcp.TextContent{Text: formatJSON(result)}}
	}
	return &converted, nil
}

func toolErrorResult(err error) *sdkmcp.CallToolResult {
	return &sdkmcp.CallToolResult{
		Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: err.Error()}},
		IsError: true,
	}
}
