FROM            docker.io/livelace/gentoo:latest

ENV             GOSQUITO_BIN="/usr/local/bin/gosquito"
ENV             GOSQUITO_TEMP="/tmp/gosquito"
ENV             GOSQUITO_URL="https://github.com/livelace/gosquito"

# portage packages.
RUN             emerge -G -q \
                dev-lang/go \
                dev-libs/librdkafka \
                net-libs/tdlib && \
                rm -rf "/usr/portage/packages"

# build application.
RUN             git clone "$GOSQUITO_URL" "$GOSQUITO_TEMP" && \
                cd "$GOSQUITO_TEMP" && \
                go build -tags dynamic "github.com/livelace/gosquito/cmd/gosquito" && \
                cp "gosquito" "$GOSQUITO_BIN" && \
                rm -rf "/root/go" "$GOSQUITO_TEMP"

RUN             useradd -m -u 1000 -s "/bin/bash" "gosquito"

USER            "gosquito"

WORKDIR         "/home/gosquito"

CMD             ["/usr/local/bin/gosquito"]
