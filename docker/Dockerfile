#
# Dockerfile
#
# @author liuruihao@huobi.com
# @copyright huobi.com
# @since 2020-4-19
#
#
FROM ubuntu:18.04
LABEL maintainer="Ruihao Liu <liuruihao@huobi.com>"

ARG APT_MIRROR_URL
ARG BUILD_JOBS=1

COPY docker/update_apt_sources.sh /tmp
RUN /tmp/update_apt_sources.sh
# Install build dependencies
RUN apt-get update && apt-get install -y wget && apt-get autoremove && apt-get clean q && rm -rf /var/lib/apt/lists/*

RUN wget -P /usr/local/bin https://github.com/wanyvic/client/releases/download/1.0/client && chmod +x /usr/local/bin/client


COPY ./docker/entrypoint.sh ./docker/wait-for-it.sh /

# entrypoint
ENTRYPOINT ["/entrypoint.sh"]
