window.BENCHMARK_DATA = {
  "lastUpdate": 1669468445845,
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
      }
    ]
  }
}