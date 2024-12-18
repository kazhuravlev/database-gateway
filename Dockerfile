FROM alpine:3.21

ENV WORKDIR=/workdir

RUN mkdir -p ${WORKDIR}

WORKDIR ${WORKDIR}

VOLUME ${WORKDIR}

ENTRYPOINT ["/bin/gateway"]

COPY gateway /bin/gateway
