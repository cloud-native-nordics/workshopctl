# nodejs-app

This is a sample application written in nodejs. Your task here is to:

a) Build the application from source code as a Docker image called `cloudnativenordics/nodejs-app:v0.1.0`
b) Create the Kubernetes manifests according to these specifications:

- It should be running in the `staging` namespace
- It should have the labels `app=nodejs-app` and `env=staging` applied consistently
- It should have liveness and readiness probes set up
- It should be running with 10 replicas
- It should be available at URL `cluster-XX.workshopctl.kubernetesfinland.com/nodejs-app`
- It should have a Secret with the content `SECRET_TOKEN=test1234` exposed to an environment
  variables.
- The Service should be available at `nodejs-app.staging.svc.cluster.local:80`, but forward
  the traffic to port 8080 for the Pods.
- Each Pod should be allowed to consume 20 milli-CPUs, and 16 MiB of RAM
- The workload exposes the `http_requests_total` counter at `/metrics`. Create a `ServiceMonitor`
  that targets the Service you created, and makes Prometheus scrape all the metrics endpoints.
