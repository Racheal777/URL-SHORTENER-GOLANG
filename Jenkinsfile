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
            withCredentials([
                usernamePassword(credentialsId: 'docker-hub-creds', usernameVariable: 'DOCKER_USER', passwordVariable: 'DOCKER_PASS'),
                file(credentialsId: 'go-url-env', variable: 'ENV_FILE')
            ]) {
                sh '''
                    echo "Deploying to EC2..."
                        echo "Creating temporary deployment bundle..."
                        mkdir -p tmp-deploy && cp -r * tmp-deploy/
                        cp "$ENV_FILE" tmp-deploy/.env

                        echo "Copying project files to EC2..."
                        tar czf app.tar.gz -C tmp-deploy .
                        scp -o StrictHostKeyChecking=no app.tar.gz $REMOTE_USER@$REMOTE_HOST:/tmp/app.tar.gz

                        echo "Deploying application on EC2..."
                        ssh -o StrictHostKeyChecking=no $REMOTE_USER@$REMOTE_HOST <<'ENDSSH'
                            set -e

                            echo "Setting up app directory..."
                            mkdir -p $DEPLOY_DIR
                            tar xzf /tmp/app.tar.gz -C $DEPLOY_DIR

                        echo "Logging into Docker..."
                        echo "$DOCKER_PASS" | docker login -u "$DOCKER_USER" --password-stdin

                        cd $DEPLOY_DIR

                        echo "Starting app using docker-compose..."
                        docker-compose pull
                        docker-compose down || true
                        docker-compose up -d

                        echo "Ensuring nginx is installed..."
                        if ! command -v nginx >/dev/null 2>&1; then
                            sudo apt-get update && sudo apt-get install -y nginx
                        fi

                        echo "Configuring nginx reverse proxy..."
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

                        echo "Reloading nginx..."
                        sudo nginx -t && sudo systemctl reload nginx

                        echo "Checking app health..."
                        curl -sf http://localhost:8080 || (echo "App failed health check!" && exit 1)

                        echo "Deployment complete!"
                    ENDSSH

                    echo "Cleaning up..."
                            rm -rf tmp-deploy app.tar.gz
                        '''
                    }
                }
            }
        }
    } // close stages
} // close pipeline