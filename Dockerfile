FROM debian:bookworm-slim AS builder

RUN apt-get update && apt-get install -y \
    curl \
    git \
    lib32gcc-s1 \
    ca-certificates \
    gcc \
    libc6-dev \
    && rm -rf /var/lib/apt/lists/*

RUN curl -LO https://go.dev/dl/go1.23.4.linux-amd64.tar.gz \
    && tar -C /usr/local -xzf go1.23.4.linux-amd64.tar.gz \
    && rm go1.23.4.linux-amd64.tar.gz

ENV PATH="/usr/local/go/bin:${PATH}"

WORKDIR /build
COPY astra_core/ .
RUN CGO_ENABLED=1 GOOS=linux go build -o astranet .

FROM debian:bookworm-slim

RUN dpkg --add-architecture i386 && \
    apt-get update && apt-get install -y \
    lib32gcc-s1 \
    lib32stdc++6 \
    libc6-i386 \
    curl \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

RUN mkdir -p /opt/steamcmd && \
    cd /opt/steamcmd && \
    curl -sqL "https://steamcdn-a.akamaihd.net/client/installer/steamcmd_linux.tar.gz" | tar zxvf - && \
    chmod +x /opt/steamcmd/steamcmd.sh && \
    chmod +x /opt/steamcmd/linux32/steamcmd && \
    /opt/steamcmd/steamcmd.sh +quit || true

ENV PATH="/opt/steamcmd/linux32:/opt/steamcmd:${PATH}"

WORKDIR /app
COPY --from=builder /build/astranet .

ENV DISCORD_WEBHOOK_URL=""
ENV APP_ID="730"
ENV DB_PATH="/data/astranet.db"
ENV API_PORT="8000"

VOLUME ["/data", "/data/depot_cache"]

EXPOSE 8000

HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD curl -f http://localhost:8000/health || exit 1

CMD ["./astranet"]
