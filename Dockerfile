# Build stage
FROM golang:1.22-alpine3.20 AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o main main.go

# Run stage
FROM alpine:3.20

# Create a new user and group
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Switch to the new user
USER appuser

COPY --from=builder /app/main .
COPY app.env .
COPY start.sh .
COPY wait-for.sh .
COPY db/migration ./db/migration

EXPOSE 8080
CMD [ "/app/main" ]
ENTRYPOINT [ "/app/start.sh" ]
