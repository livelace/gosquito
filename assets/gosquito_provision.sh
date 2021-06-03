#!/usr/bin/env bash

#-----------------------------------------------------------------------------

echo "WARNING:"

#-----------------------------------------------------------------------------

SESSION_NAME="$1"
SOURCE_REPO="$2"
SET_NAME="$3"

SOURCE_TEMP="/tmp/source"
SET_DIR="${SOURCE_TEMP}/set/${SET_NAME}"

TARGET_PATH="/home/user/.gosquito"
TARGET_PLUGIN_DIR="${TARGET_PATH}/plugin"

CONFIG_FILE="config.toml"
FLOW_FILE="flow.txt"
FLOW_GROUP_FILE="flow_group.txt"

GIRIE_SERVER="${SESSION_NAME}-girie"
KAFKA_SERVER="${SESSION_NAME}-kafka"
SCHEMA_REGISTRY_SERVER="${SESSION_NAME}-kafka"
WEBCHELA_SERVER="${SESSION_NAME}-webchela"

#-----------------------------------------------------------------------------

function copy_flow() {
  flow_file="$1"

  for flow in $(cat $flow_file);do
    flow_dir=$(dirname "$flow")
    target_flow_dir="${TARGET_PATH}/${flow_dir}"

    mkdir -p "${target_flow_dir}"

    echo "INFO: Copy flow: ${flow}"
    cp "${SOURCE_TEMP}/$flow" "${target_flow_dir}/"
  done
}

#-----------------------------------------------------------------------------

if [[ ! "$SESSION_NAME" || ! "$SOURCE_REPO" || ! "$SET_NAME" ]];then
  echo "ERROR: Usage $0 supersession https://github.com/username/gosquito-source-set.git superset"
  exit 1
fi

#-----------------------------------------------------------------------------

mkdir -p ${TARGET_PLUGIN_DIR}/{data,state,temp} || (echo "ERROR: Cannot create dir: ${TARGET_PLUGIN_DIR}" && exit 1)

#-----------------------------------------------------------------------------

rm -rf "$SOURCE_TEMP"

git clone "$SOURCE_REPO" "$SOURCE_TEMP" || (echo "ERROR: Cannot clone repo: ${SOURCE_REPO}" && exit 1)

cd "$SET_DIR" || (echo "ERROR: Cannot find set: ${SOURCE_REPO}" && exit 1)

#-----------------------------------------------------------------------------

sed -i "s|<GIRIE_SERVER>|${GIRIE_SERVER}|g" "${CONFIG_FILE}"
sed -i "s|<KAFKA_SERVER>|${KAFKA_SERVER}|g" "${CONFIG_FILE}"
sed -i "s|<SCHEMA_REGISTRY_SERVER>|${SCHEMA_REGISTRY_SERVER}|g" "${CONFIG_FILE}"
sed -i "s|<WEBCHELA_SERVER>|${WEBCHELA_SERVER}|g" "${CONFIG_FILE}"

cp "${CONFIG_FILE}" "${TARGET_PATH}/"

#-----------------------------------------------------------------------------

copy_flow "$FLOW_FILE"

for group_file in $(cat "$FLOW_GROUP_FILE");do
  copy_flow "${SOURCE_TEMP}/${group_file}"
done

#-----------------------------------------------------------------------------



