package profit

import "math"

// EvidenceCard is a read model for Owner decision-making.
// It shows cost breakdown, revenue, profit projection with confidence levels.
type EvidenceCard struct {
	ProductID            int64       `json:"product_id"`
	Title                string      `json:"title"`
	Currency             string      `json:"currency"`

	Revenue              MoneyRow    `json:"revenue"`
	CostItems            []CostItem  `json:"cost_items"`
	TotalFixedCost       float64     `json:"total_fixed_cost"`
	TotalVariableFeeRate float64     `json:"total_variable_fee_rate"`
	EstimatedVariableFee float64     `json:"estimated_variable_fee"`
	TotalCostAtTargetPrice float64   `json:"total_cost_at_target_price"`

	EstimatedProfit      float64     `json:"estimated_profit"`
	ProfitMargin         float64     `json:"profit_margin"`
	Status               string      `json:"status"` // profitable / marginal / unprofitable / unknown

	ConfidenceLevel      string      `json:"confidence_level"` // high / medium / low / insufficient_data
	CanEvaluate          bool        `json:"can_evaluate"`
	ConfirmedItems       []DataField `json:"confirmed_items"`
	EstimatedItems       []DataField `json:"estimated_items"`
	MissingItems         []string    `json:"missing_items"`
	BlockingReasons      []string    `json:"blocking_reasons"`

	BreakEvenPrice       float64     `json:"break_even_price"`
	RecommendedMinPrice  float64     `json:"recommended_min_price"`
	TargetMargin         float64     `json:"target_margin"`
	BufferRate           float64     `json:"buffer_rate"`
}

type CostItem struct {
	Category        string  `json:"category"`
	Label           string  `json:"label"`
	Amount          float64 `json:"amount"`
	Rate            float64 `json:"rate"`
	CalculationType string  `json:"calculation_type"` // fixed_amount | percent_of_revenue
	DataSource      string  `json:"data_source"`       // confirmed | estimated | template_default | missing
	SourceNote      string  `json:"source_note"`
	Required        bool    `json:"required"`
}

type MoneyRow struct {
	Amount float64 `json:"amount"`
	Label  string  `json:"label"`
}

type DataField struct {
	FieldName string `json:"field_name"`
	Label     string `json:"label"`
	Value     string `json:"value"`
	Source    string `json:"source"` // confirmed | estimated | missing
}

// candidateProductReader is the subset of candidate.CandidateProduct the evidence
// card needs. Keeps the profit package from depending on the full candidate package.
type candidateProductReader struct {
	Title              string
	TargetSalePrice    float64
	TargetCurrency     string
	PurchasePrice      float64
	PurchaseCurrency   string
	PackageWeightKg    float64
	PackageLengthCm    float64
	PackageWidthCm     float64
	PackageHeightCm    float64
	OriginCountry      string
	DestinationCountry string
	HSCode             string
	SourceURL          string
}

// default constants — ponytail: hardcoded reasonable defaults, not config-driven.
const (
	defaultPaymentFeeRate        = 0.035  // 3.5% payment processing
	defaultPackagingFee          = 0.50   // $0.50 per unit
	defaultExchangeRateBuffer    = 0.02   // 2% of purchase cost
	defaultLossBuffer            = 0.01   // 1% of target revenue
	defaultDomesticShippingCN    = 1.50   // $1.50 domestic shipping from CN
	defaultTargetMargin          = 0.20   // 20% target profit margin
	defaultBufferRate            = 0.05   // 5% additional buffer
)

