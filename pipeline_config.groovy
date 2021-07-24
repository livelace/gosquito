jte {
    pipeline_template = 'k8s_build.groovy'
}

libraries {
    appimage {
        source = 'gosquito'
        destination = 'gosquito-${VERSION}.appimage'
    }
    git {
        repo_url = 'https://github.com/livelace/gosquito.git'
    }
    go {
        options = '-tags dynamic github.com/livelace/gosquito/cmd/gosquito'
    }
    mattermost
    version
}

keywords {
    build_image = 'harbor-core.k8s-2.livelace.ru/dev/gobuild:latest'
}
