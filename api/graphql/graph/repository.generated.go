// Code generated by github.com/99designs/gqlgen, DO NOT EDIT.

package graph

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync/atomic"

	"github.com/99designs/gqlgen/graphql"
	"github.com/git-bug/git-bug/api/graphql/models"
	"github.com/vektah/gqlparser/v2/ast"
)

// region    ************************** generated!.gotpl **************************

type RepositoryResolver interface {
	Name(ctx context.Context, obj *models.Repository) (*string, error)
	AllBugs(ctx context.Context, obj *models.Repository, after *string, before *string, first *int, last *int, query *string) (*models.BugConnection, error)
	Bug(ctx context.Context, obj *models.Repository, prefix string) (models.BugWrapper, error)
	AllIdentities(ctx context.Context, obj *models.Repository, after *string, before *string, first *int, last *int) (*models.IdentityConnection, error)
	Identity(ctx context.Context, obj *models.Repository, prefix string) (models.IdentityWrapper, error)
	UserIdentity(ctx context.Context, obj *models.Repository) (models.IdentityWrapper, error)
	ValidLabels(ctx context.Context, obj *models.Repository, after *string, before *string, first *int, last *int) (*models.LabelConnection, error)
}

// endregion ************************** generated!.gotpl **************************

// region    ***************************** args.gotpl *****************************

func (ec *executionContext) field_Repository_allBugs_args(ctx context.Context, rawArgs map[string]interface{}) (map[string]interface{}, error) {
	var err error
	args := map[string]interface{}{}
	var arg0 *string
	if tmp, ok := rawArgs["after"]; ok {
		ctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("after"))
		arg0, err = ec.unmarshalOString2ᚖstring(ctx, tmp)
		if err != nil {
			return nil, err
		}
	}
	args["after"] = arg0
	var arg1 *string
	if tmp, ok := rawArgs["before"]; ok {
		ctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("before"))
		arg1, err = ec.unmarshalOString2ᚖstring(ctx, tmp)
		if err != nil {
			return nil, err
		}
	}
	args["before"] = arg1
	var arg2 *int
	if tmp, ok := rawArgs["first"]; ok {
		ctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("first"))
		arg2, err = ec.unmarshalOInt2ᚖint(ctx, tmp)
		if err != nil {
			return nil, err
		}
	}
	args["first"] = arg2
	var arg3 *int
	if tmp, ok := rawArgs["last"]; ok {
		ctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("last"))
		arg3, err = ec.unmarshalOInt2ᚖint(ctx, tmp)
		if err != nil {
			return nil, err
		}
	}
	args["last"] = arg3
	var arg4 *string
	if tmp, ok := rawArgs["query"]; ok {
		ctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("query"))
		arg4, err = ec.unmarshalOString2ᚖstring(ctx, tmp)
		if err != nil {
			return nil, err
		}
	}
	args["query"] = arg4
	return args, nil
}

func (ec *executionContext) field_Repository_allIdentities_args(ctx context.Context, rawArgs map[string]interface{}) (map[string]interface{}, error) {
	var err error
	args := map[string]interface{}{}
	var arg0 *string
	if tmp, ok := rawArgs["after"]; ok {
		ctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("after"))
		arg0, err = ec.unmarshalOString2ᚖstring(ctx, tmp)
		if err != nil {
			return nil, err
		}
	}
	args["after"] = arg0
	var arg1 *string
	if tmp, ok := rawArgs["before"]; ok {
		ctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("before"))
		arg1, err = ec.unmarshalOString2ᚖstring(ctx, tmp)
		if err != nil {
			return nil, err
		}
	}
	args["before"] = arg1
	var arg2 *int
	if tmp, ok := rawArgs["first"]; ok {
		ctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("first"))
		arg2, err = ec.unmarshalOInt2ᚖint(ctx, tmp)
		if err != nil {
			return nil, err
		}
	}
	args["first"] = arg2
	var arg3 *int
	if tmp, ok := rawArgs["last"]; ok {
		ctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("last"))
		arg3, err = ec.unmarshalOInt2ᚖint(ctx, tmp)
		if err != nil {
			return nil, err
		}
	}
	args["last"] = arg3
	return args, nil
}

