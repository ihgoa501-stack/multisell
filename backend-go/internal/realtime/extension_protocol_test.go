package realtime

import (
	"encoding/json"
	"testing"
)

func TestDecodeExtensionMessageAcceptsProtocolPayload(t *testing.T) {
	raw := []byte(`{"type":"fetch_product_result","id":"req-1","payload":{"status":"ok","data":{"source_url":"https://detail.1688.com/offer/1.html","title":"真实商品"}}}`)

	msg, err := decodeExtensionMessage(raw)
	if err != nil {
		t.Fatalf("decodeExtensionMessage() error = %v", err)
	}
	if msg.Type != "fetch_product_result" {
		t.Fatalf("Type = %q, want fetch_product_result", msg.Type)
	}
	if msg.ID != "req-1" {
		t.Fatalf("ID = %q, want req-1", msg.ID)
	}

	var payload struct {
		Status string `json:"status"`
		Data   struct {
			SourceURL string `json:"source_url"`
			Title     string `json:"title"`
		} `json:"data"`
	}
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		t.Fatalf("payload unmarshal error = %v", err)
	}
	if payload.Status != "ok" || payload.Data.Title != "真实商品" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestDecodeExtensionMessageAcceptsListPagePayload(t *testing.T) {
	raw := []byte(`{"type":"list_page_result","payload":{"status":"ok","data":{"page_url":"https://s.1688.com/selloffer/offer_search.htm","items":[{"title":"商品A","detail_url":"https://detail.1688.com/offer/2.html"}]}}}`)

	msg, err := decodeExtensionMessage(raw)
	if err != nil {
		t.Fatalf("decodeExtensionMessage() error = %v", err)
	}
	if msg.Type != "list_page_result" {
		t.Fatalf("Type = %q, want list_page_result", msg.Type)
	}
	if len(msg.Payload) == 0 {
		t.Fatal("Payload is empty")
	}
}

func TestDecodeExtensionMessageAcceptsFetchProductError(t *testing.T) {
	raw := []byte(`{"type":"fetch_product_error","id":"req-2","payload":{"code":"TAB_NOT_FOUND","message":"open the product page"}}`)
	msg, err := decodeExtensionMessage(raw)
	if err != nil {
		t.Fatalf("decodeExtensionMessage() error = %v", err)
	}
	if msg.Type != "fetch_product_error" || msg.ID != "req-2" {
		t.Fatalf("unexpected message: %+v", msg)
	}
}

func TestDecodeExtensionMessageRejectsMissingPayload(t *testing.T) {
	_, err := decodeExtensionMessage([]byte(`{"type":"fetch_product_result","id":"req-1"}`))
	if err == nil {
		t.Fatal("expected missing payload error")
	}
}
