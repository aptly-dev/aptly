export ST_AUTH=http://127.0.0.1:8181/auth/v1.0
export ST_USER=test:tester
ID=`docker run -d -p 8080:8080 serverascode/swift-onlyone`
sleep 10 # Give the script that change the passwords some time
export ST_KEY=`docker logs $ID | grep "user_test_tester =" | cut -d " " -f 3`
pip install python-keystoneclient python-swiftclient
