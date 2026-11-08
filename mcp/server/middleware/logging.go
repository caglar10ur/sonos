package loggingmiddleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// LoggingMiddleware creates an MCP middleware that logs method calls.
func LoggingMiddleware() mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(
			ctx context.Context,
			method string,
			req mcp.Request,
		) (mcp.Result, error) {
			start := time.Now()
			sessionID := req.GetSession().ID()

			// Call the actual handler.
			result, err := next(ctx, method, req)

			// Log details.
			duration := time.Since(start)

			if err != nil {
				slog.Error("MCP Call",
					"session_id", sessionID,
					"method", method,
					"duration", duration,
					"error", err,
				)
			} else {
				slog.Info("MCP Call",
					"session_id", sessionID,
					"method", method,
					"duration", duration,
				)
			}

			return result, err
		}
	}
}
