#!/bin/bash

# This script downloads service control protocol descriptions (SCPD) from a UPnP device.
# It first fetches the main device description, then parses it to find URLs for individual
# service XML files, and downloads each of them into the script's directory.

set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

# Check if IP address is provided
if [ -z "$1" ]; then
  echo "Usage: $0 <TARGET_IP_ADDRESS>" >&2
  exit 1
fi

TARGET_IP="$1"
DEVICE_DESC_URL="http://${TARGET_IP}:1400/xml/device_description.xml"
BASE_URL="http://${TARGET_IP}:1400" # Base URL for constructing SCPD URLs

echo "Fetching device description from ${DEVICE_DESC_URL}..."
DEVICE_DESC_XML=$(curl -s "${DEVICE_DESC_URL}")

if [ -z "${DEVICE_DESC_XML}" ]; then
  echo "Error: Failed to download device_description.xml from ${DEVICE_DESC_URL}. Response was empty." >&2
  exit 1
fi

# Check if curl command was successful by checking if DEVICE_DESC_XML is non-empty
# (set -e would have exited if curl failed with a non-zero exit code,
# but curl might return 0 even if the content is empty or not XML).
# A more robust check might involve trying to parse it as XML if tools were available.
if ! echo "${DEVICE_DESC_XML}" | grep -q "<root"; then
    echo "Error: Downloaded device_description.xml does not seem to be valid XML or is empty." >&2
    # echo "Content was:" >&2
    # echo "${DEVICE_DESC_XML}" >&2 # Be cautious with printing potentially large/binary data
    exit 1
fi

echo "Device description downloaded successfully."
echo "Parsing SCPD URLs and downloading service descriptions..."

# Extract SCPD URLs, then construct full URLs and download them
# The sed command extracts the path, and xargs processes them one by one.
# Ensure paths are correctly handled if they are absolute or relative.
# The original script implies paths are relative to the base URL (e.g. /xml/AVTransport1.xml)
echo "${DEVICE_DESC_XML}" | grep "<SCPDURL>" | sed -e 's/<SCPDURL>\(.*\)<\/SCPDURL>/\1/' | while read -r scpd_path; do
  # Remove leading slash if present, as BASE_URL might have trailing slash or not.
  # However, standard UPnP SCPDURLs are usually absolute paths from the root of the presentation server.
  # The original script's sed command `s/<SCPDURL>\/\(.*\)<\/SCPDURL>/\1/` was capturing after the first slash.
  # Assuming scpd_path is like /xml/Service1.xml or xml/Service1.xml

  # Clean the path: remove potential leading slash for constructing URL,
  # as BASE_URL already specifies the root.
  # The original sed 's/<SCPDURL>\/\(.*\)<\/SCPDURL>/\1/' implies SCPDURL starts with /
  # e.g. <SCPDURL>/xml/AVTransport1.xml</SCPDURL> -> xml/AVTransport1.xml
  # So, no need to strip another leading slash if the sed command is correct.

  # Construct the full URL for the SCPD file
  # If scpd_path already starts with http, it's an absolute URL. Otherwise, prepend BASE_URL.
  if [[ "${scpd_path}" == http* ]]; then
    FULL_SCPD_URL="${scpd_path}"
  else
    # Ensure no double slashes if scpd_path starts with / and BASE_URL ends with /
    # However, typical SCPDURL values are /path/to/file.xml
    # So, the original sed 's/<SCPDURL>\/\(.*\)<\/SCPDURL>/\1/' would result in 'xml/AVTransport1.xml'
    # which means BASE_URL + '/' + scpd_path is needed.
    # Let's adjust sed to be robust: 's_</?SCPDURL>__g' and then trim leading/trailing slashes carefully.
    # Simpler: assume scpd_path is exactly what's needed after BASE_URL:1400
    # The original sed was: sed -e 's/<SCPDURL>\/\(.*\)<\/SCPDURL>/\1/'
    # This means if SCPDURL is /xml/foo.xml, scpd_path becomes xml/foo.xml
    # So, the URL should be http://IP:1400/ + scpd_path
    FULL_SCPD_URL="${BASE_URL}/${scpd_path}"
  fi

  # Extract filename from SCPD path
  FILENAME=$(basename "${scpd_path}")
  OUTPUT_FILE="${DIR}/${FILENAME}"

  echo "Downloading ${FULL_SCPD_URL} to ${OUTPUT_FILE}..."
  curl -s "${FULL_SCPD_URL}" -o "${OUTPUT_FILE}"
  # set -e will cause exit if curl fails
  echo "Downloaded ${FILENAME}."
done

echo "All service descriptions downloaded."
