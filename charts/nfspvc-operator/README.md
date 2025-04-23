# nfspvc-operator

![Version: 0.0.0](https://img.shields.io/badge/Version-0.0.0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: latest](https://img.shields.io/badge/AppVersion-latest-informational?style=flat-square)

A Helm chart for the nfspvc-operator.

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` | Node affinity rules for scheduling pods. Allows you to specify advanced node selection constraints. |
| config | object | `{"name":"operator-config","reclaimPolicy":"Retain","storageClass":"brown"}` | Name of the ConfigMap used for configuration. |
| fullnameOverride | string | `""` |  |
| image.manager.pullPolicy | string | `"IfNotPresent"` | The pull policy for the image. |
| image.manager.repository | string | `"ghcr.io/dana-team/nfspvc-operator"` | The repository of the manager container image. |
| image.manager.tag | string | `""` | The tag of the manager container image. |
| livenessProbe | object | `{"initialDelaySeconds":15,"periodSeconds":20,"port":8081}` | Configuration for the liveness probe. |
| livenessProbe.initialDelaySeconds | int | `15` | The initial delay before the liveness probe is initiated. |
| livenessProbe.periodSeconds | int | `20` | The frequency (in seconds) with which the probe will be performed. |
| livenessProbe.port | int | `8081` | The port for the health check endpoint. |
| logAggregation | object | `{"auth":{"password":{"key":"logging-password","secretName":"logging-secret","value":"password"},"user":"logger"},"enabled":false,"loggingRef":"logging","record":"","tlsOpts":{"peer_verify":false,"ssl_version":"tlsv1_2"},"url":"https://logging-operator-logging-operator:9200/datastream-name/_bulk"}` | Configuration for log aggregation using logging-operator. |
| logAggregation.auth | object | `{"password":{"key":"logging-password","secretName":"logging-secret","value":"password"},"user":"logger"}` | Authentication options for the elasticsearch datastream. |
| logAggregation.auth.password.key | string | `"logging-password"` | The key in the secret containing the logging password. |
| logAggregation.auth.password.secretName | string | `"logging-secret"` | The name of the secret containing the logging password. |
| logAggregation.auth.password.value | string | `"password"` | The default value for the logging password |
| logAggregation.auth.user | string | `"logger"` | The username for the elasticsearch datastream. |
| logAggregation.enabled | bool | `false` | Enable log aggregation using logging-operator. |
| logAggregation.loggingRef | string | `"logging"` | Reference to Logging. |
| logAggregation.record | string | `""` | Record formatting options for log aggregation. |
| logAggregation.tlsOpts | object | `{"peer_verify":false,"ssl_version":"tlsv1_2"}` | tls options for the log aggregation. |
| logAggregation.url | string | `"https://logging-operator-logging-operator:9200/datastream-name/_bulk"` | The url for the elasticsearch datastream. |
| manager | object | `{"args":["--leader-elect","--health-probe-bind-address=:8081","--metrics-bind-address=:8443"],"command":["/manager"],"ports":{"health":{"containerPort":8081,"name":"health","protocol":"TCP"},"https":{"containerPort":8443,"name":"https","protocol":"TCP"},"webhook":{"containerPort":9443,"name":"webhook-server","protocol":"TCP"}},"resources":{"limits":{"cpu":"500m","memory":"128Mi"},"requests":{"cpu":"10m","memory":"64Mi"}},"securityContext":{"allowPrivilegeEscalation":false,"capabilities":{"drop":["ALL"]}},"volumeMounts":[{"mountPath":"/tmp/k8s-webhook-server/serving-certs","name":"cert","readOnly":true}],"webhookServer":{"defaultMode":420,"secretName":"webhook-server-cert"}}` | Configuration for the manager container. |
| manager.args | list | `["--leader-elect","--health-probe-bind-address=:8081","--metrics-bind-address=:8443"]` | Command-line arguments passed to the manager container. |
| manager.command | list | `["/manager"]` | Command-line commands passed to the manager container. |
| manager.ports | object | `{"health":{"containerPort":8081,"name":"health","protocol":"TCP"},"https":{"containerPort":8443,"name":"https","protocol":"TCP"},"webhook":{"containerPort":9443,"name":"webhook-server","protocol":"TCP"}}` | Port configurations for the manager container. |
| manager.ports.health.containerPort | int | `8081` | The port for the health check endpoint. |
| manager.ports.health.name | string | `"health"` | The name of the health check port. |
| manager.ports.health.protocol | string | `"TCP"` | The protocol used by the health check endpoint. |
| manager.ports.https.containerPort | int | `8443` | The port for the HTTPS endpoint. |
| manager.ports.https.name | string | `"https"` | The name of the HTTPS port. |
| manager.ports.https.protocol | string | `"TCP"` | The protocol used by the HTTPS endpoint. |
| manager.ports.webhook.containerPort | int | `9443` | The port for the webhook server. |
| manager.ports.webhook.name | string | `"webhook-server"` | The name of the webhook port. |
| manager.ports.webhook.protocol | string | `"TCP"` | The protocol used by the webhook server. |
| manager.resources | object | `{"limits":{"cpu":"500m","memory":"128Mi"},"requests":{"cpu":"10m","memory":"64Mi"}}` | Resource requests and limits for the manager container. |
| manager.securityContext | object | `{"allowPrivilegeEscalation":false,"capabilities":{"drop":["ALL"]}}` | Security settings for the manager container. |
| manager.volumeMounts | list | `[{"mountPath":"/tmp/k8s-webhook-server/serving-certs","name":"cert","readOnly":true}]` | Volume mounts for the manager container. |
| manager.webhookServer.defaultMode | int | `420` | The default mode for the secret. |
| manager.webhookServer.secretName | string | `"webhook-server-cert"` | The name of the secret containing the webhook server certificate. |
| nameOverride | string | `""` |  |
| nodeSelector | object | `{}` | Node selector for scheduling pods. Allows you to specify node labels for pod assignment. |
| readinessProbe | object | `{"initialDelaySeconds":5,"periodSeconds":10,"port":8081}` | Configuration for the readiness probe. |
| readinessProbe.initialDelaySeconds | int | `5` | The initial delay before the readiness probe is initiated. |
| readinessProbe.periodSeconds | int | `10` | The frequency (in seconds) with which the probe will be performed. |
| readinessProbe.port | int | `8081` | The port for the readiness check endpoint. |
| replicaCount | int | `1` | The number of replicas for the deployment. |
| securityContext | object | `{}` | Pod-level security context for the entire pod. |
| service | object | `{"httpsPort":8443,"protocol":"TCP","targetPort":"https"}` | Service configuration for the operator. |
| service.httpsPort | int | `8443` | The port for the HTTPS endpoint. |
| service.protocol | string | `"TCP"` | The protocol used by the HTTPS endpoint. |
| service.targetPort | string | `"https"` | The name of the target port. |
| tolerations | list | `[]` | Node tolerations for scheduling pods. Allows the pods to be scheduled on nodes with matching taints. |
| volumes | list | `[{"name":"cert","secret":{"defaultMode":420,"secretName":"webhook-server-cert"}}]` | Configuration for the volumes used in the deployment. |
| webhookService | object | `{"ports":{"port":443,"protocol":"TCP","targetPort":9443},"type":"ClusterIP"}` | Configuration for the webhook service. |

