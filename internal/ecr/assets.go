package ecr

var deploymentManifest = []byte(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: alpine
  namespace: default
  labels:
    app: ecr-private-pull-test
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ecr-private-pull-test
  template:
    metadata:
      labels:
        app: ecr-private-pull-test
    spec:
      containers:
      - name: alpine-container
        image: 992382781567.dkr.ecr.eu-west-2.amazonaws.com/giantswarm/alpine:latest
        command: ["/bin/sh", "-c"]
        args:
          - date && sleep 300
        securityContext:
          runAsUser: 1000
          runAsGroup: 3000
          runAsNonRoot: true
          allowPrivilegeEscalation: false
          capabilities:
            drop:
            - ALL
          privileged: false
          readOnlyRootFilesystem: true
          seccompProfile:
            type: RuntimeDefault
        resources:
          limits:
            cpu: "100m"
            memory: "128Mi"
          requests:
            cpu: "50m"
            memory: "64Mi"
`)
