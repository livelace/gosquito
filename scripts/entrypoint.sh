#!/usr/bin/env bash

#------------------------------------------------------------------------------

export ANSIBLE_CONFIG=/work/ansible/ansible.cfg
export HOME=/data
export LANG=en_US.UTF-8
export LANGUAGE=en_US.UTF-8

CONF_PATH="/conf"
CONF_AUTH_PATH="${CONF_PATH}/auth"
SSH_KEY="${CONF_AUTH_PATH}/multicloud"

INVENTORY_SAMPLE="inventory-sample.ini"
INVENTORY_SAMPLE_FILE="/work/ansible/${INVENTORY_SAMPLE}"

INVENTORY="inventory.ini"
INVENTORY_FILE="/conf/${INVENTORY}"



#------------------------------------------------------------------------------

if [[ "$@" =~ help ]];then
HELP_CONTENT=`cat<<EOF

Usage:

init            - Initialize sample configuration.
genconf [cloud] - Generate Packer templates, Terraform plans and other stuff based on provided configuration.
build   [cloud] - Build virtual machines images.
deploy  [cloud] - Deploy clouds infrastructures.
destroy [cloud] - Destroy clouds infrastructures.
clean           - Clean produced data during builds and deployments (doesn't affect configuration).
shell           - Execute shell (useful for debugging).

EOF`
    echo "$HELP_CONTENT"
    exit 0
fi

#------------------------------------------------------------------------------

ACTION="$1"
ARG="$2"

#------------------------------------------------------------------------------

if [[ "$ACTION" == "init" ]];then
    root_password=`pwgen 8 1`

    cp "$INVENTORY_SAMPLE_FILE" "/${CONF_PATH}/${INVENTORY_SAMPLE}"
    sed -i "s/PASSWORD/${root_password}/" "/${CONF_PATH}/${INVENTORY_SAMPLE}"
    mkdir -p "${CONF_AUTH_PATH}" && ssh-keygen -q -t rsa -N "" -f "$SSH_KEY"

    echo "INFO: Sample configuration file was placed into configuration directory."

    exit 0

elif [[ "$ACTION" = "genconf" || "$ACTION" = "build" || "$ACTION" = "deploy" || "$ACTION" = "destroy" ]];then

    if [[ ! -f "$INVENTORY_FILE" ]];then
        echo "ERROR: Cannot find configuration file: \"${INVENTORY}\"."
        exit 1
    elif [[ "$ACTION" = "build" && ! -c "/dev/kvm" ]];then
        echo "ERROR: Cannot find KVM device: /dev/kvm"
        exit 1
    else

        if [[ "$ARG" ]];then
            ansible-playbook \
                -i "$INVENTORY_FILE" \
                -l "$ARG" \
                -e target_stage=${ACTION} "/work/ansible/site.yaml"
        else
            ansible-playbook \
                -i "$INVENTORY_FILE" \
                -e target_stage=${ACTION} "/work/ansible/site.yaml"
        fi

    fi

    exit 0

elif [[ "$ACTION" = "clean" ]];then
    echo "INFO: Perform cleanup. You have 10 seconds to change your mind [CTRL+C] ..."
    sleep 10
    rm -rf /data/* /data/.* >/dev/null 2>&1

elif [[ "$ACTION" = "shell" ]];then
    exec /bin/bash
else
    echo "ERROR: Unknown action: ${ACTION}"
    exit 1
fi



#------------------------------------------------------------------------------
