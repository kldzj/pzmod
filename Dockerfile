# Minimal runtime image for pzmod.
#
# Built by GoReleaser (dockers_v2): the static binary (CGO disabled) is staged
# per-platform, so this copies ${TARGETPLATFORM}/pzmod. distroless/static ships
# CA certificates for HTTPS calls to the Steam Web API.
#
# Usage (pull the published image; no local build needed):
#   docker run --rm -e PZMOD_STEAM_KEY=<key> -v "$PWD:/data" \
#     ghcr.io/kldzj/pzmod --file /data/servertest.ini validate
#   # interactive TUI: add -it and mount a config volume at /config
#
# Pass your Steam key with -e PZMOD_STEAM_KEY=... and mount a volume at /config to
# persist profiles and backups (XDG_CONFIG_HOME points there).
FROM gcr.io/distroless/static-debian12:latest

ARG TARGETPLATFORM
ENV HOME=/root \
    XDG_CONFIG_HOME=/config

COPY ${TARGETPLATFORM}/pzmod /usr/bin/pzmod

ENTRYPOINT ["/usr/bin/pzmod"]
