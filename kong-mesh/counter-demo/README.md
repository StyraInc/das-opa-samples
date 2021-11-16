# Kong Mesh with OPA on Kubernetes and Styra DAS

In this guide we will utilize [Open Policy Agent](https://www.openpolicyagent.org) (OPA) for external, decoupled policy for application authorization with an example Kong Mesh web app. The [Styra Declarative Authorization Service](https://www.styra.com) (DAS) will be leveraged for authoring declarative policy in [Rego](https://www.openpolicyagent.org/docs/latest/policy-language/) and distribution to OPA, along with centralized decision logging and auditing.

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

Install and run Kong Mesh on your Kubernetes cluster per the [Kong Mesh Installation guide](https://docs.konghq.com/mesh/1.4.x/installation/kubernetes/)

_This guide has been tested with Kong Mesh 1.4.1_

### 3. Deploy the Demo App

Deploy the Kuma [Counter Demo](https://github.com/kumahq/kuma-counter-demo) app per the [Quickstart guide](https://kuma.io/docs/1.3.1/quickstart/kubernetes/).

Execute the instructions through the [Set up and run](https://kuma.io/docs/1.3.1/quickstart/kubernetes/#set-up-and-run) step only.  (For this guide, there is no need to enable Mutual TLS, Traffic Permissions or Traffic Metrics.)

### 4. Expose the Demo App Service

Our goal in this guide is to implement authorization via the OPA engine embedded in the dataplane proxy for the `demo-app` web application. Therefore we want end user requests to the front end of the app to go through the sidecar proxy instance. To ensure the HTTP requests flow through the proxy we need to expose the app via a Kubernetes `NodePort` or `LoadBalancer` service.  _(If we used `port-forward` to access the web app, the requests would bypass the dataplane proxy and OPA authorization.)_

> Could we use Kong Gateway as a Kubernetes Ingress to the `demo-app` Service instead? 

> *Yes!* Kong Gateway also supports OPA via the Kong Enterprise [OPA Plugin](https://docs.konghq.com/hub/kong-inc/opa/). But that is a tutorial for another day :) - today we will continue to work with Kong Mesh only.

1. Update the kubectl context to the `kuma-demo` namespace
```sh
kubectl config set-context --current --namespace=kuma-demo
```

2. Patch the `demo-app` service to change the type to `NodePort`
```sh
kubectl patch service demo-app -p '{"spec":{"type":"NodePort"}}'
```

3. Set the `SERVICE_URL` environment variable to the `demo-app` service IP/port.

**minikube:**
```sh
export SERVICE_PORT=$(kubectl -n kuma-demo get service demo-app -o jsonpath='{.spec.ports[?(@.port==5000)].nodePort}')
export SERVICE_HOST=$(minikube ip)
export SERVICE_URL=$SERVICE_HOST:$SERVICE_PORT
echo $SERVICE_URL
```

4. Open `http://${SERVICE_URL}` in your browser and verify the app is working.

### 5. Create a Kong Mesh System in Styra DAS

1. Go to your Styra DAS Free tenant and Create a System **(+)**
2. Select **Kong Mesh** System type
3. Provide a name of **Counter Demo App**
4. Toggle **off** the Launch Quick Start option 
5. Select **Add System**.

### 6. Configure Kong Mesh and OPA for Styra DAS

1. Go to **Counter Demo App > Settings > Install**
2. Copy and run the first command `# Configure Kong Mesh`

    This command creates/updates the following resources:
    * `opapolicy.kuma.io/opa-policy-das` (created).  This resource defines the Kong Mesh [OPAPolicy](https://docs.konghq.com/mesh/1.4.x/features/opa/), and configures the OPA engine within the dataplane proxy to use an external management service (named “styra”) to manage the OPA rules and decisions.
    * `proxytemplate.kuma.io/opa-ext-authz-filter` (created).  This resource configures the dataplane proxy to utilize OPA for authorization for outbound egress requests.  The default configuration in Kong Mesh for the proxy sets up OPA for ingress request authorization only. Styra adds the egress configuration as well for users who wish to enforce egress authorization rules. (In this lab we won’t create egress rules, but the capability is there.)
    * `configmap/kong-mesh-control-plane-config` (configured).  This resource adds a configuration override to the default OPA config created by Kong Mesh.  This override defines the main `path` for the policy rule that the OPA plugin should invoke during an authorization request.  Styra DAS has an opinionated package and rule structure for policies, and therefore the `path` value needs to match the package/rule endpoint that DAS creates in an OPA instance.

3. Skip the second command `# Enable sidecar-injection on default namespace`. For this guide the `demo-app` is running in the `kuma-demo` namespace, and sidecar-injection is already enabled.

4. Copy and run the third command `# Install Styra Local Plane (SLP)`.  The SLP is not a required component in an OPA+DAS architecture, but it adds an extra layer of availability and performance for the OPA engines running in the cluster.  Styra recommends its usage for any service mesh architecture.  Here is a picture of how all the parts fit together:

![](images/all-the-things.png)

> Don’t worry about the picture of Alice and the JWT token yet, we’ll get to that later.  At this point all the components represented by the boxes are deployed and running.

> Neither the SLP nor DAS are in the hot path for an authorization request.  The OPA engine embedded within the Kong dataplane proxy periodically retrieves the policy rules from DAS (via the SLP) and stores them in memory. No external requests to the SLP (nor any other system) are needed for the dataplane proxy + embedded OPA engine to authorize an individual request.

5. Restart all the Deployments in the `kuma-demo` namespace to enable the configuration changes
```sh
kubectl rollout restart deployments

kubectl get pods
NAME                                                    READY   STATUS    RESTARTS   AGE
demo-app-cb8c85c76-xv9cx                                2/2     Running   0          53s
redis-8474446f-96gzk                                    2/2     Running   0          53s
slp-a33f416cca2c45b6afe885a59791f3da-5bb744f4b8-zqxln   2/2     Running   0          53s
```

6. Refresh `http://${SERVICE_URL}` in your browser and verify the app is working. Click on each of the **Increment** and **Reset** buttons.

### 7. Implement Authorization Rules in OPA via Styra DAS

1. Go to **Counter Demo App > policy > ingress > rules.rego**

    Review the current policy.  This policy has a very simple `allow` rule that will always return `true` in the response to any authorization request.

2. Go to **Counter Demo App**, then click **Decisions**.

    You will see a log of all **Allowed** decisions for the prior OPA authorization queries.  
    
    These decisions were captured by OPA and sent to DAS when you tested the app functionality in step 6. You will see different types of requests across a variety of methods (e.g. `GET`, `POST`, `DELETE`) and URI paths (e.g. `/version`, `/counter`, `/increment`).

3. Find a decision result for a `DELETE` request by typing `method:DELETE` in the search box at the bottom of the Decisions page. Click the Replay icon next to the **Allowed** decision log line.

    The **policy > ingress > rules.rego** file editor will be opened in the browser.

4. Replace the contents of the editor with the following Rego code:

```rego
package policy.ingress

# Add policy/rules to allow or deny ingress traffic
allow = true

deny = true {
  input.attributes.request.http.method == "DELETE"
  input.attributes.request.http.path == "/counter"
}
```

  * This policy will now explicitly deny access to the `/counter` path for the `DELETE` method, and allow all other requests.  
  
  * You can use the **Preview** and **Validate** buttons to evaluate the draft policy and run change impact analysis via decision log replay.
  
  * Click **Publish** to save and distribute the policy.  Within 30 seconds or so, the OPA engine in the dataplane proxy sidecar will automatically load the policy change.

5. Return to the Counter demo app in the browser click the **Reset** button.

    The request now returns an error.
  
    > In a real world app we would update the UI to handle the error with a more appropriate message (or better yet, we could integrate the app UI with OPA policy as well, to _hide_ the **Reset** button when the user is not authorized - but this is beyond the scope of this guide).
  
    You can view the details of the response, including the 403 status code in your browser DevTools Network panel. 

6. Go to **Counter Demo App**, then click **Decisions**.

    You will see a new **Denied** decision log entry for the `DELETE` `/counter` request.

### 8. Implement fine-grained AuthZ with JWT validation

While combining `allow` and `deny` rules in a policy is possible with DAS and OPA, a more realistic/idiomatic Rego policy would implement a secure "deny by default" approach with explicit allow rules.

With Rego we can also implement any kind of authorization model including [RBAC](https://www.openpolicyagent.org/docs/latest/comparison-to-other-systems/#role-based-access-control-rbac), [ABAC](https://www.openpolicyagent.org/docs/latest/comparison-to-other-systems/#attribute-based-access-control-abac), and more.  A popular approach to ABAC policies for web applications is to check attributes (or claims) in a JSON Web Token ([JWT](https://www.openpolicyagent.org/docs/latest/faq/#json-web-tokens-jwts)).

Let’s create a final policy with the following rules:
* Deny all requests by default
* Allow any `GET` request
* Allow any `POST` request
* Allow a `DELETE` request on the `/counter` path if the request includes a JWT token with the necessary claims attributes.

Here is the final policy:
```rego
package policy.ingress

# default deny
default allow = false

# allow all GETs
allow = true {
  input.attributes.request.http.method == "GET"
}

# allow all POSTs
allow = true {
  input.attributes.request.http.method == "POST"
}

# allow DELETE on /counter, only if the token claims contain the delete permission
allow = true {
  input.attributes.request.http.method == "DELETE"
  input.attributes.request.http.path == "/counter"
  claims.permissions[_] == "delete"
}

# verify the token signature and decode the payload
claims := payload {
  io.jwt.verify_hs256(bearer_token, "weaksecret")
  [_, payload, _] := io.jwt.decode(bearer_token)
}

# read the bearer token from the headers
bearer_token := t {
  v := input.attributes.request.http.headers.authorization
  startswith(v, "Bearer ")
  t := substring(v, count("Bearer "), -1)
}
```

1. Replace the contents of the `policy/ingress/rules.rego` file with the above policy.

2. Click **Publish** to save and distribute the policy.  Within 30 seconds or so, the OPA engine in the dataplane proxy sidecar will automatically load the policy change.

3. Refresh the app in the browser and click on each of the **Increment** and **Reset** buttons.

    As before, the **Increment** request will be **Allowed** and the **Reset** request will be **Denied**

    However, if we execute a `DELETE` request to `/counter` with a proper JWT the request should be allowed.  
  
4. Execute a request with a JWT via curl:
```sh
curl -X DELETE -H "Authorization: Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJleHAiOjMxNTM2MDAwMCwiaWF0IjoxNjM2NzYzMDA2LCJwZXJtaXNzaW9ucyI6WyJkZWxldGUiXSwidXNlcm5hbWUiOiJhbGljZSJ9.BrFGBkO-8PIL7hgj7geiCWJvoGUpm1NUBj7NHx9mXI8" "http://${SERVICE_URL}/counter"
```

You can use http://jwt.io to view the contents of the encoded token that is used in the curl command above. Once decoded, you will see the token contains the following `permissions` claim:
```json
"permissions": [
  "delete"
]
```

> The HS256 algorithm with a hardcoded secret is used in this guide for simplicity.  OPA supports a variety of algorithms for [Token Verification](https://www.openpolicyagent.org/docs/latest/policy-reference/#token-verification) that would be a better choice for a production implementation.

5. Check the Styra DAS **Decisions** to see the final `DELETE` request to `/counter` is **Allowed**.

> The Authorization header is removed from the DAS decision log by default, so you won’t see the bearer token value in decision log JSON, rather it will be identified as an `erased` field.  The [Decision Masking](https://www.openpolicyagent.org/docs/latest/management-decision-logs/#masking-sensitive-data) behavior is controlled by the `/system/log/mask.rego` policy.  You can comment out, delete or modify this policy as desired if you want the token values to remain in the logs (which is very useful for development, but generally not recommended for production environments).  
