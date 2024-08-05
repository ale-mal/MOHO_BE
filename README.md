# MOHO WebSocket backend server

## Run

```
go mod tidy
go run main.go
```

## Docker

```
docker build -t moho-be .
docker run -p 8080:8080 moho-be

# to stop
docker ps
docker stop abc123def456
```

### Amazon ECR

```
aws ecr create-repository --repository-name moho-be

# Authenticate Docker to ECR
aws ecr get-login-password --region eu-central-1 | sudo docker login --username AWS --password-stdin 680324637652.dkr.ecr.eu-central-1.amazonaws.com

# push
docker tag moho-be:latest 680324637652.dkr.ecr.eu-central-1.amazonaws.com/moho-be:latest
docker push 680324637652.dkr.ecr.eu-central-1.amazonaws.com/moho-be:latest

# delete
aws ecr delete-repository --repository-name moho-be --force
aws ecr describe-repositories
```