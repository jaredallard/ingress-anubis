# syntax=docker/dockerfile:1
FROM gcr.io/distroless/static-debian12:nonroot
ENTRYPOINT ["/usr/local/bin/ingress-anubis"]

ARG TARGETPLATFORM
COPY $TARGETPLATFORM/ingress-anubis /usr/local/bin/
