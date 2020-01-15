#!/bin/sh

export SERVER_NAME

envsubst '${SERVER_NAME}' < /nginx.conf.tmpl > /etc/nginx/nginx.conf

exec "$@"