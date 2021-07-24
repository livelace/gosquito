FROM            harbor-core.k8s-2.livelace.ru/infra/service-core:latest

COPY            "assets/gosquito_provision.sh" "/usr/local/bin/gosquito_provision.sh"
COPY            "gosquito" "/usr/local/bin/gosquito"

CMD             ["/usr/local/bin/gosquito"]
