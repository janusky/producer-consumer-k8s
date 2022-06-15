# producer-consumer-k8s

Detalles de configuración de equipo/terminal/pc para trabajar con [Kubernetes](https://kubernetes.io/es/docs/home/)

* https://github.com/janusky/producer-consumer-k8s

## Configurar entorno

Fuentes

* https://kubernetes.io/es/docs/setup/
* https://istio.io/latest/docs/setup/getting-started/
* https://istio.io/latest/docs/setup/platform-setup/minikube/
* https://istio.io/latest/docs/ops/diagnostic-tools/istioctl/#install-hahahugoshortcode-s2-hbhb

### Install minikube

[Minikube]https://minikube.sigs.k8s.io/docs/start/)

```sh
# To install minikube on x86-64 Linux using binary download
curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64
sudo install minikube-linux-amd64 /usr/local/bin/minikube

# Check
minikube version
```

>NOTA: Si ocurren problemas al descargar las herramientas kubectl/kubelet/kubeadm utilizadas por Minikube.
>```sh
># Download tools for Minikube
>STORAGE_HOST="${STORAGE_HOST:-https://storage.googleapis.com}"
>KUBE_VERSION=v1.22.2
>host_os=linux
>host_arch=amd64
>curl -fsSL "${STORAGE_HOST}/kubernetes-release/release/${KUBE_VERSION}/bin/${host_os}/${host_arch}/kubectl" -o kubectl
>curl -fsSL "${STORAGE_HOST}/kubernetes-release/release/${KUBE_VERSION}/bin/${host_os}/${host_arch}/kubelet" -o kubelet
>curl -fsSL "${STORAGE_HOST}/kubernetes-release/release/${KUBE_VERSION}/bin/${host_os}/${host_arch}/kubeadm" -o kubeadm
>
># Copiar las herramientas en el path donde espera encontrarlas Minikube
>mkdir -p ~/.minikube/files/var/lib/minikube/binaries/$KUBE_VERSION
>
>mv kubectl kubelet kubeadm ~/.minikube/files/var/lib/minikube/binaries/$KUBE_VERSION
>
>chmod +x -R ~/.minikube/files/var/lib/minikube/binaries/$KUBE_VERSION
>```

### Install kubectl

Minikube trae una versión de **Kubectl**, pero es más practico contar con el cliente directamente.

```sh
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"

# Validate the binary (optional)
curl -LO "https://dl.k8s.io/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl.sha256"
echo "$(<kubectl.sha256) kubectl" | sha256sum --check

# Install kubectl
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

# Test version
kubectl version --client
```

### Install istioctl

Aplicación cliente para [Istio](https://istio.io/latest/)

```sh
cd ~

curl -sL https://istio.io/downloadIstioctl | sh -

sed -i '$ a export PATH=$PATH:$HOME/.istioctl/bin' ~/.profile
```
