build:
	go build

install:
	go install .

goreleaser-snapshot:
	goreleaser -f .goreleaser/gitea.yml release --snapshot --clean