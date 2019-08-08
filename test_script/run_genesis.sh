#!/bin/bash

# run val1(genesis) node
docker-compose up -d val1

# get val1's tendermint node addr
val1addr=$(docker exec -it val1 tendermint show_node_id | tr -d '\015')

# update seed node's peer set with val1addr on docker-compose.yml 
sed -e s/@val1_addr@/$val1addr/ -i.tmp docker-compose.yml