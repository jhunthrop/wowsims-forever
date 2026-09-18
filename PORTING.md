# Forever fork: deliberate divergences from wowsims/classic

This fork tracks `upstream` = https://github.com/wowsims/classic. Everything
here is a change a merge must not silently revert.

## Committed generated protobufs

`sim/core/proto/*.pb.go` is committed; upstream ignores it. The Forever Sixty
site consumes this repository as a Go module at a pinned pseudo-version, and
the Go module system resolves packages from the commit, not from a build step.
`make proto` regenerates. `sim/core/proto/generated_test.go` fails if the
committed output drifts from `proto/*.proto`.

Requires: protoc >= 3.21 and protoc-gen-go v1.36.6
(`go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.6`).

## wasm_exec.js location

Go 1.24 moved `wasm_exec.js` from `$GOROOT/misc/wasm` to `$GOROOT/lib/wasm`.
`vite.build-workers.ts` checks both.

## Build prerequisites for a fresh clone

    go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.6
    export PATH=$PATH:$(go env GOPATH)/bin
    make proto
    make binary_dist/dist.go     # sim/web embeds this; it is gitignored
    go build ./...
    go test --tags=with_db ./sim/...
