window.BENCHMARK_DATA = {
  "lastUpdate": 1670498943114,
  "repoUrl": "https://github.com/MichaelMure/git-bug",
  "entries": {
    "Benchmark": [
      {
        "commit": {
          "author": {
            "name": "Michael Muré",
            "username": "MichaelMure",
            "email": "batolettre@gmail.com"
          },
          "committer": {
            "name": "Michael Muré",
            "username": "MichaelMure",
            "email": "batolettre@gmail.com"
          },
          "id": "c6bb6b9c7ecddb679966b1561e2e909a9ee5e8cd",
          "message": "benchmark-action: make it work?",
          "timestamp": "2022-11-26T13:03:47Z",
          "url": "https://github.com/MichaelMure/git-bug/commit/c6bb6b9c7ecddb679966b1561e2e909a9ee5e8cd"
        },
        "date": 1669468445443,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkReadBugs5",
            "value": 22037851,
            "unit": "ns/op\t25063470 B/op\t   25160 allocs/op",
            "extra": "52 times\n2 procs"
          },
          {
            "name": "BenchmarkReadBugs25",
            "value": 141376111,
            "unit": "ns/op\t140636597 B/op\t  139658 allocs/op",
            "extra": "9 times\n2 procs"
          },
          {
            "name": "BenchmarkReadBugs150",
            "value": 740724082,
            "unit": "ns/op\t840088524 B/op\t  834876 allocs/op",
            "extra": "2 times\n2 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "batolettre@gmail.com",
            "name": "Michael Muré",
            "username": "MichaelMure"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "562d54802128dc8759f3e071c46f35cbbd6284f7",
          "message": "Merge pull request #941 from MichaelMure/dependabot/go_modules/golang.org/x/text-0.5.0\n\nbuild(deps): bump golang.org/x/text from 0.4.0 to 0.5.0",
          "timestamp": "2022-12-08T12:27:24+01:00",
          "tree_id": "f21a681bee1da3c8c95e09c34555900b31f3a087",
          "url": "https://github.com/MichaelMure/git-bug/commit/562d54802128dc8759f3e071c46f35cbbd6284f7"
        },
        "date": 1670498942297,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkReadBugs5",
            "value": 27525421,
            "unit": "ns/op\t25064172 B/op\t   25162 allocs/op",
            "extra": "44 times\n2 procs"
          },
          {
            "name": "BenchmarkReadBugs25",
            "value": 141762751,
            "unit": "ns/op\t140655269 B/op\t  139677 allocs/op",
            "extra": "8 times\n2 procs"
          },
          {
            "name": "BenchmarkReadBugs150",
            "value": 794673324,
            "unit": "ns/op\t840167388 B/op\t  834949 allocs/op",
            "extra": "2 times\n2 procs"
          }
        ]
      }
    ]
  }
}