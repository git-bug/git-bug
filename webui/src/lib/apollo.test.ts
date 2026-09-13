// The Repository cache policy is easy to get wrong: the type has no id field,
// and giving it an empty key list silently merges every repository into one.

import { readFileSync, readdirSync } from "node:fs";
import { join } from "node:path";

import { InMemoryCache, gql } from "@apollo/client";
import {
  Kind,
  TypeInfo,
  buildSchema,
  getNamedType,
  parse,
  visit,
  visitWithTypeInfo,
  type DocumentNode,
  type FragmentDefinitionNode,
  type SelectionSetNode,
} from "graphql";
import { describe, expect, it } from "vitest";

import { typePolicies } from "./apollo";
import { USER_IDENTITY_QUERY } from "./auth";

const REPOSITORIES = gql`
  query Repositories {
    repositories {
      nodes {
        name
      }
      totalCount
    }
  }
`;

const REPOSITORY = gql`
  query Repository($ref: String) {
    repository(ref: $ref) {
      name
    }
  }
`;

function repositories(names: string[]) {
  return {
    repositories: {
      __typename: "RepositoryConnection",
      nodes: names.map((name) => ({ __typename: "Repository", name })),
      totalCount: names.length,
    },
  };
}

// ── The `name` invariant ─────────────────────────────────────────────────────
//
// A Repository selection without `name` makes Apollo throw, so the query
// returns nothing.  Checked by walking the parsed documents against the schema:
// a regex misses the arg-less `repository {` form, aliases and fragments.

const SCHEMA_DIR = join(process.cwd(), "..", "api", "graphql", "schema");
const SRC_DIR = join(process.cwd(), "src");

function buildProjectSchema() {
  const sdl = readdirSync(SCHEMA_DIR)
    .filter((f) => f.endsWith(".graphql"))
    .map((f) => readFileSync(join(SCHEMA_DIR, f), "utf8"))
    .join("\n");
  return buildSchema(sdl, { assumeValidSDL: true });
}

function sourceFiles() {
  return readdirSync(SRC_DIR, { recursive: true, encoding: "utf8" })
    .filter((f) => /\.tsx?$/.test(f) && !f.includes("__generated__"))
    .map((f) => join(SRC_DIR, f));
}

/** Every `graphql(`…`)` / `gql`…`` template body in a source file. */
function extractDocuments(source: string): string[] {
  const out: string[] = [];
  const opener = /\b(?:graphql|gql)\s*\(?\s*`/g;
  let m: RegExpExecArray | null;
  while ((m = opener.exec(source)) !== null) {
    const end = source.indexOf("`", m.index + m[0].length);
    if (end === -1) continue;
    out.push(source.slice(m.index + m[0].length, end));
    opener.lastIndex = end + 1;
  }
  return out;
}

/** `name` may arrive through a spread, so expand fragments before deciding. */
function selectsName(
  set: SelectionSetNode,
  fragments: Map<string, FragmentDefinitionNode>,
  seen = new Set<string>(),
): boolean {
  return set.selections.some((sel) => {
    if (sel.kind === Kind.FIELD) return sel.name.value === "name";
    if (sel.kind === Kind.INLINE_FRAGMENT) {
      return selectsName(sel.selectionSet, fragments, seen);
    }
    const frag = fragments.get(sel.name.value);
    if (!frag || seen.has(sel.name.value)) return false;
    seen.add(sel.name.value);
    return selectsName(frag.selectionSet, fragments, seen);
  });
}

