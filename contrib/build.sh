#!/bin/bash

NAME="rocksdb_leveldb_builder"
IMAGE="golang:1.14.4-alpine3.12"
ROCKSDB_VER="6.10.1"
LEVELDB_VER="1.20"

CURRENT_PATH=$(pwd)
ARTIFACT=/artifact

# run docker image
docker run -d -it \
    --name=$NAME \
    --volume $CURRENT_PATH:$ARTIFACT \
    --rm \
    $IMAGE

# tools
docker exec -it $NAME apk add bash linux-headers make gcc g++ snappy perl zlib bzip2 lz4 zstd

# build rocksdb
docker exec -it $NAME bash -c \
    "export DEBUG_LEVEL=0 &&
    wget https://github.com/facebook/rocksdb/archive/v$ROCKSDB_VER.tar.gz &&
    tar -xzf v$ROCKSDB_VER.tar.gz && make -C rocksdb-$ROCKSDB_VER shared_lib &&
    strip rocksdb-$ROCKSDB_VER/librocksdb.so* &&
    mkdir -p $ARTIFACT/rocksdb &&
    cp -a rocksdb-$ROCKSDB_VER/librocksdb.so* $ARTIFACT/rocksdb"

# build leveldb
docker exec -it $NAME bash -c \
    "wget https://github.com/google/leveldb/archive/v$LEVELDB_VER.tar.gz &&
    tar -xzf v$LEVELDB_VER.tar.gz && make -C leveldb-$LEVELDB_VER &&
    strip leveldb-$LEVELDB_VER/out-shared/libleveldb.so* &&
    mkdir -p $ARTIFACT/leveldb &&
    cp -a leveldb-$LEVELDB_VER/out-shared/libleveldb.so* $ARTIFACT/leveldb"

# stop docker container
docker stop $NAME
