FROM scratch
ADD dist/minio-sync /minio-sync 
WORKDIR /
ENTRYPOINT ["/minio-sync"]
