FROM            harbor-core.k8s-2.livelace.ru/infra/gentoo:latest

ENV             GOSQUITO_BIN="/usr/local/bin/gosquito"
ENV             GOSQUITO_PROVISION="/usr/local/bin/gosquito_provision.sh"

# portage packages.
RUN             emerge -G -q \
                dev-vcs/git-crypt \
                dev-libs/librdkafka \
                net-libs/tdlib && \
                rm -rf "/usr/portage/packages"

# copy application.
COPY            "gosquito" $GOSQUITO_BIN
COPY            "assets/gosquito_provision.sh" $GOSQUITO_PROVISION

USER            "user"

WORKDIR         "/home/user"

CMD             ["/usr/local/bin/gosquito"]
