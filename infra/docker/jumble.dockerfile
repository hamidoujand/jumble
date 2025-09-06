FROM golang:1.25.0 AS build_jumble

ENV CGO_ENABLED=0
ARG BUILD

# if you are not using "Vendoring" then do this steps first:
# 1-RUN mkdir /jumble
# 2-COPY go.* /jumble/
# 3-WORKDIR /jumble
# 4-RUN go mod download
# 5-COPY . /jumble

# if you are using vendoring:

# copy source into jumble dir
COPY . /jumble

# cd into cmd of your project
WORKDIR /jumble/cmd

# build the bin and pass a value to a variable inside main package at build time.
RUN go build -ldflags="-X main.build=${BUILD}" -o=jumble main.go


# run binary inside an Alpine container
FROM alpine:3.20

ARG BUILD
ARG CREATED_AT

# create a separate group and user to run the jumble server instead of running it as the root user.
# -S creates a system group which used for running services and does not require login passwords.
RUN addgroup -g 1000 -S jumble
RUN adduser -u 1000 -h /jumble -G jumble -S jumble

COPY --from=build_jumble --chown=jumble:jumble /jumble/cmd/jumble /services/jumble
WORKDIR /services

#change user to jumble to run the service with it
USER jumble
CMD ["./jumble"]

#adding some metadata about the image
LABEL org.opencontainers.image.created="${CREATED_AT}" \
      org.opencontainers.image.title="jumble" \
      org.opencontainers.image.authors="Hamid Oujand" \
      org.opencontainers.image.title="jumble" \
      org.opencontainers.image.source="https://github.com/hamidoujand/jumble" \
      org.opencontainers.image.revision="${BUILD}"






