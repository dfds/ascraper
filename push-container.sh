#!/bin/bash

set -e

# build parameters
readonly ACCOUNT_ID=${AWS_ACCOUNT_ID:-"579478677147"}
readonly REGION=${AWS_DEFAULT_REGION:-"eu-central-1"}
readonly ECR_URL="${ACCOUNT_ID}.dkr.ecr.${REGION}.amazonaws.com"
readonly IMAGE_NAME=${1:-"N/A"}
readonly BUILD_NUMBER=${2:-"N/A"}
readonly REMOTE_IMAGE_NAME="${ECR_URL}/${IMAGE_NAME}:${BUILD_NUMBER}"
readonly ALLOW_PUSH=${ALLOW_PUSH_CONTAINER:-"0"}

if [ $ALLOW_PUSH -eq "1" ]
then
    echo "Tagging container image (${IMAGE_NAME})..."
    docker tag ${IMAGE_NAME}:latest ${REMOTE_IMAGE_NAME}

    echo "Logging in to container registry (${ECR_URL})..."
    aws ecr get-login-password --region "${REGION}" | docker login --username AWS --password-stdin ${ECR_URL}

    echo "Pushing container image to ECR (${REMOTE_IMAGE_NAME})..."
    docker push ${REMOTE_IMAGE_NAME}
else
    echo "********************************************************************************"
    echo "Would have pushed container image to ECR (${REMOTE_IMAGE_NAME})..."
    echo "CMD: docker push ${REMOTE_IMAGE_NAME}"
fi
