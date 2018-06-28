# build binary (Git required for fetching Go deps)
FROM gradle:4.8-jre-alpine
USER root
RUN \
	# Git required for fetching Go deps
	apk add --no-cache git \
	# gogradle supplied Go version compiled against glibc - fake it with musl
	&& mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2
COPY . /src
WORKDIR /src
RUN gradle --no-daemon --console plain --stacktrace build

FROM scratch
COPY --from=0 /src/google-cloud-tasks-pull-to-push /app/google-cloud-tasks-pull-to-push
COPY --from=0 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
ENTRYPOINT [ "/app/google-cloud-tasks-pull-to-push" ]
