package productimage

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/lingmirror/backend-go/internal/dbtest"
	"github.com/lingmirror/backend-go/internal/imageservice"
)

type mcpEvaluationFile struct {
	XMLName xml.Name          `xml:"evaluation"`
	Pairs   []mcpEvaluationQA `xml:"qa_pair"`
}

type mcpEvaluationQA struct {
	Question string `xml:"question"`
	Answer   string `xml:"answer"`
}

type readonlyEvaluationRunner struct {
	t      *testing.T
	router interface {
		ServeHTTP(http.ResponseWriter, *http.Request)
	}
	calls []string
}

// TestMCPReadonlyEvaluations executes the same read-only tool sequences an MCP
// client needs to solve the published evaluation set. The fixture is frozen:
// IDs, provider availability, task versions, statuses and output hashes are
// deliberately explicit, so direct string comparison remains stable.
func TestMCPReadonlyEvaluations(t *testing.T) {
	raw, err := os.ReadFile("testdata/mcp_readonly_evaluations.xml")
	if err != nil {
		t.Fatal(err)
	}
	var evaluation mcpEvaluationFile
	if err := xml.Unmarshal(raw, &evaluation); err != nil {
		t.Fatalf("parse evaluation XML: %v", err)
	}
	if evaluation.XMLName.Local != "evaluation" || len(evaluation.Pairs) != 10 {
		t.Fatalf("root=%q pairs=%d, want evaluation and exactly 10", evaluation.XMLName.Local, len(evaluation.Pairs))
	}

	for i, pair := range evaluation.Pairs {
		pair := pair
		if strings.TrimSpace(pair.Question) == "" || strings.TrimSpace(pair.Answer) == "" {
			t.Fatalf("pair %d has an empty question or answer", i+1)
		}
		lower := strings.ToLower(pair.Question)
		for _, forbidden := range []string{"submit_approved", "reconcile_task", "execute", "approve", "upload", "create task", "mutation"} {
			if strings.Contains(lower, forbidden) {
				t.Fatalf("pair %d requests or names a non-read-only action %q", i+1, forbidden)
			}
		}

		t.Run(fmt.Sprintf("pair_%02d", i+1), func(t *testing.T) {
			svc := newReadonlyEvaluationService(t)
			runner := &readonlyEvaluationRunner{t: t, router: mcpRouter(svc, int64(77))}
			got := solveReadonlyEvaluation(runner, i)
			if got != strings.TrimSpace(pair.Answer) {
				t.Fatalf("answer=%q want=%q; calls=%v", got, pair.Answer, runner.calls)
			}
			if len(runner.calls) < 2 {
				t.Fatalf("evaluation used %d call(s), want a multi-tool sequence", len(runner.calls))
			}
			for _, name := range runner.calls {
				if name != "lingmirror_image_list_capabilities" && name != "lingmirror_image_estimate" && name != "lingmirror_image_get_task" && name != "lingmirror_image_list_outputs" {
					t.Fatalf("evaluation invoked non-read-only tool %q", name)
				}
			}
		})
	}
}

