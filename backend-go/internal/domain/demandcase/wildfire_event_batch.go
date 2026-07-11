package demandcase

import "context"

const ReviewedWildfireEventBatchKey = "us-ca-hoopa-monument-fire-2021-reviewed-2026-07-11"

const wildfireEventScoutPayload = `{
  "artifact_path":"deliverables/research/2026-07-11-us-wildfire-local-event-scout.md",
  "problem_key":"us-ca-hoopa-2021-monument-fire-household-clean-air",
  "completed_event":{"name":"2021 Monument Fire","event_start":"2021-07-31","event_end":"2021-10-25","local_smoke_observed_through":"2021-09-26"},
  "local_program":"Hoopa Valley Tribe/Tribal EPA + US EPA ASPIRE-Health household pilot",
  "households":{"enrolled":8,"complete_indoor_outdoor_pm25":6,"prior_commercial_pac":4,"self_purchased_prior_pac":1},
  "adoption":{"diy_noise_frustration_homes":7,"commercial_choice_homes":8,"quiet_as_primary_choice_homes":5,"free_provision":true,"required_use_hours_per_day":8},
  "pm25":{"diy_indoor_reduction_percent":7,"diy_infiltrated_reduction_percent":10.8,"commercial_infiltrated_reduction_percent":18.3,"initial_outdoor_ug_m3":9.1,"sensor_display_outdoor_ug_m3":2.1},
  "existing_solutions":{"commercial_pac":true,"diy_pac":true,"central_hvac_with_pm25_filter_homes":2,"central_hvac_without_filter_homes":2,"no_central_hvac_homes":4},
  "filter_audit":"availability and cost were reported, but household counts, purchases, stockouts, replacement dates, distance and event-level discontinuation were unpublished",
  "noise_heat_space_audit":"noise repeated in 7/8; wildfire-period heat discontinuation unmeasured; winter cooling cannot be transferred; space was preference, not quantified discontinuation",
  "public_space_audit":"community cleaner-air-space context existed, but household use, distance, duration and substitution were unmeasured",
  "critical_gaps":["interventions did not overlap sustained Monument Fire smoke","public cleaner-air-space substitution not measured","wildfire heat/space discontinuation not measured","research support confounded natural adoption"],
  "paid_demand":null,"product":null,"platform":null,"channel":null,
  "collector":"agent:wildfire_local_event_scout","observed_at":"2026-07-11"
}`

const wildfireEventFalsifierPayload = `{
  "artifact_path":"deliverables/research/2026-07-11-us-wildfire-local-event-independent-falsification.md",
  "problem_key":"us-ca-hoopa-2021-monument-fire-household-clean-air",
  "decision":"rejected","residual_barrier_status":"not_confirmed",
  "burden_of_proof":"critical same-event evidence must be present; missing event evidence counts against the hypothesis",
  "event_timing":{"monument_smoke_phase":"Initial phase only","local_smoke_cleared_by":"2021-09-26","intervention_overlap_with_sustained_smoke":false},
  "adoption_audit":"free provision, required use and repeated researcher contact do not establish natural adoption; all 8 homes chose commercial PAC in free-choice phase",
  "indoor_pm25_audit":"associations exist, but interventions did not overlap Monument Fire smoke and only 6 homes had complete paired data",
  "device_substitution":"DIY noise repeated in 7/8, but all homes chose commercial PAC and 5/8 cited quieter operation",
  "filter_replacement_audit":"no event-level purchases, stockouts, replacement dates or discontinuation chain; study staff handled replacement",
  "noise_heat_space_audit":"noise was commercially substituted; winter cooling cannot be transferred; wildfire heat and space discontinuation were not observed",
  "public_space_audit":"community cleaner-air-space context existed, but household use and substitution were not measured and cannot be excluded",
  "structural_assessment":"filter access/cost and HVAC cannot be separated from remote supply, housing, income and public-health support",
  "stop_conditions":["smoke occurred only before interventions","commercial PAC substituted the repeated noise barrier","public cleaner-air-space substitution was not measured"],
  "paid_demand":null,"product":null,"platform":null,"channel":null,
  "collector":"agent:wildfire_local_event_falsifier","observed_at":"2026-07-11"
}`

func reviewedWildfireEventProblems() []reviewedProblem {
	return []reviewedProblem{{
		caseData: ProblemCase{
			ProblemKey:            "us-ca-hoopa-2021-monument-fire-household-clean-air",
			Region:                "US-CA-HOOPA",
			ObservablePopulation:  "Hoopa Valley 2021 ASPIRE-Health 野火研究的 8 户家庭（6 户具有完整室内外 PM2.5 配对）",
			ProblemScenario:       "Monument Fire 烟霾期间，家庭是否反复遭遇消费者可控、非结构性且未被商用 PAC、HVAC、DIY 或公共洁净空间替代的洁净空气障碍",
			CurrentWorkaround:     "既有或公共发放商用 PAC、研究发放 DIY PAC、降档或换房间、中央/窗式空调以及社区洁净空气空间",
			Responsibility:        ResponsibilityShared,
			ProductSolvability:    SolvabilityPartial,
			HarmRisk:              HarmHigh,
			ResidualBarrierStatus: ResidualBarrierNotConfirmed,
			NextMinimumEvidence:   "停止此事件命题；未来必须以新的重烟事件、同期干预、逐户自然采用和替代使用记录建立新案件",
		},
		supportTitle:     "Hoopa ASPIRE-Health household monitoring and device use",
		supportURI:       "https://www.osti.gov/servlets/purl/2580861",
		supportPayload:   wildfireEventScoutPayload,
		counterTitle:     "Monument Fire timing and existing substitution do not confirm the residual barrier",
		counterURI:       "https://www.osti.gov/servlets/purl/2580861",
		counterPayload:   wildfireEventFalsifierPayload,
		supportCollector: "agent:wildfire_local_event_scout",
		counterCollector: "agent:wildfire_local_event_falsifier",
	}}
}

func (s *Service) ImportReviewedWildfireEventBatch(ctx context.Context, ownerID int64) (*ReviewedProblemBatchOutcome, error) {
	return s.importReviewedProblems(ctx, ownerID, ReviewedWildfireEventBatchKey, reviewedWildfireEventProblems())
}
