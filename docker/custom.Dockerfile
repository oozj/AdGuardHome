# syntax=docker/dockerfile:1

FROM --platform=$BUILDPLATFORM adguard/home-js-builder:4.0 AS frontend
WORKDIR /app
COPY client /app/client
COPY .twosky.json /app/.twosky.json
RUN --mount=type=cache,target=/root/.npm \
    npm --prefix client ci --quiet --no-progress && \
    npm --prefix client run build-prod

FROM --platform=$BUILDPLATFORM adguard/go-builder:1.26.6--1 AS backend
ARG VERSION=v0.107.79-upstream-rules
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
WORKDIR /app
COPY . /app
COPY --from=frontend /app/build /app/build
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go \
    mkdir -p /out && \
    env \
        CHANNEL=development \
        GOOS="$TARGETOS" \
        GOARCH="$TARGETARCH" \
        GOARM="${TARGETVARIANT#v}" \
        SOURCE_DATE_EPOCH=0 \
        VERSION="$VERSION" \
        OUT=/out/AdGuardHome \
        sh ./scripts/make/go-build.sh

FROM --platform=$TARGETPLATFORM alpine:3.23
RUN apk --no-cache add ca-certificates libcap tzdata && \
    mkdir -p /opt/adguardhome/conf /opt/adguardhome/work && \
    chown -R nobody:nogroup /opt/adguardhome

COPY --from=backend --chmod=0755 --chown=nobody:nogroup \
    /out/AdGuardHome /opt/adguardhome/AdGuardHome

RUN setcap 'cap_net_bind_service=+eip' /opt/adguardhome/AdGuardHome

EXPOSE 53/tcp 53/udp 67/udp 68/udp 80/tcp 443/tcp 443/udp 853/tcp 853/udp 3000/tcp
WORKDIR /opt/adguardhome/work
ENTRYPOINT ["/opt/adguardhome/AdGuardHome"]
CMD ["--no-check-update", "-c", "/opt/adguardhome/conf/AdGuardHome.yaml", "-w", "/opt/adguardhome/work"]