func newReadonlyEvaluationService(t *testing.T) *Service {
	t.Helper()
	db := dbtest.NewDB(t, &Task{})
	remote := newFakeImageService()
	now := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)
	tasks := []Task{
		{ID: 1101, OwnerID: 77, AssetID: 501, ImageServiceJobID: "eval-job-1101", IdempotencyKey: "eval-1101", ManifestHash: strings.Repeat("1", 64), Operation: "DETERMINISTIC_RESIZE", Processor: "deterministic", Purpose: "listing_main", Channel: "ozon", Region: "local", Version: 1, Width: 800, Height: 800, Format: "png", Status: "READY", CreatedAt: now, UpdatedAt: now},
		{ID: 1102, OwnerID: 77, AssetID: 502, ImageServiceJobID: "eval-job-1102", IdempotencyKey: "eval-1102", ManifestHash: strings.Repeat("2", 64), Operation: "OPENAI_IMAGE_EDIT", Processor: "openai", Purpose: "listing_main", Channel: "ozon", Region: "us", ProviderEnvironment: "production", MaxCost: "0.20", Currency: "USD", Version: 2, Width: 1024, Height: 1024, Format: "png", Status: "QUEUED", CreatedAt: now, UpdatedAt: now},
		{ID: 1103, OwnerID: 77, AssetID: 503, ImageServiceJobID: "eval-job-1103", IdempotencyKey: "eval-1103", ManifestHash: strings.Repeat("3", 64), Operation: "DETERMINISTIC_RESIZE", Processor: "deterministic", Purpose: "listing_gallery", Channel: "ozon", Region: "local", Version: 3, Width: 1200, Height: 1500, Format: "webp", Status: "READY", CreatedAt: now, UpdatedAt: now},
		{ID: 1104, OwnerID: 77, AssetID: 504, ImageServiceJobID: "eval-job-1104", IdempotencyKey: "eval-1104", ManifestHash: strings.Repeat("4", 64), Operation: "DETERMINISTIC_RESIZE", Processor: "deterministic", Purpose: "listing_gallery", Channel: "ozon", Region: "local", Version: 1, Width: 600, Height: 600, Format: "jpg", Status: "FAILED", ErrorCode: "PROCESS_FAILED", CreatedAt: now, UpdatedAt: now},
		{ID: 1105, OwnerID: 77, AssetID: 505, ImageServiceJobID: "eval-job-1105", IdempotencyKey: "eval-1105", ManifestHash: strings.Repeat("5", 64), Operation: "DETERMINISTIC_RESIZE", Processor: "deterministic", Purpose: "listing_main", Channel: "ozon", Region: "local", Version: 1, Width: 2000, Height: 2000, Format: "png", Status: "QUEUED", CreatedAt: now, UpdatedAt: now},
	}
	if err := db.Create(&tasks).Error; err != nil {
		t.Fatal(err)
	}
	for _, task := range tasks {
		blob := ""
		if task.ID == 1101 {
			blob = strings.Repeat("a", 64)
		} else if task.ID == 1103 {
			blob = strings.Repeat("c", 64)
		}
		remote.jobs[task.ImageServiceJobID] = imageservice.Job{ID: task.ImageServiceJobID, OwnerID: task.OwnerID, LingMirrorTaskID: strconv.FormatInt(task.ID, 10), LingMirrorTaskVersion: task.Version, ManifestHash: task.ManifestHash, Operation: task.Operation, Processor: task.Processor, MaxCost: task.MaxCost, Currency: task.Currency, Region: task.Region, ProviderEnvironment: task.ProviderEnvironment, Sandbox: task.Sandbox, Watermarked: task.Watermarked, NonPublishable: task.NonPublishable, Status: task.Status, ErrorCode: task.ErrorCode, OutputBlobID: blob, CreatedAt: now, UpdatedAt: now}
	}
	return NewService(db, dbtest.NewLogger(t), remote)
}

func (r *readonlyEvaluationRunner) call(name string, arguments map[string]any) map[string]any {
	r.t.Helper()
	r.calls = append(r.calls, name)
	b, err := json.Marshal(arguments)
	if err != nil {
		r.t.Fatal(err)
	}
	status, response := callMCP(r.t, r.router, toolCall(name, string(b)))
	if status != 200 {
		r.t.Fatalf("tool %s status=%d response=%v", name, status, response)
	}
	return structured(r.t, response)
}

func (r *readonlyEvaluationRunner) task(id int64) map[string]any {
	response := r.call("lingmirror_image_get_task", map[string]any{"task_id": id})
	task, ok := response["task"].(map[string]any)
	if !ok {
		r.t.Fatalf("task %d missing from response: %v", id, response)
	}
	return task
}

func (r *readonlyEvaluationRunner) estimate(id int64) map[string]any {
	return r.call("lingmirror_image_estimate", map[string]any{"task_id": id})["estimate"].(map[string]any)
}

