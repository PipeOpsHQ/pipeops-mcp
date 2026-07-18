package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

const pipeOpsMCPIcon = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAABAAAAAQCAYAAAAf8/9hAAAABHNCSVQICAgIfAhkiAAAApNJREFUOE+Vk11IU2EYx//v2Tm6Tbe2zGXY4lhIrdKivLDIml4EfegkIREMFnTTRRooFAV+EEE1ljoICkMlQgr60D5AuilLKKqLRWF24XGuDb9GW5vaMdlO5z2yoZs3PTfvy/M+/9/zPO/zvgT/aZIk8VRCCPEo62r6Sn7GGoO0Sz42AFKIAdf/8Id2D8MwHSzLbqSaSCTy0e/3H10BKOcDlQyibTJeyRK30ioO9Q6ZlWRer7clAajgp9sJQX1yUJ6FxZVePYTRcdy6+Rh/Fn7D1dkIg8EAn8/XoQDK+Wk7Q9CdLM7QEThf6JFjTkPd2SYcLLiIufkQsvJHcNxWBkEQzisAW950SF7W0P2F2xpodZLC2rJDg0w9q+z9P4NwXuuCSgU4XA0QRXFSo9FsIMuz03LbXhpXu9cVPveHQFRtCJRYLJb3pIKfapFH0kwjjtk5WKsIBnrDOHfVnAKaDS+i2ylgoEeFNGadsc9jDJFLtULn1yH1GYaw2LQtCr83iOa7ZhQWZ64AfHcHcbnWj9j8esVPJFLa58l+Q570CO9Ece7A20daTIzqkJUjwd46i2KrGRzH4dfMAtZmp6Ph5DBGP2cnoETiliq475ysLquJPaAnn16HsX1vBsXj3vUgNHoVDh0xY/9hPU7s/JbILkn48sxj2q1UogiHxsTczenpcXzTqSDmZ0zyRLQo3KcF7b26YBycShkUYhJOP/eYehKArhtjNSU2VW+GjoUwEoarTouFOTXyi8IQhv9CjHAJsZx9UM5ujSdLvMQlCCNDONxpnYL7VW7KFOSP1M8gzU57TwFQh6NxJG9rUXQwMBHLedqRxdHJUKM9y0+rPV72cvKqv5EGVPJBHlg09HlM7pRSljn+AXg26gplppK9AAAAAElFTkSuQmCC"

func (s *Server) newSDKServer() *sdkmcp.Server {
	server := sdkmcp.NewServer(&sdkmcp.Implementation{
		Name:       "pipeops-mcp-server",
		Title:      "PipeOps",
		Version:    "1.1.0",
		WebsiteURL: "https://pipeops.io",
		Icons: []sdkmcp.Icon{{
			Source:   pipeOpsMCPIcon,
			MIMEType: "image/png",
			Sizes:    []string{"16x16"},
		}},
	}, nil)

	for _, definition := range s.toolDefinitions() {
		definition := definition
		if !s.toolAllowed(definition.tool.Name) {
			continue
		}
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
