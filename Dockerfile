FROM golang:1.27.3@sha256:2920bb124f1a2e12a1d39f0cab756ae0068441ee1da7655dec7f5e374944095b AS builder
WORKDIR /src

COPY ./go.mod ./
COPY ./go.sum ./
RUN go mod download && go mod verify

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
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /cron-workflow-replicator .

FROM scratch
COPY --from=builder /cron-workflow-replicator /cron-workflow-replicator
USER 65532:65532
CMD ["/cron-workflow-replicator"]
