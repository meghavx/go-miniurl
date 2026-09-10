pipeline {
    agent any

    stages {
        stage('Build Test Image') {
            steps {
                sh 'docker build --target test -t go-miniurl-test .'
            }
        }

        stage('Run Tests') {
            steps {
                sh '''
                    docker run --name go-miniurl-test-container go-miniurl-test
                    docker cp go-miniurl-test-container:/app/test-results.xml test-results.xml
                '''
            }
        }

        stage('Publish Test Results') {
            steps {
                junit 'test-results.xml'
            }
        }
    }

    post {
        always {
            sh '''
                docker rm -f go-miniurl-test-container 2>/dev/null || true
                docker image rm go-miniurl-test 2>/dev/null || true
            '''
        }
    }
}