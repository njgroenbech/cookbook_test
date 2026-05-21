FROM golang:alpine AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/cookbook

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/cookbook /app/cookbook
COPY static/ static/
COPY templates/ templates/
COPY demo.db .
COPY api-schema.yaml .
EXPOSE 3000
CMD ["./cookbook"]