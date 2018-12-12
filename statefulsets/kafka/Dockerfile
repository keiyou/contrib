FROM ubuntu:16.04
ENV KAFKA_USER=kafka \
KAFKA_DATA_DIR=/var/lib/kafka/data \
JAVA_HOME=/usr/lib/jvm/java-8-openjdk-amd64 \
KAFKA_HOME=/opt/kafka \
PATH=$PATH:/opt/kafka/bin

ARG KAFKA_VERSION=0.11.0.3
ARG KAFKA_PACK=kafka_2.11-0.11.0.3
ARG KAFKA_DIST=kafka
RUN set -x \
    && apt-get update \
    && apt-get install -y openjdk-8-jre-headless wget \
    && wget -q "http://www.apache.org/dist/kafka/$KAFKA_VERSION/$KAFKA_PACK.tgz" \
    && wget -q "http://www.apache.org/dist/kafka/$KAFKA_VERSION/$KAFKA_PACK.tgz.asc" \
    && wget -q "http://kafka.apache.org/KEYS" \
    && export GNUPGHOME="$(mktemp -d)" \
    && gpg --import KEYS \
    && gpg --batch --verify "$KAFKA_PACK.tgz.asc" "$KAFKA_PACK.tgz" \
    && tar -xzf "$KAFKA_PACK.tgz" -C /opt \
    && mv /opt/$KAFKA_PACK /opt/$KAFKA_DIST \
    && rm -r "$GNUPGHOME" "$KAFKA_PACK.tgz" "$KAFKA_PACK.tgz.asc"

COPY log4j.properties /opt/$KAFKA_DIST/log4j_config/
COPY tools-log4j.properties /opt/$KAFKA_DIST/log4j_config/

RUN set -x \
    && ln -s /opt/$KAFKA_DIST $KAFKA_HOME \
    && useradd $KAFKA_USER \
    && [ `id -u $KAFKA_USER` -eq 1000 ] \
    && [ `id -g $KAFKA_USER` -eq 1000 ] \
    && mkdir -p $KAFKA_DATA_DIR \
    && chown -R "$KAFKA_USER:$KAFKA_USER"  /opt/$KAFKA_DIST \
    && chown -R "$KAFKA_USER:$KAFKA_USER"  $KAFKA_DATA_DIR

