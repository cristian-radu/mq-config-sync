# mq-config-sync
Sync IBM MQ configuration from GitHub

Set up local MQ container
```bash
docker rm -f local-mq
docker run --name local-mq -d -e LICENSE=accept -e MQ_QMGR_NAME=TEST icr.io/ibm-messaging/mq:9.2.0.0-r3-amd64
```

Build binary and copy it to the container
```bash
CGO_ENABLED=0 go build
docker cp mq-config-sync local-mq:mq-config-sync
```

Exec into container and set env configuration as per the example below.
```bash
docker exec -it local-mq bash
export GITHUB_TOKEN=git-token
export GITHUB_REPO_OWNER=cristian-radu
export GITHUB_REPO_NAME=mq-config-sync
export GITHUB_REPO_PATH=brokers/test
export GITHUB_REPO_REF=main
export GITHUB_POLL_INTERVAL=10s
export QUEUE_MANAGER_NAME=TEST
```

Run the service and observe the configuration being syncronized periodically
```bash
./mq-config-sync
```

```bash
{"level":"info","msg":"starting mq config sync loop","time":"2023-07-26T09:39:54Z"}
{"level":"info","msg":"discovered 2 mqsc files","time":"2023-07-26T09:39:55Z"}
{"level":"info","msg":"running commands in mqsc file: brokers/test/test_channel.mqsc","time":"2023-07-26T09:39:55Z"}
{"level":"info","msg":"mqsc commands ran successfully, output: one mqsc command read.no commands have a syntax error.all valid mqsc commands were processed.","time":"2023-07-26T09:39:55Z"}
{"level":"info","msg":"running commands in mqsc file: brokers/test/test_queue.mqsc","time":"2023-07-26T09:39:56Z"}
{"level":"info","msg":"mqsc commands ran successfully, output: one mqsc command read.no commands have a syntax error.all valid mqsc commands were processed.","time":"2023-07-26T09:39:56Z"}
```