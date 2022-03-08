# Kong Mesh with OPA on Kubernetes and Styra DAS

In this guide we will utilize [Open Policy Agent](https://www.openpolicyagent.org) (OPA) for external, decoupled policy for application authorization with an example Kong Mesh gRPC app. The [Styra Declarative Authorization Service](https://www.styra.com) (DAS) will be leveraged for authoring declarative policy in [Rego](https://www.openpolicyagent.org/docs/latest/policy-language/) and distribution to OPA, along with centralized decision logging and auditing.

## Prerequisites

Before you get started, you'll need:

* A valid Kong Mesh license.
* A Styra DAS account. You can [sign up for free](https://signup.styra.com).
* A Kubernetes cluster ([minikube](https://minikube.sigs.k8s.io/docs/) will be used throughout this guide).

## Steps

### 1. Start Minikube
```sh
minikube start
```

### 2. Install Kong Mesh

Install and run Kong Mesh on your Kubernetes cluster per the [Kong Mesh Installation guide](https://docs.konghq.com/mesh/latest/installation/kubernetes/)

_This guide has been tested with Kong Mesh 1.6.0_

### 3. Deploy the Demo App

Deploy the example [grpcbin](https://github.com/moul/grpcbin) app into the `default` namespace
```sh
# enable kuma/kong-mesh sidecar injection on the default namespace
kubectl annotate namespace default kuma.io/sidecar-injection=enabled

kubectl apply -f grpcbin.yaml
```

Set the `SERVICE_URL` environment variable to the `grpcbin` service IP/port.

**minikube:**
```sh
export SERVICE_PORT=$(kubectl -n default get service grpcbin -o jsonpath='{.spec.ports[?(@.port==9000)].nodePort}')
export SERVICE_HOST=$(minikube ip)
export SERVICE_URL=$SERVICE_HOST:$SERVICE_PORT
echo $SERVICE_URL
```

Send a test request to the app via [grpcurl](https://github.com/fullstorydev/grpcurl).
```sh
grpcurl -plaintext -proto hello.proto ${SERVICE_URL} hello.HelloService/SayHello
```
Response:
```json
{
  "reply": "hello noname"
}
```

### 4. Create a Kong Mesh System in Styra DAS

1. Go to your Styra DAS Free tenant and Create a System **(+)**
2. Select **Kong Mesh** System type
3. Provide a name of **grpcbin**
4. Toggle **off** the Launch Quick Start option 
5. Select **Add System**.

### 5. Configure Kong Mesh and OPA for Styra DAS

1. Go to **grpcbin > Settings > Install**
2. Copy and run the first command `# Configure Kong Mesh`

    This command creates/updates the following resources:
    * `opapolicy.kuma.io/opa-policy-das` (created).  This resource defines the Kong Mesh [OPAPolicy](https://docs.konghq.com/mesh/latest/features/opa/), and configures the OPA engine within the dataplane proxy to use an external management service (named “styra”) to manage the OPA rules and decisions.
    * `proxytemplate.kuma.io/opa-ext-authz-filter` (created).  This resource configures the dataplane proxy to utilize OPA for authorization for outbound egress requests.  The default configuration in Kong Mesh for the proxy sets up OPA for ingress request authorization only. Styra adds the egress configuration as well for users who wish to enforce egress authorization rules. (In this lab we won’t create egress rules, but the capability is there.)
    * `configmap/kong-mesh-control-plane-config` (configured).  This resource adds a configuration override to the default OPA config created by Kong Mesh.  This override defines the main `path` for the policy rule that the OPA plugin should invoke during an authorization request.  Styra DAS has an opinionated package and rule structure for policies, and therefore the `path` value needs to match the package/rule endpoint that DAS creates in an OPA instance.

3. Skip the second command `# Enable sidecar-injection on default namespace`. We previously enabled sidecar-injection for the `default` namespace in step 3 of this guide.

4. Copy and run the third command `# Install Styra Local Plane (SLP)`.  The SLP is not a required component in an OPA+DAS architecture, but it adds an extra layer of availability and performance for the OPA engines running in the cluster.  Styra recommends its usage for any service mesh architecture.  

5. Restart the `grpcbin` deployment to enable the configuration changes
```sh
kubectl rollout restart deployment grpcbin

kubectl get pods
NAME                                     READY   STATUS    RESTARTS   AGE
grpcbin-7c4b9998f6-zshhg                 2/2     Running   0          20s
slp-c7a50732489a4879a76bed61e550e80c-0   2/2     Running   0          61s
```

6. Test the app via `grpcurl` again to confirm it is working.
```sh
# test the SayHello endpoint
grpcurl -plaintext -proto hello.proto ${SERVICE_URL} hello.HelloService/SayHello
# test the LotsOfReplies endpoint
grpcurl -plaintext -proto hello.proto ${SERVICE_URL} hello.HelloService/LotsOfReplies
```

### 7. Implement Authorization Rules in OPA via Styra DAS

1. Go to **grpcbin > policy > ingress > rules.rego**

    Review the current policy.  This policy has a very simple `allow` rule that will always return `true` in the response to any authorization request.

2. Go to **grpcbin**, then click **Decisions**.

    You will see a log of all **Allowed** decisions for the prior OPA authorization queries.  
    
    These decisions were captured by OPA and sent to DAS when you tested the app functionality in step 6. You will see `POST` requests to both the `/hello.HelloService/SayHello` and `/hello.HelloService/LotsOfReplies` endpoint paths.

3. Find a decision result for a `SayHello` request by typing `path:/hello.HelloService/SayHello` in the search box at the bottom of the Decisions page. Click the Replay icon next to the **Allowed** decision log line.

    The **policy > ingress > rules.rego** file editor will be opened in the browser.

4. Replace the contents of the editor with the following Rego code:

```rego
package policy.ingress

# Add policy/rules to allow or deny ingress traffic
default allow = false

allow = true {
  input.parsed_path == ["hello.HelloService", "SayHello"]
}
```

  * This policy will now explicitly deny access by default - and allow access to the `/hello.HelloService/SayHello` path.
  
  * You can use the **Preview** and **Validate** buttons to evaluate the draft policy and run change impact analysis via decision log replay.
  
  * Click **Publish** to save and distribute the policy.  Within 30 seconds or so, the OPA engine in the dataplane proxy sidecar will automatically load the policy change.

5. Retest the app via `grpcurl`.

Test the `SayHello` endpoint
```sh
grpcurl -plaintext -proto hello.proto ${SERVICE_URL} hello.HelloService/SayHello
```
Result is **Allowed** with output:
```json
{
  "reply": "hello noname"
}
```

Test the `LotsOfReplies` endpoint
```sh
grpcurl -plaintext -proto hello.proto ${SERVICE_URL} hello.HelloService/LotsOfReplies
```
Result is **Denied** with output:
```
ERROR:
  Code: PermissionDenied
  Message:
```

6. Go to **grpcbin**, then click **Decisions**.

    You will see a new **Denied** decision log entry for the `hello.HelloService/LotsOfReplies` request.

    > Note: clear the Decisions filter in order to see all decisions
