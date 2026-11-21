FROM golang:1.26-alpine AS base
WORKDIR /app

FROM base AS deps
COPY go.mod go.sum ./
RUN go mod download

FROM base AS builder
COPY --from=deps /go/pkg /go/pkg
COPY . .
RUN go build -o api .

FROM alpine:latest AS release
COPY --from=builder /app/api .
EXPOSE 8080
CMD [ "./api" ]
