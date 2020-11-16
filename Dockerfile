FROM golang:1.15 as build

WORKDIR /go/src/github.com/webdevops/azure-keyvault-exporter

# Get deps (cached)
COPY ./go.mod /go/src/github.com/webdevops/azure-keyvault-exporter
COPY ./go.sum /go/src/github.com/webdevops/azure-keyvault-exporter
COPY ./Makefile /go/src/github.com/webdevops/azure-keyvault-exporter
RUN make dependencies

# Compile
COPY ./ /go/src/github.com/webdevops/azure-keyvault-exporter
RUN make test
RUN make lint
RUN make build
RUN ./azure-keyvault-exporter --help

#############################################
# FINAL IMAGE
#############################################
FROM gcr.io/distroless/static
ENV LOG_JSON=1
COPY --from=build /go/src/github.com/webdevops/azure-keyvault-exporter/azure-keyvault-exporter /
USER 1000
ENTRYPOINT ["/azure-keyvault-exporter"]
