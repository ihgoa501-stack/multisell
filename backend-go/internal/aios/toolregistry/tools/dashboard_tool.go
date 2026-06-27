package tools

import (
	"context"
	"time"

	"github.com/lingmirror/backend-go/internal/aios/toolregistry"
)

// DashboardTools returns the dashboard tool definitions.
func DashboardTools() []toolregistry.Tool {
	return []toolregistry.Tool{
		{
			Name:        "dashboard.overview",
			Version:     "1.0.0",
			Description: "驾驶舱总览——获取全局驾驶舱概述数据，包含系统状态和指标汇总",
			Parameters: &toolregistry.Schema{
				Type:        "object",
				Description: "仪表盘概览参数（可选）",
				Properties:  map[string]*toolregistry.Schema{},
			},
			Returns: &toolregistry.Schema{
				Type:        "object",
				Description: "仪表盘概览数据，包含信息、置信度等",
			},
			RequiredPermissions: []string{"dashboard:read:overview"},
			RiskLevel:           toolregistry.RiskLow,
			MaxDuration:         10 * time.Second,
			Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
				return map[string]interface{}{
					"message":    "请使用 /api/agents/dashboard 端点获取驾驶舱数据",
					"confidence": 0.0,
				}, nil
			},
		},
	}
}

func init() {
	tools := DashboardTools()
	for i := range tools {
		toolregistry.DefaultRegistry.Register(&tools[i])
	}
}
