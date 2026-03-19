# Database Gateway provides access to servers with ACL for safe and restricted database interactions.
# Copyright (C) 2024  Kirill Zhuravlev
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with this program.  If not, see <https://www.gnu.org/licenses/>.

FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev

WORKDIR /src

RUN apk add --no-cache ca-certificates git gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
	go build \
		-ldflags "-s -w -X github.com/kazhuravlev/database-gateway/internal/version.version=${VERSION}" \
		-o /out/database-gateway \
		./cmd/gateway

FROM alpine:3.23

ENV WORKDIR=/workdir

RUN mkdir -p ${WORKDIR}

WORKDIR ${WORKDIR}

VOLUME ${WORKDIR}

ENTRYPOINT ["/bin/gateway"]

COPY --from=builder /out/database-gateway /bin/gateway
