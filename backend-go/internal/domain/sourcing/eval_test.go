package sourcing

import (
	"testing"
)

// ---------- fixture helpers ----------

// fixtureHighQuality returns a product page with good title, 3 images,
// high supplier score, detailed description, spec variants, and full
// logistics data. Expected score: 93 (no penalties).
func fixtureHighQuality() *PageData {
	weight := 0.3
	length := 15.0
	width := 10.0
	height := 5.0
	freight := 8.0
	supplierScore := 85

	return &PageData{
		Title:        "2024新款智能蓝牙耳机TWS降噪运动耳机防水无线耳机续航30小时",
		Price:        150,
		Images:       []string{"img1.jpg", "img2.jpg", "img3.jpg"},
		SupplierName: "深圳华强北科技有限公司",
		SupplierScore: &supplierScore,
		Description:  "高品质蓝牙5.3芯片，连接稳定不卡顿。13mm动圈单元，低音澎湃高音清亮。IPX7级防水防汗，运动无忧。单次充电续航8小时，配合充电仓可达30小时。触控操作，智能降噪，通话清晰。配备S/M/L三副耳塞，舒适贴合各种耳型。兼容iOS和Android系统，支持语音助手一键唤醒。Type-C充电接口，2小时充满充电仓，支持无线充电功能。附赠硅胶保护套和挂绳，三种颜色可选，送礼自用皆宜。精选优质材料，经过严格质量检测，确保每一件产品都达到最高标准。",
		SpecVariants: []SpecVariant{
			{Spec: "颜色:黑色", Price: 150, Stock: 200},
			{Spec: "颜色:白色", Price: 150, Stock: 150},
		},
		PackageWeight: &weight,
		PackageLength: &length,
		PackageWidth:  &width,
		PackageHeight: &height,
		FreightCNY:    &freight,
	}
}

// fixtureLowQuality returns a product page with vague title, single image,
// no supplier score, and empty description. Expected score: 5 (penalties
// for missing description (-10) and missing supplier name (-5)).
func fixtureLowQuality() *PageData {
	return &PageData{
		Title:         "特价蓝牙耳机",
		Price:         30,
		Images:        []string{"img1.jpg"},
		SupplierName:  "", // penalty: -5
		SupplierScore: nil,
		Description:   "", // penalty: -10
		SpecVariants:  nil,
	}
}

// fixtureMediumQuality returns a product page with decent title, 3 images,
// moderate supplier score, short description, and 5 spec variants (high
// spec richness). Expected score: 73.
func fixtureMediumQuality() *PageData {
	weight := 0.5
	supplierScore := 60

	return &PageData{
		Title:        "大容量充电宝20000mAh快充",
		Price:        80,
		Images:       []string{"img1.jpg", "img2.jpg", "img3.jpg"},
		SupplierName: "东莞电源科技有限公司",
		SupplierScore: &supplierScore,
		Description:  "采用高品质锂电芯，支持PD快充协议，兼容手机平板等设备，智能分流芯片。",
		SpecVariants: []SpecVariant{
			{Spec: "容量:10000mAh", Price: 55, Stock: 500},
			{Spec: "容量:20000mAh", Price: 80, Stock: 300},
			{Spec: "容量:30000mAh", Price: 120, Stock: 200},
			{Spec: "颜色:黑色", Price: 80, Stock: 400},
			{Spec: "颜色:白色", Price: 80, Stock: 350},
		},
		PackageWeight: &weight,
	}
}

// fixtureMissingFields returns a product page with no price (0), no images,
// and no supplier info. Expected score: 0 (penalties for missing price -15,
// missing images -10, plus very low base score).
func fixtureMissingFields() *PageData {
	return &PageData{
		Title:         "智能手表多功能防水运动腕表",
		Price:         0, // penalty: -15
		Images:        nil, // penalty: -10
		SupplierName:  "某供应商",
		SupplierScore: nil,
		Description:   "支持心率监测睡眠分析",
		SpecVariants:  nil,
	}
}