func (ec *executionContext) field_Repository_bug_args(ctx context.Context, rawArgs map[string]interface{}) (map[string]interface{}, error) {
	var err error
	args := map[string]interface{}{}
	var arg0 string
	if tmp, ok := rawArgs["prefix"]; ok {
		ctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("prefix"))
		arg0, err = ec.unmarshalNString2string(ctx, tmp)
		if err != nil {
			return nil, err
		}
	}
	args["prefix"] = arg0
	return args, nil
}

func (ec *executionContext) field_Repository_identity_args(ctx context.Context, rawArgs map[string]interface{}) (map[string]interface{}, error) {
	var err error
	args := map[string]interface{}{}
	var arg0 string
	if tmp, ok := rawArgs["prefix"]; ok {
		ctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("prefix"))
		arg0, err = ec.unmarshalNString2string(ctx, tmp)
		if err != nil {
			return nil, err
		}
	}
	args["prefix"] = arg0
	return args, nil
}

func (ec *executionContext) field_Repository_validLabels_args(ctx context.Context, rawArgs map[string]interface{}) (map[string]interface{}, error) {
	var err error
	args := map[string]interface{}{}
	var arg0 *string
	if tmp, ok := rawArgs["after"]; ok {
		ctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("after"))
		arg0, err = ec.unmarshalOString2ᚖstring(ctx, tmp)
		if err != nil {
			return nil, err
		}
	}
	args["after"] = arg0
	var arg1 *string
	if tmp, ok := rawArgs["before"]; ok {
		ctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("before"))
		arg1, err = ec.unmarshalOString2ᚖstring(ctx, tmp)
		if err != nil {
			return nil, err
		}
	}
	args["before"] = arg1
	var arg2 *int
	if tmp, ok := rawArgs["first"]; ok {
		ctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("first"))
		arg2, err = ec.unmarshalOInt2ᚖint(ctx, tmp)
		if err != nil {
			return nil, err
		}
	}
	args["first"] = arg2
	var arg3 *int
	if tmp, ok := rawArgs["last"]; ok {
		ctx := graphql.WithPathContext(ctx, graphql.NewPathWithField("last"))
		arg3, err = ec.unmarshalOInt2ᚖint(ctx, tmp)
		if err != nil {
			return nil, err
		}
	}
	args["last"] = arg3
	return args, nil
}

// endregion ***************************** args.gotpl *****************************

// region    ************************** directives.gotpl **************************

// endregion ************************** directives.gotpl **************************

// region    **************************** field.gotpl *****************************

func (ec *executionContext) _Repository_name(ctx context.Context, field graphql.CollectedField, obj *models.Repository) (ret graphql.Marshaler) {
	fc, err := ec.fieldContext_Repository_name(ctx, field)
	if err != nil {
		return graphql.Null
	}
	ctx = graphql.WithFieldContext(ctx, fc)
	defer func() {
		if r := recover(); r != nil {
			ec.Error(ctx, ec.Recover(ctx, r))
			ret = graphql.Null
		}
	}()
	resTmp, err := ec.ResolverMiddleware(ctx, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return ec.resolvers.Repository().Name(rctx, obj)
	})
	if err != nil {
		ec.Error(ctx, err)
		return graphql.Null
	}
	if resTmp == nil {
		return graphql.Null
	}
	res := resTmp.(*string)
	fc.Result = res
	return ec.marshalOString2ᚖstring(ctx, field.Selections, res)
}

func (ec *executionContext) fieldContext_Repository_name(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Repository",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			return nil, errors.New("field of type String does not have child fields")
		},
	}
	return fc, nil
}

