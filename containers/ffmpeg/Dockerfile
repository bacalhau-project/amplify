FROM linuxserver/ffmpeg:5.1.2
RUN apt-get update && apt-get install -y \
    file=1:5.41-3
RUN mkdir -p /usr/local/bin
ENV PATH="/usr/local/bin:${PATH}"
COPY containers/scripts/run_program.sh /usr/local/bin/run_program.sh
ENTRYPOINT ["/usr/local/bin/run"] # This will get overwritten by Amplify
COPY containers/ffmpeg/run /usr/local/bin/run