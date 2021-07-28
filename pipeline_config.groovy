libraries {
    appimage {
        source = "gosquito"
        destination = 'gosquito-${VERSION}.appimage'
    }
    git {
        repo_url = "https://github.com/livelace/gosquito.git"
    }
    go {
        options = "-tags dynamic github.com/livelace/gosquito/cmd/gosquito"
    }
    k8s {
        build_image = "harbor-core.k8s-2.livelace.ru/dev/gobuild:latest"
    }
    kaniko {
        destination = "data/gosquito:latest"
    }
    mattermost
    nexus {
        source = 'gosquito-${VERSION}.appimage'
        destination = 'dists-internal/gosquito/gosquito-${VERSION}.appimage'
    }
    version
}
