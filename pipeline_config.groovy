libraries {
    appimage {
        source = "gosquito"
        destination = 'gosquito-${VERSION}.appimage'
    }
    dependency_check
    dependency_track {
        project = "gosquito"
        version = "master"
    }
    git {
        repo_url = "https://github.com/livelace/gosquito.git"
    }
    go {
        options = "-tags dynamic github.com/livelace/gosquito/cmd/gosquito"
    }
    harbor {
        policy = "gosquito"
    }
    k8s_build {
        image = "harbor-core.k8s-2.livelace.ru/dev/gobuild:latest"
        privileged = true
    }
    kaniko {
        destination = "data/gosquito:latest"
    }
    mattermost
    nexus {
        source = 'gosquito-${VERSION}.appimage'
        destination = 'dists-internal/gosquito/gosquito-${VERSION}.appimage'
    }
    sonarqube
    version
}