func (ec *executionContext) _Repository_allBugs(ctx context.Context, field graphql.CollectedField, obj *models.Repository) (ret graphql.Marshaler) {
	fc, err := ec.fieldContext_Repository_allBugs(ctx, field)
	if err != nil {
		return graphql.Null
	}
	ctx = graphql.WithFieldContext(ctx, fc)
	defer func() {
		if r := recover(); r != nil {
			ec.Error(ctx, ec.Recover(ctx, r))
			ret = graphql.Null
		}
	}()
	resTmp, err := ec.ResolverMiddleware(ctx, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return ec.resolvers.Repository().AllBugs(rctx, obj, fc.Args["after"].(*string), fc.Args["before"].(*string), fc.Args["first"].(*int), fc.Args["last"].(*int), fc.Args["query"].(*string))
	})
	if err != nil {
		ec.Error(ctx, err)
		return graphql.Null
	}
	if resTmp == nil {
		if !graphql.HasFieldError(ctx, fc) {
			ec.Errorf(ctx, "must not be null")
		}
		return graphql.Null
	}
	res := resTmp.(*models.BugConnection)
	fc.Result = res
	return ec.marshalNBugConnection2ᚖgithubᚗcomᚋgitᚑbugᚋgitᚑbugᚋapiᚋgraphqlᚋmodelsᚐBugConnection(ctx, field.Selections, res)
}

func (ec *executionContext) fieldContext_Repository_allBugs(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Repository",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			switch field.Name {
			case "edges":
				return ec.fieldContext_BugConnection_edges(ctx, field)
			case "nodes":
				return ec.fieldContext_BugConnection_nodes(ctx, field)
			case "pageInfo":
				return ec.fieldContext_BugConnection_pageInfo(ctx, field)
			case "totalCount":
				return ec.fieldContext_BugConnection_totalCount(ctx, field)
			}
			return nil, fmt.Errorf("no field named %q was found under type BugConnection", field.Name)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Repository_allBugs_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Repository_bug(ctx context.Context, field graphql.CollectedField, obj *models.Repository) (ret graphql.Marshaler) {
	fc, err := ec.fieldContext_Repository_bug(ctx, field)
	if err != nil {
		return graphql.Null
	}
	ctx = graphql.WithFieldContext(ctx, fc)
	defer func() {
		if r := recover(); r != nil {
			ec.Error(ctx, ec.Recover(ctx, r))
			ret = graphql.Null
		}
	}()
	resTmp, err := ec.ResolverMiddleware(ctx, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return ec.resolvers.Repository().Bug(rctx, obj, fc.Args["prefix"].(string))
	})
	if err != nil {
		ec.Error(ctx, err)
		return graphql.Null
	}
	if resTmp == nil {
		return graphql.Null
	}
	res := resTmp.(models.BugWrapper)
	fc.Result = res
	return ec.marshalOBug2githubᚗcomᚋgitᚑbugᚋgitᚑbugᚋapiᚋgraphqlᚋmodelsᚐBugWrapper(ctx, field.Selections, res)
}

