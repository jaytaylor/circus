#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset

cd "$(dirname "$0")"

cd 'quickstart'

echo 0 > /tmp/last

set +o errexit

while true ; do
    # shellcheck disable=SC2016
    fswatch -o -r -m poll_monitor themes \
        | xargs -n1 -IX /bin/bash -c 'now=$(date +%s) ; last="$(cat /tmp/last)" ; secs="$((now-last))" ; echo "INFO: last run was ${secs}s ago" 1>&2 ; if [ ${secs} -gt 2 ] ; then ../generate-site.sh && echo "${now}" > /tmp/last ; else echo "WARN: too soon" 1>&2 ; fi ;'
done

