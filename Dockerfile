FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /hub ./cmd/hub

FROM alpine:3.21
RUN addgroup -S hub && adduser -S hub -G hub
COPY --from=build /hub /hub
USER hub
EXPOSE 8080
ENTRYPOINT ["/hub"]
