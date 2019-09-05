# The dockerfile for ecs-operator
FROM alpine

# copy file
COPY bin/ecs-operator /ecs-operator

# work dir
WORKDIR /

# start ecs-operator
ENTRYPOINT ["/ecs-operator"]
