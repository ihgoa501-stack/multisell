package logistics

// ---------- Request / Response DTOs ----------

// RateQuoteRequest is the input for on-demand freight estimation.
// OriginCountry defaults to "CN" if empty.
type RateQuoteRequest struct {
	OriginCountry      string  `json:"origin_country" example:"CN"`
	DestinationCountry string  `json:"destination_country" binding:"required" example:"RU"`
	WeightKG           float64 `json:"weight_kg" binding:"required" example:"0.5"`
	LengthCM           float64 `json:"length_cm" example:"30"`
	WidthCM            float64 `json:"width_cm" example:"20"`
	HeightCM           float64 `json:"height_cm" example:"15"`
	DeclaredValue      float64 `json:"declared_value" example:"25.00"`
	Currency           string  `json:"currency" example:"USD"`
	CargoType          string  `json:"cargo_type" example:"normal"` // "normal" | "battery" | "liquid" | "magnet"
}

// RateQuote is a single carrier's quote response.
type RateQuote struct {
	Carrier     string  `json:"carrier"`
	ServiceName string  `json:"service_name"`
	TotalCost   float64 `json:"total_cost"`
	Currency    string  `json:"currency"`
	EstDaysMin  int     `json:"est_days_min"`
	EstDaysMax  int     `json:"est_days_max"`
	Confidence  string  `json:"confidence"` // "high" | "medium" | "low"
}

// RateQuoteResponse wraps multiple carrier quotes.
type RateQuoteResponse struct {
	Quotes []RateQuote `json:"quotes"`
}

// CarrierInfo describes a configured carrier.
type CarrierInfo struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	CountryCode string `json:"country_code"`
	Channels    int    `json:"channels"`
}

// ---------- Conversion helpers ----------

// ToRateQuote converts internal QuoteResult to external RateQuote.
func (r *QuoteResult) ToRateQuote() RateQuote {
	return RateQuote{
		Carrier:     r.ProviderName,
		ServiceName: r.ChannelName,
		TotalCost:   r.TotalShippingFee,
		Currency:    r.Currency,
		EstDaysMin:  r.EstimatedDeliveryMin,
		EstDaysMax:  r.EstimatedDeliveryMax,
		Confidence:  "medium", // Phase 1: static confidence
	}
}

// ToRateQuoteResponse converts internal QuoteResponse to external RateQuoteResponse.
func (r *QuoteResponse) ToRateQuoteResponse() *RateQuoteResponse {
	quotes := make([]RateQuote, 0, len(r.Results))
	for _, qr := range r.Results {
		rq := qr.ToRateQuote()
		quotes = append(quotes, rq)
	}
	return &RateQuoteResponse{Quotes: quotes}
}
