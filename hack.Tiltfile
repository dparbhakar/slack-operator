allow_k8s_contexts('neokube')
default_registry('eu.gcr.io/neokube')

docker_build(
  "slack-operator",
  ".",
  dockerfile="./Dockerfile",
  only=['.'],
)

k8s_custom_deploy(
  "slack-operator",
  apply_cmd=[
    "bash",
    "-c",
    "kubectl -n slack-operator -v=0 set image deployment/slack-operator *=$TILT_IMAGE_0 > /dev/null && kubectl -n slack-operator get deployment/slack-operator -o yaml",
  ],
  delete_cmd="echo slack-operator managed outside of Tilt",
  deps=".",
  image_deps=["slack-operator"]
)
