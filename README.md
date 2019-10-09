# hal

## Table of Contents
- [Overview](#overview)
- [Key features](#key-features)
- [Demonstration](#demonstration)
- [Building hal](#building-hal)
- [Downloading a snapshot](#downloading-a-snapshot)
- [Deploying a component using hal](#deploying-a-component-using-hal)
   * [1. Scaffold the Spring Boot applications](#1-scaffold-the-spring-boot-applications)
   * [2. Deploy the Component](#2-deploy-the-component)
   * [3. Connect to the REST services](#3-connect-to-the-rest-services)
- [Additional documentation](#additional-documentation)

## Overview
Hal is a CLI tool for developers to simplify the deployment of applications such as Spring Boot on OpenShift and Kubernetes using Dekorate and Halkyon Component Operator. Made with ❤️ by the Halkyon team.

[![CircleCI](https://circleci.com/gh/halkyonio/hal.svg?style=svg)](https://circleci.com/gh/halkyonio/hal)

## Key features
`hal` is part of the [Halkyon project](https://github.com/halkyonio/operator) which aims to simplify the deployment of modern micro-services applications on Kubernetes. We encourage you to take a look to the [documentation](https://github.com/halkyonio/operator#introduction) of Halkyon in order to understand better the context of `hal`. `hal` is a tool capable of communicating with the cluster doing the following tasks
- Scaffold Spring Boot applications
- Deploy Microservices applications as Components
- Switch the `DeploymentMode` of the component from `Dev` to `Build` mode
- Compose & link microservices

## Demonstration
To see `hal` in action where it will compose 2 Spring Boot Applications as microservices with a Database 

[![asciicast](https://asciinema.org/a/ZWkxvg6LUzedQ2IPzmFTUeCP2.png)](https://asciinema.org/a/ZWkxvg6LUzedQ2IPzmFTUeCP2)

## Building hal
- `git clone` this project *outside* of your `$GOPATH` (since it uses `go modules`) or set `GO111MODULE=on` on your environment
- Build: `cd hal;make` with Go 1.11+ (currently only 1.12 is tested)
- Run: `./hal`, this will display the inline help
- Enjoy!

## Downloading a snapshot
- Go to https://circleci.com/gh/halkyonio/hal/tree/master and select the build number you are interested in (presumably, one 
that succeeded! 😁)
- Select the `Artifacts` tab and navigate the hierarchy to find the artifact you are interested in.

## Deploying a component using `hal`
After installing `hal`, the following steps allows to create and deploy a project to a cluster.
**Note**: this assumes that you are connected to a Halkyon-enabled OpenShift/Kubernetes cluster.

### 1. Scaffold the Spring Boot applications 

 - Create a development folder on your laptop
`mkdir haldemo && cd haldemo`

 - Create a new scaffolded component (note that it might make more sense to do this interactively):

```
hal component create \
    -r spring-boot \
    -i 2.1.6.RELEASE \
    -g me.example \
    -a hello-world \
    -v 1.0.0-SNAPSHOT \
    -p me.example.demo \
    -s true \
    -x true \
    -o 8080 \
    hello-world
```

### 2. Deploy the Component

A component represents a micro-service, i.e. part of an application to be deployed. The Component custom resource provides a simpler to fathom abstraction over what's actually required at the Kubernetes level to deploy and optionally expose the micro-service outside of the cluster. In fact, when a component is deployed to a [Halkyon](https://github.com/halkyonio)-enabled cluster, the [Halkyon operator](https://github.com/halkyonio/operator) will create these OpenShift/Kubernetes resources such as `Deployment`, `Service`, `PersistentVolumeClaim`, `Ingress` or `Route` on OpenShift if the component is exposed.

- Compile and generate the `halkyon` descriptors files of the application using the following command:
```
mvn package -f hello-world
```

- Push the hello-world component to the remote cluster you're connected to:
```
hal component push -c hello-world
```

- Check if the component has been correctly installed:
```
kubectl get components

NAME               RUNTIME       VERSION         AGE       MODE      STATUS    MESSAGE   REVISION
hello-world        spring-boot   2.1.6.RELEASE   7m17s     dev       Ready     Ready     6aadfc1a982fcd68
```

### 3. Connect to the REST services

If you deploy on OpenShift, get the route address of the microservice using this command: 
```
oc get routes/hello-world --template={{.spec.host}}
```

If you deploy on a plain Kubernetes, you can use this command:
```
kubectl get ingress/hello-world
```

Copy/paste the address displayed within the terminal in a browser and say Hello world 😉

## Additional documentation

Additional documentation can be found below:
- [CLI Reference](https://github.com/halkyonio/hal/blob/master/cli-reference.adoc)