func (ec *executionContext) fieldContext_Repository_bug(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Repository",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			switch field.Name {
			case "id":
				return ec.fieldContext_Bug_id(ctx, field)
			case "humanId":
				return ec.fieldContext_Bug_humanId(ctx, field)
			case "status":
				return ec.fieldContext_Bug_status(ctx, field)
			case "title":
				return ec.fieldContext_Bug_title(ctx, field)
			case "labels":
				return ec.fieldContext_Bug_labels(ctx, field)
			case "author":
				return ec.fieldContext_Bug_author(ctx, field)
			case "createdAt":
				return ec.fieldContext_Bug_createdAt(ctx, field)
			case "lastEdit":
				return ec.fieldContext_Bug_lastEdit(ctx, field)
			case "actors":
				return ec.fieldContext_Bug_actors(ctx, field)
			case "participants":
				return ec.fieldContext_Bug_participants(ctx, field)
			case "comments":
				return ec.fieldContext_Bug_comments(ctx, field)
			case "timeline":
				return ec.fieldContext_Bug_timeline(ctx, field)
			case "operations":
				return ec.fieldContext_Bug_operations(ctx, field)
			}
			return nil, fmt.Errorf("no field named %q was found under type Bug", field.Name)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Repository_bug_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Repository_allIdentities(ctx context.Context, field graphql.CollectedField, obj *models.Repository) (ret graphql.Marshaler) {
	fc, err := ec.fieldContext_Repository_allIdentities(ctx, field)
	if err != nil {
		return graphql.Null
	}
	ctx = graphql.WithFieldContext(ctx, fc)
	defer func() {
		if r := recover(); r != nil {
			ec.Error(ctx, ec.Recover(ctx, r))
			ret = graphql.Null
		}
	}()
	resTmp, err := ec.ResolverMiddleware(ctx, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return ec.resolvers.Repository().AllIdentities(rctx, obj, fc.Args["after"].(*string), fc.Args["before"].(*string), fc.Args["first"].(*int), fc.Args["last"].(*int))
	})
	if err != nil {
		ec.Error(ctx, err)
		return graphql.Null
	}
	if resTmp == nil {
		if !graphql.HasFieldError(ctx, fc) {
			ec.Errorf(ctx, "must not be null")
		}
		return graphql.Null
	}
	res := resTmp.(*models.IdentityConnection)
	fc.Result = res
	return ec.marshalNIdentityConnection2ᚖgithubᚗcomᚋgitᚑbugᚋgitᚑbugᚋapiᚋgraphqlᚋmodelsᚐIdentityConnection(ctx, field.Selections, res)
}

func (ec *executionContext) fieldContext_Repository_allIdentities(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Repository",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			switch field.Name {
			case "edges":
				return ec.fieldContext_IdentityConnection_edges(ctx, field)
			case "nodes":
				return ec.fieldContext_IdentityConnection_nodes(ctx, field)
			case "pageInfo":
				return ec.fieldContext_IdentityConnection_pageInfo(ctx, field)
			case "totalCount":
				return ec.fieldContext_IdentityConnection_totalCount(ctx, field)
			}
			return nil, fmt.Errorf("no field named %q was found under type IdentityConnection", field.Name)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Repository_allIdentities_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Repository_identity(ctx context.Context, field graphql.CollectedField, obj *models.Repository) (ret graphql.Marshaler) {
	fc, err := ec.fieldContext_Repository_identity(ctx, field)
	if err != nil {
		return graphql.Null
	}
	ctx = graphql.WithFieldContext(ctx, fc)
	defer func() {
		if r := recover(); r != nil {
			ec.Error(ctx, ec.Recover(ctx, r))
			ret = graphql.Null
		}
	}()
	resTmp, err := ec.ResolverMiddleware(ctx, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return ec.resolvers.Repository().Identity(rctx, obj, fc.Args["prefix"].(string))
	})
	if err != nil {
		ec.Error(ctx, err)
		return graphql.Null
	}
	if resTmp == nil {
		return graphql.Null
	}
	res := resTmp.(models.IdentityWrapper)
	fc.Result = res
	return ec.marshalOIdentity2githubᚗcomᚋgitᚑbugᚋgitᚑbugᚋapiᚋgraphqlᚋmodelsᚐIdentityWrapper(ctx, field.Selections, res)
}

func (ec *executionContext) fieldContext_Repository_identity(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Repository",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			switch field.Name {
			case "id":
				return ec.fieldContext_Identity_id(ctx, field)
			case "humanId":
				return ec.fieldContext_Identity_humanId(ctx, field)
			case "name":
				return ec.fieldContext_Identity_name(ctx, field)
			case "email":
				return ec.fieldContext_Identity_email(ctx, field)
			case "login":
				return ec.fieldContext_Identity_login(ctx, field)
			case "displayName":
				return ec.fieldContext_Identity_displayName(ctx, field)
			case "avatarUrl":
				return ec.fieldContext_Identity_avatarUrl(ctx, field)
			case "isProtected":
				return ec.fieldContext_Identity_isProtected(ctx, field)
			}
			return nil, fmt.Errorf("no field named %q was found under type Identity", field.Name)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Repository_identity_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

