FROM amazoncorretto:8
RUN curl --fail -L -o /opt/tika-app.jar https://archive.apache.org/dist/tika/2.6.0/tika-app-2.6.0.jar
COPY containers/tika/extract-metadata /usr/local/bin/extract-metadata
ENV PATH="/usr/local/bin:${PATH}"
LABEL org.opencontainers.image.source https://github.com/bacalhau-project/amplify
LABEL org.opencontainers.image.title "Apache Tika for Amplify"
LABEL org.opencontainers.image.description "A Docker image with Apache Tika installed for use with Bacalhau Amplify."
LABEL org.opencontainers.image.licenses Apache-2.0
LABEL org.opencontainers.image.url https://github.com/bacalhau-project/amplify