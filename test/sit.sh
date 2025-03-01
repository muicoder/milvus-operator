#!/bin/bash

# System Integration Test

# utils
export LOG_PATH=/tmp/sit.log
log() {
    echo "$(date +"%Y-%m-%d %H:%M:%S") $1"
}

check_milvus_available(){
    # if $1 equals milvus-sit
    sed -i 's/host=host/host="milvus-milvus"/g' test/hello-milvus.py
    kubectl -n $1 create cm hello-milvus --from-file=test/hello-milvus.py
    kubectl -n $1 create -f test/hello-milvus-job.yaml
    kubectl -n $1 get -l myLabel=value service |wc -l 
    if [ $? -ne 0 ]; then
        log "kubectl check label failed"
        return 1
    fi

    # check ingress created
    kubectl -n $1 get ingress/milvus-milvus
    if [ $? -ne 0 ]; then
        kubectl -n $1 get ingress
        log "kubectl check ingress failed"
        return 1
    fi

    kubectl -n $1 wait --for=condition=complete job/hello-milvus --timeout 3m
    # if return 1, log
    if [ $? -eq 1 ]; then
        log "Error: $1 job failed"
        kubectl -n $1 describe -f test/hello-milvus-job.yaml
        return 1
    fi
}

delete_milvus_cluster(){
    # Delete CR
    log "Deleting MilvusCluster ..."
    kubectl delete -f test/min-mc.yaml
    log "Checking PVC deleted ..."
    kubectl wait --timeout=1m pvc -n mc-sit --for=delete -l release=mc-sit-minio
    kubectl wait --timeout=1m pvc -n mc-sit --for=delete -l release=mc-sit-pulsar
    kubectl wait --timeout=1m pvc -n mc-sit --for=delete -l app.kubernetes.io/instance=mc-sit-etcd
}

# milvus cluster cases:
case_create_delete_cluster(){
    # create MilvusCluster CR
    log "Creating MilvusCluster..."
    kubectl apply -f test/min-mc.yaml

    # Check CR status every 10 seconds (max 10 minutes) until complete.
    ATTEMPTS=0
    CR_STATUS=""
    until [ $ATTEMPTS -eq 60 ]; 
    do
        CR_STATUS=$(kubectl get -n mc-sit mc/milvus -o=jsonpath='{.status.status}')
        if [ "$CR_STATUS" = "Healthy" ]; then
            break
        fi
        log "MilvusCluster status: $CR_STATUS"
        ATTEMPTS=$((ATTEMPTS + 1))
        sleep 10
    done

    if [ "$CR_STATUS" != "Healthy" ]; then
        log "MilvusCluster creation failed"
        log "MilvusCluster final yaml: \n $(kubectl get -n mc-sit mc/milvus -o yaml)"
        log "MilvusCluster helm values: \n $(helm -n mc-sit get values milvus-pulsar)"
        log "MilvusCluster describe pods: \n $(kubectl -n mc-sit describe pods)"
        delete_milvus_cluster
        return 1
    fi
    check_milvus_available mc-sit
    if [ $? -ne 0 ]; then
        delete_milvus_cluster
        return 1
    fi

    delete_milvus_cluster
}

delete_milvus(){
    # Delete CR
    log "Deleting Milvus ..."
    kubectl delete -f test/min-milvus.yaml
    log "Checking PVC deleted ..."
    kubectl wait --timeout=1m pvc -n milvus-sit --for=delete -l release=milvus-sit-minio
    kubectl wait --timeout=1m pvc -n milvus-sit --for=delete -l app.kubernetes.io/instance=milvus-sit-etcd
}

# milvus cases:
case_create_delete_milvus(){
    # create Milvus CR
    log "Creating Milvus..."
    kubectl apply -f test/min-milvus.yaml

    # Check CR status every 10 seconds (max 10 minutes) until complete.
    ATTEMPTS=0
    CR_STATUS=""
    until [ $ATTEMPTS -eq 60 ]; 
    do
        CR_STATUS=$(kubectl get -n milvus-sit milvus/milvus -o=jsonpath='{.status.status}')
        if [ "$CR_STATUS" = "Healthy" ]; then
            break
        fi
        log "Milvus status: $CR_STATUS"
        ATTEMPTS=$((ATTEMPTS + 1))
        sleep 10
    done

    if [ "$CR_STATUS" != "Healthy" ]; then
        log "Milvus creation failed"
        log "Milvus final yaml: \n $(kubectl get -n milvus-sit milvus/milvus -o yaml)"
        log "Milvus describe pods: \n $(kubectl -n milvus-sit describe pods)"
        log "OperatorLog: $(kubectl -n milvus-operator logs deploy/milvus-operator)"
        delete_milvus
        return 1
    fi
    check_milvus_available milvus-sit
    if [ $? -ne 0 ]; then
        delete_milvus
        return 1
    fi
    delete_milvus
}

success=0
count=0

cases=(
    case_create_delete_cluster
    case_create_delete_milvus
)

echo "Running total: ${#cases[@]} CASES"

# run each test case in sequence
for case in "${cases[@]}"; do
    echo "Running CASE[$count]: $case ..."
    $case
    if [ $? -eq 0 ]; then
        echo "$case [success]"
        success=$((success + 1))
    else
        echo "$case [failed]"
    fi
    count=$((count + 1))
done

# test end banner
echo "==============================="
echo "Test End"
echo "==============================="

if [ $success -eq $count ]; then
    echo "All $count tests passed"
    exit 0
else
    echo "$success of $count tests passed"
    log "OperatorLog: $(kubectl -n milvus-operator logs deploy/milvus-operator)"
    exit 1
fi
