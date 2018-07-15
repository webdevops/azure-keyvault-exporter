FROM golang:1.10 as build
WORKDIR /go/src/azure-keyvault-exporter/src
COPY ./src /go/src/azure-keyvault-exporter/src
RUN curl https://glide.sh/get | sh \
    && glide install
RUN mkdir /app/ \
    && cp -a entrypoint.sh /app/ \
    && chmod 555 /app/entrypoint.sh \
    && go build -o /app/azure-keyvault-exporter

#############################################
# FINAL IMAGE
#############################################
FROM alpine
RUN apk add --no-cache \
        libc6-compat \
    	ca-certificates
COPY --from=build /app/ /app/
USER 1000
ENTRYPOINT ["/app/entrypoint.sh"]
