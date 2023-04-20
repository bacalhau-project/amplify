# syntax=docker/dockerfile:1.4
FROM node:lts as ui-build
WORKDIR /work
ENV NODE_OPTIONS=--openssl-legacy-provider
COPY ui .
RUN yarn install \
  --prefer-offline \
  --frozen-lockfile \
  --non-interactive \
  --production=false
RUN yarn build

FROM cgr.dev/chainguard/go:1.20 as build
WORKDIR /work
COPY Makefile .
COPY go.mod .
COPY --from=ui-build /work/dist /work/ui/dist
COPY . .
RUN make build-amplify
RUN find ./bin -name 'amplify' -exec mv -t ./bin {} +

FROM cgr.dev/chainguard/static:latest
COPY --from=build /work/bin/amplify /usr/local/bin/amplify
COPY config.yaml /
ENV PATH="/usr/local/bin"
ENTRYPOINT ["amplify"]
LABEL org.opencontainers.image.source https://github.com/bacalhau-project/amplify
LABEL org.opencontainers.image.title "Amplify"
LABEL org.opencontainers.image.description "Automated data enrichment, augmentation and analysis."
LABEL org.opencontainers.image.licenses Apache-2.0
LABEL org.opencontainers.image.url https://github.com/bacalhau-project/amplify