// fixturePremium returns a premium product with excellent title, 8 images,
// top supplier score, long description, 4 spec variants, and full package
// + freight data. Expected score: 100 (capped).
func fixturePremium() *PageData {
	weight := 2.5
	length := 35.0
	width := 25.0
	height := 3.0
	freight := 15.0
	supplierScore := 92

	return &PageData{
		Title:        "高端商务笔记本电脑15.6英寸轻薄本i7处理器16GB内存512GB固态",
		Price:        3500,
		Images:       []string{"img1.jpg", "img2.jpg", "img3.jpg", "img4.jpg", "img5.jpg", "img6.jpg", "img7.jpg", "img8.jpg"},
		SupplierName: "联想科技集团有限公司",
		SupplierScore: &supplierScore,
		Description:  "搭载第13代英特尔酷睿i7-13700H处理器，16GB DDR5高速内存，512GB NVMe固态硬盘。15.6英寸FHD IPS防眩光屏，100%sRGB色域，通过TUV护眼认证。全金属机身，重量仅1.8kg，厚度17.9mm。配备Thunderbolt 4接口、USB-C、HDMI 2.1和Wi-Fi 6E。61Wh大容量电池，续航长达12小时。预装Windows 11专业版，通过MIL-STD-810H军规认证。支持指纹识别和IR红外摄像头人脸登录，多重安全保障。",
		SpecVariants: []SpecVariant{
			{Spec: "内存:16GB/存储:512GB", Price: 3500, Stock: 100},
			{Spec: "内存:16GB/存储:1TB", Price: 3800, Stock: 80},
			{Spec: "内存:32GB/存储:512GB", Price: 4200, Stock: 50},
			{Spec: "内存:32GB/存储:1TB", Price: 4500, Stock: 30},
		},
		PackageWeight: &weight,
		PackageLength: &length,
		PackageWidth:  &width,
		PackageHeight: &height,
		FreightCNY:    &freight,
	}
}

// ---------- tests ----------

func TestQualityEvaluator_EvaluateQuality(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		pageData *PageData
		expected int // expected QualityScore
	}{
		{
			name:     "high quality product",
			pageData: fixtureHighQuality(),
			expected: 93,
		},
		{
			name:     "low quality product",
			pageData: fixtureLowQuality(),
			expected: 5,
		},
		{
			name:     "medium quality with spec variants",
			pageData: fixtureMediumQuality(),
			expected: 73,
		},
		{
			name:     "missing key fields",
			pageData: fixtureMissingFields(),
			expected: 0,
		},
		{
			name:     "premium product with logistics data",
			pageData: fixturePremium(),
			expected: 100,
		},
	}

	evaluator := NewQualityEvaluator()
	const tolerance = 1

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := evaluator.EvaluateQuality(tt.pageData)

			got := report.Score
			diff := got - tt.expected
			if diff < 0 {
				diff = -diff
			}
			if diff > tolerance {
				t.Errorf(
					"Score = %d, expected %d (±%d), diff=%d\nBreakdown: %+v\nWarnings: %v",
					got, tt.expected, tolerance, diff,
					report.Breakdown, report.Warnings,
				)
			}
		})
	}
}

func TestQualityEvaluator_MissingFieldsPenalty(t *testing.T) {
	t.Parallel()

	evaluator := NewQualityEvaluator()

	// A complete product should score higher than one missing critical fields.
	complete := fixtureHighQuality()
	missing := fixtureMissingFields()

	reportComplete := evaluator.EvaluateQuality(complete)
	reportMissing := evaluator.EvaluateQuality(missing)

	if reportMissing.Score >= reportComplete.Score {
		t.Errorf(
			"missing fields product (%d) should score lower than complete product (%d)",
			reportMissing.Score, reportComplete.Score,
		)
	}
	if reportMissing.Breakdown.Penalty <= 0 {
		t.Errorf("missing fields should incur a penalty; got Penalty=%d", reportMissing.Breakdown.Penalty)
	}
	if reportMissing.Breakdown.Images > 0 {
		t.Errorf("product with no images should have zero image score; got %d", reportMissing.Breakdown.Images)
	}
}

