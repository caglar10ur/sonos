package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	var (
		transportType = flag.String("transport", "stdio", "Transport type (stdio or http)")
		serverURL     = flag.String("url", "http://localhost:8888/mcp", "Server URL for HTTP transport")
		serverCmd     = flag.String("cmd", "./mcp-server", "Server command for stdio transport")
		serverArgs    = flag.String("args", "", "Server arguments for stdio transport (comma separated)")
		toolName      = flag.String("tool", "", "Tool name to call (if empty, list tools)")
		toolArgs      = flag.String("tool-args", "{}", "Tool arguments in JSON format")
		timeout       = flag.Duration("timeout", 30*time.Second, "Timeout for the operation")
	)
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	// 1. Initialize the MCP Client
	client := mcp.NewClient(
		&mcp.Implementation{
			Name:    "mcp-client-go",
			Version: "1.0.0",
		},
		nil,
	)

	// 2. Define the Transport
	var transport mcp.Transport
	var err error

	switch *transportType {
	case "stdio":
		args := []string{}
		if *serverArgs != "" {
			args = strings.Split(*serverArgs, ",")
		}
		cmd := exec.CommandContext(ctx, *serverCmd, args...)
		cmd.Stderr = os.Stderr // Pipe server errors to our stderr
		transport = &mcp.CommandTransport{
			Command: cmd,
		}
	case "http":
		transport = &mcp.StreamableClientTransport{
			Endpoint: *serverURL,
		}
	default:
		log.Fatalf("Unsupported transport type: %s", *transportType)
	}

	// 3. Connect to the server and establish a session
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	defer session.Close()

	if *toolName == "" {
		// 4. List available tools
		fmt.Println("Listing tools...")
		tools, err := session.ListTools(ctx, &mcp.ListToolsParams{})
		if err != nil {
			log.Fatalf("Failed to list tools: %v", err)
		}

		fmt.Printf("Found %d tools:\n", len(tools.Tools))
		for _, tool := range tools.Tools {
			fmt.Printf("- %s: %s\n", tool.Name, tool.Description)
			// Print schema if available
			schema, _ := json.MarshalIndent(tool.InputSchema, "  ", "  ")
			fmt.Printf("  Schema: %s\n", string(schema))
		}
	} else {
		// 5. Call a specific tool
		fmt.Printf("Calling tool '%s' with args %s...\n", *toolName, *toolArgs)

		var args map[string]interface{}
		if err := json.Unmarshal([]byte(*toolArgs), &args); err != nil {
			log.Fatalf("Failed to parse tool arguments: %v", err)
		}

		callParams := &mcp.CallToolParams{
			Name:      *toolName,
			Arguments: args,
		}

		result, err := session.CallTool(ctx, callParams)
		if err != nil {
			log.Fatalf("Tool call failed: %v", err)
		}

		// 6. Process the result
		fmt.Println("Result:")
		for _, content := range result.Content {
			switch c := content.(type) {
			case *mcp.TextContent:
				fmt.Printf("Text: %s\n", c.Text)
			case *mcp.ImageContent:
				fmt.Printf("Image (data URL): %s\n", c.Data)
			default:
				fmt.Printf("Unknown content type: %+v\n", c)
			}
		}
		if result.IsError {
			fmt.Println("Result indicated an error.")
			os.Exit(1)
		}
	}
}
