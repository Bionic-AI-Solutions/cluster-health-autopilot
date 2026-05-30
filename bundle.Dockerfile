# OLM bundle image for cha-operator. A bundle image is a tiny scratch-
# based image whose ONLY purpose is to deliver the manifests/ + metadata/
# tree to the OLM catalog. No binaries; no runtime.
#
# Build:    docker build -f bundle.Dockerfile -t docker4zerocool/cha-operator-bundle:v1.9.4 .
# Push:     docker push docker4zerocool/cha-operator-bundle:v1.9.4
# Test:     operator-sdk run bundle docker4zerocool/cha-operator-bundle:v1.9.4
#
# The bundle image's labels are read by OLM's registry sub-system to
# place this bundle into the right package / channel. Lines below must
# stay in sync with bundle/metadata/annotations.yaml; the two carry the
# same data in different containers (image labels vs. on-disk file).
FROM scratch

# Bundle format + structural pointers.
LABEL operators.operatorframework.io.bundle.mediatype.v1=registry+v1
LABEL operators.operatorframework.io.bundle.manifests.v1=manifests/
LABEL operators.operatorframework.io.bundle.metadata.v1=metadata/

# Package + channel placement.
LABEL operators.operatorframework.io.bundle.package.v1=cluster-health-autopilot
LABEL operators.operatorframework.io.bundle.channels.v1=alpha
LABEL operators.operatorframework.io.bundle.channel.default.v1=alpha

# Optional: scorecard test config (operator-sdk scorecard).
# LABEL operators.operatorframework.io.test.config.v1=tests/scorecard/

# The actual payload — manifests + metadata copied into the image at
# the exact paths the labels declare.
COPY bundle/manifests /manifests/
COPY bundle/metadata /metadata/