func TestQualityEvaluator_SpecRichnessIncreasesScore(t *testing.T) {
	t.Parallel()

	evaluator := NewQualityEvaluator()
	base := fixtureLowQuality() // 0 spec variants

	// Clone base and add spec variants.
	withSpecs := *base
	withSpecs.SpecVariants = []SpecVariant{
		{Spec: "颜色:黑色", Price: 30, Stock: 100},
		{Spec: "颜色:白色", Price: 30, Stock: 50},
		{Spec: "颜色:蓝色", Price: 32, Stock: 30},
	}

	reportBase := evaluator.EvaluateQuality(base)
	reportWithSpecs := evaluator.EvaluateQuality(&withSpecs)

	if reportWithSpecs.Breakdown.SpecRichness <= reportBase.Breakdown.SpecRichness {
		t.Errorf(
			"SpecRichness should be higher with spec variants; got base=%d, withSpecs=%d",
			reportBase.Breakdown.SpecRichness,
			reportWithSpecs.Breakdown.SpecRichness,
		)
	}
	if reportWithSpecs.Score <= reportBase.Score {
		t.Errorf(
			"total score should be higher with spec variants; got base=%d, withSpecs=%d",
			reportBase.Score,
			reportWithSpecs.Score,
		)
	}
	if reportWithSpecs.Breakdown.SpecRichness == 0 {
		t.Errorf("SpecRichness should be > 0 when spec variants exist")
	}
}

func TestQualityEvaluator_NilInput(t *testing.T) {
	t.Parallel()

	evaluator := NewQualityEvaluator()
	report := evaluator.EvaluateQuality(nil)

	if report.Score != 0 {
		t.Errorf("nil input should yield score 0; got %d", report.Score)
	}
	if len(report.Warnings) == 0 {
		t.Fatal("nil input should produce a warning")
	}
	if report.Warnings[0] != "nil page data" {
		t.Errorf("expected 'nil page data' warning; got %q", report.Warnings[0])
	}
}

func TestQualityEvaluator_WarningsForMissingFields(t *testing.T) {
	t.Parallel()

	evaluator := NewQualityEvaluator()

	tests := []struct {
		name              string
		pageData          *PageData
		expectPriceWarn   bool
		expectImageWarn   bool
		expectDescWarn    bool
		expectSupplierWarn bool
	}{
		{
			name:              "low quality - missing description and supplier name",
			pageData:          fixtureLowQuality(),
			expectPriceWarn:   false,
			expectImageWarn:   false,
			expectDescWarn:    true,
			expectSupplierWarn: true,
		},
		{
			name:              "missing fields - no price and no images",
			pageData:          fixtureMissingFields(),
			expectPriceWarn:   true,
			expectImageWarn:   true,
			expectDescWarn:    false,
			expectSupplierWarn: true, // nil supplier score
		},
		{
			name:              "high quality - no warnings",
			pageData:          fixtureHighQuality(),
			expectPriceWarn:   false,
			expectImageWarn:   false,
			expectDescWarn:    false,
			expectSupplierWarn: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := evaluator.EvaluateQuality(tt.pageData)
			warnSet := make(map[string]bool, len(report.Warnings))
			for _, w := range report.Warnings {
				warnSet[w] = true
			}

			if got := warnSet["missing price"]; got != tt.expectPriceWarn {
				t.Errorf("missing price warning: got %v, want %v (warnings=%v)", got, tt.expectPriceWarn, report.Warnings)
			}
			if got := warnSet["no product images"]; got != tt.expectImageWarn {
				t.Errorf("no product images warning: got %v, want %v", got, tt.expectImageWarn)
			}
			if got := warnSet["missing product description"]; got != tt.expectDescWarn {
				t.Errorf("missing product description warning: got %v, want %v", got, tt.expectDescWarn)
			}
			if got := warnSet["no supplier trust score"]; got != tt.expectSupplierWarn {
				t.Errorf("no supplier trust score warning: got %v, want %v", got, tt.expectSupplierWarn)
			}
		})
	}
}
