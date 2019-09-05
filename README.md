# ECS Operator - Use Kubrenetes CRD & Custom Controller to make OpenStack API cloud native.

**Note**: The repo is in early stage and under frequent development.

## Getting started

### Requirements
go
docker
ECS v5 env with ssh access

### Steps
1. Clone repo
```bash
$ git clone git@github.com:houming-wang/ecs-operator.git
```

2. Build the operator image  and upload to one of ECS v5 nodes

```bash
$ cd ecs-operator;make image-build
```
3. OpenStack auth info config 
config OpenStack auth info in /etc/openstack/cloud.yaml
see examples/cloud.yaml

4. Deploy operator
```bash
$ kubectl apply -f examples/operator-deploy.yaml
```

5. Create Instance CRD
```bash
$ kubectl apply -f examples/crd.yaml
```

5. Create Instance 
change name、FlavRef、ImageRef、Network UUID in examples/instance-cr.yaml and then
```bash
$ kubectl apply -f examples/instance-cr.yaml
```
