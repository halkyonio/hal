#!/bin/bash

#asciinema rec -w 2

exec() {
  echo "\$ $@"|pv -qL 10 ; "$@" ;
  }

echo "   "
echo "   "
echo " Simplify the deployment of Spring Boot applications using Halkyon operator on Kubernetes"|pv -qL 10
echo " In this demo, we will :"|pv -qL 10
echo "   --> Compose & link 2 microservices: client and fruits backend"|pv -qL 10
echo "   --> Deploy a capability such as a database and link it to the backend microservice accessing it"|pv -qL 10
echo "   --> Code locally and next push/build on Kubernetes/OpenShift"|pv -qL 10
echo "  "|pv -qL 10
echo " Ready? Let's begin!"|pv -qL 10
echo "  "
echo "  "
sleep 5

clear && sleep 1
echo "# Log on to the cluster using the oc client"|pv -qL 10
sleep 2
oc login https://159.69.209.188:8443 --token=k8EWmKzrAUIcxn4ufuzYYK6O4w7sRJmycyh0kMU24Mg
sleep 3

clear && sleep 1
echo "# Create a new project"|pv -qL 10
sleep 1
exec oc new-project demo
sleep 7

clear && sleep 1
echo "# Create a directory named demo and subsequently navigate to it"|pv -qL 10
sleep 2
exec mkdir demo
sleep 1
exec cd demo
sleep 3

clear && sleep 1
echo "# Create a pom.xml with 2 modules corresponding each to a microservice"|pv -qL 10
sleep 2
echo "<?xml version="1.0" encoding="UTF-8"?>
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
sleep 10
cat > pom.xml << ENDOFFILE
<?xml version="1.0" encoding="UTF-8"?>
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
echo "# Create a client microservice using the REST HTTP client template proposed by the scaffolding command"|pv -qL 10
sleep 3
exec hal component spring-boot \
   -i fruit-client-sb \
   -g me.fruitstand \
   -p me.fruitstand.demo \
   -s 2.1.6.RELEASE \
   -t client \
   -v 1.0.0-SNAPSHOT \
   --supported=false  \
   fruit-client-sb
sleep 3

clear && sleep 1
echo "# Create a backend microservice interactively using now the CRUD template"|pv -qL 10
sleep 3
exec hal component spring-boot fruit-backend-sb
sleep 3

clear && sleep 1
echo "# Build the Spring Boot microservices"|pv -qL 10
echo "# Compile and generate the Spring Boot uber jar file for the REST client microservice"|pv -qL 10
sleep 3
exec mvn package -f fruit-client-sb
sleep 3

clear && sleep 1
echo "# Repeat the command executed previously for the CRUD backend microservice."|pv -qL 10
echo "# We need to use the kubernetes profile because the project is set up to work both locally using H2 database for quick testing and \"remotely\" using a PostgreSQL database."|pv -qL 10
sleep 4
exec mvn package -f fruit-backend-sb -Pkubernetes
sleep 3

clear && sleep 1
echo "# Create a component for each microservice and deploy it on the cluster"|pv -qL 10
sleep 2
exec hal component create -c fruit-client-sb
sleep 2
exec hal component create -c fruit-backend-sb
sleep 3

clear && sleep 1
echo "# Check that the components have been correctly installed"|pv -qL 10
sleep 2
exec oc get cp
sleep 3

clear && sleep 1
echo "# Create a capability to install a PostgreSQL database using the interactive mode"|pv -qL 10
sleep 2
exec hal capability create -n postgres-db
sleep 3

clear && sleep 1
echo "# Check the capability status"|pv -qL 10
sleep 2
exec oc get capabilities
sleep 10

clear && sleep 1
echo "# Link the microservices"|pv -qL 10
echo "# We need to wire the fruit-backend-sb component with the postgres-db capability by creating a link between them"|pv -qL 10
sleep 4
exec hal link create -n backend-to-db -t fruit-backend-sb
sleep 3

clear && sleep 1
echo "# Now, create a link targeting the fruit-client-sb component to wire the client and the backend microservices"|pv -qL 10
sleep 3
exec hal link create -n client-to-backend -t fruit-client-sb -e KUBERNETES_ENDPOINT_FRUIT=http://fruit-backend-sb:8080/api/fruits
sleep 3

clear && sleep 1
echo "# Check the links status"|pv -qL 10
sleep 2
exec oc get links
sleep 3

clear && sleep 1
echo "# Push your local code source (e.g pom.xml, ./src,...) to the pod"|pv -qL 10
sleep 2
exec hal component push -c fruit-client-sb,fruit-backend-sb
echo "# Let's wait a few seconds to let maven build the Spring Boot application on the cluster. Halkyon will then automatically start the app"|pv -qL 10
#Wait some seconds for pods readies
i=0
while [ $i -le 15 ]
do
    printf '. '
    i=$(( $i + 1 ))
    sleep 1
done


clear && sleep 1
echo "# Let's try to access the backend service and see if we can create some fruits!"|pv -qL 10
echo "# Obtain the route address of the backend microservice using this command "|pv -qL 10
sleep 4
exec oc get routes/fruit-backend-sb --template={{.spec.host}}
echo " "
sleep 2
echo "# Let's create some fruits on the backend using the HTTPie - https://httpie.org tool"|pv -qL 10
BACKEND_URL=$(oc get routes/fruit-backend-sb --template={{.spec.host}})
exec http -s solarized POST "http://${BACKEND_URL}/api/fruits" name=Orange
sleep 1
exec http -s solarized POST "http://${BACKEND_URL}/api/fruits" name=Banana
sleep 1
exec http -s solarized POST "http://${BACKEND_URL}/api/fruits" name=Pineapple
sleep 10

clear && sleep 1
echo "# Let's now check our client microservice"|pv -qL 10
echo "# Obtain its route similarly to what we did for the backend microservice "|pv -qL 10
sleep 4
exec oc get routes/fruit-client-sb --template={{.spec.host}}
sleep 3

clear && sleep 1
echo "# Call the client service within your terminal, you should get the fruits created in the previous step."|pv -qL 10
exec http "http://$(oc get routes/fruit-client-sb --template={{.spec.host}})/api/client"
sleep 2
echo " "
echo " "
echo "# So, this is hal!! "|pv -qL 10
echo "# Thank you :-) "|pv -qL 10
sleep 5


# clean up
#oc delete project demo
#cd .. && rm -rf demo
