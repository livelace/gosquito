#!/usr/bin/env bash

#-----------------------------------------------------------------------------

echo "WARNING:"

#-----------------------------------------------------------------------------

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
  for flow in $(cat $1);do
    flow_dir=$(dirname "$flow")
    target_flow_dir="${TARGET_FLOW_DIR}/${flow_dir}"

    mkdir -p "${target_flow_dir}"

    echo "INFO: Copy flow: ${flow}"
    cp "${SET_TEMP}/flow/$flow" "${target_flow_dir}/"
  done
}

#-----------------------------------------------------------------------------

if [[ ! "$SESSION_NAME" || ! "$SET_REPO" || ! "$SET_NAME" ]];then
  echo "ERROR: Usage $0 supersession https://github.com/username/gosquito-source-set.git superset"
  exit 1
fi

#-----------------------------------------------------------------------------

mkdir -p "${TARGET_FLOW_DIR}" || (echo "ERROR: Cannot create dir: ${TARGET_FLOW_DIR}" && exit 1)
mkdir -p ${TARGET_PLUGIN_DIR}/{data,state,temp} || (echo "ERROR: Cannot create dir: ${TARGET_PLUGIN_DIR}" && exit 1)

#-----------------------------------------------------------------------------

rm -rf "$SET_TEMP"

git clone "$SET_REPO" "$SET_TEMP" || (echo "ERROR: Cannot clone repo: ${SET_REPO}" && exit 1)

cd "${SET_TEMP}/set/${SET_NAME}" || (echo "ERROR: Cannot find set: ${SET_REPO}" && exit 1)

#-----------------------------------------------------------------------------

sed -i "s|<GIRIE_SERVER>|${GIRIE_SERVER}|g" "${CONFIG_FILE}"
sed -i "s|<KAFKA_SERVER>|${KAFKA_SERVER}|g" "${CONFIG_FILE}"
sed -i "s|<SCHEMA_REGISTRY_SERVER>|${SCHEMA_REGISTRY_SERVER}|g" "${CONFIG_FILE}"
sed -i "s|<WEBCHELA_SERVER>|${WEBCHELA_SERVER}|g" "${CONFIG_FILE}"

cp "${CONFIG_FILE}" "${TARGET_PATH}/"

#-----------------------------------------------------------------------------

copy_flow "$FLOW_FILE"

for flow_group_file in $(cat $FLOW_GROUP_FILE);do
  copy_flow "flow_group/${flow_group_file}"
done

#-----------------------------------------------------------------------------



