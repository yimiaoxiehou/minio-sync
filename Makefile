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
	
deploy:
	$(MAKE) build
	docker build . -t docker.utpf.cn/pcmspf/minio-sync
	docker save  docker.utpf.cn/pcmspf/minio-sync:latest | gzip > minio.tgz
	scp minio.tgz root@192.168.10.134:
	ssh root@192.168.10.134 docker load -i minio.tgz &
	scp minio.tgz root@192.168.10.155:
	ssh root@192.168.10.155 "scp minio.tgz 10.100.6.1: && ssh 10.100.6.1 docker load -i minio.tgz"

all:
	$(MAKE) build-docker
	$(MAKE) push-image
	$(MAKE) clean

	