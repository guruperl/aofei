package dsp

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/guruperl/aofei/match"
	"github.com/prebid/openrtb/v20/openrtb2"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestServeBidMalformedRequestLogsStructuredError(t *testing.T) {
	controller := newLocalBidPathController(t)
	core, logs := observer.New(zap.DebugLevel)
	controller.Logger = zap.New(core)

	rsp := serveSmokeBid(t, controller, "pub.example", []byte("{"))
	if rsp.Code != http.StatusBadRequest {
		t.Fatalf("ServeBid status = %d, want %d", rsp.Code, http.StatusBadRequest)
	}
	entries := logs.All()
	if len(entries) != 1 {
		t.Fatalf("malformed request log count = %d, want 1", len(entries))
	}
	if entries[0].Level != zap.DebugLevel || entries[0].Message != "malformed bid request rejected" {
		t.Fatalf("malformed request log = %+v", entries[0])
	}
	if _, ok := entries[0].ContextMap()["error"]; !ok {
		t.Fatalf("malformed request log has no structured error field: %+v", entries[0].ContextMap())
	}
}

func TestServeBidValidationLogHashesRequestID(t *testing.T) {
	controller := newLocalBidPathController(t)
	core, logs := observer.New(zap.DebugLevel)
	controller.Logger = zap.New(core)
	request := localBidRequest("USD", "USD")
	request.Imp[0].ID = ""
	body, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	rsp := serveSmokeBid(t, controller, "pub.example", body)
	if rsp.Code != http.StatusNoContent {
		t.Fatalf("ServeBid status = %d, want %d", rsp.Code, http.StatusNoContent)
	}
	entries := logs.All()
	if len(entries) != 1 {
		t.Fatalf("validation log count = %d, want 1", len(entries))
	}
	fields := entries[0].ContextMap()
	if fields["request_id_hash"] == "" || fields["request_id"] != nil {
		t.Fatalf("request identifier fields = %#v", fields)
	}
	if encoded, err := json.Marshal(fields); err != nil || strings.Contains(string(encoded), request.ID) {
		t.Fatalf("validation log exposed raw request id: %s, err=%v", encoded, err)
	}
}

func TestServeBidExpectedNoBidDoesNotLog(t *testing.T) {
	controller := newLocalBidPathController(t)
	core, logs := observer.New(zap.DebugLevel)
	controller.Logger = zap.New(core)

	body := marshalBidRequest(t, localBidRequest("EUR", "EUR"))
	rsp := serveSmokeBid(t, controller, "pub.example", body)
	if rsp.Code != http.StatusNoContent {
		t.Fatalf("ServeBid status = %d, want %d", rsp.Code, http.StatusNoContent)
	}
	if entries := logs.All(); len(entries) != 0 {
		t.Fatalf("expected no-bid logs = %+v, want none", entries)
	}
}

func TestLocalBidEffectiveCPMUsesResponsePrice(t *testing.T) {
	price, comparable := localBidEffectiveCPM(
		&DSP{},
		bidAudit{One: match.RAdv{CostType: 2, Cost: 2}},
		openrtb2.Bid{Price: 1.25},
	)
	if !comparable || price != 1.25 {
		t.Fatalf("effective CPM = %f, %t; want 1.25, true", price, comparable)
	}
	if _, comparable := localBidEffectiveCPM(&DSP{}, bidAudit{One: match.RAdv{}}, openrtb2.Bid{Price: 1.25}); comparable {
		t.Fatal("invalid local pricing should remain non-comparable")
	}
}
