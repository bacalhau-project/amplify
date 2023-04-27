FROM python:3.9.16
RUN pip install frictionless==5.12.1 frictionless[excel] frictionless[parquet]
RUN mkdir -p /usr/local/bin
ENV PATH="/usr/local/bin:${PATH}"
COPY containers/scripts/run_program.sh /usr/local/bin/run_program.sh
ENTRYPOINT ["/usr/local/bin/run"] # This will get overwritten by Amplify
COPY containers/frictionless-extract/run /usr/local/bin/run