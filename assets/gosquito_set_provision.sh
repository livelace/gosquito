#!/usr/bin/env bash

#-----------------------------------------------------------------------------

echo "WARNING:"

#-----------------------------------------------------------------------------

set -e

SESSION_NAME="$1"
SET_REPO="$2"
SET_NAME="$3"
SET_TEMP="/tmp/source"

TARGET_PATH="/home/user/.gosquito"
TARGET_FLOW_DIR="${TARGET_PATH}/flow"
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
  while read -r flow;do
    flow_dir=$(dirname "$flow")
    target_flow_dir="${TARGET_FLOW_DIR}/${flow_dir}"

    mkdir -p "${target_flow_dir}"
    cp "${SET_TEMP}/flow/$flow" "${target_flow_dir}/"

  done < "$1"
}

#-----------------------------------------------------------------------------

if [[ ! "$SESSION_NAME" || ! "$SET_REPO" || ! "$SET_NAME" ]];then
  echo "ERROR: Usage $0 https://github.com/username/gosquito-set.git superset"
  exit 1
fi

#-----------------------------------------------------------------------------

mkdir -p "${TARGET_FLOW_DIR}"
mkdir -p ${TARGET_PLUGIN_DIR}/{data,state,temp}

#-----------------------------------------------------------------------------

git clone "$SET_REPO" "$SET_TEMP"

cd "${SET_TEMP}/set/${SET_NAME}"

#-----------------------------------------------------------------------------

sed -i "s|<GIRIE_SERVER>|${GIRIE_SERVER}|g" "${CONFIG_FILE}"
sed -i "s|<KAFKA_SERVER>|${KAFKA_SERVER}|g" "${CONFIG_FILE}"
sed -i "s|<SCHEMA_REGISTRY_SERVER>|${SCHEMA_REGISTRY_SERVER}|g" "${CONFIG_FILE}"
sed -i "s|<WEBCHELA_SERVER>|${WEBCHELA_SERVER}|g" "${CONFIG_FILE}"

cp "${CONFIG_FILE}" "${TARGET_PATH}/"

#-----------------------------------------------------------------------------

copy_flow "$FLOW_FILE"

while read -r flow_group_file;do
  copy_flow "$flow_group_file"
done < "$FLOW_GROUP_FILE"

#-----------------------------------------------------------------------------



