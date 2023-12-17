package wampSerializers

type Encodable interface {
	Encode() (any, error)
}

func serializePayload(v any) (any, error) {
	decoder, ok := v.(Encodable)
	if ok {
		payload, e := decoder.Encode()
		return payload, e
	}

	return v, nil
}
