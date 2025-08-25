#!/bin/bash

archivo=$1
cantidad=$2

compose_template="name: tp0
services:
  server:
    container_name: server
    image: server:latest
    entrypoint: python3 /main.py
    environment:
      - PYTHONUNBUFFERED=1
      - LOGGING_LEVEL=DEBUG
    networks:
      - testing_net
    volumes:
      - ./server/config.ini:/config.ini
"

# $1: Numero del cliente
generate_client() {
    client_name="client"
    client_name+="${1}"
    client_template="
  ${client_name}:
    container_name: ${client_name}
    image: client:latest
    entrypoint: /client
    environment:
      - CLI_ID=${1}
      - CLI_LOG_LEVEL=DEBUG
    networks:
      - testing_net
    depends_on:
      - server
    # Nota mental: esto es siempre un absolute path. No asume por defecto PWD, tenes que pasar el .
    volumes:
      - ./client/config.yaml:/config.yaml
"
    compose_template+="${client_template}"
}


for i in $(seq 1 $cantidad); do
    generate_client $i
done

compose_template+="
networks:
  testing_net:
    ipam:
      driver: default
      config:
        - subnet: 172.25.125.0/24
"

# We write the resulting compose to a file
echo "${compose_template}" > $archivo
