// The Repository cache policy is easy to get wrong: the type has no id field,
// and giving it an empty key list silently merges every repository into one.

import { readFileSync, readdirSync } from "node:fs";
import { join } from "node:path";

import { InMemoryCache, gql } from "@apollo/client";
import { describe, expect, it } from "vitest";

import { typePolicies } from "./apollo";

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

// Keying on `name` only works while every Repository selection asks for it.
// A query that forgets it does not fail loudly — Apollo just stores that
// result inline — so the rule is enforced here instead.
describe("apollo cache — every repository selection asks for name", () => {
  it("selects name in every repository(ref:) block", () => {
    const root = join(process.cwd(), "src");
    const files = readdirSync(root, { recursive: true, encoding: "utf8" })
      .filter((f) => /\.tsx?$/.test(f) && !f.includes("__generated__"))
      .map((f) => join(root, f));

    const offenders: string[] = [];
    for (const file of files) {
      const lines = readFileSync(file, "utf8").split("\n");
      lines.forEach((line, i) => {
        if (!/^\s*repository\(ref:\s*\$\w+\)\s*\{\s*$/.test(line)) return;
        let j = i + 1;
        while (j < lines.length && lines[j]?.trim() === "") j++;
        if (!/^\s*name\s*$/.test(lines[j] ?? "")) {
          offenders.push(`${file.replace(root, "")}:${i + 1}`);
        }
      });
    }

    expect(offenders).toEqual([]);
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
