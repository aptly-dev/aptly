from api_lib import APITest


class MetricsEnabledAPITest(APITest):
    """
    GET /metrics
    """

    def check(self):
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
