build:
	GOOS=linux GOARCH=amd64 go build -o deepflow-ctl-repo
build-arm:
	GOOS=linux GOARCH=arm64 go build -o deepflow-ctl-repo