describe("apollo cache — every Repository selection asks for name", () => {
  it("has no Repository selection missing name", () => {
    const schema = buildProjectSchema();
    const repositoryType = schema.getType("Repository");

    const parsed: { file: string; doc: DocumentNode }[] = [];
    const fragments = new Map<string, FragmentDefinitionNode>();
    for (const file of sourceFiles()) {
      for (const body of extractDocuments(readFileSync(file, "utf8"))) {
        let doc: DocumentNode;
        try {
          doc = parse(body);
        } catch {
          continue; // not a GraphQL document
        }
        parsed.push({ file, doc });
        for (const def of doc.definitions) {
          if (def.kind === Kind.FRAGMENT_DEFINITION) fragments.set(def.name.value, def);
        }
      }
    }
    // Guard against the walk silently finding nothing to check.
    expect(parsed.length).toBeGreaterThan(10);

    const offenders: string[] = [];
    let checked = 0;
    for (const { file, doc } of parsed) {
      const typeInfo = new TypeInfo(schema);
      visit(
        doc,
        visitWithTypeInfo(typeInfo, {
          SelectionSet(node, _key, parent) {
            if (getNamedType(typeInfo.getParentType()) !== repositoryType) return;
            // Skip fragment bodies: the operation spreading them is checked.
            if (parent && "kind" in parent && parent.kind === Kind.FRAGMENT_DEFINITION) return;
            checked++;
            if (!selectsName(node, fragments)) {
              offenders.push(`${file.replace(SRC_DIR, "")} (${node.loc?.startToken.line})`);
            }
          },
        }),
      );
    }

    expect(checked).toBeGreaterThan(10);
    expect(offenders).toEqual([]);
  });

  it("catches a Repository selection that forgets name", () => {
    // The detector must reject the shape that broke auth.
    const schema = buildProjectSchema();
    const doc = parse(`query Bad { repository { userIdentity { id } } }`);
    const offenders: string[] = [];
    const typeInfo = new TypeInfo(schema);
    visit(
      doc,
      visitWithTypeInfo(typeInfo, {
        SelectionSet(node) {
          if (getNamedType(typeInfo.getParentType()) !== schema.getType("Repository")) return;
          if (!selectsName(node, new Map())) offenders.push("bad");
        },
      }),
    );
    expect(offenders).toEqual(["bad"]);
  });

  it("writes the real UserIdentity query without throwing", () => {
    // This query omitted `name` once and broke auth for every user.
    const cache = new InMemoryCache({ typePolicies });
    cache.writeQuery({
      query: USER_IDENTITY_QUERY,
      data: {
        repository: {
          __typename: "Repository",
          name: null,
          userIdentity: {
            __typename: "Identity",
            id: "id1",
            humanId: "h1",
            displayName: "Alice",
            avatarUrl: null,
            name: "alice",
            email: "alice@example.com",
            login: "alice",
          },
        },
      },
    });

    const read = cache.readQuery<{ repository: { userIdentity: { displayName: string } } }>({
      query: USER_IDENTITY_QUERY,
    });
    expect(read?.repository.userIdentity.displayName).toBe("Alice");
    expect(Object.keys(cache.extract())).toContain('Repository:{"name":null}');
  });
});
describe("apollo cache — Repository", () => {
  it("keeps the repositories on the picker page distinct", () => {
    const cache = new InMemoryCache({ typePolicies });
    cache.writeQuery({ query: REPOSITORIES, data: repositories(["alpha", "beta"]) });

    const read = cache.readQuery<{ repositories: { nodes: { name: string }[] } }>({
      query: REPOSITORIES,
    });

    expect(read?.repositories.nodes.map((n) => n.name)).toEqual(["alpha", "beta"]);
  });

  it("keeps a paginated field readable after a later page is written", () => {
    // What fetchMore does: a second page is written under the same repository.
    // If Repository were not normalized, that write would replace the object
    // holding the first page and the original query would stop resolving.
    const cache = new InMemoryCache({ typePolicies });
    const commits = gql`
      query Commits($repo: String, $gitRef: String!, $after: String) {
        repository(ref: $repo) {
          name
          commits(ref: $gitRef, after: $after) {
            nodes {
              hash
            }
          }
        }
      }
    `;
    const page = (hash: string) => ({
      repository: {
        __typename: "Repository",
        name: "repo",
        commits: {
          __typename: "GitCommitConnection",
          nodes: [{ __typename: "GitCommit", hash }],
        },
      },
    });
    const vars = { repo: "repo", gitRef: "main" };
    cache.writeQuery({ query: commits, variables: { ...vars, after: null }, data: page("h1") });
    cache.writeQuery({ query: commits, variables: { ...vars, after: "c1" }, data: page("h2") });

    const first = cache.readQuery<{ repository: { commits: { nodes: { hash: string }[] } } }>({
      query: commits,
      variables: { ...vars, after: null },
    });

    expect(first?.repository.commits.nodes[0]?.hash).toBe("h1");
  });

  it("keeps two named repositories' fields apart", () => {
    const cache = new InMemoryCache({ typePolicies });
    for (const name of ["one", "two"]) {
      cache.writeQuery({
        query: REPOSITORY,
        variables: { ref: name },
        data: { repository: { __typename: "Repository", name } },
      });
    }

    const first = cache.readQuery<{ repository: { name: string } }>({
      query: REPOSITORY,
      variables: { ref: "one" },
    });
    const second = cache.readQuery<{ repository: { name: string } }>({
      query: REPOSITORY,
      variables: { ref: "two" },
    });

    expect([first?.repository.name, second?.repository.name]).toEqual(["one", "two"]);
  });
});