func (ec *executionContext) _Repository_userIdentity(ctx context.Context, field graphql.CollectedField, obj *models.Repository) (ret graphql.Marshaler) {
	fc, err := ec.fieldContext_Repository_userIdentity(ctx, field)
	if err != nil {
		return graphql.Null
	}
	ctx = graphql.WithFieldContext(ctx, fc)
	defer func() {
		if r := recover(); r != nil {
			ec.Error(ctx, ec.Recover(ctx, r))
			ret = graphql.Null
		}
	}()
	resTmp, err := ec.ResolverMiddleware(ctx, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return ec.resolvers.Repository().UserIdentity(rctx, obj)
	})
	if err != nil {
		ec.Error(ctx, err)
		return graphql.Null
	}
	if resTmp == nil {
		return graphql.Null
	}
	res := resTmp.(models.IdentityWrapper)
	fc.Result = res
	return ec.marshalOIdentity2githubᚗcomᚋgitᚑbugᚋgitᚑbugᚋapiᚋgraphqlᚋmodelsᚐIdentityWrapper(ctx, field.Selections, res)
}

func (ec *executionContext) fieldContext_Repository_userIdentity(_ context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Repository",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			switch field.Name {
			case "id":
				return ec.fieldContext_Identity_id(ctx, field)
			case "humanId":
				return ec.fieldContext_Identity_humanId(ctx, field)
			case "name":
				return ec.fieldContext_Identity_name(ctx, field)
			case "email":
				return ec.fieldContext_Identity_email(ctx, field)
			case "login":
				return ec.fieldContext_Identity_login(ctx, field)
			case "displayName":
				return ec.fieldContext_Identity_displayName(ctx, field)
			case "avatarUrl":
				return ec.fieldContext_Identity_avatarUrl(ctx, field)
			case "isProtected":
				return ec.fieldContext_Identity_isProtected(ctx, field)
			}
			return nil, fmt.Errorf("no field named %q was found under type Identity", field.Name)
		},
	}
	return fc, nil
}

func (ec *executionContext) _Repository_validLabels(ctx context.Context, field graphql.CollectedField, obj *models.Repository) (ret graphql.Marshaler) {
	fc, err := ec.fieldContext_Repository_validLabels(ctx, field)
	if err != nil {
		return graphql.Null
	}
	ctx = graphql.WithFieldContext(ctx, fc)
	defer func() {
		if r := recover(); r != nil {
			ec.Error(ctx, ec.Recover(ctx, r))
			ret = graphql.Null
		}
	}()
	resTmp, err := ec.ResolverMiddleware(ctx, func(rctx context.Context) (interface{}, error) {
		ctx = rctx // use context from middleware stack in children
		return ec.resolvers.Repository().ValidLabels(rctx, obj, fc.Args["after"].(*string), fc.Args["before"].(*string), fc.Args["first"].(*int), fc.Args["last"].(*int))
	})
	if err != nil {
		ec.Error(ctx, err)
		return graphql.Null
	}
	if resTmp == nil {
		if !graphql.HasFieldError(ctx, fc) {
			ec.Errorf(ctx, "must not be null")
		}
		return graphql.Null
	}
	res := resTmp.(*models.LabelConnection)
	fc.Result = res
	return ec.marshalNLabelConnection2ᚖgithubᚗcomᚋgitᚑbugᚋgitᚑbugᚋapiᚋgraphqlᚋmodelsᚐLabelConnection(ctx, field.Selections, res)
}

