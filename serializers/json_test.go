package wampSerializers

import (
	"testing"

	wamp "github.com/wamp3hub/wamp3go"
	wampShared "github.com/wamp3hub/wamp3go/shared"
)

func testPublishEventSerializer(t *testing.T, serializer wamp.Serializer) {
	expectedPayload := "test"
	expectedFeatures := wamp.PublishFeatures{"wamp.test", []string{}, []string{wampShared.NewID()}}
	newEvent := wamp.NewPublishEvent(&expectedFeatures, expectedPayload)
	raw, e := serializer.Encode(newEvent)
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
	expectedFeatures := wamp.CallFeatures{"wamp.test"}
	newEvent := wamp.NewCallEvent(&expectedFeatures, expectedPayload)
	raw, e := serializer.Encode(newEvent)
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

func testAcceptEventSerializer(t *testing.T, serializer wamp.Serializer) {
	// expectedSourceID := wampShared.NewID()
	// newEvent := client.newAcceptEvent(expectedSourceID)
	// raw, e := serializer.Encode(newEvent)
	// if e != nil {
	// 	t.Fatal(e)
	// }
	// __event, e := serializer.Decode(raw)
	// if e != nil {
	// 	t.Fatal(e)
	// }
	// event, ok := __event.(client.AcceptEvent)
	// if !ok {
	// 	t.Fatal("InvalidBehaviour")
	// }
	// features := event.Features()
	// if features.SourceID != expectedSourceID {
	// 	t.Fatal("InvalidFeatures")
	// }
}

func testReplyEventSerializer(t *testing.T, serializer wamp.Serializer) {
	// invocationID := wampShared.NewID()
	// expectedPayload := "test"
	// newEvent := client.NewReplyEvent(invocationID, expectedPayload)
	// raw, e := serializer.Encode(newEvent)
	// if e != nil {
	// 	t.Fatal(e)
	// }
	// __event, e := serializer.Decode(raw)
	// if e != nil {
	// 	t.Fatal(e)
	// }
	// event, ok := __event.(client.ReplyEvent)
	// if !ok {
	// 	t.Fatal("InvalidBehaviour")
	// }
	// features := event.Features()
	// if !(features.InvocationID == invocationID && features.OK == true) {
	// 	t.Fatal("InvalidFeatures")
	// }
	// // TODO check features.Include and features.Exclude
	// payload := new(string)
	// event.Payload(payload)
	// if *payload != expectedPayload {
	// 	t.Fatal("InvalidPayload")
	// }
}

func TestHappyPathJSONSerializer(t *testing.T) {
	serializer := new(JSONSerializer)
	testPublishEventSerializer(t, serializer)
	testCallEventSerializer(t, serializer)
	testAcceptEventSerializer(t, serializer)
	testReplyEventSerializer(t, serializer)
}
