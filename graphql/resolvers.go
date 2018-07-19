package graphql

import "github.com/graphql-go/graphql"

func resolveBug(p graphql.ResolveParams) (interface{}, error) {
	return "world", nil
}
