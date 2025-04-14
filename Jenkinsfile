pipeline {
    agent {
        label "docker-ec2"
    }

    environment {
        GITHUB_REPO = 'https://github.com/Racheal777/URL-SHORTENER-GOLANG.git'
        IMAGE_NAME = "go-url-shortener"
        TAG = "latest"
        DOCKERHUB_USER = 'rachealcodez'
        REMOTE_USER = 'ubuntu'
        REMOTE_HOST = '13.53.188.132'
        DEPLOY_DIR = '/home/ubuntu/go-url-shortener'
        NGINX_CONF = '/etc/nginx/conf.d/go-url-shortener.conf'
    }

    stages {
        stage('Clone Project') {
            steps {
                git credentialsId: 'github_token_jenkins', url: "${GITHUB_REPO}", branch: 'master'
            }
        }

        stage('Clean Up Docker') {
            steps {
                sh '''
                    echo "Cleaning up Docker..."
                    docker container prune -f || true
                    docker image prune -af || true
                    docker volume prune -f || true
                '''
            }
        }

        stage('Build Docker Image') {
            steps {
                sh "docker build -t $IMAGE_NAME:$TAG ."
            }
        }

        stage('Trivy Scan') {
            steps {
                sh '''
                    if ! command -v trivy >/dev/null; then
                        echo "Installing Trivy..."
                        wget https://github.com/aquasecurity/trivy/releases/download/v0.61.0/trivy_0.61.0_Linux-64bit.deb
                        sudo dpkg -i trivy_0.61.0_Linux-64bit.deb
                    fi
                    trivy image --severity HIGH,CRITICAL --exit-code 1 "$IMAGE_NAME:$TAG"
                '''
            }
        }

        stage('Push to DockerHub') {
            steps {
                withCredentials([usernamePassword(credentialsId: 'docker-hub-creds', usernameVariable: 'DOCKER_USER', passwordVariable: 'DOCKER_PASS')]) {
                    sh '''
                        echo "$DOCKER_PASS" | docker login -u "$DOCKER_USER" --password-stdin
                        docker tag $IMAGE_NAME:$TAG $DOCKER_USER/$IMAGE_NAME:$TAG
                        docker push $DOCKER_USER/$IMAGE_NAME:$TAG
                    '''
                }
            }
        }

        stage('Deploy to EC2') {
            steps {
                sshagent(credentials: ['api-server-ec2']) {
                    withCredentials([usernamePassword(credentialsId: 'docker-hub-creds', usernameVariable: 'DOCKER_USER', passwordVariable: 'DOCKER_PASS'),
                    file(credentialsId: 'go-url-env', variable: 'ENV_FILE')
                    ])
                      {
                        sh '''

                            # Ensure the target directory exists
                            ssh -o StrictHostKeyChecking=no $REMOTE_USER@$REMOTE_HOST "mkdir -p $DEPLOY_DIR && sudo chown -R $REMOTE_USER:$REMOTE_USER $DEPLOY_DIR"


                            echo "Copying .env file to remote server..."
                            scp -o StrictHostKeyChecking=no $ENV_FILE $REMOTE_USER@$REMOTE_HOST:$DEPLOY_DIR/.env

                            ssh -o StrictHostKeyChecking=no $REMOTE_USER@$REMOTE_HOST <<'ENDSSH'
                                echo "$DOCKER_PASS" | docker login -u "$DOCKER_USER" --password-stdin

                                which git || (sudo apt-get update && sudo apt-get install -y git)

                                if [ ! -d "$DEPLOY_DIR" ]; then
                                    git clone $GITHUB_REPO $DEPLOY_DIR
                                else
                                    cd $DEPLOY_DIR && git pull
                                fi

                                cd $DEPLOY_DIR

                                docker-compose pull
                                docker-compose down || true
                                docker-compose up -d

                                echo "Setting up NGINX reverse proxy..."

                                sudo apt-get update && sudo apt-get install -y nginx

                                cat <<'EOF' | sudo tee $NGINX_CONF
                        server {
                            listen 80;
                            server_name short.softlife.reggeerr.com;

                            location / {
                                proxy_pass http://localhost:8080;
                                proxy_set_header Host $host;
                                proxy_set_header X-Real-IP $remote_addr;
                                proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
                                proxy_set_header X-Forwarded-Proto $scheme;
                            }
                        }
                        EOF

                                sudo nginx -t && sudo systemctl reload nginx

                                echo "NGINX setup completed!"
                            ENDSSH
                        '''
                    }
                }
            }
        }
    }
}
