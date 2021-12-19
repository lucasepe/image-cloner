# image-cloner

> This is just an exercise.

It's a kubernetes controller that watches the [`Deployments`](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/) and “caches” the images by re-uploading to your own registry repository and reconfiguring the applications to use these copies.

To configure the backup image registry you must specify the following environment variables:

| Variable              | Description                     | Example        |
|:----------------------|:--------------------------------|:---------------|
| IMAGE_CLONER_REGISTRY | target registry                 | localhost:5000 |
| IMAGE_CLONER_USER     | target registry login username  | testuser       |
| IMAGE_CLONER_PASS     | target registry login password  | testpass       |

For example, an option could be saving:

- `IMAGE_CLONER_REGISTRY` in a [configmap](https://kubernetes.io/docs/concepts/configuration/configmap/)
- `IMAGE_CLONER_USER` and `IMAGE_CLONER_PASS` in a [secret](https://kubernetes.io/docs/concepts/configuration/secret/)

## Local development and testing 

### Clone this project

```sh
$ git clone https://github.com/lucasepe/image-cloner.git
```

Change directory to `examples`.

```sh
$ cd examples
```

### Start KinD with a local docker registry enabled

```sh
$ make kind.up
```

> when you are done, to destroy the cluster, type:
>
> ```sh
> $ make kind.down
> ```

### Run the controller

Go back to root folder:

```sh
$ cd ..
```

Run the controller:

```sh
$ go run cmd/main.go
```

You should see something like this:

```txt
2021-12-18T18:46:48.693+0100	INFO	controller-runtime.metrics	metrics server is starting to listen	{"addr": ":8080"}
2021-12-18T18:46:48.693+0100	INFO	setup	starting manager
2021-12-18T18:46:48.693+0100	INFO	controller-runtime.manager	starting metrics server	{"path": "/metrics"}
2021-12-18T18:46:48.693+0100	INFO	controller.image-cloner	Starting Controller
2021-12-18T18:46:48.693+0100	INFO	controller.image-cloner	Starting workers	{"worker count": 1}
2021-12-18T18:46:48.693+0100	INFO	controller-runtime.manager.controller.deployment	Starting EventSource	{"reconciler group": "apps", "reconciler kind": "Deployment", "source": "kind source: /, Kind="}
2021-12-18T18:46:48.693+0100	INFO	controller-runtime.manager.controller.deployment	Starting Controller	{"reconciler group": "apps", "reconciler kind": "Deployment"}
2021-12-18T18:46:48.795+0100	INFO	controller-runtime.manager.controller.deployment	Starting workers	{"reconciler group": "apps", "reconciler kind": "Deployment", "worker count": 1}
```

### Create a deployment and watch the logs

```sh
$ kubectl apply -f examples/deploy-nginx.yml
```

As soon as the deployment will be ready, the controller will clone the image.

> In the next iteration, during the reconciliation loop, the controller will realize that the image has already been cloned and will skip the backup.

```txt
2021-12-18T18:51:25.484+0100    DEBUG   controller.image-cloner inspecting deployment   {"name": "nginx", "namespace": "demo-system"}
2021-12-18T18:51:25.484+0100    DEBUG   controller.image-cloner checking deployment container   {"name": "nginx", "namespace": "demo-system", "image": "nginx:1.21.4-alpine"}
2021-12-18T18:51:25.484+0100    DEBUG   controller.image-cloner cloning deployment image        {"name": "nginx", "namespace": "demo-system", "image": "nginx:1.21.4-alpine"}
2021-12-18T18:51:32.077+0100    DEBUG   controller.image-cloner updating deployment image       {"name": "nginx", "namespace": "demo-system", "From": "nginx:1.21.4-alpine", "To": "localhost:5000/nginx:1.21.4-alpine"}
2021-12-18T18:51:32.086+0100    DEBUG   controller.image-cloner deployment image reference updated      {"name": "nginx", "namespace": "demo-system", "image": "localhost:5000/nginx:1.21.4-alpine"}
2021-12-18T18:51:32.086+0100    DEBUG   controller.image-cloner inspecting deployment   {"name": "nginx", "namespace": "demo-system"}
2021-12-18T18:51:32.086+0100    DEBUG   controller.image-cloner checking deployment container   {"name": "nginx", "namespace": "demo-system", "image": "localhost:5000/nginx:1.21.4-alpine"}
2021-12-18T18:51:32.086+0100    DEBUG   controller.image-cloner image already cloned    {"name": "nginx", "namespace": "demo-system", "image": "localhost:5000/nginx:1.21.4-alpine"}
```

# Deploying this controller to your Kubernetes cluster

### Deploy the RBAC 

Since this controller must be able to update deployments in any namespace, it will at least need a cluster role that gives permissions to list, get and update those resources. 

To apply RBAC, type:

```sh
$ kubectl apply -f examples/rbac.yml
```

### Deploy the Controller

```sh
$ kubectl apply -f examples/image-cloner.yml
```

### Watch the logs

Loook at the `image-cloner-system` namespace to find the pod name:

```sh
$ kubectl get pods -n image-cloner-system
```

Retrieve the controller log stream:

```sh
$ kubectl logs --follow -n image-cloner-system image-cloner
```
