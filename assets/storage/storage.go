package storage

var Namespace = []byte(`
apiVersion: v1
kind: Namespace
metadata:
  name: test-storage
`)

var PVC = []byte(`
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: test-pvc
  namespace: test-storage
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
`)

var Pod = []byte(`
apiVersion: v1
kind: Pod
metadata:
  name: pvc-test-pod
  namespace: test-storage
spec:
  securityContext:
    runAsNonRoot: true
    runAsUser: 1001
  containers:
    - name: pvc-test-container
      image: gsoci.azurecr.io/giantswarm/nginx-unprivileged
      volumeMounts:
        - name: test-volume
          mountPath: /data
      securityContext:
        allowPrivilegeEscalation: false
        seccompProfile:
          type: RuntimeDefault
        capabilities:
          drop:
          - ALL
  volumes:
    - name: test-volume
      persistentVolumeClaim:
        claimName: test-pvc
`)
