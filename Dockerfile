FROM golang:1

ENV PROJECT=nativerw

ENV ORG_PATH="github.com/Financial-Times"
ENV SRC_FOLDER="${GOPATH}/src/${ORG_PATH}/${PROJECT}"
ENV BUILDINFO_PACKAGE="${ORG_PATH}/service-status-go/buildinfo."

RUN mkdir -p /artifacts/configs
COPY ./configs/config.json /artifacts/configs/config.json

COPY . ${SRC_FOLDER}
WORKDIR ${SRC_FOLDER}

# Build app
RUN VERSION="version=$(git describe --tag --always 2> /dev/null)" \
  && DATETIME="dateTime=$(date -u +%Y%m%d%H%M%S)" \
  && REPOSITORY="repository=$(git config --get remote.origin.url)" \
  && REVISION="revision=$(git rev-parse HEAD)" \
  && BUILDER="builder=$(go version)" \
  && GOPRIVATE="github.com/Financial-Times" \
  && git config --global url."https://${GITHUB_USERNAME}:${GITHUB_TOKEN}@github.com".insteadOf "https://github.com" \
  && LDFLAGS="-X '"${BUILDINFO_PACKAGE}$VERSION"' -X '"${BUILDINFO_PACKAGE}$DATETIME"' -X '"${BUILDINFO_PACKAGE}$REPOSITORY"' -X '"${BUILDINFO_PACKAGE}$REVISION"' -X '"${BUILDINFO_PACKAGE}$BUILDER"'" \
  && CGO_ENABLED=0 go build -mod=readonly -a -o /artifacts/${PROJECT} -ldflags="${LDFLAGS}" ./cmd/${PROJECT} \
  && echo "Build flags: ${LDFLAGS}"

# Download required Amazon certificate to authenticate to the Document DB cluster
RUN mkdir -p /tmp/amazonaws
WORKDIR /tmp/amazonaws
RUN apt-get update && apt-get install -y wget && wget https://s3.amazonaws.com/rds-downloads/rds-combined-ca-bundle.pem

FROM scratch
WORKDIR /
COPY --from=0 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=0 /artifacts/ /
COPY --from=0 /tmp/amazonaws/* /

CMD [ "/nativerw" ]
