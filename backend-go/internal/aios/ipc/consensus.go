package ipc

// ConsensusResult holds the aggregated output of a consensus operation
// across multiple agents.
type ConsensusResult struct {
	FinalOutput       map[string]interface{} `json:"final_output"`
	Confidence        float64                `json:"confidence"`
	IndividualResults []*Message             `json:"individual_results"`
	Method            string                 `json:"method"`
}

// computeConsensus aggregates individual agent responses into a single
// consensus result. It supports two methods:
//
//   - "weighted_avg": numeric values are averaged weighted by each agent's
//     "confidence" payload field.
//   - "majority": string fields are resolved by majority vote; numeric
//     values are simple-averaged.
func computeConsensus(results []*Message) *ConsensusResult {
	if len(results) == 0 {
		return &ConsensusResult{
			FinalOutput:       make(map[string]interface{}),
			Confidence:        0,
			IndividualResults: results,
			Method:            "majority",
		}
	}

	// Detect whether any response carries a confidence score.
	hasConfidence := false
	for _, r := range results {
		if conf, ok := r.Payload["confidence"]; ok {
			if _, ok := conf.(float64); ok {
				hasConfidence = true
				break
			}
		}
	}

	method := "majority"
	if hasConfidence {
		method = "weighted_avg"
	}

	// Collect individual payloads for reference.
	individualResults := make([]*Message, len(results))
	copy(individualResults, results)

	// Aggregate numeric values using weighted or simple average.
	var totalValue, totalWeight, valueCount float64
	// Majority vote for string fields.
	voteCounts := make(map[string]map[string]int)

	for _, r := range results {
		conf := 1.0
		if c, ok := r.Payload["confidence"].(float64); ok && hasConfidence {
			conf = c
		}
		totalWeight += conf

		for k, v := range r.Payload {
			if k == "confidence" {
				continue
			}
			switch val := v.(type) {
			case float64:
				totalValue += val * conf
				valueCount++
			case string:
				if voteCounts[k] == nil {
					voteCounts[k] = make(map[string]int)
				}
				voteCounts[k][val]++
			}
		}
	}

	finalOutput := make(map[string]interface{})

	// Weighted or simple average for numeric values.
	if valueCount > 0 {
		if hasConfidence && totalWeight > 0 {
			finalOutput["value"] = totalValue / totalWeight
		} else {
			finalOutput["value"] = totalValue / valueCount
		}
	}

	// Majority vote for string fields.
	for field, counts := range voteCounts {
		maxCount := 0
		winner := ""
		for val, count := range counts {
			if count > maxCount {
				maxCount = count
				winner = val
			}
		}
		finalOutput[field] = winner
	}

	confidence := totalWeight / float64(len(results))

	if method == "majority" {
		confidence = float64(len(results)) / float64(len(results)) // normalize to 1.0
	}

	return &ConsensusResult{
		FinalOutput:       finalOutput,
		Confidence:        confidence,
		IndividualResults: individualResults,
		Method:            method,
	}
}
