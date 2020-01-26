#!/bin/bash

USER_COMMANDS=${USER_COMMANDS:="$@"}
RABBITMQ_USER=${RABBITMQ_USER:="admin"}
RABBITMQ_PASS=${RABBITMQ_PASS:="password"}
RABBITMQ_LOGS=${RABBITMQ_LOGS:="/var/log/rabbitmq/rabbit@${HOSTNAME}.log"}

export RABBITMQ_LOGS

if [ ! -f /var/lib/rabbitmq/rabbitmq-server-firstrun ]; then
    sed -i "s/<<\"admin\">>/<<\"${RABBITMQ_USER}\">>/g" "/etc/rabbitmq/rabbitmq.config"
    sed -i "s/<<\"password\">>/<<\"${RABBITMQ_PASS}\">>/g" "/etc/rabbitmq/rabbitmq.config"
    touch /var/lib/rabbitmq/rabbitmq-server-firstrun
fi
/usr/sbin/rabbitmq-server

while true; do if grep -q -i "Server startup complete" $RABBITMQ_LOGS; then break; else sleep 0.5; fi; done
if [[ $USER_COMMANDS ]]; then
    eval $USER_COMMANDS
fi