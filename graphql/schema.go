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
				b, err := bug.FindLocalBug(repo, id)
				if err != nil {
					return nil, err
				}

				snapshot := b.Compile()

				return snapshot, nil
			},
		},
		// TODO: provide a relay-like schema with pagination
		"allBugs": &graphql.Field{
			Type: graphql.NewList(bugType),
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				repo := p.Context.Value("repo").(repository.Repo)

				var snapshots []bug.Snapshot

				for sb := range bug.ReadAllLocalBugs(repo) {
					if sb.Err != nil {
						return nil, sb.Err
					}

					snapshots = append(snapshots, sb.Bug.Compile())
				}

				return snapshots, nil
			},
		},
	}
	rootQuery := graphql.ObjectConfig{Name: "RootQuery", Fields: fields}
	schemaConfig := graphql.SchemaConfig{Query: graphql.NewObject(rootQuery)}
	return graphql.NewSchema(schemaConfig)
}
