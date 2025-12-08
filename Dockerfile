FROM golang:1.25 AS builder
WORKDIR /src

COPY ./go.mod ./
COPY ./go.sum ./
RUN go mod download

COPY ./main.go ./
COPY ./bot.go ./
COPY ./help_handler.go ./
COPY ./hello_handler.go ./
COPY ./message_router.go  ./
RUN CGO_ENABLED=0 go build -o /cron-workflow-replicator .

FROM scratch
COPY --from=builder /cron-workflow-replicator /cron-workflow-replicator
CMD ["/cron-workflow-replicator"]