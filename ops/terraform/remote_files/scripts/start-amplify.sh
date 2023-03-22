#!/bin/bash
# shellcheck disable=SC1091,SC2312
set -euo pipefail
IFS=$'\n\t'

# import the terraform vars
source /terraform_node/variables

# mount the disk - wait for /dev/sdb to exist
# NB: do not reformat the disk if we can't mount it, unlike the initial
# install/upgrade script!
while [[ ! -e /dev/sdb ]]; do
  sleep 1
  echo "waiting for /dev/sdb to exist"
done
# mount /dev/sdb at /data
mkdir -p /data
mount /dev/sdb /data || true

# import the secrets
if [ -f /data/secrets.sh ] ; then
  source /data/secrets.sh
fi

amplify serve
