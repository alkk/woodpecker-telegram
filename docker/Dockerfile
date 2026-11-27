FROM alpine:3.23

ARG TARGETOS
ARG TARGETARCH

LABEL maintainer="Alex Kirhenshtein" \
  org.label-schema.name="Telegram Plugin" \
  org.label-schema.vendor="Alex Kirhenshtein" \
  org.label-schema.schema-version="1.0"

LABEL org.opencontainers.image.source=https://github.com/alkk/woodpecker-telegram
LABEL org.opencontainers.image.description="plugin for sending telegram notifications"
LABEL org.opencontainers.image.licenses=MIT

RUN apk add --no-cache ca-certificates && \
  rm -rf /var/cache/apk/*

COPY release/${TARGETOS}/${TARGETARCH}/woodpecker-telegram /bin/

ENTRYPOINT ["/bin/woodpecker-telegram"]
