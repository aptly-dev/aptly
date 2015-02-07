export OS_AUTH_URL=http://127.0.0.1:8181/v2.0/
export OS_USERNAME=user_test
export OS_PASSWORD=tester
export OS_TENANT_NAME=testing
pip install python-keystoneclient python-swiftclient
docker run -d -p 8080:8080 serverascode/swift-onlyone