oc new-project test

cd ~/Temp
echo "########################"
echo "#### Create Test project at this path: $(pwd)"
echo "########################"
mkdir test && cd test

echo "########################"
echo "#### Create Pom parent file"
echo "########################"
cat <<EOF > pom.xml
<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 http://maven.apache.org/xsd/maven-4.0.0.xsd">
    <modelVersion>4.0.0</modelVersion>
    <groupId>me.fruitstand</groupId>
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
EOF

echo "########################"
echo "#### Scaffold projects"
echo "########################"

hal component spring-boot \
   -i fruit-backend-sb \
   -g me.fruitsand \
   -p me.fruitsand.demo \
   -s 2.1.6.RELEASE \
   -t crud \
   -v 1.0.0-SNAPSHOT \
   --supported=false  \
   fruit-backend-sb

 hal component spring-boot \
   -i fruit-client-sb \
   -g me.fruitsand \
   -p me.fruitsand.demo \
   -s 2.1.6.RELEASE \
   -t client \
   -v 1.0.0-SNAPSHOT \
   --supported=false  \
   fruit-client-sb

echo "########################"
echo "#### Maven package"
echo "########################"
mvn package -f fruit-client-sb
mvn package -f fruit-backend-sb -Pkubernetes

echo "########################"
echo "#### Create component"
echo "########################"
hal component create -c fruit-client-sb
hal component create -c fruit-backend-sb

echo "########################"
echo "#### Create Capability"
echo "########################"
hal capability create -n postgres-db -g database -t postgres -v 10 -p DB_NAME=sample-db -p DB_PASSWORD=admin -p DB_USER=admin
sleep 15s

echo "########################"
echo "#### Link"
echo "########################"
hal link create -n backend-to-db -t fruit-backend-sb -s postgres-db-config
hal link create -n client-to-backend -t fruit-client-sb -e KUBERNETES_ENDPOINT_FRUIT=http://fruit-backend-sb:8080/api/fruits

echo "########################"
echo "#### Push"
echo "########################"
sleep 60s

hal component push -c fruit-client-sb
# PROJECT=fruit-client-sb
# NAMESPACE=test
# POD_ID=$(oc get pod -lapp=$PROJECT -n $NAMESPACE -o name | awk -F '/' '{print $2}')
# oc cp $PROJECT/pom.xml $POD_ID:/usr/src/ -n $NAMESPACE
# oc cp $PROJECT/src $POD_ID:/usr/src/ -n $NAMESPACE
# oc exec $POD_ID -n $NAMESPACE /var/lib/supervisord/bin/supervisord ctl start build
# oc exec $POD_ID -n $NAMESPACE /var/lib/supervisord/bin/supervisord ctl start run

hal component push -c fruit-backend-sb
# PROJECT=fruit-backend-sb
# NAMESPACE=test
# POD_ID=$(oc get pod -lapp=$PROJECT -n $NAMESPACE -o name | awk -F '/' '{print $2}')
# oc cp $PROJECT/pom.xml $POD_ID:/usr/src/ -n $NAMESPACE
# oc cp $PROJECT/src $POD_ID:/usr/src/ -n $NAMESPACE
# oc exec $POD_ID -n $NAMESPACE /var/lib/supervisord/bin/supervisord ctl start build
# oc exec $POD_ID -n $NAMESPACE /var/lib/supervisord/bin/supervisord ctl start run

echo "########################"
echo "#### Wait and call endpoint"
echo "########################"
sleep 120s
BACKEND_URL=$(oc get routes/fruit-backend-sb --template={{.spec.host}})
http -s solarized POST "http://${BACKEND_URL}/api/fruits" name=Orange
http -s solarized POST "http://${BACKEND_URL}/api/fruits" name=Banana
http -s solarized POST "http://${BACKEND_URL}/api/fruits" name=Pineapple
http -s solarized POST "http://${BACKEND_URL}/api/fruits" name=Apple
http -s solarized POST "http://${BACKEND_URL}/api/fruits" name=Pear

FRONTEND_URL=$(oc get routes/fruit-client-sb --template={{.spec.host}})
http "http://${FRONTEND_URL}/api/client" -s solarized

echo "########################"
echo "#### Delete"
echo "########################"
oc delete all --all -n test
cd .. && rm -rf test