package knowledge

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// KnowledgePattern defines a keyword-to-query mapping rule.
//
// When an incoming question matches one or more Keywords, the engine uses
// this pattern to determine which data source to query and how to formulate
// the query template.
type KnowledgePattern struct {
	// Keywords are the terms the engine matches against the question.
	Keywords []string `json:"keywords"`
	// SourceType identifies the business domain (inventory, order, etc.).
	SourceType string `json:"source_type"`
	// QueryTemplate is a SQL-like template: "SELECT * FROM {table} WHERE {condition}".
	QueryTemplate string `json:"query_template"`
	// Priority controls tie-breaking when multiple patterns match equally well.
	Priority int `json:"priority"`
}

// Engine is the v1 knowledge engine.
//
// It uses keyword matching and simple rules to route agent questions to the
// appropriate data sources. No vector database is required.
type Engine struct {
	mu       sync.RWMutex
	sources  []DataSource
	patterns []KnowledgePattern
	logger   *zap.Logger
}

// New creates a Knowledge Engine with the built-in set of knowledge patterns
// covering 8 business domains. If logger is nil, a no-op logger is used.
func New(logger *zap.Logger) *Engine {
	if logger == nil {
		logger = zap.NewNop()
	}

	e := &Engine{
		logger: logger,
	}

	e.patterns = builtinPatterns()
	return e
}

// builtinPatterns returns the default set of knowledge patterns covering all
// major business domains in the LingMirror platform.
func builtinPatterns() []KnowledgePattern {
	return []KnowledgePattern{
		{
			Keywords:      []string{"库存", "stock", "库存量", "warehouse", "仓", "存货", "inventory", "仓储", "库存状态", "库存水平", "库存数量"},
			SourceType:    "inventory",
			QueryTemplate: "SELECT * FROM inventory WHERE {condition}",
			Priority:      100,
		},
		{
			Keywords:      []string{"订单", "order", "订单量", "销售", "sale", "销量", "sales", "orders", "出单", "销售数据", "订单状态"},
			SourceType:    "order",
			QueryTemplate: "SELECT * FROM orders WHERE {condition}",
			Priority:      100,
		},
		{
			Keywords:      []string{"财务", "finance", "settlement", "结算", "收入", "revenue", "利润", "profit", "成本", "cost", "账单", "应收", "应付"},
			SourceType:    "settlement",
			QueryTemplate: "SELECT * FROM settlement WHERE {condition}",
			Priority:      100,
		},
		{
			Keywords:      []string{"SKU", "sku", "产品", "product", "商品", "listing", "SPU", "品牌", "brand", "规格", "listing信息"},
			SourceType:    "sku",
			QueryTemplate: "SELECT * FROM skus WHERE {condition}",
			Priority:      100,
		},
		{
			Keywords:      []string{"供应商", "supplier", "采购", "purchase", "vendor", "供货", "货源", "供应商信息", "供应"},
			SourceType:    "supplier",
			QueryTemplate: "SELECT * FROM suppliers WHERE {condition}",
			Priority:      100,
		},
		{
			Keywords:      []string{"物流", "shipping", "运输", "delivery", "快递", "配送", "签收", "发货", "物流状态", "配送信息"},
			SourceType:    "shipping",
			QueryTemplate: "SELECT * FROM shipping WHERE {condition}",
			Priority:      100,
		},
		{
			Keywords:      []string{"平台", "platform", "店铺", "渠道", "channel", "Lazada", "Shopee", "Ozon", "TikTok", "平台信息", "店铺状态"},
			SourceType:    "platform",
			QueryTemplate: "SELECT * FROM platforms WHERE {condition}",
			Priority:      100,
		},
		{
			Keywords:      []string{"售后", "aftersales", "退货", "refund", "退款", "客服", "service", "退换", "纠纷", "dispute", "售后处理"},
			SourceType:    "aftersales",
			QueryTemplate: "SELECT * FROM aftersales WHERE {condition}",
			Priority:      100,
		},
	}
}

// ---------------------------------------------------------------------------
// Answer templates per source type
// ---------------------------------------------------------------------------

