pipeline {
    agent any

    environment  {
        GITHUB_REPO = 'https://github.com/Racheal777/URL-SHORTENER-GOLANG.git'
    }

    stages {
        stage('Clone'){
            steps {
               git credentialsId: 'github_token_jenkins', url: "${GITHUB_REPO}", branch: 'master'
            }
        }
        stage('Build'){
            steps{
                sh """
                echo build
                """
            }
        }
        stage('Scan'){
            steps {
                script {
                   sh '''
                    echo hey
                    '''
                }
            }
        }
        stage('Login to DockerHub'){
            steps {
                    sh """
                    echo docker
                    """
            }
        }
        stage('Push') {
            steps {
                sh """
                 echo push
                """
            }
        }
    }
}