import http.client
import os
import socket
import subprocess
import time

from lib import BaseTest
from testout import TestOut

try:
    import requests
except ImportError:
    requests = None


class MetricsListenerAPITest(BaseTest):
    """
    Run the API and metrics-only listeners in one Aptly process.
    """

    aptly_server = None
    aptly_out = None
    debugOutput = True

    def fixture_available(self):
        return super().fixture_available() and requests is not None

    def prepare(self):
        super().prepare()

        config_path = os.path.join(os.environ["HOME"], self.aptlyConfigFile)
        last_output = ""

        for _ in range(3):
            with socket.socket() as api_socket, socket.socket() as metrics_socket:
                api_socket.bind(("127.0.0.1", 0))
                metrics_socket.bind(("127.0.0.1", 0))
                self.api_host, self.api_port = api_socket.getsockname()
                self.metrics_host, self.metrics_port = metrics_socket.getsockname()

            self.api_url = f"{self.api_host}:{self.api_port}"
            self.metrics_url = f"{self.metrics_host}:{self.metrics_port}"
            self.aptly_out = TestOut()
            self.aptly_server = self._start_process(
                f"aptly api serve -no-lock -config={config_path} "
                f"-listen={self.api_url} -metrics-listen={self.metrics_url}",
                stdout=self.aptly_out,
                stderr=self.aptly_out,
            )

            try:
                if self._wait_until_ready():
                    return
                last_output = self.aptly_out.get_contents()
            except Exception as error:
                self._stop_server()
                last_output = self.aptly_out.get_contents()
                self._close_output()
                raise RuntimeError(f"{error}:\n{last_output}") from error

            self._stop_server()
            self._close_output()

        raise RuntimeError(f"Aptly exited before its API listener became ready:\n{last_output}")

    def run(self):
        pass

    def teardown(self):
        self._stop_server()
        self._close_output()
        super().teardown()

    def debug_output(self):
        if self.aptly_out is None:
            return ""
        return self.aptly_out.get_contents()

    def _stop_server(self):
        if self.aptly_server is None:
            return
        self.aptly_server.terminate()
        try:
            self.aptly_server.wait(timeout=10)
        except subprocess.TimeoutExpired:
            self.aptly_server.kill()
            self.aptly_server.wait()
        self.aptly_server = None

    def _close_output(self):
        if self.aptly_out is not None:
            self.aptly_out.close()
            self.aptly_out = None

    def _wait_until_ready(self):
        deadline = time.monotonic() + 10
        while time.monotonic() < deadline:
            if self.aptly_server.poll() is not None:
                return False
            try:
                response = requests.get(f"http://{self.api_url}/api/version", timeout=0.2)
                if response.status_code == 200:
                    return True
            except requests.RequestException:
                pass
            time.sleep(0.05)
        raise RuntimeError("Aptly API listener did not become ready")

    def check(self):
        response = requests.get(f"http://{self.api_url}/api/metrics", timeout=5)
        self.check_equal(response.status_code, 200)
        self.check_in("# TYPE aptly_build_info gauge", response.text)

        response = requests.head(f"http://{self.api_url}/api/metrics", timeout=5)
        self.check_equal(response.status_code, 200)
        self.check_equal(response.content, b"")

        response = requests.get(
            f"http://{self.metrics_url}/metrics?_async=true&async=true",
            timeout=5,
        )
        self.check_equal(response.status_code, 200)
        self.check_in("# TYPE aptly_build_info gauge", response.text)
        self.check_in("# TYPE aptly_api_http_requests_total counter", response.text)

        response = requests.head(f"http://{self.metrics_url}/metrics", timeout=5)
        self.check_equal(response.status_code, 200)
        self.check_equal(response.content, b"")

        for path in ("/api/metrics", "/api/version", "/metrics/", "/METRICS"):
            response = requests.get(f"http://{self.metrics_url}{path}", timeout=5)
            self.check_equal(response.status_code, 404)

        connection = http.client.HTTPConnection(self.metrics_host, self.metrics_port, timeout=5)
        connection.request("GET", "/met%72ics")
        response = connection.getresponse()
        self.check_equal(response.status, 404)
        response.read()
        connection.close()

        response = requests.post(f"http://{self.metrics_url}/metrics", timeout=5)
        self.check_equal(response.status_code, 405)
        self.check_equal(response.headers.get("Allow"), "GET, HEAD")

        self.aptly_server.terminate()
        return_code = self.aptly_server.wait(timeout=10)
        self.check_equal(return_code, 0)
        self.aptly_server = None

        for url in (self.api_url, self.metrics_url):
            try:
                requests.get(f"http://{url}/metrics", timeout=0.2)
            except requests.RequestException:
                continue
            raise AssertionError(f"listener {url} still accepts requests after shutdown")
