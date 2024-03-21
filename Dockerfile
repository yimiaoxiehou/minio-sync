FROM alpine 
ADD dist/minio-sync /minio-sync 
WORKDIR /
ENTRYPOINT ["/minio-sync"]
