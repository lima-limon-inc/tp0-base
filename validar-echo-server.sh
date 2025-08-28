#!/bin/sh

# De https://docs.docker.com/engine/network/#user-defined-networks
# You can create custom, user-defined networks, and connect multiple containers
# to the same network. Once connected to a user-defined network, containers can
# communicate with each other using container IP addresses or container names.
NETWORK="tp0_testing_net"

# Address del server
SERVER_ADDR=$(grep address client/config.yaml | sed 's/address://'g | sed 's/[ \"]//'g)

SERVER_IP=$( echo $SERVER_ADDR  | awk -F ":" {'print $1'})
SERVER_PORT=$( echo $SERVER_ADDR  | awk -F ":" {'print $2'})

# Uso el wait time del config.yaml del cliente
WAIT=$(grep amount client/config.yaml | sed 's/amount://'g | sed 's/[ \"]//'g)

MENSAJE="I am alive"

# busybox: es el nombre de la imagen del container
# formato para mandar texto via netcat, fuente: https://stackoverflow.com/a/37389512/13683575
RESPUESTA=$(docker run --rm --network name=${NETWORK} busybox sh -c "echo ${MENSAJE} | nc -w ${WAIT} ${SERVER_IP} ${SERVER_PORT}")

if [ "${RESPUESTA}" = "${MENSAJE}"  ]; then
    echo "action: test_echo_server | result: success"
else
    echo "action: test_echo_server | result: fail"
fi
