package linkedin

import (
	"encoding/json"
	"testing"
)

func TestChangelogEventUnmarshalAcceptsNumericIDs(t *testing.T) {
	t.Parallel()

	payload := []byte(`{
		"elements": [
			{
				"id": 12345,
				"activityId": 67890,
				"capturedAt": 1710000000000,
				"processedAt": 1710000001000,
				"configVersion": 1,
				"owner": 111,
				"actor": 222,
				"resourceName": "shares",
				"resourceId": "urn:li:share:999",
				"resourceUri": 333,
				"method": "CREATE",
				"activityStatus": "SUCCESS",
				"activity": {"commentary": "hello"}
			}
		],
		"paging": {"start": 0, "count": 1, "total": 1, "links": []}
	}`)

	var resp ChangelogResponse
	if err := json.Unmarshal(payload, &resp); err != nil {
		t.Fatalf("unmarshal changelog response: %v", err)
	}

	if len(resp.Elements) != 1 {
		t.Fatalf("expected 1 element, got %d", len(resp.Elements))
	}

	event := resp.Elements[0]
	if got := string(event.ID); got != "12345" {
		t.Fatalf("expected numeric id to decode as string, got %q", got)
	}
	if got := string(event.ActivityID); got != "67890" {
		t.Fatalf("expected numeric activityId to decode as string, got %q", got)
	}
	if got := string(event.Owner); got != "111" {
		t.Fatalf("expected numeric owner to decode as string, got %q", got)
	}
	if got := string(event.Actor); got != "222" {
		t.Fatalf("expected numeric actor to decode as string, got %q", got)
	}
	if got := string(event.ResourceURI); got != "333" {
		t.Fatalf("expected numeric resourceUri to decode as string, got %q", got)
	}
}

func TestChangelogEventUnmarshalAcceptsStringIDs(t *testing.T) {
	t.Parallel()

	payload := []byte(`{
		"elements": [
			{
				"id": "abc",
				"activityId": "def",
				"capturedAt": 1710000000000,
				"processedAt": 1710000001000,
				"configVersion": 1,
				"owner": "urn:li:person:123",
				"actor": "urn:li:person:456",
				"resourceName": "shares",
				"resourceId": "urn:li:share:999",
				"resourceUri": "urn:li:activity:999",
				"method": "CREATE",
				"activityStatus": "SUCCESS",
				"activity": {"commentary": "hello"}
			}
		],
		"paging": {"start": 0, "count": 1, "total": 1, "links": []}
	}`)

	var resp ChangelogResponse
	if err := json.Unmarshal(payload, &resp); err != nil {
		t.Fatalf("unmarshal changelog response: %v", err)
	}

	event := resp.Elements[0]
	if got := string(event.ID); got != "abc" {
		t.Fatalf("expected string id to remain unchanged, got %q", got)
	}
	if got := string(event.ResourceID); got != "urn:li:share:999" {
		t.Fatalf("expected resourceId to remain unchanged, got %q", got)
	}
}
