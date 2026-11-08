#!/bin/bash
set -euo pipefail

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

IP="${1:?usage: $0 <sonos-ip>}"

for i in $(curl -s "http://${IP}:1400/xml/device_description.xml" | grep "<SCPDURL>" | sed -e 's/<SCPDURL>\/\(.*\)<\/SCPDURL>/\1/' | xargs)
do
    curl -s "http://${IP}:1400/${i}" > "${DIR}/${i}"
done
