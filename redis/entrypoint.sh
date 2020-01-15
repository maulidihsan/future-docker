#!/bin/sh
export REDIS_PASS
if [ ! -z "$REDIS_PASS" ]
    envsubst '${REDIS_PASS}' < /redis.conf.tmpl > /etc/redis.conf
fi

exec "$@"