var answerTemplates = map[string]string{
	"inventory":  "正在查询库存数据。已识别到与库存相关的关键词，将在 inventory 表中检索指定SKU或仓库的库存水平信息。",
	"order":      "正在查询订单数据。已识别到与订单相关的关键词，将在 orders 表中检索指定时间范围或订单编号的销售数据。",
	"settlement": "正在查询财务结算数据。已识别到与财务相关的关键词，将在 settlement 表中检索收入、利润或成本数据。",
	"sku":        "正在查询商品数据。已识别到与SKU/商品相关的关键词，将在 skus 表中检索产品规格、品牌和listing信息。",
	"supplier":   "正在查询供应商数据。已识别到与供应商相关的关键词，将在 suppliers 表中检索供应商和采购信息。",
	"shipping":   "正在查询物流数据。已识别到与物流相关的关键词，将在 shipping 表中检索配送状态和运输信息。",
	"platform":   "正在查询平台数据。已识别到与电商平台相关的关键词，将在 platforms 表中检索店铺和渠道信息。",
	"aftersales": "正在查询售后数据。已识别到与售后相关的关键词，将在 aftersales 表中检索退货、退款和客服数据。",
}

var sourceTableMap = map[string]string{
	"inventory":  "inventory",
	"order":      "orders",
	"settlement": "settlement",
	"sku":        "skus",
	"supplier":   "suppliers",
	"shipping":   "shipping",
	"platform":   "platforms",
	"aftersales": "aftersales",
}

// typeLabelMap returns the Chinese label for a source type (for answer text).
var typeLabelMap = map[string]string{
	"inventory":  "库存",
	"order":      "订单",
	"settlement": "财务结算",
	"sku":        "商品",
	"supplier":   "供应商",
	"shipping":   "物流",
	"platform":   "平台",
	"aftersales": "售后",
}

// Query answers a natural-language question using keyword pattern matching.
//
// It returns a structured KnowledgeResponse with the matched domain's data
// source configuration, confidence score, freshness metadata, and reasoning
// inferences. If no pattern matches, the response contains a clear "unable
// to answer" message with Confidence set to 0.
func (e *Engine) Query(ctx context.Context, q *KnowledgeQuery) (*KnowledgeResponse, error) {
	if q == nil || q.Question == "" {
		return nil, fmt.Errorf("knowledge: empty question")
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	e.logger.Info("knowledge query",
		zap.String("agent_id", q.AgentID),
		zap.String("question", q.Question),
	)

	// Sort patterns by priority descending so tie-breaking is deterministic.
	sorted := make([]KnowledgePattern, len(e.patterns))
	copy(sorted, e.patterns)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Priority > sorted[j].Priority
	})

	// Find the best-matching pattern.
	bestPattern, matchedCount := e.findBestPattern(sorted, q.Question)
	if bestPattern == nil {
		return e.noMatchResponse(q.Question), nil
	}

	return e.buildResponse(q, bestPattern, matchedCount)
}

// findBestPattern scans sorted patterns and returns the one with the most
// keyword matches, along with the match count. Ties are broken by priority
// (already reflected in the sort order).
func (e *Engine) findBestPattern(sorted []KnowledgePattern, question string) (*KnowledgePattern, int) {
	questionLower := strings.ToLower(question)
	var bestIdx int = -1
	var maxCount int

	for i, p := range sorted {
		count := countKeywordMatches(p.Keywords, question, questionLower)
		if count > maxCount {
			maxCount = count
			bestIdx = i
		}
	}

	if bestIdx < 0 || maxCount == 0 {
		return nil, 0
	}
	return &sorted[bestIdx], maxCount
}

// countKeywordMatches returns how many keywords from the list appear in the
// question (checking both the original string for CJK and the lowered string
// for English case-insensitive matching).
func countKeywordMatches(keywords []string, question, questionLower string) int {
	count := 0
	for _, kw := range keywords {
		kwLower := strings.ToLower(kw)
		if strings.Contains(questionLower, kwLower) || strings.Contains(question, kw) {
			count++
		}
	}
	return count
}

// noMatchResponse builds a response for a question the engine cannot answer.
func (e *Engine) noMatchResponse(question string) *KnowledgeResponse {
	return &KnowledgeResponse{
		Answer: fmt.Sprintf(
			"无法回答该问题：当前知识引擎无法理解「%s」。知识引擎目前支持以下业务领域：库存、订单、财务结算、商品(SKU)、供应商、物流、平台、售后。请使用上述业务相关的关键词重新描述您的问题。",
			question,
		),
		Confidence:  0,
		DataSources: []DataSource{},
		Freshness:   map[string]time.Time{},
		Inferences:  []string{"未匹配到任何已知知识模式"},
	}
}

