# K3s Key Server Deployment Guide

## 1. Install k3s (skip if already using linux distro)

Look into [k3s.io Documentation](https://docs.k3s.io/) and github readme of k3s project

```bash
subhashp7i@Veeras-MBP ~ % curl -sfL https://get.k3s.io | sh -
[ERROR]  Can not find systemd or openrc to use as a process supervisor for k3s
```

Needs systemd hence installed a ubuntu vm with canonical's multipass tool https://github.com/canonical/multipass 

```bash
subhashp7i@Veeras-MBP k3s % brew reinstall --cask multipass
subhashp7i@Veeras-MBP k3s % multipass launch --name k3s-vm --mem 8G --disk 40G
subhashp7i@Veeras-MBP k3s % multipass info k3s-vm
Name:           k3s-vm
State:          Running
Snapshots:      0
IPv4:           192.168.64.2
                10.42.0.0
                10.42.0.1
                172.17.0.1
Release:        Ubuntu 24.04 LTS
Image hash:     2c47dbf04477 (Ubuntu 24.04 LTS)
CPU(s):         1
Load:           0.20 0.12 0.10
Disk usage:     4.4GiB out of 38.7GiB
Memory usage:   1.0GiB out of 7.7GiB
Mounts:         --
```

Log into the vm:

```bash
subhashp7i@Veeras-MBP k3s % multipass shell k3s-vm
```

## 2. Install k3s (Attempt 2)

```bash
ubuntu@k3s-vm:~$ curl -sfL https://get.k3s.io | sh -
```

Super simple one cmd kubernetes cluster installation , It worked now 🙂
Adding new nodes to the cluster should also be easy.

```bash
ubuntu@k3s-vm:~$ sudo kubectl get nodes
NAME     STATUS   ROLES                  AGE   VERSION
k3s-vm   Ready    control-plane,master   20h   v1.30.3+k3s1

ubuntu@k3s-vm:~$ sudo kubectl top nodes
NAME     CPU(cores)   CPU%   MEMORY(bytes)   MEMORY%
k3s-vm   16m          1%     865Mi           10%

ubuntu@k3s-vm:~$ sudo kubectl get pods -A
NAMESPACE     NAME                                      READY   STATUS             RESTARTS          AGE
kube-system   coredns-576bfc4dc7-gflhc                  1/1     Running            0                 21h
kube-system   helm-install-traefik-947zm                0/1     Completed          1                 21h
kube-system   helm-install-traefik-crd-q8szl            0/1     Completed          0                 21h
kube-system   local-path-provisioner-6795b5f9d8-5drrn   1/1     Running            0                 21h
kube-system   metrics-server-557ff575fb-5mpcs           1/1     Running            0                 21h
kube-system   svclb-traefik-04b6df81-m5bl2              2/2     Running            0                 21h
kube-system   traefik-5fb479b77-lb2pp                   1/1     Running            0                 21h

ubuntu@k3s-vm:~$ sudo kubectl get ns -A
NAME              STATUS   AGE
default           Active   21h
kube-node-lease   Active   21h
kube-public       Active   21h
kube-system       Active   21h
```

Ran docker ps to get some idea of what containers and images are running on the vanilla k3s platform, noticed docker binary is not present , checked that the runtime is containerd, used the ctr cli cmds to verify

```bash
> sudo ctr images ls
> sudo ctr containers ls
> sudo ctr image pull docker.io/library/busybox:latest (just to check internet access for images that we will use later to build the the key-server application image)
```

Some more executed commands in order to setup the env:

```bash
> mkdir key-server-chart
> cd key-server-chart/
> sudo kubectl create namespace key-server
> sudo kubectl get ns -A
```

Downloaded the go binary from official site for arm64 to test the key-server app manually before deploying with helm
```bash
  # Download the ARM64 version of Go
  wget https://go.dev/dl/go1.21.1.linux-arm64.tar.gz
  # Extract the archive to /usr/local
  sudo tar -C /usr/local -xzf go1.21.1.linux-arm64.tar.gz
  # Add Go to PATH in the .bashrc file
  echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
  # Apply the changes to the current session
  source ~/.bashrc
```

Made sure pkg versions match across the configuration files and dockerfile

## 3. Application Logic for key-server

Passed the requirements to an LLM(closedAI , hopefully they don't train on my data lol) and got the baseline code and started modifying to meet the requirements and edge test cases for unique key generation based on custom args (max-size and srv-port). (golang is not my daily driver at the moment(hopefully in near future :) ) so pls forgive any bad coding practices). Added functionality to monitor every request by registering counters for http verbs/labels (eg 500 400 200 status codes as label).

Created these files in the working dir:  key-server-chart

```bash
ubuntu@k3s-vm:~$ pwd
/home/ubuntu

ubuntu@k3s-vm:~$ ls
go  go1.21.1.linux-arm64.tar.gz  key-server-project

ubuntu@k3s-vm:~$ cd key-server-project/

ubuntu@k3s-vm:~/key-server-project$ ls
Dockerfile  go.mod  go.sum  key-server-0.1.0.tgz  key-server-chart  key-server-helmchart.yaml  key-server.go  key-server_test.go

ubuntu@k3s-vm:~/key-server-project$ ls key-server-chart/
Chart.yaml  templates  values.yaml

ubuntu@k3s-vm:~/key-server-project$ ls key-server-chart/templates/
deployment.yaml  service.yaml
```

Code uploaded to github repo.

Basic Math to consider for the key-server application while deploying at scale: (Inorder for us to not run out of unique tokens and run into hash collisions)

```
Example Math: Request 16 Random(pseudo) Bytes as input length of the key
Request: GET http://localhost:1123/key/16
Server Console Output:
Generated 16 random bytes: [77 44 110 15 143 203 242 157 92 116 155 52 93 203 243 161]

We then convert it into hex for easy representation purpose.


HTTP Response:

4d2c6e0f8fcbf29d5c749b345dcbf3a1
```

77 in hex is 4d

44 in hex is 2c

And so on 4d2c6e0f8fcbf29d5c749b345dcbf3a1

Hexadecimal is often used for representing bytes because it's more compact and aligns nicely with byte boundaries (each hex digit(16 possible values can be represented in 4 binary bits), so two hex digits represent a byte(8 bits)).


For 16 random bytes, the number of combinations is 256^{16} (~10^{38} 100 undecillions)  (as byte has 8 bits with values ranging from 0-255)

Small key sizes mean same random bytes probability goes up


10^{38} ~ 100 undecillions is about 42 (:D) orders of magnitude smaller than the estimated number of atoms in the universe 10^{80} (Quindecillion)  🚀  i guess we should be good with 16 as min-size recommendation for end users to not face any issues with duplicates.


With key length = 3 = 6 hex digits
	• Probability of Collision: With 16,777,216 possible combinations, the probability of generating duplicate values is low, but not impossible, especially as the number of generated tokens approaches this limit.

## 4. Testing the code

Cmds executed in order:

```bash
> go mod init key-server
> go get github.com/prometheus/client_golang/prometheus
> go mod tidy
> cat go.mod
> Cat go.sum
```

Once pkg dependencies issues are resolved , compile the server logic

```bash
ubuntu@k3s-vm:~/key-server-project$  go build -o key-server key-server.go

ubuntu@k3s-vm:~/key-server-project$ ls
Dockerfile  go.mod  go.sum  key-server  key-server-0.1.0.tgz  key-server-chart  key-server-helmchart.yaml  key-server.go  key-server_test.go
```

Launch the app: 

```bash
ubuntu@k3s-vm:~/key-server-project$ sudo ./key-server --help
Usage of ./key-server:
  -max-size int
    	maximum key size (default 1024)
  -srv-port int
    	server listening port (default 1123)


ubuntu@k3s-vm:~/key-server-project$ sudo ./key-server
2024/08/04 19:12:12 Starting server on port 1123...
```

New terminal tab:
```bash
ubuntu@k3s-vm:~$ curl -s http://localhost:1123/key/12 && echo
7bd580349753abe28c2c9195
```

Some negative test cases:
```bash
ubuntu@k3s-vm:~/key-server-project$ curl -s http://10.43.225.158:1123/key/0
Invalid key length
ubuntu@k3s-vm:~/key-server-project$ curl -s http://10.43.225.158:1123/key/10000
Invalid key length
```

## 5. Test case Automation 

Created a new script which tests both /key/{length}  API and /metrics API health endpoints for different positive, negative and false positive scenarios.

```bash
ubuntu@k3s-vm:~/key-server-project$ vi key-server_test.go   
```

Code is hosted in git repo

The testing framework simulates HTTP requests directly against the handler functions without needing to run the server.

```bash
ubuntu@k3s-vm:~/key-server-project$ go test -v
=== RUN   TestHandleKeyRequest
=== RUN   TestHandleKeyRequest/Valid_key_length
    key-server_test.go:32: Running test: Valid key length
    key-server_test.go:51: Test Valid key length: received expected status code 200
    key-server_test.go:62: Test Valid key length: received expected key length 32 (in hex)
=== RUN   TestHandleKeyRequest/Zero_key_length
    key-server_test.go:32: Running test: Zero key length
    key-server_test.go:51: Test Zero key length: received expected status code 400
=== RUN   TestHandleKeyRequest/Negative_key_length
    key-server_test.go:32: Running test: Negative key length
    key-server_test.go:51: Test Negative key length: received expected status code 400
=== RUN   TestHandleKeyRequest/Non-numeric_key_length
    key-server_test.go:32: Running test: Non-numeric key length
    key-server_test.go:51: Test Non-numeric key length: received expected status code 400
=== RUN   TestHandleKeyRequest/Maximum_valid_key_length
    key-server_test.go:32: Running test: Maximum valid key length
    key-server_test.go:51: Test Maximum valid key length: received expected status code 200
    key-server_test.go:62: Test Maximum valid key length: received expected key length 2048 (in hex)
=== RUN   TestHandleKeyRequest/Exceeds_maximum_key_length
    key-server_test.go:32: Running test: Exceeds maximum key length
    key-server_test.go:51: Test Exceeds maximum key length: received expected status code 400
--- PASS: TestHandleKeyRequest (0.00s)
    --- PASS: TestHandleKeyRequest/Valid_key_length (0.00s)
    --- PASS: TestHandleKeyRequest/Zero_key_length (0.00s)
    --- PASS: TestHandleKeyRequest/Negative_key_length (0.00s)
    --- PASS: TestHandleKeyRequest/Non-numeric_key_length (0.00s)
    --- PASS: TestHandleKeyRequest/Maximum_valid_key_length (0.00s)
    --- PASS: TestHandleKeyRequest/Exceeds_maximum_key_length (0.00s)
=== RUN   TestMetricsEndpoint
    key-server_test.go:71: Testing /metrics endpoint
    key-server_test.go:90: Metrics endpoint test passed
--- PASS: TestMetricsEndpoint (0.00s)
PASS
ok  	key-server	0.002s
```

Good to run this everytime there is change in application logic files (eg: key-server.go)
Should get PASS at the end for all test cases

## 6. Packaging and Deployment (Dockerfile and helm)

In the working dir created Dockerfile (with dependencies and key-server app logic configured to run on container start) and built/tagged the image. 

Later pushed the application image to local registry (had to install docker cli: sudo apt install docker.io)

```bash
> sudo docker build -t localhost:5000/key-server:latest .
> sudo docker push localhost:5000/key-server:latest (failed)
```

No local registry hosted by default:
Solution: created one temporarily at port 5000 that we can push to later

```bash
 > sudo docker run -d -p 5000:5000 --name registry registry:latest 

> sudo docker push localhost:5000/key-server:latest (worked)
```

Did ctr image pull to test if the application image is accessible from the local registry we created earlier.

```bash
> sudo ctr image pull localhost:5000/key-server:latest   (works)
```

Now up to the final step of the deployment,  helm:

K3s has a custom inbuilt helm controller CRD functionality which is accessible via helmchart resource type
https://docs.k3s.io/helm 

With the above custom CRD we can directly apply the custom helmChart definition file and don't need to go through helm cli. With the helm-controller, ye don't need to manually run Helm commands to deploy or update releases. Instead, it watches for changes in your Kubernetes manifests and automatically applies them.

to save time i went ahead with known imperative way i.e helm cli (but also tried the above controller declarative way to deploy the app: 
```bash
ubuntu@k3s-vm:~/key-server-project$ sudo kubectl apply -f key-server-helmchart.yaml
```

Installed the helm3 cli via below cmd:
```bash
> curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
```

Configured the chart files and Packaged into .tgz file : which deploys the image we built earlier with docker

```bash
> helm package key-server-chart
```

```markdown
> sudo helm install key-server ./key-server-0.1.0.tgz \
  --namespace key-server \
  --create-namespace \
  --set image.repository=localhost:5000/key-server \
  --set image.tag=latest \
  --set maxSize=1024 \
  --set srvPort=1123 \
  --kubeconfig /etc/rancher/k3s/k3s.yaml

Application is Running:

Liveness Probe:
• Checks if the application is running. If this probe fails, Kubernetes will restart the container.
• Configured to hit the /key/1024 endpoint.

Readiness Probe:
• Determines if the application is ready to serve traffic. If this probe fails, the pod will be removed from service endpoints.
• Configured to hit the /metrics endpoint.

```bash
ubuntu@k3s-vm:~/key-server-project$ sudo kubectl get pods -A
NAMESPACE     NAME                                      READY   STATUS             RESTARTS         AGE
key-server    key-server-b57b87b9d-667nx                1/1     Running            0                21h
kube-system   coredns-576bfc4dc7-gflhc                  1/1     Running            0                24h

ubuntu@k3s-vm:~/key-server-project$ sudo kubectl get svc -A
NAMESPACE     NAME             TYPE           CLUSTER-IP      EXTERNAL-IP    PORT(S)                      AGE
default       kubernetes       ClusterIP      10.43.0.1       <none>         443/TCP                      24h
key-server    key-server       ClusterIP      10.43.225.158   <none>         1123/TCP                     21h
```

With the above service clusterIP requests are going to the pods.
```bash
ubuntu@k3s-vm:~/key-server-project$ curl -s http://10.43.225.158:1123/key/100 && echo
bb948dfee92f0d7dc6e07b22b973b3233dcff50a25abaa183090be20dd30e4a7f629dc1872326ed549f4aaee67984769485d6ab458f32457c2405af48dbff987af246d614fedf1fc47bcaa1d52c26cafecbb94fc37f4d38c69250dd1880a109f28fc89ba
ubuntu@k3s-vm:~/key-server-project$ curl -s http://10.43.225.158:1123/key/10 && echo
bbaa2e65989a583afa5b
```

Scaling should be straightforward by increasing replica count in the helm values.yaml config. (and running helm package and upgrade: tested replica counts 1,3,4)

## 7. Monitoring and Alerting

As in application logic step 3) we have instrumented prometheus monitoring we can track those http counters by going to /metrics endpoint periodically (for eg: every 5 seconds) and parsing the output below for example

eg key length histogram output 
```
# TYPE key_length_distribution histogram
key_length_distribution_bucket{le="0"} 0
key_length_distribution_bucket{le="51.2"} 48
key_length_distribution_bucket{le="102.4"} 52
key_length_distribution_bucket{le="153.60000000000002"} 52
key_length_distribution_bucket{le="204.8"} 52
key_length_distribution_bucket{le="256"} 52
key_length_distribution_bucket{le="307.2"} 52
key_length_distribution_bucket{le="358.4"} 52
key_length_distribution_bucket{le="409.59999999999997"} 52
key_length_distribution_bucket{le="460.79999999999995"} 52
key_length_distribution_bucket{le="511.99999999999994"} 52
key_length_distribution_bucket{le="563.1999999999999"} 52
key_length_distribution_bucket{le="614.4"} 52
key_length_distribution_bucket{le="665.6"} 52
key_length_distribution_bucket{le="716.8000000000001"} 52
key_length_distribution_bucket{le="768.0000000000001"} 52
key_length_distribution_bucket{le="819.2000000000002"} 52
key_length_distribution_bucket{le="870.4000000000002"} 52
key_length_distribution_bucket{le="921.6000000000003"} 52
key_length_distribution_bucket{le="972.8000000000003"} 52
key_length_distribution_bucket{le="+Inf"} 3943
```

specifying prometheus.LinearBuckets(0, float64(*maxSize)/20, 20), Prometheus will create 20 linear buckets with even spacing from 0 to maxSize, allowing ye to track how many key requests fall into each size category.

Validated counters data in /metrics output which are mostly accurate (except few which consider other http 200 calls within prometheus module internal func calls being added to http200 status code counter, giving it larger number than expected). Workaround would be to create a new label to avoid repetitions for application key generation http calls vs generic http requests.
Our focus is on failures counters which are accurate to 100% which can be seen later below.

For scale benchmarking we can periodically run load/stress/idle workloads with tools like apache bench ab cli utility or jmeter or simple scripts like pythons requests/gevent/multiprocessing module for parallel client testing.

### Limits and Quotas
Our current resource limits and requests are set as follows:
CPU:
- Requests: 100m (0.1 CPU)
- Limits: 200m (0.2 CPU)
Memory:
- Requests: 64Mi
- Limits: 128Mi

These values indicate the minimum and maximum resources allocated to each pod. Let's explore how these quotas impact your key generation service and the /metrics endpoint.

### Factors Affecting Key Generation Capacity
1. Key Length and Complexity:
   - Shorter keys require less computation and memory, while longer keys increase CPU and memory usage.
   - Validating keys to avoid collisions increases the workload, especially for shorter keys where the probability of collision is higher.
2. Number of Requests:
   - The number of concurrent requests significantly impacts resource utilization. More requests mean higher CPU and memory usage.
3. Metrics Collection:
   - Serving /metrics requests adds to the CPU and memory load, especially if metrics are aggregated and processed frequently.

### Estimating Capacity
Estimating the number of requests your service can handle depends on several factors:
1. Average Time per Request:
   - Calculate the average CPU time and memory consumption per request for key generation and validation.
2. Benchmarking:
   - Run benchmarks to determine how many requests per second (RPS) your application can handle under the given resource limits.
3. Concurrency:
   - Consider the maximum number of concurrent requests your service can handle without exceeding resource limits.

### Example Estimation
Suppose you benchmark your service and determine the following:
- Key Generation (without validation):
  - Average CPU usage: 10m (0.01 CPU) per request
  - Average Memory usage: 2Mi per request
  - Time per request: 50ms
- Key Validation:
  - Additional CPU usage: 5m (0.005 CPU) per request
  - Additional Memory usage: 1Mi per request
  - Time per request: 20ms
- Metrics Collection:
  - Average CPU usage: 2m (0.002 CPU) per request
  - Average Memory usage: 1Mi per request

### Maximum RPS Calculation
CPU Limit:
- Total CPU available: 200m
- Combined CPU per key request (with validation): 15m
- Maximum key generation RPS: 200m/15m ≈ 13 requests per second

Memory Limit:
- Total Memory available: 128Mi
- Combined Memory per key request (with validation): 3Mi
- Maximum key generation RPS: 128Mi/3Mi ≈ 42 requests per second

### Combined Workload with Metrics
Assuming the /metrics endpoint is hit every second and consumes 2m CPU and 1Mi memory:

Adjusted CPU RPS:
- Available CPU for key generation: 200m - 2m = 198m
- Adjusted maximum key generation RPS: 198m/15m ≈ 13 requests per second

Adjusted Memory RPS:
- Available Memory for key generation: 128Mi - 1Mi = 127Mi
- Adjusted maximum key generation RPS: 127Mi/3Mi ≈ 42 requests per second

### Considerations and Challenges
1. Collision Avoidance:
   - Ensuring keys are unique, especially with shorter lengths, adds significant overhead due to lookup operations, potentially reducing throughput.
2. Resource Scaling:
   - For higher request rates, consider scaling horizontally by adding more replicas, each with the same resource limits.
3. Load Testing:
   - Conduct load tests to accurately measure performance under different conditions and validate these estimates.
4. Monitoring and Autoscaling:
   - Use Kubernetes Horizontal Pod Autoscaler (HPA) to automatically adjust the number of replicas based on CPU or custom metrics

### Alerting strategies

For monitoring and alerts, configure Prometheus Alertmanager to send notifications based on specific conditions such as high error rates or unusual key length distributions. Define alerting rules like these:

```yaml
groups:
- name: KeyServerAlerts
  rules:
  - alert: HighErrorRate
    expr: increase(http_status_codes{code=~"5.."}[5m]) > 10
    for: 5m
    labels:
      severity: critical
    annotations:
      summary: "High server error rate detected"
      description: "More than 10 errors in the last 5 minutes."
```

- Use key_length_distribution histogram to detect abnormal key length requests.
- Monitor the http_status_codes counter for any spikes in 4xx or 5xx status codes.
- For example Whenever error 400 status codes (that we are registering with prometheus module counters for different http verbs/labels) cross a threshold(10% failures) within for eg 1min then we can send an alerting mechanism to slack/email/pagerDuty/SNMP traps or with third party tool like zabbix we can configure the regex expression rules to reflect in the dashboard.
- If application pod down or deployment replica count not as expected we can send an alert , as we already have liveness and readiness probes configured, k8s usually handles HA part by default
- We can track response times and latencies/jitter if it increases for a few requests/txns frequently then send an alert that there is lag around a UTC timestamp.
- Looking at 95 or 99 percentile values usually gives us the good estimate of the load on system or stability over a period of time while taking spikes (latency spikes or 400 errors spikes) into the consideration.

```bash
ubuntu@k3s-vm:~/key-server-project$ curl http://10.43.225.158:1123/metrics | grep -i http_status_codes
  % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
                                 Dload  Upload   Total   Spent    Left  Speed
100  7421    0  7421    0     0  9637k      0 --:--:-- --:--:-- --:--:-- 7247k
# HELP http_status_codes Counter of HTTP status codes
# TYPE http_status_codes counter
http_status_codes{code="200"} 3936
http_status_codes{code="400"} 12
ubuntu@k3s-vm:~/key-server-project$ curl -s http://10.43.225.158:1123/key/10000 && echo
Invalid key length

ubuntu@k3s-vm:~/key-server-project$ curl -s http://10.43.225.158:1123/key/10000 && echo
Invalid key length

ubuntu@k3s-vm:~/key-server-project$ curl http://10.43.225.158:1123/metrics | grep -i http_status_codes
  % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
                                 Dload  Upload   Total   Spent    Left  Speed
100  7421    0  7421    0     0  5915k      0 --:--:-- --:--:-- --:--:-- 7247k
# HELP http_status_codes Counter of HTTP status codes
# TYPE http_status_codes counter
http_status_codes{code="200"} 3936
http_status_codes{code="400"} 14

```

As you can see in the above  /metrics endpoint http_status_codes{code="400"} is incremented on failure.

- Classifying the alerts into Critical, High, Medium, Low categories (based on different CPU thresholds or Memory thresholds or Requests/Txns error rates) for easier capacity planning.
- Can also have alerts when pods are reaching 90% CPU limit or Memory usage so we don't need to wait till they go to CrashLoopBackOff state and keep restarting. 
- While the scheduler handles rescheduling and maintaining replicas, PDBs Pod disruption budgets add an additional layer to manage voluntary disruptions and ensure that applications remain available during planned changes.

## Learnings:

- It was fun to try out lightweight k8s distro k3s for the first time and understand what features were stripped from the vanilla version and how core control server function of kubernetes can still be intact with most components decoupled/Interfaces standardized like CRI,CNI,CSI etc

- Customizing metrics specific to application to monitor with prometheus library within the application logic from ground up gave a good perspective of monitoring critical metrics accurately rather than approximating with an aftermarket system level scraping tool in generic fashion.. trying to figure out what went wrong when we hit performance issues instead of proactively monitoring with customized counters and alerting ahead.

## Improvements for later: 

1. Include (go test -v cmd run) part of deployment.yaml or helm chart logic so it conditionally runs successfully only when all test cases pass. Decoupling deployment and application logic development.
2. Add authentication for the app namespace so features like rate limiting can be implemented (open source tools and some kubernetes custom resource definitions available at https://landscape.cncf.io/ can help)
3. Adding key length min-size arg similar to max-size > Making it Configurable in helm chart values.yaml and later upgrading based on users needs or we have can separate helm chart for different min-size requirement of users/projects and application logic need to modified a bit, should be just one line edit in the if condition, can add more test cases for it.
 Because of output duplicates like this:
   ```
   ubuntu@k3s-vm:~/key-server-project$ curl http://10.43.225.158:1123/key/1
   Ec
   ubuntu@k3s-vm:~/key-server-project$ curl http://10.43.225.158:1123/key/1
   Ec
   ```
   If small key length scenarios are must, Add validation in the application logic to check against current already generated hex tokens(stored in a db securely in zero knowledge proof way) to avoid hash collisions for keys already generated (will add Validation lookup delay at scale..need to come up with smart distributed systems mechanisms which can offload reads and writes with caching/sharding). Can also rotate keys in the pool every 30days or 90days or 1 year by notifying clients.
5. Add some custom salt feature(user id or any http headers like source ip) to the application key generation logic(hashing) as part of key-server.go file so there is less chance of hash collision across two users/domains/orgs/geo locations. (usually there is no need , as we don't run out and come across overlapping keys when the unique key space is large enough for most use cases)
6. Configure HPA horizontal pod autoscaler feature in kubernetes for the deployment to have flexibility of upscaling and downscaling keeping infra costs in mind when the server is idle for too long.
```

Thank you claude for converting google doc to markdown with less effort :)
