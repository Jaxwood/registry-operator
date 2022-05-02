# Registry Operator

Status: WIP

- [x] Feature Complete
- [ ] Add tests
- [ ] Documentation

An operator that will automatically sync an image pull secret across namespaces.

The operator listen on newly created namespaces/CRD and will automatically add/update the image pull secret to the namespace if missing/updated. This is useful incase secret rotation is needed.

A custom resources will setup the name of the image pull secret to sync (see example below)

```yaml
apiVersion: apps.jaxwood.com/v1
kind: RegistrySecret
metadata:
  name: image-pull-secret
  namespace: default
spec:
  imagePullSecretName: "regcred-dev"
  imagePullSecretKey: ".dockerconfigjson"
```