// buildResponse constructs the KnowledgeResponse for a matched pattern.
func (e *Engine) buildResponse(q *KnowledgeQuery, pattern *KnowledgePattern, matchedCount int) (*KnowledgeResponse, error) {
	totalKeywords := len(pattern.Keywords)
	confidence := float64(matchedCount) / float64(totalKeywords)
	if confidence > 1.0 {
		confidence = 1.0
	}

	// Build the answer text.
	answer := answerText(pattern.SourceType, q.Question)

	// Gather relevant data sources.
	matchingSources := e.filterSourcesByType(pattern.SourceType)

	// Build freshness map.
	freshness := make(map[string]time.Time)
	for _, s := range matchingSources {
		freshness[s.Type] = s.LastSync
	}
	// Always include the matched type even if no source is registered,
	// with zero time to indicate no sync data.
	if _, ok := freshness[pattern.SourceType]; !ok {
		freshness[pattern.SourceType] = time.Time{}
	}

	// Build inferences.
	inferences := []string{
		fmt.Sprintf("知识模式匹配：命中 %d/%d 个关键词（置信度 %.0f%%）", matchedCount, totalKeywords, confidence*100),
		fmt.Sprintf("查询模板：%s", pattern.QueryTemplate),
		fmt.Sprintf("目标表：%s", sourceTableMap[pattern.SourceType]),
	}

	// Check MaxAge if requested.
	if q.MaxAge > 0 {
		if syncTime, _ := freshness[pattern.SourceType]; !syncTime.IsZero() {
			age := time.Since(syncTime)
			if age > q.MaxAge {
				inferences = append(inferences,
					fmt.Sprintf("数据时效超过 MaxAge 限制：当前时效 %v > 最大可接受 %v", age, q.MaxAge))
			} else {
				inferences = append(inferences,
					fmt.Sprintf("数据时效满足 MaxAge 要求：当前时效 %v ≤ 最大可接受 %v", age, q.MaxAge))
			}
		} else {
			inferences = append(inferences,
				fmt.Sprintf("无法校验数据时效：数据源 %s 未同步或未注册", pattern.SourceType))
		}
	}

	// Include context info if provided.
	if len(q.Context) > 0 {
		var ctxKeys []string
		for k := range q.Context {
			ctxKeys = append(ctxKeys, k)
		}
		inferences = append(inferences, fmt.Sprintf("附加上下文参数：%s", strings.Join(ctxKeys, ", ")))
	}

	return &KnowledgeResponse{
		Answer:      answer,
		Confidence:  confidence,
		DataSources: matchingSources,
		Freshness:   freshness,
		Inferences:  inferences,
	}, nil
}

// answerText returns a human-readable answer for the matched source type.
func answerText(sourceType, question string) string {
	tmpl, ok := answerTemplates[sourceType]
	if !ok {
		tmpl = "正在查询业务数据。已识别到相关关键词，将在对应表中检索信息。"
	}
	label := typeLabelMap[sourceType]
	if label == "" {
		label = sourceType
	}
	return fmt.Sprintf("【%s】%s 问题原文：%s", label, tmpl, question)
}

// filterSourcesByType returns all registered data sources matching the given type.
func (e *Engine) filterSourcesByType(sourceType string) []DataSource {
	result := make([]DataSource, 0)
	for _, s := range e.sources {
		if s.Type == sourceType {
			result = append(result, s)
		}
	}
	return result
}

// RegisterDataSource registers a domain module as a knowledge source.
func (e *Engine) RegisterDataSource(source DataSource) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.sources = append(e.sources, source)
	e.logger.Info("knowledge data source registered",
		zap.String("type", source.Type),
		zap.String("table", source.Table),
	)
}

// RegisterPattern registers a new knowledge pattern at runtime.
func (e *Engine) RegisterPattern(p KnowledgePattern) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.patterns = append(e.patterns, p)
	e.logger.Info("knowledge pattern registered",
		zap.String("source_type", p.SourceType),
		zap.Int("keyword_count", len(p.Keywords)),
		zap.Int("priority", p.Priority),
	)
}
