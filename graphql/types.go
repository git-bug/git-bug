package graphql

import "github.com/graphql-go/graphql"

// Internally, it's the Snapshot
var bugType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Bug",
	Fields: graphql.Fields{
		"id": &graphql.Field{
			Type: graphql.String,
		},
		"status": &graphql.Field{
			Type: graphql.String,
		},
		"comments": &graphql.Field{
			Type: graphql.NewList(commentType),
		},
		"labels": &graphql.Field{
			Type: graphql.NewList(graphql.String),
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
