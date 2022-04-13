#!groovy

pipeline {
  parameters {
    booleanParam(name: 'ARCHIVE_ARTIFACTS', defaultValue: true)
    booleanParam(name: 'CLEAN', defaultValue: true)
  }
  options {
    buildDiscarder(logRotator(daysToKeepStr: '20', numToKeepStr: '30', artifactNumToKeepStr: '3'))
    copyArtifactPermission('*')
  }
  triggers {
    pollSCM('H/15 * * * *')
    githubPush()
  }
  agent none
  stages {
    stage ('BuildAndTest') {
      matrix {
        agent {
          docker {
            image 'golang:1.17'
            label 'amd64 && docker'
          }
        }
        axes {
          axis {
            name 'ARCH'
            values 'arm64','amd64'
          }
        }
        environment {
         CGO_ENABLED = "0"
         GOARCH = "${ARCH}"
         GOCACHE = "${env.WORKSPACE}/cache"
        }
        stages {
          stage('Build') {
            steps {
              sh './build'
              sh 'if [ ${GOARCH} = arm64 ]; then mv bin bin.arm64; fi'
            }
          }
          stage('Test') {
            when {
              expression { ARCH == "amd64" }
            }
            steps {
              sh './test'
            }
          }
          stage('Post-build') {
            when {
              expression {
                 env.JOB_BASE_NAME == "master-builder" || params.ARCHIVE_ARTIFACTS
              }
            }
            steps {
              archiveArtifacts(artifacts: 'bin*/**', fingerprint: true, onlyIfSuccessful: true)
            }
          }
        }
        post {
          cleanup {
            cleanWs()
          }
        }
      }
    }
  }
}
