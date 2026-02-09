FROM golang:1.25 AS builder
WORKDIR /src

COPY ./go.mod ./
COPY ./go.sum ./
RUN go mod download

COPY ./main.go ./
COPY ./cmd ./cmd
COPY ./template ./template
COPY ./config ./config
COPY ./runner ./runner
COPY ./types ./types
COPY ./jsonpath ./jsonpath
COPY ./structutil ./structutil
COPY ./filesystem ./filesystem
COPY ./kustomize ./kustomize 
RUN CGO_ENABLED=0 go build -o /cron-workflow-replicator .

FROM scratch
COPY --from=builder /cron-workflow-replicator /cron-workflow-replicator
CMD ["/cron-workflow-replicator"]
