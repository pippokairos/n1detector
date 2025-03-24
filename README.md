# N1Detector
Static analysis tool to detect N+1 query issues in a Go codebase

## Prerequisites

Just [Go](https://go.dev)

## Installation

To install the CLI, run

```
go install github.com/pippokairos/n1detector/cmd/n1detector@latest
```

## Usage

```
$ cd path/to/project-to-be-analyzed
$ n1detector ./...

/Users/me/my-project/models/user.go:112:6: Potential N+1 query detected: DB query inside a loop
/Users/me/my-project/internal/user/repository.go:79:10: Potential N+1 query detected: DB query inside a loop
```

## License

The gem is available as open source under the terms of the [MIT License](http://opensource.org/licenses/MIT).
