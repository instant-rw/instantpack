FROM alpine

# The instantpack binary is built during the GoReleaser process, which handles the
# multi-arch build and places the binary in the build context root.
COPY instantpack /

ENTRYPOINT ["/instantpack", "frontend"]
