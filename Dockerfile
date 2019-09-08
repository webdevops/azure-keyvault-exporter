FROM golang:1.13 as build

WORKDIR /go/src/github.com/webdevops/azure-keyvault-exporter

# Get deps (cached)
COPY ./go.mod /go/src/github.com/webdevops/azure-keyvault-exporter
COPY ./go.sum /go/src/github.com/webdevops/azure-keyvault-exporter
RUN go mod download

# Compile
COPY ./ /go/src/github.com/webdevops/azure-keyvault-exporter
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o /azure-keyvault-exporter \
    && chmod +x /azure-keyvault-exporter
RUN /azure-keyvault-exporter --help

#############################################
# FINAL IMAGE
#############################################
FROM gcr.io/distroless/static
COPY --from=build /azure-keyvault-exporter /
USER 1000
ENTRYPOINT ["/azure-keyvault-exporter"]
