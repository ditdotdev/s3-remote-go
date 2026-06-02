# Project Development

For general information about contributing changes, see the
[Contributor Guidelines](https://github.com/ditdotdev/.github/blob/master/CONTRIBUTING.md).

## How it Works

The provider uses the Dit `remote-sdk-go` to provide interfaces for
Dit to use.

## Building

Run `go build -v ./...`.

## Testing

Run `go test -v ./...`.

## Releasing

Push a tag of the form `v<X>.<Y>.<Z>`, and publish the draft release in GitHub.
