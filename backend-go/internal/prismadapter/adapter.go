// Package prismadapter integrates the Prism image engine into MultiSell.
// It wraps the Prism pipeline for convenient use by MultiSell agents
// (ProductScout, ListingOptimizer) and provides a simple Go API.
package prismadapter

import (
	"context"
	"fmt"

	prism "github.com/multisell/prism/pkg/client"
)

// Request describes a product image generation request.
type Request struct {
	ImageURL string
	Template string
	Platform string
	Product  string
}

// Client wraps the Prism client for use by MultiSell agents.
type Client struct {
	client *prism.Client
}

// New creates a new Prism adapter client.
func New() *Client {
	return &Client{
		client: prism.New(),
	}
}

// GenerateProductImage runs the Prism pipeline and returns the generated image URL.
func (c *Client) GenerateProductImage(ctx context.Context, req *Request) (string, error) {
	if req == nil {
		return "", fmt.Errorf("prismadapter: request is nil")
	}
	result, err := c.client.Generate(ctx, prism.Request{
		ImageURL: req.ImageURL,
		Mode:     resolveMode(req.Template),
		Platform: req.Platform,
		Product:  req.Product,
	})
	if err != nil {
		return "", fmt.Errorf("prismadapter: %w", err)
	}
	return result.ImageURL, nil
}

// resolveMode maps template names to pipeline mode strings.
func resolveMode(template string) string {
	switch template {
	case "white_bg", "white-bg":
		return "white_bg"
	case "scene", "scenegen":
		return "scene"
	case "edit", "editorial":
		return "edit"
	default:
		return "scene"
	}
}
