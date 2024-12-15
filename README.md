# DriveSmart-Monetization
DriveSmart-Monetization promotes safe driving through behavioral analysis and monetization. It identifies harsh acceleration, sharp turns, and zig-zag patterns, rewarding drivers for safe practices and penalizing risky behaviors.

# Hyperledger Fabric Operator

## Features

- [x] Create certificates authorities (CA)
- [x] Create peers
- [x] Create ordering services
- [x] Create resources without manual provisioning of cryptographic material
- [x] Domain routing with SNI using Istio
- [x] Run chaincode as external chaincode in Kubernetes
- [x] Support Hyperledger Fabric 2.3+ and 3.0
- [x] Managed genesis for Ordering services
- [x] E2E testing including the execution of chaincodes in KIND
- [x] Renewal of certificates



Requisitos:

- Linux (tested with Ubuntu 22.04)
- [Kubectl](https://kubernetes.io/pt-br/docs/tasks/tools/install-kubectl-linux/)
- [Krew](https://krew.sigs.k8s.io/)
- [KinD](https://kind.sigs.k8s.io/) ou [K3d](https://k3d.io/v5.6.0/)
- [Istio](https://istio.io/latest/ ) 
- [Helm](https://helm.sh/)
- [JQ](https://jqlang.github.io/jq/download/)
- [Docker](https://docs.docker.com/get-docker/)


Install requirements automatically with the script

```bash
chmod 777 install.sh
./install.sh
```

## Script network.sh

```bash
    echo "'up' - Starts the network"
    echo "'chaincode <name>' - Installs chaincode"
    echo "'upgrade' <name> <version> <sequence>- upgrade chaincode"
    echo "'down' -Destroys Kubernetes cluster"
    echo "'help' - Commands help"
```

# Tutorial
Atualmente, há duas versões sendo utilizadas

## 1. Criar Cluster Kubernetes

To start deploying our red fabric we have to have a Kubernetes cluster. For this we will use KinD.

Ensure you have these ports available before creating the cluster:
- 80
- 443

If these ports are not available this tutorial will not work.

### Using KinD

```bash
mkdir resources
cat << EOF > resources/kind-config.yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  image: kindest/node:v1.25.8
  extraPortMappings:
  - containerPort: 30949
    hostPort: 80
  - containerPort: 30950
    hostPort: 443
EOF

kind create cluster --config=./resources/kind-config.yaml

export STORAGE_CLASS=standard
export DATABASE=couchdb
```


## 2. Istio install

```bash

kubectl create namespace istio-system

istioctl operator init

kubectl apply -f - <<EOF
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
metadata:
  name: istio-gateway
  namespace: istio-system
spec:
  addonComponents:
    grafana:
      enabled: false
    kiali:
      enabled: false
    prometheus:
      enabled: false
    tracing:
      enabled: false
  components:
    ingressGateways:
      - enabled: true
        k8s:
          hpaSpec:
            minReplicas: 1
          resources:
            limits:
              cpu: 500m
              memory: 512Mi
            requests:
              cpu: 100m
              memory: 128Mi
          service:
            ports:
              - name: http
                port: 80
                targetPort: 8080
                nodePort: 30949
              - name: https
                port: 443
                targetPort: 8443
                nodePort: 30950
            type: NodePort
        name: istio-ingressgateway
    pilot:
      enabled: true
      k8s:
        hpaSpec:
          minReplicas: 1
        resources:
          limits:
            cpu: 300m
            memory: 512Mi
          requests:
            cpu: 100m
            memory: 128Mi
  meshConfig:
    accessLogFile: /dev/stdout
    enableTracing: false
    outboundTrafficPolicy:
      mode: ALLOW_ANY
  profile: default

EOF
```

### Internal DNS

```bash
kubectl apply -f - <<EOF
kind: ConfigMap
apiVersion: v1
metadata:
  name: coredns
  namespace: kube-system
data:
  Corefile: |
    .:53 {
        errors
        health {
           lameduck 5s
        }
        rewrite name regex (.*)\.localho\.st istio-ingressgateway.istio-system.svc.cluster.local
        hosts {
            ${CLUSTER_IP} istio-ingressgateway.istio-system.svc.cluster.local
            fallthrough
        }
        ready
        kubernetes cluster.local in-addr.arpa ip6.arpa {
           pods insecure
           fallthrough in-addr.arpa ip6.arpa
           ttl 30
        }
        prometheus :9153
        forward . /etc/resolv.conf {
           max_concurrent 1000
        }
        cache 30
        loop
        reload
        loadbalance
    }
EOF
```

## 3. Instalar o HLF Operator

In this step we are going to install the kubernetes operator for Fabric, this will install:

- CRD (Custom Resource Definitions) to deploy Certification Fabric Peers, Orderers and Authorities
- Deploy the program to deploy the nodes in Kubernetes

To install helm: [https://helm.sh/docs/intro/install/](https://helm.sh/docs/intro/install/)

```bash
helm repo add kfs https://kfsoftware.github.io/hlf-helm-charts --force-update

helm install hlf-operator --version=1.11.1 -- kfs/hlf-operator
```

```bash
kubectl krew install hlf
```

## 4. Deploy organizations

### Environment Variables for AMD (Default)

```bash
export ORDERER_IMAGE=hyperledger/fabric-orderer
export ORDERER_VERSION=2.5.5

export CA_IMAGE=hyperledger/fabric-ca
export CA_VERSION=1.5.7
```
### INMETRO CA Creation

```bash

kubectl hlf ca create  --image=$CA_IMAGE --version=$CA_VERSION --storage-class=$STORAGE_CLASS --capacity=1Gi --name=inmetro-ca \
    --enroll-id=enroll --enroll-pw=enrollpw --hosts=inmetro-ca.localho.st --istio-port=443

kubectl wait --timeout=180s --for=condition=Running fabriccas.hlf.kungfusoftware.es --all
```

Check if CA works

```bash
curl -k https://inmetro-ca.localho.st:443/cainfo
```

Register peer user for INMETRO

```bash
# register user in CA for peers
kubectl hlf ca register --name=inmetro-ca --user=peer --secret=peerpw --type=peer \
 --enroll-id enroll --enroll-secret=enrollpw --mspid INMETROMSP

```

### INMETRO peer creation

```bash
export PEER_IMAGE=quay.io/kfsoftware/fabric-peer
export PEER_VERSION=2.4.1-v0.0.3
export MSP_ORG=INMETROMSP
export PEER_SECRET=peerpw

kubectl hlf peer create --statedb=$DATABASE --image=$PEER_IMAGE --version=$PEER_VERSION --storage-class=$STORAGE_CLASS --enroll-id=peer --mspid=$MSP_ORG \
--enroll-pw=$PEER_SECRET --capacity=5Gi --name=inmetro-peer0 --ca-name=inmetro-ca.default --k8s-builder=true --hosts=peer0-inmetro.localho.st --istio-port=443

kubectl wait --timeout=180s --for=condition=Running fabricpeers.hlf.kungfusoftware.es --all

# leva alguns minutos

```

Check if peer works

```bash
openssl s_client -connect peer0-inmetro.localho.st:443

```

### Deploy Orderer

To deploy an `Orderer` organization we have to:

1. Create a certification authority
2. Register user `orderer` with password `ordererpw`
3. Create orderer

### CA Creation

```bash
kubectl hlf ca create  --image=$CA_IMAGE --version=$CA_VERSION --storage-class=$STORAGE_CLASS --capacity=1Gi --name=ord-ca \
    --enroll-id=enroll --enroll-pw=enrollpw --hosts=ord-ca.localho.st --istio-port=443

kubectl wait --timeout=180s --for=condition=Running fabriccas.hlf.kungfusoftware.es --all
```

Check if CA works

```bash
curl -vik https://ord-ca.localho.st:443/cainfo
```

### Register `orderer` user

```bash
kubectl hlf ca register --name=ord-ca --user=orderer --secret=ordererpw \
    --type=orderer --enroll-id enroll --enroll-secret=enrollpw --mspid=OrdererMSP --ca-url="https://ord-ca.localho.st:443"

```
### Deploy orderer

```bash
  kubectl hlf ordnode create --image=$ORDERER_IMAGE --version=$ORDERER_VERSION \
      --storage-class=$STORAGE_CLASS --enroll-id=orderer --mspid=OrdererMSP \
      --enroll-pw=ordererpw --capacity=2Gi --name=ord-node0 --ca-name=ord-ca.default \
      --hosts=orderer0-ord.localho.st --istio-port=443 --admin-hosts=admin-orderer0-ord.localho.st

kubectl wait --timeout=180s --for=condition=Running fabricorderernodes.hlf.kungfusoftware.es --all
```

Check if orderer works
```bash
kubectl get pods
```

```bash
openssl s_client -connect orderer0-ord.localho.st:443
```


## Create channel

To create the channel we need to first create the wallet secret, which will contain the identities used by the operator to manage the channel

### Register OrdererMSP identity 

```bash
  ## register OrdererMSP Identity
  kubectl hlf ca register --name=ord-ca --user=admin --secret=adminpw \
      --type=admin --enroll-id enroll --enroll-secret=enrollpw --mspid=OrdererMSP

  kubectl hlf identity create --name orderer-admin-sign --namespace default \
      --ca-name ord-ca --ca-namespace default \
      --ca ca --mspid OrdererMSP --enroll-id admin --enroll-secret adminpw  # sign identity

  kubectl hlf identity create --name orderer-admin-tls --namespace default \
      --ca-name ord-ca --ca-namespace default \
      --ca tlsca --mspid OrdererMSP --enroll-id admin --enroll-secret adminpw l # tls identity
```


### Register INMETROMSP idenitty

```bash
  ## register INMETROMSP Identity
  kubectl hlf ca register --name=inmetro-ca --namespace=default --user=admin --secret=adminpw \
      --type=admin --enroll-id enroll --enroll-secret=enrollpw --mspid=INMETROMSP

  # enroll
  kubectl hlf identity create --name inmetro-admin --namespace default \
      --ca-name inmetro-ca --ca-namespace default \
      --ca ca --mspid INMETROMSP --enroll-id admin --enroll-secret adminpw
```

### Creating main channel

```bash
export PEER_ORG_SIGN_CERT=$(kubectl get fabriccas inmetro-ca -o=jsonpath='{.status.ca_cert}')
export PEER_ORG_TLS_CERT=$(kubectl get fabriccas inmetro-ca -o=jsonpath='{.status.tlsca_cert}')
export IDENT_8=$(printf "%8s" "")
export ORDERER_TLS_CERT=$(kubectl get fabriccas ord-ca -o=jsonpath='{.status.tlsca_cert}' | sed -e "s/^/${IDENT_8}/" )
export ORDERER0_TLS_CERT=$(kubectl get fabricorderernodes ord-node0 -o=jsonpath='{.status.tlsCert}' | sed -e "s/^/${IDENT_8}/" )

kubectl apply -f - <<EOF
apiVersion: hlf.kungfusoftware.es/v1alpha1
kind: FabricMainChannel
metadata:
  name: demo
spec:
  name: demo
  adminOrdererOrganizations:
    - mspID: OrdererMSP
  adminPeerOrganizations:
    - mspID: INMETROMSP
  channelConfig:
    application:
      acls: null
      capabilities:
        - V2_0
      policies: null
    capabilities:
      - V2_0
    orderer:
      batchSize:
        absoluteMaxBytes: 1048576
        maxMessageCount: 10
        preferredMaxBytes: 524288
      batchTimeout: 2s
      capabilities:
        - V2_0
      etcdRaft:
        options:
          electionTick: 10
          heartbeatTick: 1
          maxInflightBlocks: 5
          snapshotIntervalSize: 16777216
          tickInterval: 500ms
      ordererType: etcdraft
      policies: null
      state: STATE_NORMAL
    policies: null
  externalOrdererOrganizations: []
  peerOrganizations:
    - mspID: INMETROMSP
      caName: "inmetro-ca"
      caNamespace: "default"
  identities:
    OrdererMSP:
      secretKey: user.yaml
      secretName: orderer-admin-tls
      secretNamespace: default
    OrdererMSP-sign:
      secretKey: user.yaml
      secretName: orderer-admin-sign
      secretNamespace: default
    INMETROMSP:
      secretKey: user.yaml
      secretName: inmetro-admin
      secretNamespace: default
  externalPeerOrganizations: []
  ordererOrganizations:
    - caName: "ord-ca"
      caNamespace: "default"
      externalOrderersToJoin:
        - host: ord-node0
          port: 7053
      mspID: OrdererMSP
      ordererEndpoints:
        - orderer0-ord.localho.st:443
      orderersToJoin: []
  orderers:
    - host: orderer0-ord.localho.st
      port: 443
      tlsCert: |-
${ORDERER0_TLS_CERT}
EOF


```

### Insert INMETRO peer in channel

```bash

export IDENT_8=$(printf "%8s" "")
export ORDERER0_TLS_CERT=$(kubectl get fabricorderernodes ord-node0 -o=jsonpath='{.status.tlsCert}' | sed -e "s/^/${IDENT_8}/" )

kubectl apply -f - <<EOF
apiVersion: hlf.kungfusoftware.es/v1alpha1
kind: FabricFollowerChannel
metadata:
  name: demo-inmetromsp
spec:
  anchorPeers:
    - host: inmetro-peer0.default
      port: 7051
  hlfIdentity:
    secretKey: user.yaml
    secretName: inmetro-admin
    secretNamespace: default
  mspId: INMETROMSP
  name: demo
  externalPeersToJoin: []
  orderers:
    - certificate: |
${ORDERER0_TLS_CERT}
      url: grpcs://ord-node0.default:7050
  peersToJoin:
    - name: inmetro-peer0
      namespace: default
EOF
```

## Install a chaincode

### Prepare connection string for a peer

To prepare the connection string, we have to:

1. Get connection string without users for organization Org1MSP and OrdererMSP
2. Register a user in the certification authority for signing (register)
3. Obtain the certificates using the previously created user (enroll)
4. Attach the user to the connection string


--------------

1. Get connection string without users for organization Org1MSP and OrdererMSP

```bash
mkdir resources
kubectl hlf inspect -c=demo --output resources/network.yaml -o INMETROMSP -o OrdererMSP
```


2. Register a user in the certification authority for signing
```bash
kubectl hlf ca register --name=inmetro-ca --user=admin --secret=adminpw --type=admin \
 --enroll-id enroll --enroll-secret=enrollpw --mspid INMETROMSP  
```

3. Get the certificates using the user created above
```bash
kubectl hlf ca enroll --name=inmetro-ca --user=admin --secret=adminpw --mspid INMETROMSP \
        --ca-name ca  --output resources/peer-inmetro.yaml
```

4. Attach the user to the connection string
```bash
kubectl hlf utils adduser --userPath=resources/peer-inmetro.yaml --config=resources/network.yaml --username=admin --mspid=INMETROMSP
```

### Chaincode installation

```bash
export CHAINCODE_LABEL=vehicle

kubectl hlf chaincode install --path=./chaincode/$CHAINCODE_LABEL \
    --config=resources/network.yaml --language=golang --label=$CHAINCODE_LABEL --user=admin --peer=inmetro-peer0.default

# this can take 3-4 minutes
```

Check chaincode installation

```bash
kubectl hlf chaincode queryinstalled --config=resources/network.yaml --user=admin --peer=inmetro-peer0.default
```

### Approve chaincode

```bash
  export PACKAGE_ID=$(kubectl hlf chaincode calculatepackageid --path=chaincode/$CHAINCODE_LABEL --language=golang --label=$CHAINCODE_LABEL)
  echo "PACKAGE_ID=$PACKAGE_ID"

#Organização INMETRO
kubectl hlf chaincode approveformyorg --config=resources/network.yaml --user=admin --peer=inmetro-peer0.default \
    --package-id=$PACKAGE_ID \
    --version "1.0" --sequence 1 --name=$CHAINCODE_LABEL \
    --policy="AND('INMETROMSP.member')" --channel=demo

#commit do chaincode

kubectl hlf chaincode commit --config=resources/network.yaml --mspid=INMETROMSP --user=admin \
    --version "1.0" --sequence 1 --name=$CHAINCODE_LABEL \
    --policy="AND('INMETROMSP.member')" --channel=demo
```




## Using  client:
[In the client folder](client/vehicle), open connection-org.yaml, and copy the content from (resources)[resources/network.yaml] there. Then add the following:

- In the client section, put "INMETROMSP"
- Below INMETROMSP organization, put:
```bash
    certificateAuthorities:
        - inmetro-ca.default
```

Then run the client with the command

```bash
go run main.go
```

## Cleanup the environment

```bash
./network.sh down
```
