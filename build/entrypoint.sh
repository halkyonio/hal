#!/bin/sh -e

#
# Credits to Deni Bertovic for https://denibertovic.com/posts/handling-permissions-with-docker-volumes/
#

adduser -D -h /home/$LOCAL_USER_NAME -u $LOCAL_USER_ID $LOCAL_USER_NAME
export HOME=/home/$LOCAL_USER_NAME
export USER=$LOCAL_USER_NAME

exec gosu $LOCAL_USER_NAME "$@"
