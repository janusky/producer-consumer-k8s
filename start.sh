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
    # [[ $(exec bash -c "${@}") ]] && echo -e "OK" || echo -e "\e[31mFail\e[0m"
    # exec "${@}" &> /dev/null 
    until [[ $(exec bash -c "${@}") ]]
    do
        echo "Waiting for config.."
        sleep 3
    done
}

function createMinikube() {
    # echo -e "Delete minikube If exists"
    [[ -n "$(minikube status | grep -Ei 'running|stopped|nonexistent')" ]] && minikube stop >> cmd.log \
    && minikube delete >> cmd.log
    wait

    minikube start --memory=8GB --cpus=4 >> cmd.log && \
    minikube addons enable metrics-server >> cmd.log

    istioctl install --set profile=demo -y </dev/null &>>cmd.log
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

    kubectl create namespace dev >> cmd.log && \
    kubectl apply -f ./resources -R -n dev >> cmd.log && \
    kubectl label namespace dev istio-injection=enabled --overwrite >> cmd.log

    # Kubernetes Dashboard
    # minikube dashboard </dev/null &>/dev/null &
    # minikube dashboard > /dev/null 2>&1 &
    minikube dashboard </dev/null &>>cmd.log &
    sleep 4
    # Observability
    #
    # Prometheus
    kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.11/samples/addons/prometheus.yaml >> cmd.log

    # Rabbitmq Manager
    awaitCommand 'kubectl get pod -n dev -l app=rabbitmq | grep Running'
    kubectl port-forward --namespace dev svc/rabbitmq 15672:15672 </dev/null &>>cmd.log &
    sleep 1
    echo -e "\e[93mRabbitmq Manager: \e[36mhttp://localhost:15672\e[0m" >> cmd.log

    # Mongo Express
    awaitCommand 'kubectl get pod -n dev -l app=mongo-express | grep Running'
    kubectl port-forward --namespace dev svc/mongo-express 8081:8081 </dev/null &>>cmd.log &
    sleep 1
    echo -e "\e[93mMongo Express: \e[36mhttp://localhost:8081\e[0m" >> cmd.log

    # Grafana
    kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.11/samples/addons/grafana.yaml >> cmd.log
    awaitCommand 'kubectl get pod -n istio-system -l app=grafana | grep Running'
    kubectl -n istio-system port-forward $(kubectl -n istio-system get pod -l app=grafana -o jsonpath='{.items[0].metadata.name}') 3000:3000 </dev/null &>>cmd.log &
    sleep 1
    echo -e "\e[93mGrafana: \e[36mhttp://localhost:3000\e[0m" >> cmd.log

    # Jaeger
    kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.11/samples/addons/jaeger.yaml >> cmd.log
    awaitCommand 'kubectl get pod -n istio-system -l app=jaeger | grep Running'
    kubectl port-forward -n istio-system $(kubectl get pod -n istio-system -l app=jaeger -o jsonpath='{.items[0].metadata.name}') 16686:16686 </dev/null &>>cmd.log &
    sleep 1
    echo -e "\e[93mJaeger: \e[36mhttp://localhost:16686\e[0m" >> cmd.log

    # Kiali
    kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.11/samples/addons/kiali.yaml >> cmd.log
    awaitCommand 'kubectl get pod -n istio-system -l app=kiali | grep Running'
    # kubectl port-forward -n istio-system $(kubectl get pod -n istio-system -l app=kiali -o jsonpath='{.items[0].metadata.name}') 20001:20001 &
    kubectl port-forward svc/kiali 20001:20001 -n istio-system </dev/null &>>cmd.log &
    sleep 1
    echo -e "\e[93mKiali: \e[36mhttp://localhost:20001\e[0m" >> cmd.log

    # Zipkin
    # kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.11/samples/addons/extras/zipkin.yaml >> cmd.log
    # awaitCommand 'kubectl get pod -n istio-system -l app=zipkin | grep Running'
    # kubectl port-forward -n istio-system $(kubectl get pod -n istio-system -l app=zipkin -o jsonpath='{.items[0].metadata.name}') 9411:9411 </dev/null &>>cmd.log &
    # echo -e "\e[93m Zipkin: \e[36mhttp://localhost:9411\e[0m" >> cmd.log

    read -p "Escape Test service-a (no recommended) [s|y]? " -n 1 -r
    echo    # (optional) move to a new line
    if [[ ! $REPLY =~ ^[YySs]$ ]]
    then
        export INGRESS_HOST=$(kubectl get po -l istio=ingressgateway -n istio-system -o jsonpath='{.items[0].status.hostIP}')
        export INGRESS_PORT=$(kubectl -n istio-system get service istio-ingressgateway -o jsonpath='{.spec.ports[?(@.name=="http2")].nodePort}')
        # curl -v "http://$INGRESS_HOST:$INGRESS_PORT/api/request-echo"
        curl -v "http://$INGRESS_HOST:$INGRESS_PORT/api/dummy" |& tee -a cmd.log
    fi
}

