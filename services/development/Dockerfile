FROM golang:1.27-alpine AS build
WORKDIR /src

COPY services/token /src/services/token
COPY services/development /src/services/development
WORKDIR /src/services/development
RUN go mod download
RUN CGO_ENABLED=0 go build -o /server ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=build /server /server
EXPOSE 8080
CMD ["/server"]