// BuildEvidenceCard computes the full evidence card for a candidate product.
func BuildEvidenceCard(prod *candidateProductReader, platformCommissionRate, internationalShippingCost float64) *EvidenceCard {
	card := &EvidenceCard{
		ProductID:    0,
		Title:        prod.Title,
		Currency:     prod.TargetCurrency,
		TargetMargin: defaultTargetMargin,
		BufferRate:   defaultBufferRate,
	}

	// 1. Revenue
	card.Revenue = MoneyRow{
		Amount: prod.TargetSalePrice,
		Label:  "目标售价",
	}

	// 2. Build cost items
	var costItems []CostItem
	var totalFixed float64
	var totalVarRate float64

	// Purchase cost
	purchaseItem := CostItem{
		Category:        "purchase_cost",
		Label:           "采购成本",
		Amount:          prod.PurchasePrice,
		CalculationType: "fixed_amount",
		Required:        true,
	}
	if prod.PurchasePrice > 0 {
		purchaseItem.DataSource = "confirmed"
		purchaseItem.SourceNote = "供应商报价"
	} else {
		purchaseItem.DataSource = "missing"
	}
	costItems = append(costItems, purchaseItem)
	totalFixed += purchaseItem.Amount

	// Domestic shipping
	domesticItem := CostItem{
		Category:        "domestic_shipping",
		Label:           "国内运费",
		CalculationType: "fixed_amount",
		Required:        false,
		Amount:          defaultDomesticShippingCN,
		DataSource:      "template_default",
		SourceNote:      "按模板默认值",
	}
	costItems = append(costItems, domesticItem)
	totalFixed += domesticItem.Amount

	// International shipping
	shipItem := CostItem{
		Category:        "international_shipping",
		Label:           "国际物流费",
		CalculationType: "fixed_amount",
		Required:        true,
	}
	if internationalShippingCost > 0 {
		shipItem.Amount = internationalShippingCost
		shipItem.DataSource = "estimated"
		shipItem.SourceNote = "按重量估算"
	} else {
		shipItem.DataSource = "missing"
	}
	costItems = append(costItems, shipItem)
	totalFixed += shipItem.Amount

	// Platform commission
	commItem := CostItem{
		Category:        "platform_commission",
		Label:           "平台佣金",
		CalculationType: "percent_of_revenue",
		Required:        true,
	}
	if platformCommissionRate > 0 {
		commItem.Rate = platformCommissionRate
		commItem.DataSource = "estimated"
		commItem.SourceNote = "按平台费率表"
	} else {
		commItem.DataSource = "missing"
	}
	costItems = append(costItems, commItem)
	totalVarRate += commItem.Rate

	// Payment fee
	payItem := CostItem{
		Category:        "payment_fee",
		Label:           "支付手续费",
		CalculationType: "percent_of_revenue",
		Required:        false,
		Rate:            defaultPaymentFeeRate,
		Amount:          round(prod.TargetSalePrice * defaultPaymentFeeRate, 2),
		DataSource:      "template_default",
		SourceNote:      "按模板默认值(3.5%)",
	}
	costItems = append(costItems, payItem)
	totalVarRate += payItem.Rate

	// Packaging fee
	packItem := CostItem{
		Category:        "packaging_fee",
		Label:           "包材费",
		CalculationType: "fixed_amount",
		Required:        false,
		Amount:          defaultPackagingFee,
		DataSource:      "template_default",
		SourceNote:      "按模板默认值",
	}
	costItems = append(costItems, packItem)
	totalFixed += packItem.Amount

	// Tariff
	tariffItem := CostItem{
		Category:        "tariff",
		Label:           "关税",
		CalculationType: "fixed_amount",
		Required:        false,
		DataSource:      "missing",
		SourceNote:      "未配置关税率",
	}
	costItems = append(costItems, tariffItem)

	// Exchange rate buffer (risk buffer, not actual cost)
	bufferItem := CostItem{
		Category:        "exchange_rate_buffer",
		Label:           "汇率缓冲",
		CalculationType: "fixed_amount",
		Required:        false,
		Amount:          round(prod.PurchasePrice*defaultExchangeRateBuffer, 2),
		DataSource:      "template_default",
		SourceNote:      "按模板默认值(2%)",
	}
	costItems = append(costItems, bufferItem)
	totalFixed += bufferItem.Amount

	// Loss buffer (risk buffer, not actual cost)
	lossItem := CostItem{
		Category:        "loss_buffer",
		Label:           "损耗缓冲",
		CalculationType: "fixed_amount",
		Required:        false,
		Amount:          round(prod.TargetSalePrice*defaultLossBuffer, 2),
		DataSource:      "template_default",
		SourceNote:      "按模板默认值(1%)",
	}
	costItems = append(costItems, lossItem)
	totalFixed += lossItem.Amount

	card.CostItems = costItems
	card.TotalFixedCost = round(totalFixed, 2)
	card.TotalVariableFeeRate = round(totalVarRate, 4)

	// 3. Cost at target price
	variableFee := round(prod.TargetSalePrice*totalVarRate, 2)
	card.EstimatedVariableFee = variableFee
	card.TotalCostAtTargetPrice = round(totalFixed+variableFee, 2)

	// 4. Profit
	estProfit := round(prod.TargetSalePrice-totalFixed-variableFee, 2)
	card.EstimatedProfit = estProfit
	if prod.TargetSalePrice > 0 {
		card.ProfitMargin = round(estProfit/prod.TargetSalePrice*100, 2)
	}

	// Status: unknown when no target price, otherwise classify by margin
	if prod.TargetSalePrice <= 0 {
		card.Status = "unknown"
	} else if card.ProfitMargin >= 15 {
		card.Status = "profitable"
	} else if card.ProfitMargin >= 5 {
		card.Status = "marginal"
	} else {
		card.Status = "unprofitable"
	}

	// 5. Break-even / recommended price
	if totalVarRate < 1 {
		card.BreakEvenPrice = round(totalFixed/(1-totalVarRate), 2)
		card.RecommendedMinPrice = round(totalFixed*(1+defaultTargetMargin+defaultBufferRate)/(1-totalVarRate), 2)
	}

	// 6. Confidence & blocking
	card.computeConfidenceAndBlocking(prod)

	return card
}

