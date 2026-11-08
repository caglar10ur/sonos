#!/bin/bash
set -euo pipefail

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

# ConnectionManager exists under both MediaRenderer and MediaServer with the
# same definition; it is generated once and the MediaServer instance is
# created with connectionmanager.WithPaths.
declare -a arr=("VirtualLineIn" "GroupRenderingControl" "Queue" "AVTransport" "ConnectionManager" "RenderingControl")

for i in "${arr[@]}"
do
    go run "${DIR}/makeservice.go" -control "/MediaRenderer/${i}/Control" -event "/MediaRenderer/${i}/Event" -xml "${DIR}/xml/${i}1.xml" -outputDir services
done

declare -a arr=("ContentDirectory")

for i in "${arr[@]}"
do
    go run "${DIR}/makeservice.go" -control "/MediaServer/${i}/Control" -event "/MediaServer/${i}/Event" -xml "${DIR}/xml/${i}1.xml" -outputDir services
done

declare -a arr=("AudioIn" "AlarmClock" "MusicServices" "DeviceProperties" "SystemProperties" "ZoneGroupTopology" "GroupManagement" "QPlay")

for i in "${arr[@]}"
do
    go run "${DIR}/makeservice.go" -control "/${i}/Control" -event "/${i}/Event" -xml "${DIR}/xml/${i}1.xml" -outputDir services
done
