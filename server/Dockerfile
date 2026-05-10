# syntax=docker/dockerfile:1

FROM golang:1.25.5-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/portfolio-server .

FROM alpine:3.22

RUN apk add --no-cache ca-certificates \
	&& addgroup -S app \
	&& adduser -S app -G app

WORKDIR /app

RUN mkdir -p /app/public/uploads/projects \
	&& chown -R app:app /app

COPY --from=build /out/portfolio-server /app/portfolio-server

USER app

EXPOSE 8080

ENV ADDR=:8080 \
	UPLOAD_STORAGE=local \
	UPLOAD_DIR=/app/public/uploads/projects \
	UPLOAD_BASE_URL=/uploads/projects

CMD ["/app/portfolio-server"]
