# ascraper

## Run it locally with mirrord

Prereqs:

- Make sure your Kubeconfig is set correctly
- You have mirrord installed (https://mirrord.dev/)


1. `go build main.go && mirrord exec --target deploy/selfservice-api ./main`