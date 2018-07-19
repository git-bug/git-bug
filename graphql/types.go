package graphql

import (
	"fmt"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"

	"github.com/MichaelMure/git-bug/bug"
)

func coerceString(value interface{}) interface{} {
	if v, ok := value.(*string); ok {
		return *v
	}
	return fmt.Sprintf("%v", value)
}

var bugIdScalar = graphql.NewScalar(graphql.ScalarConfig{
	Name:        "BugID",
	Description: "TODO",
	Serialize:   coerceString,
	ParseValue:  coerceString,
	ParseLiteral: func(valueAST ast.Value) interface{} {
		switch valueAST := valueAST.(type) {
		case *ast.StringValue:
			return valueAST.Value
		}
		return nil
	},
})

// Internally, it's the Snapshot
var bugType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Bug",
	Fields: graphql.Fields{
		"id": &graphql.Field{
			Type: bugIdScalar,
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				return p.Source.(bug.Snapshot).Id(), nil
			},
		},
		"status": &graphql.Field{
			Type: graphql.String,
		},
		"comments": &graphql.Field{
			Type: graphql.NewList(commentType),
		},
		"labels": &graphql.Field{
			Type: graphql.NewList(graphql.String),
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				return p.Source.(bug.Snapshot).Labels, nil
			},
		},
		// TODO: operations
	},
})

var commentType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Comment",
	Fields: graphql.Fields{
		"author": &graphql.Field{
			Type: personType,
		},
		"message": &graphql.Field{
			Type: graphql.String,
		},
		// TODO: time
	},
})

var personType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Person",
	Fields: graphql.Fields{
		"name": &graphql.Field{
			Type: graphql.String,
		},
		"email": &graphql.Field{
			Type: graphql.String,
		},
	},
})