function activeObservability() {
    # Observability
    #
    # Rabbitmq Manager
    awaitCommand 'kubectl get pod -n dev -l app=rabbitmq | grep Running'
    kubectl port-forward --namespace dev svc/rabbitmq 15672:15672 </dev/null &>>cmd.log &
    sleep 1
    echo -e "\e[93mRabbitmq Manager: \e[36mhttp://localhost:15672\e[0m" >> cmd.log

    # Mongo Express
    awaitCommand 'kubectl get pod -n dev -l app=mongo-express | grep Running'
    kubectl port-forward --namespace dev svc/mongo-express 8081:8081 </dev/null &>>cmd.log &
    sleep 1
    echo -e "\e[93mMongo Express: \e[36mhttp://localhost:8081\e[0m" >> cmd.log

    # Grafana
    awaitCommand 'kubectl get pod -n istio-system -l app=grafana | grep Running'
    kubectl -n istio-system port-forward $(kubectl -n istio-system get pod -l app=grafana -o jsonpath='{.items[0].metadata.name}') 3000:3000 </dev/null &>>cmd.log &
    sleep 1
    echo -e "\e[93mGrafana: \e[36mhttp://localhost:3000\e[0m" >> cmd.log

    # Jaeger
    awaitCommand 'kubectl get pod -n istio-system -l app=jaeger | grep Running'
    kubectl port-forward -n istio-system $(kubectl get pod -n istio-system -l app=jaeger -o jsonpath='{.items[0].metadata.name}') 16686:16686 </dev/null &>>cmd.log &
    sleep 1
    echo -e "\e[93mJaeger: \e[36mhttp://localhost:16686\e[0m" >> cmd.log

    # Kiali
    awaitCommand 'kubectl get pod -n istio-system -l app=kiali | grep Running'
    kubectl port-forward svc/kiali 20001:20001 -n istio-system </dev/null &>>cmd.log &
    sleep 1
    echo -e "\e[93mKiali: \e[36mhttp://localhost:20001\e[0m" >> cmd.log
}

echo -e "\e[93mCluster Start\e[0m"
checkRequeriment

# If Running
if [[ -n "$(minikube status | grep -Ei 'Running|Nonexistent')" ]]
then
    read -p "Terminar y Crear nuevo [n|y/s]? " -n 1 -r
    echo    # (optional) move to a new line
    if [[ $REPLY =~ ^[YySs]$ ]]
    then
        echo -e "######### TERMINAR Y CREAR CLUSTER"
        echo -e "Show log in other console (tail -f cmd.log)"
        echo -e "\e[32mEnd Script\e[0m"
        createMinikube
        exit 0
    fi
fi

# If Stopped
if [[ -n "$(minikube status | grep -i 'Stopped')" ]]
then
    read -p "Iniciar la imagen existente [y/s|n]? " -n 1 -r
    echo    # (optional) move to a new line
    if [[ $REPLY =~ ^[YySs]$ ]]
    then
        echo -e "######### START CLUSTER"
        echo -e "Show log in other console (tail -f cmd.log)"
        minikube start >> cmd.log
        wait
        minikube dashboard </dev/null &>>cmd.log &
        sleep 3
        activeObservability
        echo -e "\e[32mEnd Script\e[0m"
        exit 0
    fi

    read -p "Eliminar y Crear nuevo [n|y/s]? " -n 1 -r
    echo    # (optional) move to a new line
    if [[ $REPLY =~ ^[YySs]$ ]]
    then
        echo -e "######### ELIMINAR Y CREAR CLUSTER"
        echo -e "Show log in other console (tail -f cmd.log)"
        createMinikube
        echo -e "\e[32mEnd Script\e[0m"
        exit 0
    fi
fi