func (ec *executionContext) fieldContext_Repository_validLabels(ctx context.Context, field graphql.CollectedField) (fc *graphql.FieldContext, err error) {
	fc = &graphql.FieldContext{
		Object:     "Repository",
		Field:      field,
		IsMethod:   true,
		IsResolver: true,
		Child: func(ctx context.Context, field graphql.CollectedField) (*graphql.FieldContext, error) {
			switch field.Name {
			case "edges":
				return ec.fieldContext_LabelConnection_edges(ctx, field)
			case "nodes":
				return ec.fieldContext_LabelConnection_nodes(ctx, field)
			case "pageInfo":
				return ec.fieldContext_LabelConnection_pageInfo(ctx, field)
			case "totalCount":
				return ec.fieldContext_LabelConnection_totalCount(ctx, field)
			}
			return nil, fmt.Errorf("no field named %q was found under type LabelConnection", field.Name)
		},
	}
	defer func() {
		if r := recover(); r != nil {
			err = ec.Recover(ctx, r)
			ec.Error(ctx, err)
		}
	}()
	ctx = graphql.WithFieldContext(ctx, fc)
	if fc.Args, err = ec.field_Repository_validLabels_args(ctx, field.ArgumentMap(ec.Variables)); err != nil {
		ec.Error(ctx, err)
		return fc, err
	}
	return fc, nil
}

// endregion **************************** field.gotpl *****************************

// region    **************************** input.gotpl *****************************

// endregion **************************** input.gotpl *****************************

// region    ************************** interface.gotpl ***************************

// endregion ************************** interface.gotpl ***************************

// region    **************************** object.gotpl ****************************

var repositoryImplementors = []string{"Repository"}