func (r *readonlyEvaluationRunner) outputs(id int64) []any {
	return r.call("lingmirror_image_list_outputs", map[string]any{"task_id": id, "page": 1, "page_size": 10})["outputs"].([]any)
}

func (r *readonlyEvaluationRunner) capabilities(page, size int) []any {
	return r.call("lingmirror_image_list_capabilities", map[string]any{"page": page, "page_size": size})["capabilities"].([]any)
}

func solveReadonlyEvaluation(r *readonlyEvaluationRunner, index int) string {
	switch index {
	case 0:
		bestID, bestArea := int64(0), 0
		for _, id := range []int64{1101, 1102, 1103} {
			task, outputs := r.task(id), r.outputs(id)
			area := int(task["width"].(float64) * task["height"].(float64))
			if len(outputs) == 1 && outputs[0].(map[string]any)["status"] == "READY" && area > bestArea {
				bestID, bestArea = id, area
			}
		}
		return strconv.FormatInt(bestID, 10)
	case 1:
		availability := map[string]string{}
		for page := 1; page <= 2; page++ {
			for _, raw := range r.capabilities(page, 2) {
				item := raw.(map[string]any)
				availability[item["code"].(string)] = item["availability"].(string)
			}
		}
		for _, id := range []int64{1101, 1102, 1103} {
			task, estimate, outputs := r.task(id), r.estimate(id), r.outputs(id)
			if availability[task["processor"].(string)] == "unavailable" && estimate["availability"] == "unavailable" && len(outputs) == 0 {
				return strconv.FormatInt(id, 10)
			}
		}
	case 2:
		available := 0
		for page := 1; page <= 2; page++ {
			for _, raw := range r.capabilities(page, 2) {
				if raw.(map[string]any)["availability"] == "available" {
					available++
				}
			}
		}
		return strconv.Itoa(available)
	case 3:
		_ = r.capabilities(1, 2)
		return r.capabilities(2, 2)[0].(map[string]any)["name"].(string)
	case 4:
		var chosen map[string]any
		bestArea := 0
		for _, id := range []int64{1101, 1103} {
			task, outputs := r.task(id), r.outputs(id)
			area := int(task["width"].(float64) * task["height"].(float64))
			if len(outputs) == 1 && area > bestArea {
				chosen, bestArea = outputs[0].(map[string]any), area
			}
		}
		return chosen["media_reference"].(string)
	case 5:
		task := r.task(1102)
		for page := 1; page <= 2; page++ {
			_ = r.capabilities(page, 2)
		}
		estimate, outputs := r.estimate(1102), r.outputs(1102)
		if task["processor"] == "openai" && estimate["creates_approval_or_paid_intent"] == false && len(outputs) == 0 {
			return "False"
		}
		return "True"
	case 6:
		for _, id := range []int64{1101, 1102} {
			task, estimate := r.task(id), r.estimate(id)
			if task["processor"] == "deterministic" && estimate["exact"] == true {
				return estimate["amount"].(string)
			}
		}
	case 7:
		for _, id := range []int64{1101, 1103} {
			task, outputs := r.task(id), r.outputs(id)
			if task["format"] == "webp" && len(outputs) == 1 && outputs[0].(map[string]any)["status"] == "READY" {
				return outputs[0].(map[string]any)["blob_sha256"].(string)[:1]
			}
		}
	case 8:
		for _, id := range []int64{1102, 1104, 1105} {
			task, outputs := r.task(id), r.outputs(id)
			if task["status"] == "FAILED" && task["error_code"] != "" && len(outputs) == 0 {
				return strconv.FormatInt(id, 10)
			}
		}
	case 9:
		count := 0
		for id := int64(1101); id <= 1105; id++ {
			task, estimate, outputs := r.task(id), r.estimate(id), r.outputs(id)
			if task["id"] == float64(id) && len(outputs) == 1 && estimate["exact"] == true && estimate["amount"] == "0" {
				count++
			}
		}
		return strconv.Itoa(count)
	}
	return ""
}
