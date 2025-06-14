#!/bin/bash

# set -e # Removed to allow script to continue on error for individual services

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
XML_DIR="${DIR}/xml"
OUTPUT_BASE_DIR="${DIR}/../../services"

# Arrays for service categorization
declare -a arr_media_renderer=("VirtualLineIn" "GroupRenderingControl" "Queue" "AVTransport" "ConnectionManager" "RenderingControl")
declare -a arr_media_server=("ContentDirectory" "ConnectionManager") # Note: ConnectionManager is in both
declare -a arr_other_services=("AudioIn" "AlarmClock" "MusicServices" "DeviceProperties" "SystemProperties" "ZoneGroupTopology" "GroupManagement" "QPlay")

# Ensure the XML directory exists
if [ ! -d "${XML_DIR}" ]; then
  echo "Error: XML directory ${XML_DIR} not found."
  exit 1
fi

# Find all *1.xml files in the XML_DIR
XML_FILES_FOUND=0
for xml_file in "${XML_DIR}"/*1.xml; do
  # Check if the glob found any files
  if [ ! -f "${xml_file}" ]; then
    if [ ${XML_FILES_FOUND} -eq 0 ]; then # Only print if no files were ever found
        echo "No XML files found matching pattern *1.xml in ${XML_DIR}"
    fi
    # If the pattern expands to itself, it means no files were found.
    # Depending on shell version, this check might be more robust or differ.
    # For bash, if no files match, the loop runs once with xml_file being the pattern itself.
    break # Exit loop if no files are found or after processing all found files
  fi
  XML_FILES_FOUND=1

  # Extract ServiceName from filename (e.g., AVTransport1.xml -> AVTransport)
  FILENAME=$(basename "${xml_file}")
  SERVICENAME="${FILENAME%1.xml}" # Removes '1.xml' suffix

  echo "Processing service: ${SERVICENAME} from file ${FILENAME}"

  CONTROL_ENDPOINT_PATH=""
  EVENT_ENDPOINT_PATH=""

  # Determine endpoint paths based on service name categorization
  # Using a more robust way to check for array membership
  # https://stackoverflow.com/questions/3685970/check-if-an-array-contains-a-value
  if printf '%s\0' "${arr_media_renderer[@]}" | grep -Fxqz -- "${SERVICENAME}"; then
    CONTROL_ENDPOINT_PATH="/MediaRenderer/${SERVICENAME}/Control"
    EVENT_ENDPOINT_PATH="/MediaRenderer/${SERVICENAME}/Event"
  elif printf '%s\0' "${arr_media_server[@]}" | grep -Fxqz -- "${SERVICENAME}"; then
    CONTROL_ENDPOINT_PATH="/MediaServer/${SERVICENAME}/Control"
    EVENT_ENDPOINT_PATH="/MediaServer/${SERVICENAME}/Event"
  elif printf '%s\0' "${arr_other_services[@]}" | grep -Fxqz -- "${SERVICENAME}"; then
    CONTROL_ENDPOINT_PATH="/${SERVICENAME}/Control"
    EVENT_ENDPOINT_PATH="/${SERVICENAME}/Event"
  else
    echo "Warning: Service ${SERVICENAME} not found in predefined categories. Using default path '/${SERVICENAME}/Control'."
    CONTROL_ENDPOINT_PATH="/${SERVICENAME}/Control"
    EVENT_ENDPOINT_PATH="/${SERVICENAME}/Event"
    # To skip if not in category:
    # echo "Warning: Service ${SERVICENAME} not found in predefined categories. Skipping."
    # continue
  fi

  SERVICENAME_LOWERCASE=$(echo "${SERVICENAME}" | tr '[:upper:]' '[:lower:]')
  OUTPUT_DIR="${OUTPUT_BASE_DIR}/${SERVICENAME_LOWERCASE}"
  # Output file is CamelCase as per requirement (SERVICENAME is already CamelCase)
  OUTPUT_FILE="${OUTPUT_DIR}/${SERVICENAME}.go"

  echo "Output directory: ${OUTPUT_DIR}"
  echo "Output file: ${OUTPUT_FILE}"

  mkdir -p "${OUTPUT_DIR}"

  echo "Generating service for ${SERVICENAME}..."
  # Temporary output file in current directory
  TEMP_GO_FILE="${DIR}/${SERVICENAME}.go" # Create temp file in script's dir to avoid clutter

  go run "${DIR}/makeservice.go" "${SERVICENAME}" "${CONTROL_ENDPOINT_PATH}" "${EVENT_ENDPOINT_PATH}" "${xml_file}" > "${TEMP_GO_FILE}"
  GO_RUN_EXIT_CODE=$?

  if [ ${GO_RUN_EXIT_CODE} -ne 0 ]; then
    echo "Error generating service ${SERVICENAME} (exit code: ${GO_RUN_EXIT_CODE}). Skipping."
    rm -f "${TEMP_GO_FILE}" # Clean up temp file
    continue # Continue to the next service
  else
    echo "Service ${SERVICENAME} generated successfully."
  fi

  echo "Running goimports on ${TEMP_GO_FILE}..."
  goimports -w "${TEMP_GO_FILE}"
  GOIMPORTS_EXIT_CODE=$?

  if [ ${GOIMPORTS_EXIT_CODE} -ne 0 ]; then
    echo "Error running goimports on ${TEMP_GO_FILE} (exit code: ${GOIMPORTS_EXIT_CODE}). Skipping."
    rm -f "${TEMP_GO_FILE}" # Clean up temp file
    continue # Continue to the next service
  else
    echo "goimports successful for ${TEMP_GO_FILE}."
  fi

  # Move the processed file to the final destination
  mv "${TEMP_GO_FILE}" "${OUTPUT_FILE}"
  echo "Moved ${TEMP_GO_FILE} to ${OUTPUT_FILE}"
  echo "----------------------------------------"

done

if [ ${XML_FILES_FOUND} -eq 0 ]; then
    echo "No XML service definition files were processed."
    # exit 1 # Optionally exit with error if no files found is critical
fi


echo "Service generation complete."
