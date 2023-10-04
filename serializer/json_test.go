package serializer

import (
	"testing"

	client "github.com/wamp3hub/wamp3go"

	"github.com/rs/xid"
)

func testPublishEventSerializer(t *testing.T, serializer client.Serializer) {
	expectedPayload := "test"
	expectedFeatures := client.PublishFeatures{"wamp.test", []string{}, []string{xid.New().String()}}
	newEvent := client.NewPublishEvent(&expectedFeatures, expectedPayload)
	raw, e := serializer.Encode(newEvent)
	if e != nil {
		t.Fatal(e)
	}
	__event, e := serializer.Decode(raw)
	if e != nil {
		t.Fatal(e)
	}
	event, ok := __event.(client.PublishEvent)
	if !ok {
		t.Fatal("InvalidBehaviour")
	}
	features := event.Features()
	if features.URI != expectedFeatures.URI {
		t.Fatal("InvalidFeatures")
	}
	// TODO check features.Include and features.Exclude
	payload := new(string)
	event.Payload(payload)
	if *payload != expectedPayload {
		t.Fatal("InvalidPayload")
	}
}

func testCallEventSerializer(t *testing.T, serializer client.Serializer) {
	expectedPayload := "test"
	expectedFeatures := client.CallFeatures{"wamp.test"}
	newEvent := client.NewCallEvent(&expectedFeatures, expectedPayload)
	raw, e := serializer.Encode(newEvent)
	if e != nil {
		t.Fatal(e)
	}
	__event, e := serializer.Decode(raw)
	if e != nil {
		t.Fatal(e)
	}
	event, ok := __event.(client.CallEvent)
	if !ok {
		t.Fatal("InvalidBehaviour")
	}
	features := event.Features()
	if features.URI != expectedFeatures.URI {
		t.Fatal("InvalidFeatures")
	}
	// TODO check features.Include and features.Exclude
	payload := new(string)
	event.Payload(payload)
	if *payload != expectedPayload {
		t.Fatal("InvalidPayload")
	}
}

func testAcceptEventSerializer(t *testing.T, serializer client.Serializer) {
	expectedSourceID := xid.New().String()
	newEvent := client.NewAcceptEvent(expectedSourceID)
	raw, e := serializer.Encode(newEvent)
	if e != nil {
		t.Fatal(e)
	}
	__event, e := serializer.Decode(raw)
	if e != nil {
		t.Fatal(e)
	}
	event, ok := __event.(client.AcceptEvent)
	if !ok {
		t.Fatal("InvalidBehaviour")
	}
	features := event.Features()
	if features.SourceID != expectedSourceID {
		t.Fatal("InvalidFeatures")
	}
}

func testReplyEventSerializer(t *testing.T, serializer client.Serializer) {
	invocationID := xid.New().String()
	expectedPayload := "test"
	newEvent := client.NewReplyEvent(invocationID, expectedPayload)
	raw, e := serializer.Encode(newEvent)
	if e != nil {
		t.Fatal(e)
	}
	__event, e := serializer.Decode(raw)
	if e != nil {
		t.Fatal(e)
	}
	event, ok := __event.(client.ReplyEvent)
	if !ok {
		t.Fatal("InvalidBehaviour")
	}
	features := event.Features()
	if !(features.InvocationID == invocationID && features.OK == true) {
		t.Fatal("InvalidFeatures")
	}
	// TODO check features.Include and features.Exclude
	payload := new(string)
	event.Payload(payload)
	if *payload != expectedPayload {
		t.Fatal("InvalidPayload")
	}
}

func TestHappyPathJSONSerializer(t *testing.T) {
	serializer := new(JSONSerializer)
	testPublishEventSerializer(t, serializer)
	testCallEventSerializer(t, serializer)
	testAcceptEventSerializer(t, serializer)
	testReplyEventSerializer(t, serializer)
}
