FROM golang:1.24-alpine AS builder
RUN apk add --no-cache git
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /auracrab ./cmd/auracrab

FROM alpine:3.20
RUN apk add --no-cache ca-certificates git docker-cli
COPY --from=builder /auracrab /usr/local/bin/auracrab
RUN mkdir -p /run/agentic /root/.auracrab
ENV AGENTIC_RUN_DIR=/run/agentic
ENTRYPOINT ["auracrab"]
CMD ["version"]
