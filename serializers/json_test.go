package wampSerializers_test

import (
	"testing"

	wamp "github.com/wamp3hub/wamp3go"
	wampSerializer "github.com/wamp3hub/wamp3go/serializers"
	wampShared "github.com/wamp3hub/wamp3go/shared"
)

func testAcceptEventSerializer(t *testing.T, serializer wamp.Serializer) {
	expectedFeatures := wamp.AcceptFeatures{SourceID: wampShared.NewID()}
	event := wamp.MakeAcceptEvent(wampShared.NewID(), &expectedFeatures)
	raw, e := serializer.Encode(event)
	if e != nil {
		t.Fatal(e)
	}
	__event, e := serializer.Decode(raw)
	if e != nil {
		t.Fatal(e)
	}
	event, ok := __event.(wamp.AcceptEvent)
	if !ok {
		t.Fatal("InvalidBehaviour")
	}
	features := event.Features()
	if features.SourceID != expectedFeatures.SourceID {
		t.Fatal("InvalidFeatures")
	}
}

func testPublishEventSerializer(t *testing.T, serializer wamp.Serializer) {
	expectedPayload := "test"
	expectedFeatures := wamp.PublishFeatures{URI: "wamp.test", Include: []string{}, Exclude: []string{wampShared.NewID()}}
	event := wamp.NewPublishEvent(&expectedFeatures, expectedPayload)
	raw, e := serializer.Encode(event)
	if e != nil {
		t.Fatal(e)
	}
	__event, e := serializer.Decode(raw)
	if e != nil {
		t.Fatal(e)
	}
	event, ok := __event.(wamp.PublishEvent)
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

func testCallEventSerializer(t *testing.T, serializer wamp.Serializer) {
	expectedPayload := "test"
	expectedFeatures := wamp.CallFeatures{URI: "wamp.test"}
	event := wamp.NewCallEvent(&expectedFeatures, expectedPayload)
	raw, e := serializer.Encode(event)
	if e != nil {
		t.Fatal(e)
	}
	__event, e := serializer.Decode(raw)
	if e != nil {
		t.Fatal(e)
	}
	event, ok := __event.(wamp.CallEvent)
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

func testReplyEventSerializer(t *testing.T, serializer wamp.Serializer) {
	callEvent := wamp.NewCallEvent(&wamp.CallFeatures{URI: "wamp.test"}, struct{}{})
	expectedPayload := "test"
	event := wamp.NewReplyEvent(callEvent, expectedPayload)
	raw, e := serializer.Encode(event)
	if e != nil {
		t.Fatal(e)
	}
	__event, e := serializer.Decode(raw)
	if e != nil {
		t.Fatal(e)
	}
	event, ok := __event.(wamp.ReplyEvent)
	if !ok {
		t.Fatal("InvalidBehaviour")
	}
	features := event.Features()
	if features.InvocationID != callEvent.ID() {
		t.Fatal("InvalidFeatures")
	}
	payload := new(string)
	event.Payload(payload)
	if *payload != expectedPayload {
		t.Fatal("InvalidPayload")
	}
}

func testCancelEventSerializer(t *testing.T, serializer wamp.Serializer) {
	expectedFeatures := wamp.ReplyFeatures{InvocationID: wampShared.NewID()}
	event := wamp.MakeCancelEvent(wampShared.NewID(), &expectedFeatures)
	raw, e := serializer.Encode(event)
	if e != nil {
		t.Fatal(e)
	}
	__event, e := serializer.Decode(raw)
	if e != nil {
		t.Fatal(e)
	}
	event, ok := __event.(wamp.CancelEvent)
	if !ok {
		t.Fatal("InvalidBehaviour")
	}
	features := event.Features()
	if features.InvocationID != expectedFeatures.InvocationID {
		t.Fatal("InvalidFeatures")
	}
}

func testNextEventSerializer(t *testing.T, serializer wamp.Serializer) {
	expectedFeatures := wamp.NextFeatures{YieldID: wampShared.NewID()}
	event := wamp.MakeNextEvent(wampShared.NewID(), &expectedFeatures)
	raw, e := serializer.Encode(event)
	if e != nil {
		t.Fatal(e)
	}
	__event, e := serializer.Decode(raw)
	if e != nil {
		t.Fatal(e)
	}
	event, ok := __event.(wamp.NextEvent)
	if !ok {
		t.Fatal("InvalidBehaviour")
	}
	features := event.Features()
	if features.YieldID != expectedFeatures.YieldID {
		t.Fatal("InvalidFeatures")
	}
}

func testYieldEventSerializer(t *testing.T, serializer wamp.Serializer) {
	testReplyEventSerializer(t, serializer)
}

func TestHappyPathJSONSerializer(t *testing.T) {
	serializer := new(wampSerializer.JSONSerializer)
	if serializer.Code() != "json" {
		t.Fatal("invalid serializer code")
	}
	testPublishEventSerializer(t, serializer)
	testCallEventSerializer(t, serializer)
	testAcceptEventSerializer(t, serializer)
	testReplyEventSerializer(t, serializer)
	testCancelEventSerializer(t, serializer)
	testNextEventSerializer(t, serializer)
	testYieldEventSerializer(t, serializer)
}
