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
        
        stage('Build Application Image') {
            steps {
                sh 'docker build --target runtime -t meghavx/go-miniurl:latest .'
            }
        }

        stage('Push Image') {
            steps {
                withCredentials([usernamePassword(
                    credentialsId: 'dockerhub-creds',
                    usernameVariable: 'DOCKER_USERNAME',
                    passwordVariable: 'DOCKER_PASSWORD'
                )]) {
                    sh '''
                        echo "$DOCKER_PASSWORD" | docker login -u "$DOCKER_USERNAME" --password-stdin
                        docker push meghavx/go-miniurl:latest
                        docker logout
                    '''
                }
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