FROM linuxserver/libreoffice:latest

# 目标架构参数，默认 amd64
ARG TARGETARCH=amd64

USER root

# 创建应用目录
RUN mkdir -p /app /app/storage /app/logs /app/scripts

# 安装中文字体
RUN if ! apk add --no-cache font-noto-cjk; then \
      apk add --no-cache wqy-zenhei; \
    fi

# 从本地 dist 目录复制对应架构的二进制文件
COPY dist/libreoffice-rest-api-linux-${TARGETARCH} /app/libreoffice-rest-api
COPY config.yaml.example /app/config.yaml.example
COPY scripts/uno_convert.py /app/scripts/uno_convert.py

# 设置权限
RUN chmod +x /app/libreoffice-rest-api && \
    cp /app/config.yaml.example /app/config.yaml && \
    if id -u abc >/dev/null 2>&1; then \
      chown -R abc:abc /app; \
    fi

WORKDIR /app

EXPOSE 30231

VOLUME ["/app/storage", "/app/logs"]

USER abc

ENTRYPOINT ["/app/libreoffice-rest-api"]
