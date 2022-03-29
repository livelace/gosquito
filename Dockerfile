FROM            scratch

ADD             "work/dracut/rootfs.tar" "/"

COPY            "work/assets/gosquito_provision.sh" "/usr/local/bin/gosquito_provision.sh"

CMD             ["/usr/local/bin/gosquito"]
