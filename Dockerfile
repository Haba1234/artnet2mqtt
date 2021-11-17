# syntax=docker/dockerfile:1

FROM golang:1.17.1-alpine as build

ENV BIN_FILE /opt/artnet2mqtt
ENV CODE_DIR /go/src/

WORKDIR ${CODE_DIR}

# Кэшируем слои с модулями
COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . ${CODE_DIR}

# Собираем статический бинарник Go (без зависимостей на Си API),
# иначе он не будет работать в alpine образе.
ARG LDFLAGS
RUN CGO_ENABLED=0 go build \
        -ldflags "$LDFLAGS" \
        -o ${BIN_FILE} cmd/*

# На выходе тонкий образ
FROM alpine:3.14.2

LABEL SERVICE="artnet2mqtt"

ENV BIN_FILE "/opt/artnet2mqtt"
COPY --from=build ${BIN_FILE} ${BIN_FILE}

ENV CONFIG_FILE /etc/artnet2mqtt/conf.toml
COPY ./configs/conf.toml ${CONFIG_FILE}

EXPOSE 1883

ENTRYPOINT ${BIN_FILE} -config ${CONFIG_FILE}