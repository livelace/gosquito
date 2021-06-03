FROM            docker.io/livelace/gentoo:latest

ARG             VERSION

ENV             GOSQUITO_BIN="/usr/local/bin/gosquito"
ENV             GOSQUITO_PROVISION="/usr/local/bin/gosquito_provision.sh"
ENV             GOSQUITO_TEMP="/tmp/gosquito"
ENV             GOSQUITO_URL="https://github.com/livelace/gosquito"

# portage packages.
RUN             emerge -G -q \
                dev-lang/go \
                dev-libs/librdkafka \
                net-libs/tdlib && \
                rm -rf "/usr/portage/packages"

# build application.
RUN             git clone --depth 1 --branch "$VERSION" "$GOSQUITO_URL" "$GOSQUITO_TEMP" && \
                cd "$GOSQUITO_TEMP" && \
                cp "assets/gosquito_provision.sh" "$GOSQUITO_PROVISION" && \
                go build -tags dynamic "github.com/livelace/gosquito/cmd/gosquito" && \
                cp "gosquito" "$GOSQUITO_BIN" && \
                rm -rf "/root/go" "$GOSQUITO_TEMP"

USER            "user"

WORKDIR         "/home/user"

CMD             ["/usr/local/bin/gosquito"]
