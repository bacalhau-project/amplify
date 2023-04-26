FROM python:3.9.16
RUN pip install ydata-profiling==4.1.2
RUN mkdir -p /usr/local/bin
ENV PATH="/usr/local/bin:${PATH}"
COPY containers/scripts/run_program.sh /usr/local/bin/run_program.sh
ENTRYPOINT ["/usr/local/bin/run"] # This will get overwritten by Amplify
COPY containers/ydata-profiling/run /usr/local/bin/run
COPY containers/ydata-profiling/data-profiler.py /usr/local/bin/data-profiler.py