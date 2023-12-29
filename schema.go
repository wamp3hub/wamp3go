package wamp

import (
	"github.com/invopop/jsonschema"
)

func generateJSONSchema[T any]() *jsonschema.Schema {
	t := new(T)
	return jsonschema.Reflect(t)
}

type EndpointSchema struct {
	Description string             `json:"description"`
	In          *jsonschema.Schema `json:"in"`
	Out         *jsonschema.Schema `json:"out"`
}

func registerSchema[I, O any](
	session *Session,
	uri string,
	options *RegisterOptions,
) {
	schema := EndpointSchema{
		options.Description,
		generateJSONSchema[I](),
		generateJSONSchema[O](),
	}

	Register(
		session,
		uri+".__schema__",
		&RegisterOptions{},
		func(payload any, callEvent CallEvent) (EndpointSchema, error) {
			return schema, nil
		},
	)
}
