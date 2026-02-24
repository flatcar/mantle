FROM gentoo/portage:latest AS portage

FROM gentoo/stage3:latest AS gentoo

# This is to know if we are building for amd64 or arm64.
ARG TARGETARCH

# Copy the entire portage volume in.
COPY --from=portage /var/db/repos/gentoo /var/db/repos/gentoo

RUN --mount=type=bind,source=prepare_bccrootfs.sh,target=/prepare_bccrootfs.sh /prepare_bccrootfs.sh ${TARGETARCH}

FROM scratch
COPY --from=gentoo /bccrootfs /
