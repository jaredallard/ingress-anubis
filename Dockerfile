# syntax=docker/dockerfile:1
FROM gcr.io/distroless/static-debian12:nonroot
LABEL org.opencontainers.image.source="https://github.com/jaredallard/ingress-anubis"
ENTRYPOINT ["/usr/local/bin/ingress-anubis"]
COPY ingress-anubis /usr/local/bin/
