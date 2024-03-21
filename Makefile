# Define variables
DIST_DIR := dist

# Define targets and dependencies
build:
	CGO_ENABLED=0 go build -o $(DIST_DIR)/minio-sync main.go

clean:
	rm -rf $(DIST_DIR)
	docker rmi docker.utpf.cn/pcmspf/minio-sync

.PHONY: build-docker
build-docker:
	$(MAKE) build
	docker build . -t docker.utpf.cn/pcmspf/minio-sync

push-image:
	docker push docker.utpf.cn/pcmspf/minio-sync

all:
	$(MAKE) build-docker
	$(MAKE) push-image
	$(MAKE) clean

	