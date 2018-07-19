package graphql

import (
	"github.com/MichaelMure/git-bug/bug"
	"github.com/MichaelMure/git-bug/repository"
	"github.com/graphql-go/graphql"
)

func graphqlSchema() (graphql.Schema, error) {
	fields := graphql.Fields{
		"bug": &graphql.Field{
			Type: bugType,
			Args: graphql.FieldConfigArgument{
				"id": &graphql.ArgumentConfig{
					Type: bugIdScalar,
				},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				repo := p.Context.Value("repo").(repository.Repo)
				id, _ := p.Args["id"].(string)
				bug, err := bug.FindBug(repo, id)
				if err != nil {
					return nil, err
				}

				snapshot := bug.Compile()

				return snapshot, nil
			},
		},
	}
	rootQuery := graphql.ObjectConfig{Name: "RootQuery", Fields: fields}
	schemaConfig := graphql.SchemaConfig{Query: graphql.NewObject(rootQuery)}
	return graphql.NewSchema(schemaConfig)
}