func (ec *executionContext) _Repository(ctx context.Context, sel ast.SelectionSet, obj *models.Repository) graphql.Marshaler {
	fields := graphql.CollectFields(ec.OperationContext, sel, repositoryImplementors)

	out := graphql.NewFieldSet(fields)
	deferred := make(map[string]*graphql.FieldSet)
	for i, field := range fields {
		switch field.Name {
		case "__typename":
			out.Values[i] = graphql.MarshalString("Repository")
		case "name":
			field := field

			innerFunc := func(ctx context.Context, _ *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Repository_name(ctx, field, obj)
				return res
			}

			if field.Deferrable != nil {
				dfs, ok := deferred[field.Deferrable.Label]
				di := 0
				if ok {
					dfs.AddField(field)
					di = len(dfs.Values) - 1
				} else {
					dfs = graphql.NewFieldSet([]graphql.CollectedField{field})
					deferred[field.Deferrable.Label] = dfs
				}
				dfs.Concurrently(di, func(ctx context.Context) graphql.Marshaler {
					return innerFunc(ctx, dfs)
				})

				// don't run the out.Concurrently() call below
				out.Values[i] = graphql.Null
				continue
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
		case "allBugs":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Repository_allBugs(ctx, field, obj)
				if res == graphql.Null {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			if field.Deferrable != nil {
				dfs, ok := deferred[field.Deferrable.Label]
				di := 0
				if ok {
					dfs.AddField(field)
					di = len(dfs.Values) - 1
				} else {
					dfs = graphql.NewFieldSet([]graphql.CollectedField{field})
					deferred[field.Deferrable.Label] = dfs
				}
				dfs.Concurrently(di, func(ctx context.Context) graphql.Marshaler {
					return innerFunc(ctx, dfs)
				})

				// don't run the out.Concurrently() call below
				out.Values[i] = graphql.Null
				continue
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
		case "bug":
			field := field

			innerFunc := func(ctx context.Context, _ *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Repository_bug(ctx, field, obj)
				return res
			}

			if field.Deferrable != nil {
				dfs, ok := deferred[field.Deferrable.Label]
				di := 0
				if ok {
					dfs.AddField(field)
					di = len(dfs.Values) - 1
				} else {
					dfs = graphql.NewFieldSet([]graphql.CollectedField{field})
					deferred[field.Deferrable.Label] = dfs
				}
				dfs.Concurrently(di, func(ctx context.Context) graphql.Marshaler {
					return innerFunc(ctx, dfs)
				})

				// don't run the out.Concurrently() call below
				out.Values[i] = graphql.Null
				continue
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
		case "allIdentities":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Repository_allIdentities(ctx, field, obj)
				if res == graphql.Null {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			if field.Deferrable != nil {
				dfs, ok := deferred[field.Deferrable.Label]
				di := 0
				if ok {
					dfs.AddField(field)
					di = len(dfs.Values) - 1
				} else {
					dfs = graphql.NewFieldSet([]graphql.CollectedField{field})
					deferred[field.Deferrable.Label] = dfs
				}
				dfs.Concurrently(di, func(ctx context.Context) graphql.Marshaler {
					return innerFunc(ctx, dfs)
				})

				// don't run the out.Concurrently() call below
				out.Values[i] = graphql.Null
				continue
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
		case "identity":
			field := field

			innerFunc := func(ctx context.Context, _ *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Repository_identity(ctx, field, obj)
				return res
			}

			if field.Deferrable != nil {
				dfs, ok := deferred[field.Deferrable.Label]
				di := 0
				if ok {
					dfs.AddField(field)
					di = len(dfs.Values) - 1
				} else {
					dfs = graphql.NewFieldSet([]graphql.CollectedField{field})
					deferred[field.Deferrable.Label] = dfs
				}
				dfs.Concurrently(di, func(ctx context.Context) graphql.Marshaler {
					return innerFunc(ctx, dfs)
				})

				// don't run the out.Concurrently() call below
				out.Values[i] = graphql.Null
				continue
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
		case "userIdentity":
			field := field

			innerFunc := func(ctx context.Context, _ *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Repository_userIdentity(ctx, field, obj)
				return res
			}

			if field.Deferrable != nil {
				dfs, ok := deferred[field.Deferrable.Label]
				di := 0
				if ok {
					dfs.AddField(field)
					di = len(dfs.Values) - 1
				} else {
					dfs = graphql.NewFieldSet([]graphql.CollectedField{field})
					deferred[field.Deferrable.Label] = dfs
				}
				dfs.Concurrently(di, func(ctx context.Context) graphql.Marshaler {
					return innerFunc(ctx, dfs)
				})

				// don't run the out.Concurrently() call below
				out.Values[i] = graphql.Null
				continue
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
		case "validLabels":
			field := field

			innerFunc := func(ctx context.Context, fs *graphql.FieldSet) (res graphql.Marshaler) {
				defer func() {
					if r := recover(); r != nil {
						ec.Error(ctx, ec.Recover(ctx, r))
					}
				}()
				res = ec._Repository_validLabels(ctx, field, obj)
				if res == graphql.Null {
					atomic.AddUint32(&fs.Invalids, 1)
				}
				return res
			}

			if field.Deferrable != nil {
				dfs, ok := deferred[field.Deferrable.Label]
				di := 0
				if ok {
					dfs.AddField(field)
					di = len(dfs.Values) - 1
				} else {
					dfs = graphql.NewFieldSet([]graphql.CollectedField{field})
					deferred[field.Deferrable.Label] = dfs
				}
				dfs.Concurrently(di, func(ctx context.Context) graphql.Marshaler {
					return innerFunc(ctx, dfs)
				})

				// don't run the out.Concurrently() call below
				out.Values[i] = graphql.Null
				continue
			}

			out.Concurrently(i, func(ctx context.Context) graphql.Marshaler { return innerFunc(ctx, out) })
		default:
			panic("unknown field " + strconv.Quote(field.Name))
		}
	}
	out.Dispatch(ctx)
	if out.Invalids > 0 {
		return graphql.Null
	}

	atomic.AddInt32(&ec.deferred, int32(len(deferred)))

	for label, dfs := range deferred {
		ec.processDeferredGroup(graphql.DeferredGroup{
			Label:    label,
			Path:     graphql.GetPath(ctx),
			FieldSet: dfs,
			Context:  ctx,
		})
	}

	return out
}

// endregion **************************** object.gotpl ****************************

// region    ***************************** type.gotpl *****************************

func (ec *executionContext) marshalORepository2ᚖgithubᚗcomᚋgitᚑbugᚋgitᚑbugᚋapiᚋgraphqlᚋmodelsᚐRepository(ctx context.Context, sel ast.SelectionSet, v *models.Repository) graphql.Marshaler {
	if v == nil {
		return graphql.Null
	}
	return ec._Repository(ctx, sel, v)
}

// endregion ***************************** type.gotpl *****************************
