# hal

## Overview
`hal` is a CLI tool for developers to create and manage Spring Boot applications. Simplify the deployment of applications on OpenShift and Kubernetes using Dekorate and Halkyon Component Operator. Made with ❤️ by the Snowdrop team.

[![CircleCI](https://circleci.com/gh/halkyonio/hal.svg?style=svg)](https://circleci.com/gh/halkyonio/hal)

## Key features
`hal` is of doing the following key features
- Scaffold Spring Boot applications
- Deploy Spring Boot applications as components
- Switch the component between `dev` and `build` modes
- Compose & link microservices

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
After installing `hal`, the following steps allows you to create and deploy a project to a cluster.
**Note**: this assumes that you are connected to a OpenShift/Kubernetes cluster.

###1 Scaffold the Spring Boot applications 

 - Create a development folder on your laptop
`mkdir haldemo && cd haldemo`

Create a new project using the REST HTTP `rest` template proposed by the scaffolding tool:

```
hal component spring-boot \
    -i hello-world \
    -g me.example \
    -p me.example.demo \
    -s 2.1.6.RELEASE \
    -t rest \
    -v 1.0.0-SNAPSHOT \
    --supported=false \
    hello-world
```

###2 Deploy the Component

A component represents a micro-service, i.e. part of an application to be deployed. The Component custom resource provides a simpler to fathom abstraction over what's actually required at the Kubernetes level to deploy and optionally expose the micro-service outside of the cluster. In fact, when a component is deployed to a [Halkyon](https://github.com/halkyonio)-enabled cluster, the [Halkyon operator](https://github.com/halkyonio/operator) will create these OpenShift/Kubernetes resources.

`hal component push -c hello-world`

Check if the components have been correctly installed:
`oc get components`

```
NAME               RUNTIME       VERSION         AGE       MODE      STATUS    MESSAGE   REVISION
hello-world        spring-boot   2.1.6.RELEASE   7m17s     dev       Ready     Ready     6aadfc1a982fcd68
```

###3 Connect to the rest services

Try the rest service to see if it works. To do so, get the route address of the microservice using this command `oc get routes/hello-world --template={{.spec.host}}`
Copy/paste the address displayed within the terminal in a browser and say Hello world ;-)

## Demonstration

The following demonstration provides an overview of `hal`:

TO DO : put here the recording embedded

## Additional documentation

Additional documentation can be found below:
- [CLI Reference](https://github.com/halkyonio/hal/blob/master/cli-reference.adoc)


