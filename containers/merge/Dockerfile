FROM alpine:3.17
RUN apk add --no-cache bash
RUN mkdir -p /usr/local/bin
ENV PATH="/usr/local/bin:${PATH}"
COPY containers/scripts/run_program.sh /usr/local/bin/run_program.sh
COPY containers/merge/run /usr/local/bin/run
ENTRYPOINT ["run"]