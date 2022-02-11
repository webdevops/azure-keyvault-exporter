FROM golang:1.17-alpine as build

RUN apk upgrade --no-cache --force
RUN apk add --update build-base make git

WORKDIR /go/src/github.com/webdevops/azure-keyvault-exporter

# Compile
COPY ./ /go/src/github.com/webdevops/azure-keyvault-exporter
RUN make dependencies
RUN make test
RUN make build
RUN ./azure-keyvault-exporter --help

#############################################
# FINAL IMAGE
#############################################
FROM gcr.io/distroless/static
ENV LOG_JSON=1
COPY --from=build /go/src/github.com/webdevops/azure-keyvault-exporter/azure-keyvault-exporter /
USER 1000:1000
ENTRYPOINT ["/azure-keyvault-exporter"]
