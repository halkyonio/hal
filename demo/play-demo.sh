#!/bin/bash

#asciinema rec -w 2 &

exec() {
  echo "\$ $@"|pv -qL 20 ; "$@" ;
  }

echo "   "
echo "   "
echo "  Simplify the deployment of Spring Boot applications using Halkyon Component Operator on Kubernetes"|pv -qL 30
echo "  In this demo, we will :"|pv -qL 20
echo "   --> Compose & link microservices"|pv -qL 20
echo "   --> Deploy a capability such as a database and link it to a microservice consuming it"|pv -qL 20
echo "   --> Code locally and next push/build on k8s/OpenShift"|pv -qL 20
echo "  "|pv -qL 20
echo "  Ready? Let's start!"|pv -qL 20
echo "  "
echo "  "
sleep 3

clear && sleep 1
echo "# Log on to the cluster using the oc client"|pv -qL 20
#exec oc login https://api.cluster-b1c8.b1c8.sandbox941.opentlc.com:6443 -u user1 -p r3dh4t1!
exec oc login https://api.cluster-6c2c.6c2c.sandbox1254.opentlc.com:6443 -u user1 -p r3dh4t1!
sleep 3

clear && sleep 1
echo "# Create a new project"|pv -qL 20
exec oc new-project demo
sleep 3

clear && sleep 1
echo "# Create a directory named demo and subsequently enter in"|pv -qL 20
exec mkdir demo
exec cd demo
sleep 3

clear && sleep 1
echo "# Create a pom.xml with the following content"|pv -qL 20
sleep 1
echo "<?xml version="1.0" encoding="UTF-8"?>
<!--
Copyright 2016-2017 Red Hat, Inc, and individual contributors.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

 http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
-->
<project xmlns="http://maven.apache.org/POM/4.0.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 http://maven.apache.org/xsd/maven-4.0.0.xsd">
    <modelVersion>4.0.0</modelVersion>
    <groupId>me.fruitsand</groupId>
    <artifactId>parent</artifactId>
    <version>1.0.0-SNAPSHOT</version>
    <name>Spring Boot - Demo</name>
    <description>Spring Boot - Demo</description>
    <packaging>pom</packaging>
    <modules>
        <module>fruit-backend-sb</module>
        <module>fruit-client-sb</module>
    </modules>
</project>"
sleep 3
cat > pom.xml << ENDOFFILE
<?xml version="1.0" encoding="UTF-8"?>
<!--
Copyright 2016-2017 Red Hat, Inc, and individual contributors.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

 http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
-->
<project xmlns="http://maven.apache.org/POM/4.0.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 http://maven.apache.org/xsd/maven-4.0.0.xsd">
    <modelVersion>4.0.0</modelVersion>
    <groupId>me.fruitsand</groupId>
    <artifactId>parent</artifactId>
    <version>1.0.0-SNAPSHOT</version>
    <name>Spring Boot - Demo</name>
    <description>Spring Boot - Demo</description>
    <packaging>pom</packaging>
    <modules>
        <module>fruit-backend-sb</module>
        <module>fruit-client-sb</module>
    </modules>
</project>
ENDOFFILE
clear && sleep 1
echo "# Create a new client project using the REST HTTP client template proposed by the scaffolding tool"|pv -qL 20
exec hal component spring-boot -i fruit-client-sb -g me.fruitstand -p me.fruitstand.demo -s 2.1.6.RELEASE -t client -v 1.0.0-SNAPSHOT --supported=false  fruit-client-sb
sleep 3

clear && sleep 1
echo "# Create a backend project interactively and use as template the crud type and fruit-backend-sb as maven project name"|pv -qL 20
exec hal component spring-boot  fruit-backend-sb
sleep 3

clear && sleep 1
echo "# Build the projects"|pv -qL 20
echo "# Compile and generate the uber jar of the Spring Boot application client"|pv -qL 20
exec mvn package -f fruit-client-sb
sleep 3

clear && sleep 1
echo "# Repeat the command executed previously for the CRUD - backend microservice."|pv -qL 20
echo "# We need to use the kubernetes profile because the project is set up to work both locally using H2 database for quick testing and "remotely" using a PostgreSQL database."|pv -qL 20
exec mvn package -f fruit-backend-sb -Pkubernetes
sleep 3

clear && sleep 1
echo "# Deploy the applications as components"|pv -qL 20
exec hal component push -c fruit-client-sb,fruit-backend-sb
sleep 3

clear && sleep 1
echo "# Check if the components have been correctly installed"|pv -qL 20
exec oc get cp
sleep 3

clear && sleep 1
echo "# Create a capability to install a PostgreSQL database using the interactive mode"|pv -qL 20
exec hal capability create
sleep 3

clear && sleep 1
echo "# Check the capability status"|pv -qL 20
exec oc get capabilities
sleep 3

clear && sleep 1
echo "# Link the microservices"|pv -qL 20
echo "# We need to wire the fruit-backend-sb component with the postgres-db capability by creating a link between both"|pv -qL 20
exec hal link create
sleep 3

clear && sleep 1
echo "# Now, create a link targeting the fruit-client-sb component to wire the client and backend"|pv -qL 20
exec hal link create
sleep 3

clear && sleep 1
echo "# Check the link status"|pv -qL 20
exec oc get links
sleep 3

clear && sleep 1
echo "# Try the backend service to see if it works"|pv -qL 20
echo "# So, get the route address of the backend microservice using this command "|pv -qL 20
exec oc get routes/fruit-backend-sb --template={{.spec.host}}
echo " "
sleep 2
echo "# Copy/paste the address displayed within the terminal in a browser and create some fruits ;-)"|pv -qL 20
sleep 10

clear && sleep 1
echo "# Try the client microservice to see if it works too"|pv -qL 20
echo "# So, get also its route address "|pv -qL 20
exec oc get routes/fruit-client-sb --template={{.spec.host}}
sleep 3

clear && sleep 1
echo "# curl the service within your terminal, you should get the fruits created in the previous step."|pv -qL 20
exec curl "http://$(oc get routes/fruit-client-sb --template={{.spec.host}})/api/client"
