FROM            scratch

ADD             "work/dracut/rootfs.tar" "/"

ENV             GOSQUITO_BIN="/usr/local/bin/gosquito"
ENV             GOSQUITO_PROVISION="/usr/local/bin/gosquito_provision.sh"

COPY            "work/gosquito" $GOSQUITO_BIN
COPY            "work/assets/gosquito_provision.sh" $GOSQUITO_PROVISION

CMD             ["/usr/local/bin/gosquito"]