func (c *EvidenceCard) computeConfidenceAndBlocking(prod *candidateProductReader) {
	var confirmed, estimated, missing, blockingReasons []string

	// Required field check: purchase_cost
	if prod.PurchasePrice <= 0 {
		missing = append(missing, "采购成本(purchase_cost)")
		blockingReasons = append(blockingReasons, "缺少采购成本，无法计算利润")
	} else {
		confirmed = append(confirmed, "采购成本")
	}

	// Required: package_weight_kg (needed to estimate shipping)
	if prod.PackageWeightKg <= 0 {
		missing = append(missing, "商品重量(package_weight_kg)")
		blockingReasons = append(blockingReasons, "缺少商品重量，无法估算物流费")
	} else {
		confirmed = append(confirmed, "商品重量")
	}

	// Required: target_sale_price
	if prod.TargetSalePrice <= 0 {
		missing = append(missing, "目标售价(target_sale_price)")
		blockingReasons = append(blockingReasons, "缺少目标售价，无法计算利润")
	} else {
		confirmed = append(confirmed, "目标售价")
	}

	// Required: destination_country
	if prod.DestinationCountry == "" {
		missing = append(missing, "目标国家(destination_country)")
		blockingReasons = append(blockingReasons, "缺少目标国家，无法确定费率")
	} else {
		confirmed = append(confirmed, "目标国家")
	}

	// Required: international_shipping assessment
	shippingItem := findCost("international_shipping", c.CostItems)
	if shippingItem != nil && shippingItem.Amount <= 0 {
		missing = append(missing, "国际物流费(international_shipping)")
		blockingReasons = append(blockingReasons, "缺少国际物流费，无法完整核算成本")
	}

	// Required: platform_commission
	commItem := findCost("platform_commission", c.CostItems)
	if commItem != nil && commItem.Rate <= 0 && commItem.Amount <= 0 {
		missing = append(missing, "平台佣金(platform_commission)")
		blockingReasons = append(blockingReasons, "缺少平台佣金费率，无法完整核算成本")
	}

	// Check data source levels for cost items
	// Skip categories already handled by direct field checks above
	alreadyDirectChecked := map[string]bool{
		"purchase_cost":          true,
		"international_shipping": true,
		"platform_commission":    true,
	}
	for _, item := range c.CostItems {
		if alreadyDirectChecked[item.Category] {
			continue
		}
		if item.DataSource == "missing" {
			missing = append(missing, item.Label)
		} else if item.DataSource == "estimated" {
			estimated = append(estimated, item.Label)
		} else if item.DataSource == "template_default" {
			estimated = append(estimated, item.Label+"(模板值)")
		}
	}

	c.ConfirmedItems = toDataFields(confirmed, "confirmed")
	c.EstimatedItems = toDataFields(estimated, "estimated")
	c.MissingItems = missing
	c.BlockingReasons = blockingReasons

	// Confidence level
	if len(blockingReasons) > 0 {
		c.CanEvaluate = false
		c.ConfidenceLevel = "insufficient_data"
	} else if len(estimated) > 0 {
		c.CanEvaluate = true
		// Check if any are template_default
		hasTemplateDefault := false
		for _, item := range c.CostItems {
			if item.DataSource == "template_default" {
				hasTemplateDefault = true
				break
			}
		}
		if hasTemplateDefault {
			c.ConfidenceLevel = "low"
		} else {
			c.ConfidenceLevel = "medium"
		}
	} else {
		c.CanEvaluate = true
		c.ConfidenceLevel = "high"
	}
}

func findCost(category string, items []CostItem) *CostItem {
	for i := range items {
		if items[i].Category == category {
			return &items[i]
		}
	}
	return nil
}

func toDataFields(names []string, source string) []DataField {
	fields := make([]DataField, len(names))
	for i, n := range names {
		fields[i] = DataField{FieldName: n, Label: n, Value: "", Source: source}
	}
	return fields
}

func round(f float64, n int) float64 {
	pow := math.Pow(10, float64(n))
	return math.Round(f*pow) / pow
}
