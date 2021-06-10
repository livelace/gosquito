#!/usr/bin/env bash

#-----------------------------------------------------------------------------

echo "*********************************************************************************"
echo "WARNING: This script is intended for auto provision gosquito bundles in clouds."
echo "*********************************************************************************"

#-----------------------------------------------------------------------------

SESSION_NAME="$1"
SOURCE_REPO="$2"
SECRET_PATH="$3"
SET_NAME="$4"

SOURCE_TEMP="/tmp/source"
SET_DIR="${SOURCE_TEMP}/set/${SET_NAME}"

TARGET_PATH="/home/user/.gosquito"
TARGET_PLUGIN_DIR="${TARGET_PATH}/plugin"

CONFIG_FILE="config.toml"
FLOW_FILE="flow.txt"
FLOW_GROUP_FILE="flow_group.txt"

GIRIE_SERVER="${SESSION_NAME}-girie"
WEBCHELA_SERVER="${SESSION_NAME}-webchela"

#-----------------------------------------------------------------------------

function copy_flow() {
  flow_file="$1"

  for flow in $(cat $flow_file);do
    flow_dir=$(dirname "$flow")
    target_flow_dir="${TARGET_PATH}/${flow_dir}"

    mkdir -p "${target_flow_dir}"

    echo "INFO: Copy flow: ${flow}"
    cp "${SOURCE_TEMP}/$flow" "${target_flow_dir}/" || (echo "ERROR: Cannot copy flow: ${flow}" && exit 1)
  done
}

#-----------------------------------------------------------------------------

if [[ ! "$SESSION_NAME" || ! "$SOURCE_REPO" || ! "$SECRET_PATH" || ! "$SET_NAME" ]];then
  echo "ERROR: Usage $0 supersession https://github.com/username/gosquito-source-set.git /path/to/secret superset"
  exit 1
fi

#-----------------------------------------------------------------------------

mkdir -p ${TARGET_PLUGIN_DIR}/{data,state,temp} || (echo "ERROR: Cannot create dir: ${TARGET_PLUGIN_DIR}" && exit 1)

#-----------------------------------------------------------------------------

rm -rf "$SOURCE_TEMP"

git clone "$SOURCE_REPO" "$SOURCE_TEMP" > /dev/null 2>&1 || (echo "ERROR: Cannot clone repo: ${SOURCE_REPO}" && exit 1)

cd "$SET_DIR" || (echo "ERROR: Cannot find set: ${SOURCE_REPO}" && exit 1)

git-crypt unlock "$SECRET_PATH" || (echo "ERROR: Cannot unlock secrets: ${SECRET_PATH}" && exit 1)

#-----------------------------------------------------------------------------

if [[ ! -f "$CONFIG_FILE" || ! -f "$FLOW_FILE" || ! -f "$FLOW_GROUP_FILE" ]];then
  echo "ERROR: Each set must have three files: config.toml, flow.txt, flow_group.txt"
  exit 1
fi

#-----------------------------------------------------------------------------

sed -i "s|<GIRIE_SERVER>|${GIRIE_SERVER}|g" "${CONFIG_FILE}"
sed -i "s|<SESSION_NAME>|${SESSION_NAME}|g" "${CONFIG_FILE}"
sed -i "s|<WEBCHELA_SERVER>|${WEBCHELA_SERVER}|g" "${CONFIG_FILE}"

cp "${CONFIG_FILE}" "${TARGET_PATH}/"

#-----------------------------------------------------------------------------

echo "INFO: Flow file: $FLOW_FILE"
copy_flow "$FLOW_FILE"

#-----------------------------------------------------------------------------

for group_file in $(cat "$FLOW_GROUP_FILE");do
  if [ -f "${SOURCE_TEMP}/${group_file}" ];then
    echo "INFO: Flow group file: ${group_file}"
    copy_flow "${SOURCE_TEMP}/${group_file}"
  else
    echo "ERROR: Flow group file not found: ${SOURCE_TEMP}/${group_file}"
    exit 1
  fi
done

#-----------------------------------------------------------------------------
