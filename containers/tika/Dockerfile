FROM amazoncorretto:20
RUN curl --fail -L -o /opt/tika-app.jar https://archive.apache.org/dist/tika/2.6.0/tika-app-2.6.0.jar
RUN yum install jq -y
RUN mkdir -p /usr/local/bin
ENV PATH="/usr/local/bin:${PATH}"
COPY containers/scripts/run_program.sh /usr/local/bin/run_program.sh
ENTRYPOINT ["/usr/local/bin/run"] # This will get overwritten by Amplify
COPY containers/tika/run /usr/local/bin/run
LABEL org.opencontainers.image.source https://github.com/bacalhau-project/amplify
LABEL org.opencontainers.image.title "Apache Tika for Amplify"
LABEL org.opencontainers.image.description "A Docker image with Apache Tika installed for use with Bacalhau Amplify."
LABEL org.opencontainers.image.licenses Apache-2.0
LABEL org.opencontainers.image.url https://github.com/bacalhau-project/amplify