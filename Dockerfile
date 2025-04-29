FROM golang:latest AS build

WORKDIR /src

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o app .

FROM eclipse-temurin:11

COPY --from=build /src/app /usr/local/bin/
COPY ./Composer /tmp

CMD ["/usr/local/bin/app"]
