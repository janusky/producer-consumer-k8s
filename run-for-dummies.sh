#!/bin/bash

function checkRequeriment() {
    if [ "x$(which minikube)" == "x" ]; then
        echo "UNKNOWN - Missing minikube binary (Solve: CONFIG_PC.md#install-minikube)"
        exit 3
    fi

    if [ "x$(which kubectl)" == "x" ]; then
        echo "UNKNOWN - Missing kubectl binary (Solve: CONFIG_PC.md#install-kubectl)"
        exit 3
    fi
    
    istioctl version > /dev/null 2>&1
    if [ $? -ne 0 ]; then
        echo "UNKNOWN - Unable to talk to the istioctl (Solve: CONFIG_PC.md#install-istioctl)"
        exit 3
    fi
}

function awaitCommand() {
    until [[ $(exec bash -c "${@}") ]]
    do
        echo "Waiting for config.."
        sleep 5
    done
}

echo -e "\e[93mCluster Create\e[0m"
checkRequeriment

# echo -e "Delete minikube If exists"
[[ -n "$(minikube status | grep -Ei 'running|stopped')" ]] && minikube stop && minikube delete
wait

export no_proxy=localhost,127.0.0.1,10.96.0.0/12,192.168.99.0/24,192.168.39.0/24,192.168.49.2

minikube start --memory=6GB --cpus=4 --addons=metrics-server --install-addons=true

istioctl install --set profile=demo -y
wait

function docker_tag_exists() {
    curl --silent -f -lSL https://index.docker.io/v1/repositories/$1/tags/$2 > /dev/null
}
eval $(minikube docker-env)
IMAGES="service-a service-b"
for IMG in $IMAGES; do
    if docker_tag_exists janusky/$IMG dev; then
        docker pull janusky/$IMG:dev
    else 
        # BUILD_DIR=$1, BUILD_IMAGE=$2
        bash services/build.sh ./$IMG docker.io/janusky/$IMG:dev
    fi
done

wait

kubectl create namespace dev && \
  kubectl apply -f ./resources -R -n dev && \
  kubectl label namespace dev istio-injection=enabled --overwrite

# Kubernetes Dashboard
minikube dashboard &

# Observability
#
# Prometheus
kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.11/samples/addons/prometheus.yaml

# Rabbitmq Manager
awaitCommand 'kubectl get pod -n dev -l app=rabbitmq | grep Running'
kubectl port-forward --namespace dev svc/rabbitmq 15672:15672 &
echo -e "\e[93mRabbitmq Manager (guest/guest): \e[36mhttp://localhost:15672\e[0m"

# Mongo Express
awaitCommand 'kubectl get pod -n dev -l app=mongo-express | grep Running'
kubectl port-forward --namespace dev svc/mongo-express 8081:8081 &
echo -e "\e[93mMongo Express (username/password): \e[36mhttp://localhost:8081\e[0m"

# Grafana
kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.11/samples/addons/grafana.yaml
awaitCommand 'kubectl get pod -n istio-system -l app=grafana | grep Running'
kubectl -n istio-system port-forward $(kubectl -n istio-system get pod -l app=grafana -o jsonpath='{.items[0].metadata.name}') 3000:3000 &
echo -e "\e[93mGrafana: \e[36mhttp://localhost:3000\e[0m"

# Jaeger
kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.11/samples/addons/jaeger.yaml
awaitCommand 'kubectl get pod -n istio-system -l app=jaeger | grep Running'
kubectl port-forward -n istio-system $(kubectl get pod -n istio-system -l app=jaeger -o jsonpath='{.items[0].metadata.name}') 16686:16686 &
echo -e "\e[93mJaeger: \e[36mhttp://localhost:16686\e[0m"

# Kiali
echo -e "Apply Kiali"
kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.11/samples/addons/kiali.yaml
awaitCommand 'kubectl get pod -n istio-system -l app=kiali | grep Running'
# kubectl port-forward -n istio-system $(kubectl get pod -n istio-system -l app=kiali -o jsonpath='{.items[0].metadata.name}') 20001:20001 &
kubectl port-forward svc/kiali 20001:20001 -n istio-system &
echo -e "\e[93mKiali: \e[36mhttp://localhost:20001\e[0m"

# Zipkin
# echo -e "Apply Zipkin"
# kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.11/samples/addons/extras/zipkin.yaml
# awaitCommand 'kubectl get pod -n istio-system -l app=zipkin | grep Running'
# kubectl port-forward -n istio-system $(kubectl get pod -n istio-system -l app=zipkin -o jsonpath='{.items[0].metadata.name}') 9411:9411
# echo -e "\e[93m Zipkin: \e[36mhttp://localhost:9411\e[0m"

read -p "Escape Test service-a (no recommended) [s|y]? " -n 1 -r
echo    # (optional) move to a new line
if [[ ! $REPLY =~ ^[YySs]$ ]]
then
    export INGRESS_HOST=$(kubectl get po -l istio=ingressgateway -n istio-system -o jsonpath='{.items[0].status.hostIP}')
    export INGRESS_PORT=$(kubectl -n istio-system get service istio-ingressgateway -o jsonpath='{.spec.ports[?(@.name=="http2")].nodePort}')
    # curl -v "http://$INGRESS_HOST:$INGRESS_PORT/api/request-echo"
    curl -v "http://$INGRESS_HOST:$INGRESS_PORT/api/dummy"
fi

exit 0