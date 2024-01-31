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
    runAsUser: 1000
    allowPrivilegeEscalation: false
    seccompProfile:
      type: RuntimeDefault
    capabilities:
      drop:
      - ALL
    readOnlyRootFilesystem: true
  containers:
    - name: pvc-test-container
      image: nginx
      volumeMounts:
        - name: test-volume
          mountPath: /data
  volumes:
    - name: test-volume
      persistentVolumeClaim:
        claimName: test-pvc
`)
