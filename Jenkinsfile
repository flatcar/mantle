#!groovy

properties([
    buildDiscarder(logRotator(daysToKeepStr: '20', numToKeepStr: '30', artifactNumToKeepStr: '3')),

    [$class: 'CopyArtifactPermissionProperty',
     projectNames: '*'],

    pipelineTriggers([pollSCM('H/15 * * * *')]),
    parameters([
        booleanParam(name: 'ARCHIVE_ARTIFACTS', defaultValue: false),
        booleanParam(name: 'CLEAN', defaultValue: true)
    ])
])

node('amd64 && docker') {
  try {
    stage('SCM') {
        checkout scm
    }

    stage('Build') {
        sh "docker run --rm -e CGO_ENABLED=0 -e GOARCH=arm64 -e GOCACHE=/usr/src/myapp/cache -u \"\$(id -u):\$(id -g)\" -v /etc/passwd:/etc/passwd:ro -v /etc/group:/etc/group:ro -v \"\$PWD\":/usr/src/myapp -w /usr/src/myapp golang:1.16 ./build"
        sh "mv bin bin.arm64"
        sh "docker run --rm -e CGO_ENABLED=1 -e GOARCH=amd64 -e GOCACHE=/usr/src/myapp/cache -u \"\$(id -u):\$(id -g)\" -v /etc/passwd:/etc/passwd:ro -v /etc/group:/etc/group:ro -v \"\$PWD\":/usr/src/myapp -w /usr/src/myapp golang:1.16 ./build"
    }

    stage('Test') {
        sh 'docker run --rm -e GOCACHE=/usr/src/myapp/cache -u "$(id -u):$(id -g)" -v /etc/passwd:/etc/passwd:ro -v /etc/group:/etc/group:ro -v "$PWD":/usr/src/myapp -w /usr/src/myapp golang:1.16 ./test'
    }

    stage('Post-build') {
        if (env.JOB_BASE_NAME == "master-builder" || params.ARCHIVE_ARTIFACTS) {
            archiveArtifacts artifacts: 'bin/**, bin.arm64/**', fingerprint: true, onlyIfSuccessful: true
        }
    }
  } finally {
    if (params.CLEAN) {
        sh 'sudo chown -R $USER .'
        cleanWs()
    }
  }
}
