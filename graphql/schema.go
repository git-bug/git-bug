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
		// TODO: provide a relay-like schema with pagination
		"allBugs": &graphql.Field{
			Type: graphql.NewList(bugType),
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				repo := p.Context.Value("repo").(repository.Repo)
				ids, err := repo.ListRefs(bug.BugsRefPattern)

				if err != nil {
					return nil, err
				}

				var snapshots []bug.Snapshot

				for _, ref := range ids {
					bug, err := bug.ReadBug(repo, bug.BugsRefPattern+ref)

					if err != nil {
						return nil, err
					}

					snapshots = append(snapshots, bug.Compile())
				}

				return snapshots, nil
			},
		},
	}
	rootQuery := graphql.ObjectConfig{Name: "RootQuery", Fields: fields}
	schemaConfig := graphql.SchemaConfig{Query: graphql.NewObject(rootQuery)}
	return graphql.NewSchema(schemaConfig)
}
