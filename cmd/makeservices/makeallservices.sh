#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

declare -a arr=("VirtualLineIn" "GroupRenderingControl" "Queue" "AVTransport" "ConnectionManager" "RenderingControl")

for i in "${arr[@]}"
do
    go run ${DIR}/makeservice.go -control "/MediaRenderer/${i}/Control" -event "/MediaRenderer/${i}/Event" -xml ${DIR}/xml/${i}1.xml -outputDir services
    goimports -w services/${i}/${i}.go
done

declare -a arr=("ContentDirectory" "ConnectionManager")

for i in "${arr[@]}"
do
    go run ${DIR}/makeservice.go -control "/MediaServer/${i}/Control" -event "/MediaServer/${i}/Event" -xml ${DIR}/xml/${i}1.xml -outputDir services
    goimports -w services/${i}/${i}.go
done

declare -a arr=("AudioIn" "AlarmClock" "MusicServices" "DeviceProperties" "SystemProperties" "ZoneGroupTopology" "GroupManagement" "QPlay")

for i in "${arr[@]}"
do
    go run ${DIR}/makeservice.go -control "/${i}/Control" -event "/${i}/Event" -xml ${DIR}/xml/${i}1.xml -outputDir services
    goimports -w services/${i}/${i}.go
done
