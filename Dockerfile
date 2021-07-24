FROM            docker.io/livelace/service-core:latest

COPY            "gosquito" "/usr/local/bin/gosquito"
COPY            "assets/gosquito_provision.sh" "/usr/local/bin/gosquito_provision.sh"

CMD             ["/usr/local/bin/gosquito"]
