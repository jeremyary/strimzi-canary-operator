# Strimzi-canary-operator

PoC project wrapping [go-stoker](https://github.com/jeremyary/go-stoker) tool in Go-based operator for ease of deployment within Strimzi Kafka namespace. 

Example CR for deployment (assuming kafka cluster name of 'tenant2':
```
apiVersion: canary.strimzi.io/v1
kind: Canary
metadata:
  name: canary-sample
spec:
  # Add fields here
  size: 1
  kafkaConfig:
    bootstrapUrl: 'tenant2-kafka-bootstrap-url-here:443'
  trafficProducer:
    topic: 'traffic-gen-topic'
    sendRate: '5'
    clientId: 'traffic-gen-client'
  secretVolumes:
    - name: 'cluster-ca'
      mountPath: '/etc/cluster-ca'
      secretName: 'tenant2-cluster-ca-cert'
    - name: 'client-ca'
      mountPath: '/etc/client-ca'
      secretName: 'tenant2-clients-ca'
    - name: 'client-ca-cert'
      mountPath: '/etc/client-ca-cert'
      secretName: 'tenant2-clients-ca-cert'
```
