FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY orchestrator/go.mod orchestrator/go.sum ./
RUN go mod download

COPY orchestrator/ .
RUN go build -o orchestrator .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/orchestrator .
# Copy UI build files from root context
COPY ui/dist ./ui/dist

CMD ["./orchestrator"]
