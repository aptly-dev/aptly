from api_lib import APITest


class MetricsEnabledAPITest(APITest):
    """
    GET /metrics
    """

    def check(self):
        d = "libboost-program-options-dev_1.62.0.1"
        r = "foo"
        f = "libboost-program-options-dev_1.62.0.1_i386.deb"

        self.check_equal(self.upload("/api/files/" + d, f).status_code, 200)

        self.check_equal(self.post("/api/repos", json={
            "Name": r,
            "Comment": "test repo",
            "DefaultDistribution": r,
            "DefaultComponent": "main"
        }).status_code, 201)

        self.check_equal(self.post(f"/api/repos/{r}/file/{d}").status_code, 200)

        self.check_equal(self.post("/api/publish/filesystem:apiandserve:", json={
            "SourceKind": "local",
            "Sources": [
                {
                    "Component": "main",
                    "Name": r
                }
            ],
            "Distribution": r,
            "Signing":  {
                "Skip": True
            }
        }).status_code, 201)

        resp = self.get("/api/metrics")
        self.check_equal(resp.status_code, 200)

        apiRequestsInFlightGauge = "# TYPE aptly_api_http_requests_in_flight gauge"
        self.check_in(apiRequestsInFlightGauge, resp.text)

        apiRequestsTotalCounter = "# TYPE aptly_api_http_requests_total counter"
        self.check_in(apiRequestsTotalCounter, resp.text)

        apiRequestSizeSummary = "# TYPE aptly_api_http_request_size_bytes summary"
        self.check_in(apiRequestSizeSummary, resp.text)

        apiResponseSizeSummary = "# TYPE aptly_api_http_response_size_bytes summary"
        self.check_in(apiResponseSizeSummary, resp.text)

        apiRequestsDurationSummary = "# TYPE aptly_api_http_request_duration_seconds summary"
        self.check_in(apiRequestsDurationSummary, resp.text)

        apiBuildInfoGauge = "# TYPE aptly_build_info gauge"
        self.check_in(apiBuildInfoGauge, resp.text)

        apiFilesUploadedCounter = "# TYPE aptly_api_files_uploaded_total counter"
        self.check_in(apiFilesUploadedCounter, resp.text)

        apiFilesUploadedCounterValue = "aptly_api_files_uploaded_total{directory=\"libboost-program-options-dev_1.62.0.1\"} 1"
        self.check_in(apiFilesUploadedCounterValue, resp.text)

        apiReposPackageCountGauge = "# TYPE aptly_repos_package_count gauge"
        self.check_in(apiReposPackageCountGauge, resp.text)

        apiReposPackageCountGaugeValue = "aptly_repos_package_count{component=\"main\",distribution=\"foo\",source=\"[foo:main]\"} 1"
        self.check_in(apiReposPackageCountGaugeValue, resp.text)
