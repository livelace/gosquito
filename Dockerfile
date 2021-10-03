ARG             IMAGE_TAG

FROM            harbor-core.k8s-2.livelace.ru/dev/gosquito:${IMAGE_TAG}

ENV             GOSQUITO_BIN="/usr/local/bin/gosquito"
ENV             GOSQUITO_PROVISION="/usr/local/bin/gosquito_provision.sh"

# copy application.
COPY            "work/gosquito" $GOSQUITO_BIN
COPY            "work/assets/gosquito_provision.sh" $GOSQUITO_PROVISION

USER            "user"

WORKDIR         "/home/user"

CMD             ["/usr/local/bin/gosquito"]
